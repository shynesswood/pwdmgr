package service_test

import (
	"path/filepath"
	"testing"

	"pwdmgr/internal/service"
	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupBoundRepo 创建一个已绑定远程的本地仓库（InitLocalVault + BindRemoteRepo 完成）。
func setupBoundRepo(t *testing.T) (local string, remote string) {
	t.Helper()
	remote = initBareRemote(t)
	local = freshDir(t)
	require.NoError(t, service.InitLocalVault(local, testPassword))
	require.NoError(t, service.BindRemoteRepo(local, remote, testPassword))
	return
}

// ---------------------------------------------------------------------------
// S1 — 无本地变更，纯 pull
// ---------------------------------------------------------------------------

func TestS1_SyncVault_NoLocalChanges_PurePull(t *testing.T) {
	requireGit(t)
	local, remote := setupBoundRepo(t)

	// "另一台设备"向远程推送新条目
	updateRemoteVault(t, remote, testPassword, func(v *vault.Vault) {
		v.AddEntry(vault.Entry{ID: "from-other", Name: "OtherDevice", Password: "pwd", UpdatedAt: 500})
	})

	err := service.SyncVault(local, testPassword)
	require.NoError(t, err)

	entries, err := service.ListEntries(local, testPassword)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "OtherDevice", entries[0].Name)
}

// ---------------------------------------------------------------------------
// S2 — 有本地变更，远程无变更
// ---------------------------------------------------------------------------

func TestS2_SyncVault_LocalChanges_NoRemoteChanges(t *testing.T) {
	requireGit(t)
	local, remote := setupBoundRepo(t)

	require.NoError(t, service.AddEntry(local, testPassword, "LocalNew", "u", "p", "", nil))

	err := service.SyncVault(local, testPassword)
	require.NoError(t, err)

	// 验证远程也包含新条目
	verify := loadClonedVault(t, remote, testPassword)
	assert.Len(t, verify.Entries, 1)
	assert.Equal(t, "LocalNew", verify.Entries[0].Name)
}

// ---------------------------------------------------------------------------
// S3 — 有本地变更，远程也有变更（不同条目）
// ---------------------------------------------------------------------------

func TestS3_SyncVault_BothChanged_DifferentEntries(t *testing.T) {
	requireGit(t)
	local, remote := setupBoundRepo(t)

	// 远程新增条目 B
	updateRemoteVault(t, remote, testPassword, func(v *vault.Vault) {
		v.AddEntry(vault.Entry{ID: "b", Name: "B", UpdatedAt: 100})
	})

	// 本地新增条目 A
	require.NoError(t, service.AddEntry(local, testPassword, "A", "u", "p", "", nil))

	err := service.SyncVault(local, testPassword)
	require.NoError(t, err)

	entries, err := service.ListEntries(local, testPassword)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2, "should have both A and B after merge")

	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	assert.True(t, names["A"])
	assert.True(t, names["B"])
}

// ---------------------------------------------------------------------------
// S4 — 有本地变更，远程也有变更（同条目冲突）
// ---------------------------------------------------------------------------

func TestS4_SyncVault_BothChanged_SameEntryConflict(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	// 初始条目 X 推到远程
	seed := vault.NewVault()
	seed.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "original", UpdatedAt: 50})
	pushVaultToRemote(t, remote, testPassword, seed)

	// 本地绑定拉取
	local := freshDir(t)
	require.NoError(t, service.BindRemoteRepo(local, remote, testPassword))

	// "另一设备"修改条目 X（较旧时间戳）
	updateRemoteVault(t, remote, testPassword, func(v *vault.Vault) {
		for i, e := range v.Entries {
			if e.ID == "x" {
				v.Entries[i].Password = "remote-updated"
				v.Entries[i].UpdatedAt = 100
			}
		}
	})

	// 本地修改条目 X（较新时间戳）
	vaultPath := filepath.Join(local, "vault.dat")
	lv, err := storage.LoadVault(vaultPath, testPassword)
	require.NoError(t, err)
	for i, e := range lv.Entries {
		if e.ID == "x" {
			lv.Entries[i].Password = "local-updated"
			lv.Entries[i].UpdatedAt = 200
		}
	}
	require.NoError(t, storage.SaveVault(vaultPath, testPassword, lv))

	err = service.SyncVault(local, testPassword)
	require.NoError(t, err)

	result := loadLocalVault(t, local, testPassword)
	x := entryByID(result.Entries, "x")
	require.NotNil(t, x)
	assert.Equal(t, "local-updated", x.Password,
		"entry with newer UpdatedAt (local=200) should win over remote (100)")
}

// ---------------------------------------------------------------------------
// S5 — pull 失败时恢复本地 vault
// ---------------------------------------------------------------------------

func TestS5_SyncVault_PullFail_RecoverLocalVault(t *testing.T) {
	requireGit(t)
	local, _ := setupBoundRepo(t)

	require.NoError(t, service.AddEntry(local, testPassword, "Important", "u", "p", "", nil))

	// 把 remote 改成不可达地址
	gitExec(t, local, "remote", "set-url", "origin", "/nonexistent/remote/path")

	err := service.SyncVault(local, testPassword)
	require.Error(t, err, "sync with unreachable remote should fail")

	// vault.dat 应该被恢复，本地数据不丢失
	assert.True(t, vaultFileExists(local), "vault.dat should be restored after failure")

	entries, err := service.ListEntries(local, testPassword)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "local entry should survive failed sync")
	assert.Equal(t, "Important", entries[0].Name)
}

// ---------------------------------------------------------------------------
// S6 — BindRepo 后首次 Sync
// ---------------------------------------------------------------------------

func TestS6_SyncVault_FirstSyncAfterBind(t *testing.T) {
	requireGit(t)
	local, _ := setupBoundRepo(t)

	err := service.SyncVault(local, testPassword)
	require.NoError(t, err, "first sync after bind should succeed")
}

// ---------------------------------------------------------------------------
// S7 — 工作区清理策略验证
// ---------------------------------------------------------------------------

func TestS7_SyncVault_WorkspaceCleanupStrategy(t *testing.T) {
	requireGit(t)
	local, remote := setupBoundRepo(t)

	// 远程推送一个条目
	updateRemoteVault(t, remote, testPassword, func(v *vault.Vault) {
		v.AddEntry(vault.Entry{ID: "remote-e", Name: "Remote", UpdatedAt: 100})
	})

	// 本地新增条目（vault.dat 有未提交变更）
	require.NoError(t, service.AddEntry(local, testPassword, "Local", "u", "p", "", nil))

	err := service.SyncVault(local, testPassword)
	require.NoError(t, err)

	// 验证最终 vault 是合并结果
	entries, err := service.ListEntries(local, testPassword)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 2)

	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	assert.True(t, names["Local"], "local entry should survive workspace cleanup")
	assert.True(t, names["Remote"], "remote entry should be pulled in")
}

// ---------------------------------------------------------------------------
// S8 — 空路径校验
// ---------------------------------------------------------------------------

func TestS8_SyncVault_EmptyPath(t *testing.T) {
	err := service.SyncVault("", testPassword)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "仓库路径不能为空")
}
