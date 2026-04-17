package vault_test

import (
	"testing"

	"pwdmgr/internal/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVault_EntriesIsEmptySlice(t *testing.T) {
	v := vault.NewVault()
	require.NotNil(t, v)
	assert.Equal(t, 1, v.Version)
	assert.NotNil(t, v.Entries, "Entries should be non-nil empty slice, not nil")
	assert.Empty(t, v.Entries)
}

func TestNewEntry_GeneratesIDAndTimestamp(t *testing.T) {
	e := vault.NewEntry("GitHub", "user", "pass", "note", []string{"dev", "DEV"})
	assert.NotEmpty(t, e.ID)
	assert.Equal(t, "GitHub", e.Name)
	assert.Equal(t, "user", e.Username)
	assert.Equal(t, "pass", e.Password)
	assert.Equal(t, "note", e.Note)
	assert.Greater(t, e.UpdatedAt, int64(0))
	// tags should be normalized: lowercased and deduplicated
	assert.Equal(t, []string{"dev"}, e.Tags)
}

func TestVault_AddEntry(t *testing.T) {
	v := vault.NewVault()
	e := vault.NewEntry("Test", "u", "p", "", nil)
	v.AddEntry(e)
	assert.Len(t, v.Entries, 1)
	assert.Equal(t, e.ID, v.Entries[0].ID)
}

func TestVault_UpdateEntry(t *testing.T) {
	v := vault.NewVault()
	e := vault.NewEntry("Original", "u", "old-pass", "", nil)
	v.AddEntry(e)

	updated := e
	updated.Password = "new-pass"
	v.UpdateEntry(updated)

	assert.Len(t, v.Entries, 1)
	assert.Equal(t, "new-pass", v.Entries[0].Password)
	assert.GreaterOrEqual(t, v.Entries[0].UpdatedAt, e.UpdatedAt, "UpdatedAt should be refreshed")
}

// D1 — 软删除：条目保留在 Entries，但打上 DeletedAt 标记并刷新 UpdatedAt。
func TestVault_DeleteEntry_SoftDelete(t *testing.T) {
	v := vault.NewVault()
	e1 := vault.NewEntry("Keep", "u", "p", "", nil)
	e2 := vault.NewEntry("Delete", "u", "p", "", nil)
	v.AddEntry(e1)
	v.AddEntry(e2)
	beforeUpdate := e2.UpdatedAt

	v.DeleteEntry(e2.ID)

	assert.Len(t, v.Entries, 2, "软删除后条目仍保留在 Entries 中")

	deleted := findEntry(v.Entries, e2.ID)
	require.NotNil(t, deleted)
	assert.True(t, deleted.IsDeleted(), "DeletedAt 应被设置")
	assert.Greater(t, deleted.DeletedAt, int64(0))
	assert.GreaterOrEqual(t, deleted.UpdatedAt, beforeUpdate, "删除应刷新 UpdatedAt")
	assert.Equal(t, deleted.DeletedAt, deleted.UpdatedAt, "删除时 UpdatedAt 与 DeletedAt 应同步")

	kept := findEntry(v.Entries, e1.ID)
	require.NotNil(t, kept)
	assert.False(t, kept.IsDeleted())
}

// D1b — ActiveEntries 过滤掉已软删除的条目。
func TestVault_ActiveEntries_FilterDeleted(t *testing.T) {
	v := vault.NewVault()
	e1 := vault.NewEntry("Keep", "u", "p", "", nil)
	e2 := vault.NewEntry("Delete", "u", "p", "", nil)
	v.AddEntry(e1)
	v.AddEntry(e2)
	v.DeleteEntry(e2.ID)

	active := v.ActiveEntries()
	assert.Len(t, active, 1)
	assert.Equal(t, e1.ID, active[0].ID)
}

func TestVault_DeleteEntry_NonExistentID(t *testing.T) {
	v := vault.NewVault()
	e := vault.NewEntry("A", "u", "p", "", nil)
	v.AddEntry(e)

	v.DeleteEntry("nonexistent-id")
	assert.Len(t, v.Entries, 1, "deleting non-existent ID should be a no-op")
	assert.False(t, v.Entries[0].IsDeleted())
}

