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

	entries, err := service.ListEntries(local, testPassword, "")
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

	require.NoError(t, service.AddEntry(local, testPassword, "", "LocalNew", "u", "p", "", nil))

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
	require.NoError(t, service.AddEntry(local, testPassword, "", "A", "u", "p", "", nil))

	err := service.SyncVault(local, testPassword)
	require.NoError(t, err)

	entries, err := service.ListEntries(local, testPassword, "")
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

	require.NoError(t, service.AddEntry(local, testPassword, "", "Important", "u", "p", "", nil))

	// 把 remote 改成不可达地址
	gitExec(t, local, "remote", "set-url", "origin", "/nonexistent/remote/path")

	err := service.SyncVault(local, testPassword)
	require.Error(t, err, "sync with unreachable remote should fail")

	// vault.dat 应该被恢复，本地数据不丢失
	assert.True(t, vaultFileExists(local), "vault.dat should be restored after failure")

	entries, err := service.ListEntries(local, testPassword, "")
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
	require.NoError(t, service.AddEntry(local, testPassword, "", "Local", "u", "p", "", nil))

	err := service.SyncVault(local, testPassword)
	require.NoError(t, err)

	// 验证最终 vault 是合并结果
	entries, err := service.ListEntries(local, testPassword, "")
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

// ---------------------------------------------------------------------------
// D7 — 软删除同步后不会被远程"复活"
//   - 本地删除条目 X，SyncVault 成功推送 DeletedAt 到远程
//   - 再次 SyncVault（模拟拉取），X 依然处于删除状态
//   - 对前端可见的条目列表不应包含 X
// ---------------------------------------------------------------------------

func TestD7_SyncVault_SoftDeletePropagates(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	// 初始远程有 X
	seed := vault.NewVault()
	seed.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "p", UpdatedAt: 50})
	pushVaultToRemote(t, remote, testPassword, seed)

	// 本地绑定拉取
	local := freshDir(t)
	require.NoError(t, service.BindRemoteRepo(local, remote, testPassword))

	entries, err := service.ListEntries(local, testPassword, "")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	// 本地删除 X
	require.NoError(t, service.DeleteEntry(local, testPassword, "x"))

	// 同步，推送删除标记到远程
	require.NoError(t, service.SyncVault(local, testPassword))

	// 对用户不可见
	visible, err := service.ListEntries(local, testPassword, "")
	require.NoError(t, err)
	assert.Empty(t, visible, "软删除后条目不应出现在列表中")

	// 远程 vault 内应保留 DeletedAt 标记的条目，而非"消失"
	remoteVault := loadClonedVault(t, remote, testPassword)
	require.Len(t, remoteVault.Entries, 1)
	assert.True(t, remoteVault.Entries[0].IsDeleted(), "远程也应有 DeletedAt 标记")

	// 再次 SyncVault，确保条目不会被"复活"
	require.NoError(t, service.SyncVault(local, testPassword))
	visible, err = service.ListEntries(local, testPassword, "")
	require.NoError(t, err)
	assert.Empty(t, visible, "再次同步后条目仍不可见")
}

// ---------------------------------------------------------------------------
// SP-S1 — 双端在不同空间各自新增条目，同步后两个空间都正确合并
// ---------------------------------------------------------------------------

func TestSPS1_SyncVault_DifferentSpacesMergeIndependently(t *testing.T) {
	requireGit(t)
	local, remote := setupBoundRepo(t)

	// 本地创建 work 空间并新增条目
	work, err := service.CreateSpace(local, testPassword, "工作")
	require.NoError(t, err)
	require.NoError(t, service.AddEntry(local, testPassword, work.ID, "Jira", "u", "p", "", nil))

	// 远程（模拟另一设备）新增 personal 空间 + 条目
	updateRemoteVault(t, remote, testPassword, func(v *vault.Vault) {
		p, err := v.AddSpace("个人")
		require.NoError(t, err)
		v.AddEntry(vault.NewEntryInSpace(p.ID, "Email", "u", "p", "", nil))
	})

	require.NoError(t, service.SyncVault(local, testPassword))

	// 合并后应有 3 个空间（默认 + 工作 + 个人），各自条目独立
	spaces, err := service.ListSpaces(local, testPassword)
	require.NoError(t, err)
	names := map[string]bool{}
	for _, s := range spaces {
		names[s.Name] = true
	}
	assert.True(t, names["工作"])
	assert.True(t, names["个人"])

	// work 空间下仅有 Jira
	workEntries, err := service.ListEntries(local, testPassword, work.ID)
	require.NoError(t, err)
	require.Len(t, workEntries, 1)
	assert.Equal(t, "Jira", workEntries[0].Name)
}

// ---------------------------------------------------------------------------
// SP-S2 — 远程软删除的空间同步到本地后被过滤
// ---------------------------------------------------------------------------

func TestSPS2_SyncVault_RemoteDeletedSpaceHidden(t *testing.T) {
	requireGit(t)
	local, remote := setupBoundRepo(t)

	// 本地创建 archived 空间
	archived, err := service.CreateSpace(local, testPassword, "存档")
	require.NoError(t, err)
	require.NoError(t, service.SyncVault(local, testPassword))

	// 远程删除该空间（时间戳更新）
	updateRemoteVault(t, remote, testPassword, func(v *vault.Vault) {
		sp := v.FindSpace(archived.ID)
		require.NotNil(t, sp)
		sp.DeletedAt = 9999999999
		sp.UpdatedAt = 9999999999
	})

	require.NoError(t, service.SyncVault(local, testPassword))

	spaces, err := service.ListSpaces(local, testPassword)
	require.NoError(t, err)
	for _, s := range spaces {
		assert.NotEqual(t, archived.ID, s.ID, "远程已删除的空间应过滤")
	}
}

// ---------------------------------------------------------------------------
// D8 — 本地删除后，"另一设备"对同一条目做了时间更晚的修改 → 条目被恢复
// ---------------------------------------------------------------------------

func TestD8_SyncVault_NewerRemoteUpdateRestoresDeleted(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	// 远程初始有 X（updated_at=50）
	seed := vault.NewVault()
	seed.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "original", UpdatedAt: 50})
	pushVaultToRemote(t, remote, testPassword, seed)

	// 本地绑定并拉取
	local := freshDir(t)
	require.NoError(t, service.BindRemoteRepo(local, remote, testPassword))

	// 本地在 t=100 左右删除 X（DeletedAt 较小）
	vaultPath := filepath.Join(local, "vault.dat")
	lv, err := storage.LoadVault(vaultPath, testPassword)
	require.NoError(t, err)
	for i := range lv.Entries {
		if lv.Entries[i].ID == "x" {
			lv.Entries[i].DeletedAt = 100
			lv.Entries[i].UpdatedAt = 100
		}
	}
	require.NoError(t, storage.SaveVault(vaultPath, testPassword, lv))

	// 另一设备在 t=300 修改 X（更晚）
	updateRemoteVault(t, remote, testPassword, func(v *vault.Vault) {
		for i := range v.Entries {
			if v.Entries[i].ID == "x" {
				v.Entries[i].Password = "restored"
				v.Entries[i].UpdatedAt = 300
			}
		}
	})

	require.NoError(t, service.SyncVault(local, testPassword))

	entries, err := service.ListEntries(local, testPassword, "")
	require.NoError(t, err)
	require.Len(t, entries, 1, "远程更新时间戳更晚应恢复条目")
	assert.Equal(t, "restored", entries[0].Password)
	assert.Equal(t, int64(300), entries[0].UpdatedAt)
	assert.False(t, entries[0].IsDeleted())
}
