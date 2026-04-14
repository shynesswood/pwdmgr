package service_test

import (
	"path/filepath"
	"testing"

	"pwdmgr/internal/service"
	"pwdmgr/internal/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// X1 — 密码错误：解密失败且 vault 文件不被损坏
// ---------------------------------------------------------------------------

func TestX1_WrongPassword(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	pwd := []byte("correct-password")
	wrongPwd := []byte("wrong-password")

	require.NoError(t, service.InitLocalVault(dir, pwd))
	require.NoError(t, service.AddEntry(dir, pwd, "Site", "u", "p", "", nil))

	_, err := service.ListEntries(dir, wrongPwd)
	assert.Error(t, err, "wrong password should fail decryption")

	entries, err := service.ListEntries(dir, pwd)
	require.NoError(t, err, "vault should not be corrupted by wrong password attempt")
	assert.Len(t, entries, 1)
}

// ---------------------------------------------------------------------------
// X2 — 远程 URL 不可达
// ---------------------------------------------------------------------------

func TestX2_UnreachableRemoteURL(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	err := service.BindRemoteRepo(dir, "/nonexistent/path/to/repo.git", testPassword)
	require.Error(t, err)
	// git 的错误信息应该包含有用信息，不应只是 "exit status 128"
	assert.NotEqual(t, "exit status 128", err.Error(),
		"error message should be readable")
}

// ---------------------------------------------------------------------------
// X3 — repo_root 目录不存在
// ---------------------------------------------------------------------------

func TestX3_NonExistentRepoRoot(t *testing.T) {
	requireGit(t)
	badPath := filepath.Join(t.TempDir(), "this", "does", "not", "exist")

	err := service.InitLocalVault(badPath, testPassword)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// X4 — 空密码
// ---------------------------------------------------------------------------

func TestX4_EmptyPassword(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	emptyPwd := []byte("")

	require.NoError(t, service.InitLocalVault(dir, emptyPwd))
	require.NoError(t, service.AddEntry(dir, emptyPwd, "EmptyPwdSite", "u", "p", "", nil))

	entries, err := service.ListEntries(dir, emptyPwd)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "EmptyPwdSite", entries[0].Name)
}

// ---------------------------------------------------------------------------
// X5 — 重复 BindRemoteRepo
// ---------------------------------------------------------------------------

func TestX5_DuplicateBindRemoteRepo(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)
	dir := freshDir(t)

	require.NoError(t, service.InitLocalVault(dir, testPassword))
	require.NoError(t, service.BindRemoteRepo(dir, remote, testPassword))

	// 再次绑定同一远程
	err := service.BindRemoteRepo(dir, remote, testPassword)
	assert.NoError(t, err, "duplicate BindRemoteRepo should not error (AddRemote tolerant)")
}

// ---------------------------------------------------------------------------
// X6 — BindRemoteRepo 空参数
// ---------------------------------------------------------------------------

func TestX6_BindRemoteRepo_EmptyParams(t *testing.T) {
	err := service.BindRemoteRepo("", "https://example.com/repo.git", testPassword)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "仓库路径不能为空")

	err = service.BindRemoteRepo(t.TempDir(), "", testPassword)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "远程仓库地址不能为空")
}

// ---------------------------------------------------------------------------
// X7 — Pull/Push 空路径
// ---------------------------------------------------------------------------

func TestX7_PullPush_EmptyPath(t *testing.T) {
	err := service.PullVault("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "仓库路径不能为空")

	err = service.PushVault("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "仓库路径不能为空")
}

// ---------------------------------------------------------------------------
// X — SyncVault 用错误密码
// ---------------------------------------------------------------------------

func TestX_SyncVault_WrongPassword(t *testing.T) {
	requireGit(t)
	local, _ := setupBoundRepo(t)
	require.NoError(t, service.AddEntry(local, testPassword, "S", "u", "p", "", nil))

	err := service.SyncVault(local, []byte("wrong"))
	assert.Error(t, err, "sync with wrong password should fail on LoadVault")
}

// ---------------------------------------------------------------------------
// X — BindRemoteRepo 双端合并时密码不匹配
// ---------------------------------------------------------------------------

func TestX_BindRemoteRepo_PasswordMismatch(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	// 远程用 pwdA 加密
	pwdA := []byte("password-A")
	rv := vault.NewVault()
	rv.AddEntry(vault.Entry{ID: "r1", Name: "Remote", UpdatedAt: 1})
	pushVaultToRemote(t, remote, pwdA, rv)

	// 本地用 pwdB 加密
	pwdB := []byte("password-B")
	dir := freshDir(t)
	lv := vault.NewVault()
	lv.AddEntry(vault.Entry{ID: "l1", Name: "Local", UpdatedAt: 1})
	saveLocalVault(t, dir, pwdB, lv)

	// Bind 时传 pwdB：能解密本地 vault，但 pull 后无法用 pwdB 解密远程 vault
	err := service.BindRemoteRepo(dir, remote, pwdB)
	assert.Error(t, err, "should fail when remote vault was encrypted with different password")
}