// D1c — 对已软删除的 ID 再次调用 DeleteEntry 应为 no-op，不覆盖原有 DeletedAt。
func TestVault_DeleteEntry_Idempotent(t *testing.T) {
	v := vault.NewVault()
	e := vault.NewEntry("A", "u", "p", "", nil)
	v.AddEntry(e)

	v.DeleteEntry(e.ID)
	firstDeletedAt := findEntry(v.Entries, e.ID).DeletedAt
	require.Greater(t, firstDeletedAt, int64(0))

	v.DeleteEntry(e.ID)
	secondDeletedAt := findEntry(v.Entries, e.ID).DeletedAt
	assert.Equal(t, firstDeletedAt, secondDeletedAt, "对已删除条目再次删除应为 no-op")
}

// B5 — 合并相同条目
func TestMergeVault_SameEntries(t *testing.T) {
	entry := vault.Entry{ID: "shared-id", Name: "Same", Password: "p", UpdatedAt: 100}

	local := vault.NewVault()
	local.AddEntry(entry)

	remote := vault.NewVault()
	remote.AddEntry(entry)

	merged := vault.MergeVault(local, remote)
	assert.Len(t, merged.Entries, 1)
	assert.Equal(t, "shared-id", merged.Entries[0].ID)
}

// B6 — 合并不同条目
func TestMergeVault_DifferentEntries(t *testing.T) {
	local := vault.NewVault()
	local.AddEntry(vault.Entry{ID: "a", Name: "A", UpdatedAt: 100})

	remote := vault.NewVault()
	remote.AddEntry(vault.Entry{ID: "b", Name: "B", UpdatedAt: 100})

	merged := vault.MergeVault(local, remote)
	assert.Len(t, merged.Entries, 2)

	ids := map[string]bool{}
	for _, e := range merged.Entries {
		ids[e.ID] = true
	}
	assert.True(t, ids["a"])
	assert.True(t, ids["b"])
}

// B7 — 同 ID 不同时间戳，取较新的
func TestMergeVault_SameID_NewerTimestampWins(t *testing.T) {
	local := vault.NewVault()
	local.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "local-newer", UpdatedAt: 200})

	remote := vault.NewVault()
	remote.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "remote-older", UpdatedAt: 100})

	merged := vault.MergeVault(local, remote)
	assert.Len(t, merged.Entries, 1)
	assert.Equal(t, "local-newer", merged.Entries[0].Password)
	assert.Equal(t, int64(200), merged.Entries[0].UpdatedAt)
}

func TestMergeVault_SameID_RemoteNewerWins(t *testing.T) {
	local := vault.NewVault()
	local.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "local-older", UpdatedAt: 100})

	remote := vault.NewVault()
	remote.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "remote-newer", UpdatedAt: 200})

	merged := vault.MergeVault(local, remote)
	assert.Len(t, merged.Entries, 1)
	assert.Equal(t, "remote-newer", merged.Entries[0].Password)
}

func TestMergeVault_BothEmpty(t *testing.T) {
	merged := vault.MergeVault(vault.NewVault(), vault.NewVault())
	assert.NotNil(t, merged)
	assert.Empty(t, merged.Entries)
}

func TestVault_FilterByTags(t *testing.T) {
	v := vault.NewVault()
	v.AddEntry(vault.Entry{ID: "1", Tags: []string{"work", "email"}})
	v.AddEntry(vault.Entry{ID: "2", Tags: []string{"personal"}})
	v.AddEntry(vault.Entry{ID: "3", Tags: []string{"work"}})

	result := v.FilterByTags([]string{"work"})
	assert.Len(t, result, 2)

	result = v.FilterByTags([]string{"work", "email"})
	assert.Len(t, result, 1)
	assert.Equal(t, "1", result[0].ID)

	result = v.FilterByTags(nil)
	assert.Len(t, result, 3, "nil tags should return all")
}

func TestVault_TagStats(t *testing.T) {
	v := vault.NewVault()
	v.AddEntry(vault.Entry{ID: "1", Tags: []string{"work", "email"}})
	v.AddEntry(vault.Entry{ID: "2", Tags: []string{"work"}})

	stats := v.TagStats()
	assert.Equal(t, 2, stats["work"])
	assert.Equal(t, 1, stats["email"])
}

