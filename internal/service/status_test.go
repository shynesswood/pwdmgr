package service_test

import (
	"testing"

	"pwdmgr/internal/service"
	"pwdmgr/internal/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// R1 — 空目录状态（非 git repo）
// ---------------------------------------------------------------------------

func TestR1_RepoStatus_EmptyDir(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)

	status, err := service.GetRepoStatus(dir)
	require.NoError(t, err)

	assert.False(t, status.IsGitRepo)
	assert.False(t, status.HasRemote)
	assert.False(t, status.HasLocalVault)
	assert.False(t, status.HasUncommitted)
	assert.Empty(t, status.CurrentBranch)
	assert.Empty(t, status.RemoteURL)
}

// ---------------------------------------------------------------------------
// R2 — 已初始化 + 已绑定远程
// ---------------------------------------------------------------------------

func TestR2_RepoStatus_InitializedWithRemote(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	remote := initBareRemote(t)

	require.NoError(t, service.InitLocalVault(dir, testPassword))
	require.NoError(t, service.BindRemoteRepo(dir, remote, testPassword))

	status, err := service.GetRepoStatus(dir)
	require.NoError(t, err)

	assert.True(t, status.IsGitRepo)
	assert.True(t, status.HasRemote)
	assert.True(t, status.HasLocalVault)
	assert.False(t, status.HasUncommitted, "InitLocalVault auto-commits")
	assert.NotEmpty(t, status.CurrentBranch)
	assert.Equal(t, remote, status.RemoteURL)
}

// ---------------------------------------------------------------------------
// R3 — 有未提交变更
// ---------------------------------------------------------------------------

func TestR3_RepoStatus_HasUncommitted(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	remote := initBareRemote(t)

	require.NoError(t, service.InitLocalVault(dir, testPassword))
	require.NoError(t, service.BindRemoteRepo(dir, remote, testPassword))

	require.NoError(t, service.AddEntry(dir, testPassword, "NewSite", "user", "pass", "", nil))

	status, err := service.GetRepoStatus(dir)
	require.NoError(t, err)
	assert.True(t, status.HasUncommitted)
}

// ---------------------------------------------------------------------------
// R4 — 远程有数据
// ---------------------------------------------------------------------------

func TestR4_RepoStatus_RemoteHasData(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	v := vault.NewVault()
	v.AddEntry(vault.Entry{ID: "e1", Name: "X", UpdatedAt: 1})
	pushVaultToRemote(t, remote, testPassword, v)

	dir := freshDir(t)
	gitExec(t, dir, "init")
	gitCfg(t, dir)
	gitExec(t, dir, "remote", "add", "origin", remote)

	status, err := service.GetRepoStatus(dir)
	require.NoError(t, err)
	assert.True(t, status.RemoteHasData)
}
