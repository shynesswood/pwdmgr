package service_test

import (
	"testing"

	"pwdmgr/internal/service"
	"pwdmgr/internal/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// B1 — 全新目录 + 本地 vault → 推送
// ---------------------------------------------------------------------------

func TestB1_BindRemoteRepo_NewDirLocalVaultPush(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)
	dir := freshDir(t)

	require.NoError(t, service.InitLocalVault(dir, testPassword))
	require.NoError(t, service.AddEntry(dir, testPassword, "Site", "u", "p", "", nil))

	err := service.BindRemoteRepo(dir, remote, testPassword)
	require.NoError(t, err)

	// 验证远程仓库已有 vault.dat
	verify := loadClonedVault(t, remote, testPassword)
	assert.Len(t, verify.Entries, 1)
	assert.Equal(t, "Site", verify.Entries[0].Name)
}

// ---------------------------------------------------------------------------
// B2 — 已有 git repo + 本地 vault → 推送
// ---------------------------------------------------------------------------

func TestB2_BindRemoteRepo_ExistingGitLocalVaultPush(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)
	dir := freshDir(t)

	gitExec(t, dir, "init")
	gitCfg(t, dir)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	err := service.BindRemoteRepo(dir, remote, testPassword)
	require.NoError(t, err)

	verify := loadClonedVault(t, remote, testPassword)
	assert.NotNil(t, verify)
}

// ---------------------------------------------------------------------------
// B3 — 全新空目录拉取远程
// ---------------------------------------------------------------------------

func TestB3_BindRemoteRepo_EmptyLocalPullFromRemote(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	// 先从"另一台设备"推送数据到远程
	v := vault.NewVault()
	v.AddEntry(vault.Entry{ID: "remote-e1", Name: "RemoteSite", Username: "u", Password: "p", UpdatedAt: 100})
	pushVaultToRemote(t, remote, testPassword, v)

	// 新的空目录执行 BindRemoteRepo
	dir := freshDir(t)
	err := service.BindRemoteRepo(dir, remote, testPassword)
	require.NoError(t, err)

	assert.True(t, vaultFileExists(dir))

	entries, err := service.ListEntries(dir, testPassword)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "RemoteSite", entries[0].Name)
}

// ---------------------------------------------------------------------------
// B4 — 已有 git repo + 无 vault 拉取
// ---------------------------------------------------------------------------

func TestB4_BindRemoteRepo_ExistingGitNoVaultPull(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	v := vault.NewVault()
	v.AddEntry(vault.Entry{ID: "r1", Name: "R", UpdatedAt: 1})
	pushVaultToRemote(t, remote, testPassword, v)

	dir := freshDir(t)
	gitExec(t, dir, "init")
	gitCfg(t, dir)

	err := service.BindRemoteRepo(dir, remote, testPassword)
	require.NoError(t, err)

	assert.True(t, vaultFileExists(dir))
}

// ---------------------------------------------------------------------------
// B5 — 双端相同条目
// ---------------------------------------------------------------------------

func TestB5_BindRemoteRepo_BothHaveSameEntries(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	entry := vault.Entry{ID: "shared", Name: "Shared", Password: "same", UpdatedAt: 100}

	remoteVault := vault.NewVault()
	remoteVault.AddEntry(entry)
	pushVaultToRemote(t, remote, testPassword, remoteVault)

	dir := freshDir(t)
	localVault := vault.NewVault()
	localVault.AddEntry(entry)
	saveLocalVault(t, dir, testPassword, localVault)

	err := service.BindRemoteRepo(dir, remote, testPassword)
	require.NoError(t, err)

	result := loadLocalVault(t, dir, testPassword)
	assert.Len(t, result.Entries, 1)
	assert.Equal(t, "shared", result.Entries[0].ID)
}

// ---------------------------------------------------------------------------
// B6 — 双端不同条目
// ---------------------------------------------------------------------------

func TestB6_BindRemoteRepo_BothHaveDifferentEntries(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	remoteVault := vault.NewVault()
	remoteVault.AddEntry(vault.Entry{ID: "a", Name: "A", UpdatedAt: 100})
	pushVaultToRemote(t, remote, testPassword, remoteVault)

	dir := freshDir(t)
	localVault := vault.NewVault()
	localVault.AddEntry(vault.Entry{ID: "b", Name: "B", UpdatedAt: 100})
	saveLocalVault(t, dir, testPassword, localVault)

	err := service.BindRemoteRepo(dir, remote, testPassword)
	require.NoError(t, err)

	result := loadLocalVault(t, dir, testPassword)
	assert.Len(t, result.Entries, 2)

	ids := map[string]bool{}
	for _, e := range result.Entries {
		ids[e.ID] = true
	}
	assert.True(t, ids["a"], "should contain remote entry A")
	assert.True(t, ids["b"], "should contain local entry B")
}

// ---------------------------------------------------------------------------
// B7 — 双端同 ID 不同时间戳
// ---------------------------------------------------------------------------

func TestB7_BindRemoteRepo_SameIDNewerTimestampWins(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	remoteVault := vault.NewVault()
	remoteVault.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "remote-old", UpdatedAt: 100})
	pushVaultToRemote(t, remote, testPassword, remoteVault)

	dir := freshDir(t)
	localVault := vault.NewVault()
	localVault.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "local-new", UpdatedAt: 200})
	saveLocalVault(t, dir, testPassword, localVault)

	err := service.BindRemoteRepo(dir, remote, testPassword)
	require.NoError(t, err)

	result := loadLocalVault(t, dir, testPassword)
	assert.Len(t, result.Entries, 1)
	assert.Equal(t, "local-new", result.Entries[0].Password,
		"entry with newer UpdatedAt should win")
}

// ---------------------------------------------------------------------------
// B8 — 双空初始化
// ---------------------------------------------------------------------------

func TestB8_BindRemoteRepo_BothEmpty(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)
	dir := freshDir(t)

	err := service.BindRemoteRepo(dir, remote, testPassword)
	require.NoError(t, err)

	assert.True(t, vaultFileExists(dir))

	result := loadLocalVault(t, dir, testPassword)
	assert.NotNil(t, result.Entries)
	assert.Empty(t, result.Entries, "both empty should result in empty entries")
}

// ---------------------------------------------------------------------------
// helper: clone remote 并读取 vault
// ---------------------------------------------------------------------------

func loadClonedVault(t *testing.T, remote string, password []byte) *vault.Vault {
	t.Helper()
	clone := t.TempDir()
	gitExec(t, "", "clone", remote, clone)
	return loadLocalVault(t, clone, password)
}