// ---------------------------------------------------------------------------
// 软删除合并场景
// ---------------------------------------------------------------------------

// D2 — 合并：本地软删除 + 远程旧版本 → 保留删除标记（删除胜出）。
func TestMergeVault_LocalDelete_BeatsOlderRemote(t *testing.T) {
	local := vault.NewVault()
	local.AddEntry(vault.Entry{ID: "x", Name: "X", UpdatedAt: 200, DeletedAt: 200})

	remote := vault.NewVault()
	remote.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "old", UpdatedAt: 100})

	merged := vault.MergeVault(local, remote)
	require.Len(t, merged.Entries, 1)
	e := merged.Entries[0]
	assert.True(t, e.IsDeleted(), "删除标记应保留")
	assert.Equal(t, int64(200), e.UpdatedAt)
}

// D3 — 合并：远程更新时间戳更大 → 覆盖本地删除，相当于"恢复"条目。
func TestMergeVault_NewerRemoteUpdate_OverridesLocalDelete(t *testing.T) {
	local := vault.NewVault()
	local.AddEntry(vault.Entry{ID: "x", Name: "X", UpdatedAt: 100, DeletedAt: 100})

	remote := vault.NewVault()
	remote.AddEntry(vault.Entry{ID: "x", Name: "X", Password: "new-from-other-device", UpdatedAt: 200})

	merged := vault.MergeVault(local, remote)
	require.Len(t, merged.Entries, 1)
	e := merged.Entries[0]
	assert.False(t, e.IsDeleted(), "更新更晚时应覆盖删除标记")
	assert.Equal(t, "new-from-other-device", e.Password)
	assert.Equal(t, int64(200), e.UpdatedAt)
}

// D4 — 合并：本地软删除，远程从未有过该条目 → 保留删除标记，后续同步时删除可传播。
func TestMergeVault_LocalDelete_RemoteMissing(t *testing.T) {
	local := vault.NewVault()
	local.AddEntry(vault.Entry{ID: "x", Name: "X", UpdatedAt: 200, DeletedAt: 200})

	remote := vault.NewVault()

	merged := vault.MergeVault(local, remote)
	require.Len(t, merged.Entries, 1)
	assert.True(t, merged.Entries[0].IsDeleted())
}

// ---------------------------------------------------------------------------
// 多空间 — 模型与 CRUD
// ---------------------------------------------------------------------------

// SP1 — NewVault 自动包含默认空间，且默认空间 ID 固定。
func TestNewVault_HasDefaultSpace(t *testing.T) {
	v := vault.NewVault()
	spaces := v.ActiveSpaces()
	require.Len(t, spaces, 1)
	assert.Equal(t, vault.DefaultSpaceID, spaces[0].ID)
	assert.Equal(t, vault.DefaultSpaceName, spaces[0].Name)
	assert.Greater(t, spaces[0].CreatedAt, int64(0))
}

// SP2 — NewEntry / NewEntryInSpace 的默认空间归属。
func TestNewEntry_DefaultSpaceAssigned(t *testing.T) {
	e := vault.NewEntry("A", "u", "p", "", nil)
	assert.Equal(t, vault.DefaultSpaceID, e.SpaceID)

	work := vault.NewEntryInSpace("work", "A", "u", "p", "", nil)
	assert.Equal(t, "work", work.SpaceID)

	blank := vault.NewEntryInSpace("", "A", "u", "p", "", nil)
	assert.Equal(t, vault.DefaultSpaceID, blank.SpaceID, "空 spaceID 应回退默认")
}

// SP3 — AddSpace：名称去空格、重复名称返回 ErrSpaceNameDuplicate、空名返回 ErrSpaceNameEmpty。
func TestVault_AddSpace(t *testing.T) {
	v := vault.NewVault()

	s, err := v.AddSpace("  工作  ")
	require.NoError(t, err)
	assert.Equal(t, "工作", s.Name)
	assert.NotEmpty(t, s.ID)
	assert.NotEqual(t, vault.DefaultSpaceID, s.ID)

	_, err = v.AddSpace("工作")
	assert.ErrorIs(t, err, vault.ErrSpaceNameDuplicate)

	_, err = v.AddSpace("   ")
	assert.ErrorIs(t, err, vault.ErrSpaceNameEmpty)
}

