package service_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pwdmgr/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// I1 — 正常创建：空目录 → InitLocalVault → .git + vault.dat + 自动提交
// ---------------------------------------------------------------------------

func TestI1_InitLocalVault_Normal(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)

	err := service.InitLocalVault(dir, testPassword)
	require.NoError(t, err)

	assert.DirExists(t, filepath.Join(dir, ".git"))
	assert.FileExists(t, filepath.Join(dir, "vault.dat"))

	v := loadLocalVault(t, dir, testPassword)
	assert.NotNil(t, v.Entries)
	assert.Empty(t, v.Entries, "new vault should have no entries")

	out := gitExec(t, dir, "log", "--oneline")
	assert.Contains(t, out, "init vault")
}

// ---------------------------------------------------------------------------
// I2 — 已有 vault 报错
// ---------------------------------------------------------------------------

func TestI2_InitLocalVault_AlreadyExists(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)

	require.NoError(t, service.InitLocalVault(dir, testPassword))

	err := service.InitLocalVault(dir, testPassword)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "本地 vault 已存在")
}

// ---------------------------------------------------------------------------
// I3 — 已有 git repo：跳过 git init，仅创建 vault 并提交
// ---------------------------------------------------------------------------

func TestI3_InitLocalVault_ExistingGitRepo(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)

	gitExec(t, dir, "init")
	gitCfg(t, dir)

	err := service.InitLocalVault(dir, testPassword)
	require.NoError(t, err)

	assert.FileExists(t, filepath.Join(dir, "vault.dat"))

	out := gitExec(t, dir, "log", "--oneline")
	assert.Contains(t, out, "init vault")

	status := gitExec(t, dir, "status", "--porcelain")
	assert.Empty(t, strings.TrimSpace(status), "vault.dat should be committed")
}

// ---------------------------------------------------------------------------
// I4 — 空路径校验
// ---------------------------------------------------------------------------

func TestI4_InitLocalVault_EmptyPath(t *testing.T) {
	err := service.InitLocalVault("", testPassword)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "仓库路径不能为空")
}

// ---------------------------------------------------------------------------
// I — 目录不存在时 git init 应失败
// ---------------------------------------------------------------------------

func TestI_InitLocalVault_NonExistentDir(t *testing.T) {
	requireGit(t)
	dir := filepath.Join(t.TempDir(), "does", "not", "exist")

	err := service.InitLocalVault(dir, testPassword)
	assert.Error(t, err)

	_, statErr := os.Stat(filepath.Join(dir, "vault.dat"))
	assert.True(t, os.IsNotExist(statErr))
}