// SP4 — RenameSpace：默认空间受保护；空名/重名被拒绝；普通空间可重命名并刷新 UpdatedAt。
func TestVault_RenameSpace(t *testing.T) {
	v := vault.NewVault()
	work, err := v.AddSpace("工作")
	require.NoError(t, err)
	personal, err := v.AddSpace("个人")
	require.NoError(t, err)

	// 默认空间禁止改名
	err = v.RenameSpace(vault.DefaultSpaceID, "新默认")
	assert.ErrorIs(t, err, vault.ErrSpaceProtected)

	// 重名
	err = v.RenameSpace(work.ID, "个人")
	assert.ErrorIs(t, err, vault.ErrSpaceNameDuplicate)

	// 空名
	err = v.RenameSpace(work.ID, "  ")
	assert.ErrorIs(t, err, vault.ErrSpaceNameEmpty)

	// 不存在
	err = v.RenameSpace("nonexistent", "XX")
	assert.ErrorIs(t, err, vault.ErrSpaceNotFound)

	// 正常
	oldUpdate := v.FindSpace(work.ID).UpdatedAt
	err = v.RenameSpace(work.ID, "Work Space")
	require.NoError(t, err)
	renamed := v.FindSpace(work.ID)
	assert.Equal(t, "Work Space", renamed.Name)
	assert.GreaterOrEqual(t, renamed.UpdatedAt, oldUpdate)

	_ = personal // 避免 unused
}

// SP5 — DeleteSpace：默认空间受保护；非空空间禁止删除；空空间软删除。
func TestVault_DeleteSpace(t *testing.T) {
	v := vault.NewVault()
	work, err := v.AddSpace("工作")
	require.NoError(t, err)
	personal, err := v.AddSpace("个人")
	require.NoError(t, err)

	// 默认空间受保护
	assert.ErrorIs(t, v.DeleteSpace(vault.DefaultSpaceID), vault.ErrSpaceProtected)

	// 空间下有条目 → 禁止删除
	v.AddEntry(vault.NewEntryInSpace(work.ID, "Job", "u", "p", "", nil))
	assert.ErrorIs(t, v.DeleteSpace(work.ID), vault.ErrSpaceNotEmpty)

	// 空空间 → 软删除
	require.NoError(t, v.DeleteSpace(personal.ID))
	assert.True(t, v.FindSpace(personal.ID).IsDeleted())
	assert.Equal(t, 2, len(v.ActiveSpaces()), "仅剩默认+工作空间")

	// 再次删除同 ID → ErrSpaceNotFound（已软删除）
	assert.ErrorIs(t, v.DeleteSpace(personal.ID), vault.ErrSpaceNotFound)
}

// SP6 — EntriesInSpace 仅返回该空间下的活跃条目。
func TestVault_EntriesInSpace(t *testing.T) {
	v := vault.NewVault()
	work, _ := v.AddSpace("工作")

	v.AddEntry(vault.NewEntryInSpace(vault.DefaultSpaceID, "Email", "u", "p", "", nil))
	w1 := vault.NewEntryInSpace(work.ID, "Jira", "u", "p", "", nil)
	v.AddEntry(w1)
	w2 := vault.NewEntryInSpace(work.ID, "GitLab", "u", "p", "", nil)
	v.AddEntry(w2)

	v.DeleteEntry(w2.ID)

	defEntries := v.EntriesInSpace(vault.DefaultSpaceID)
	assert.Len(t, defEntries, 1)
	assert.Equal(t, "Email", defEntries[0].Name)

	workEntries := v.EntriesInSpace(work.ID)
	require.Len(t, workEntries, 1)
	assert.Equal(t, "Jira", workEntries[0].Name)

	// 空 spaceID → 默认空间
	blank := v.EntriesInSpace("")
	assert.Len(t, blank, 1)
}

// SP7 — EnsureDefaultSpace 兼容旧 vault：补全默认空间，把无 SpaceID 条目归入默认空间。
func TestVault_EnsureDefaultSpace_MigratesLegacy(t *testing.T) {
	legacy := &vault.Vault{
		Version: 1,
		Entries: []vault.Entry{
			{ID: "e1", Name: "Old", UpdatedAt: 100},
			{ID: "e2", Name: "Also", UpdatedAt: 100, SpaceID: ""},
		},
	}
	legacy.EnsureDefaultSpace()

	require.Len(t, legacy.Spaces, 1)
	assert.Equal(t, vault.DefaultSpaceID, legacy.Spaces[0].ID)
	for _, e := range legacy.Entries {
		assert.Equal(t, vault.DefaultSpaceID, e.SpaceID, "旧 entry 应归入默认空间")
	}
}

// SP8 — Merge 支持 Spaces：按 UpdatedAt 取较新一方，兼顾软删除。
func TestMergeVault_MergesSpaces(t *testing.T) {
	// 本地新增 work 空间，远程没有 → 合并保留
	local := vault.NewVault()
	work, _ := local.AddSpace("工作")

	// 远程有 personal 空间，本地没有 → 合并新增
	remote := vault.NewVault()
	remote.Spaces = append(remote.Spaces, vault.Space{
		ID: "personal", Name: "个人", CreatedAt: 50, UpdatedAt: 50,
	})

	merged := vault.MergeVault(local, remote)
	ids := map[string]bool{}
	for _, s := range merged.Spaces {
		ids[s.ID] = true
	}
	assert.True(t, ids[vault.DefaultSpaceID])
	assert.True(t, ids[work.ID])
	assert.True(t, ids["personal"])
}

// SP9 — Merge 空间冲突：同 ID，取 UpdatedAt 较大者（包括软删除标记）。
func TestMergeVault_SpaceConflict_NewerWins(t *testing.T) {
	sid := "custom"
	local := vault.NewVault()
	local.Spaces = append(local.Spaces, vault.Space{
		ID: sid, Name: "本地名", CreatedAt: 100, UpdatedAt: 200,
	})

	remote := vault.NewVault()
	remote.Spaces = append(remote.Spaces, vault.Space{
		ID: sid, Name: "远程名", CreatedAt: 100, UpdatedAt: 100,
	})

	merged := vault.MergeVault(local, remote)
	for _, s := range merged.Spaces {
		if s.ID == sid {
			assert.Equal(t, "本地名", s.Name, "UpdatedAt 更大的一方胜出")
			assert.Equal(t, int64(200), s.UpdatedAt)
		}
	}
}

// SP10 — Merge 软删除的空间也按 UpdatedAt 胜出。
func TestMergeVault_DeletedSpace_BeatsOlderRemote(t *testing.T) {
	sid := "to-delete"
	local := vault.NewVault()
	local.Spaces = append(local.Spaces, vault.Space{
		ID: sid, Name: "X", CreatedAt: 50, UpdatedAt: 200, DeletedAt: 200,
	})
	remote := vault.NewVault()
	remote.Spaces = append(remote.Spaces, vault.Space{
		ID: sid, Name: "X", CreatedAt: 50, UpdatedAt: 100,
	})

	merged := vault.MergeVault(local, remote)
	var got *vault.Space
	for i := range merged.Spaces {
		if merged.Spaces[i].ID == sid {
			got = &merged.Spaces[i]
		}
	}
	require.NotNil(t, got)
	assert.True(t, got.IsDeleted(), "删除标记应保留")
}

// ---------------------------------------------------------------------------
// 批量移动 (MV) — vault 层
// ---------------------------------------------------------------------------

// MV1 — 基本移动：把一批条目迁到另一个空间，UpdatedAt 被刷新。
func TestVault_MoveEntries_Basic(t *testing.T) {
	v := vault.NewVault()
	work, _ := v.AddSpace("工作")

	e1 := vault.NewEntryInSpace(vault.DefaultSpaceID, "A", "u", "p", "", nil)
	e2 := vault.NewEntryInSpace(vault.DefaultSpaceID, "B", "u", "p", "", nil)
	e3 := vault.NewEntryInSpace(vault.DefaultSpaceID, "C", "u", "p", "", nil)
	e1.UpdatedAt = 100
	e2.UpdatedAt = 100
	e3.UpdatedAt = 100
	v.AddEntry(e1)
	v.AddEntry(e2)
	v.AddEntry(e3)

	moved := v.MoveEntries([]string{e1.ID, e2.ID}, work.ID)
	assert.Equal(t, 2, moved)

	got1 := findEntry(v.Entries, e1.ID)
	got2 := findEntry(v.Entries, e2.ID)
	got3 := findEntry(v.Entries, e3.ID)
	require.NotNil(t, got1)
	require.NotNil(t, got2)
	require.NotNil(t, got3)

	assert.Equal(t, work.ID, got1.SpaceID)
	assert.Equal(t, work.ID, got2.SpaceID)
	assert.Equal(t, vault.DefaultSpaceID, got3.SpaceID, "未包含的条目不受影响")

	assert.Greater(t, got1.UpdatedAt, int64(100), "被移动的条目 UpdatedAt 应刷新")
	assert.Equal(t, int64(100), got3.UpdatedAt, "未移动的条目 UpdatedAt 保持")
}

// MV2 — 目标空间为空字符串时回退到默认空间。
func TestVault_MoveEntries_EmptyTargetFallsBackToDefault(t *testing.T) {
	v := vault.NewVault()
	work, _ := v.AddSpace("工作")

	e := vault.NewEntryInSpace(work.ID, "X", "u", "p", "", nil)
	v.AddEntry(e)

	moved := v.MoveEntries([]string{e.ID}, "")
	assert.Equal(t, 1, moved)

	got := findEntry(v.Entries, e.ID)
	require.NotNil(t, got)
	assert.Equal(t, vault.DefaultSpaceID, got.SpaceID)
}

// MV3 — 已经在目标空间 / 已软删除 / 不存在 的 ID 被静默跳过。
func TestVault_MoveEntries_SkipsInvalidEntries(t *testing.T) {
	v := vault.NewVault()
	work, _ := v.AddSpace("工作")
	other, _ := v.AddSpace("其它")

	inTarget := vault.NewEntryInSpace(work.ID, "Already", "u", "p", "", nil)
	deleted := vault.NewEntryInSpace(other.ID, "Deleted", "u", "p", "", nil)
	normal := vault.NewEntryInSpace(other.ID, "Normal", "u", "p", "", nil)
	v.AddEntry(inTarget)
	v.AddEntry(deleted)
	v.AddEntry(normal)
	v.DeleteEntry(deleted.ID)

	moved := v.MoveEntries([]string{inTarget.ID, deleted.ID, normal.ID, "ghost-id"}, work.ID)
	assert.Equal(t, 1, moved, "只有 normal 条目真正被移动")

	gotDeleted := findEntry(v.Entries, deleted.ID)
	require.NotNil(t, gotDeleted)
	assert.True(t, gotDeleted.IsDeleted())
	assert.Equal(t, other.ID, gotDeleted.SpaceID, "软删除条目不应被移动")

	gotNormal := findEntry(v.Entries, normal.ID)
	require.NotNil(t, gotNormal)
	assert.Equal(t, work.ID, gotNormal.SpaceID)
}

// MV4 — 空 ID 列表返回 0，不做任何修改。
func TestVault_MoveEntries_EmptyIDs(t *testing.T) {
	v := vault.NewVault()
	work, _ := v.AddSpace("工作")
	e := vault.NewEntryInSpace(vault.DefaultSpaceID, "A", "u", "p", "", nil)
	v.AddEntry(e)

	moved := v.MoveEntries(nil, work.ID)
	assert.Equal(t, 0, moved)

	got := findEntry(v.Entries, e.ID)
	require.NotNil(t, got)
	assert.Equal(t, vault.DefaultSpaceID, got.SpaceID)
}

// ---------------------------------------------------------------------------
// 测试辅助
// ---------------------------------------------------------------------------

func findEntry(entries []vault.Entry, id string) *vault.Entry {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
	}
	return nil
}
