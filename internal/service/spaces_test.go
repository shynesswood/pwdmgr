package service_test

import (
	"testing"

	"pwdmgr/internal/service"
	"pwdmgr/internal/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// SP-I1 — 初始化后 ListSpaces 仅含默认空间
// ---------------------------------------------------------------------------

func TestSPI1_ListSpaces_DefaultOnly(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	spaces, err := service.ListSpaces(dir, testPassword)
	require.NoError(t, err)
	require.Len(t, spaces, 1)
	assert.Equal(t, vault.DefaultSpaceID, spaces[0].ID)
}

// ---------------------------------------------------------------------------
// SP-I2 — CreateSpace 创建新空间，ListSpaces 按名称排序（默认空间置顶）
// ---------------------------------------------------------------------------

func TestSPI2_CreateSpace_ListOrder(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	_, err := service.CreateSpace(dir, testPassword, "个人")
	require.NoError(t, err)
	_, err = service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)

	spaces, err := service.ListSpaces(dir, testPassword)
	require.NoError(t, err)
	require.Len(t, spaces, 3)
	assert.Equal(t, vault.DefaultSpaceID, spaces[0].ID, "默认空间应置顶")
}

// ---------------------------------------------------------------------------
// SP-I3 — CreateSpace 重复名称失败
// ---------------------------------------------------------------------------

func TestSPI3_CreateSpace_DuplicateName(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	_, err := service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)

	_, err = service.CreateSpace(dir, testPassword, "工作")
	assert.ErrorIs(t, err, vault.ErrSpaceNameDuplicate)
}

// ---------------------------------------------------------------------------
// SP-I4 — RenameSpace：默认空间受保护
// ---------------------------------------------------------------------------

func TestSPI4_RenameSpace_ProtectDefault(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	err := service.RenameSpace(dir, testPassword, vault.DefaultSpaceID, "新默认")
	assert.ErrorIs(t, err, vault.ErrSpaceProtected)
}

// ---------------------------------------------------------------------------
// SP-I5 — RenameSpace 正常流程
// ---------------------------------------------------------------------------

func TestSPI5_RenameSpace_Success(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	sp, err := service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)

	require.NoError(t, service.RenameSpace(dir, testPassword, sp.ID, "Work Space"))

	spaces, err := service.ListSpaces(dir, testPassword)
	require.NoError(t, err)
	var got *vault.Space
	for i := range spaces {
		if spaces[i].ID == sp.ID {
			got = &spaces[i]
		}
	}
	require.NotNil(t, got)
	assert.Equal(t, "Work Space", got.Name)
}

// ---------------------------------------------------------------------------
// SP-I6 — DeleteSpace：非空空间拒绝删除
// ---------------------------------------------------------------------------

func TestSPI6_DeleteSpace_NotEmptyRejected(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	sp, err := service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)

	require.NoError(t, service.AddEntry(dir, testPassword, sp.ID, "Jira", "u", "p", "", nil))

	err = service.DeleteSpace(dir, testPassword, sp.ID)
	assert.ErrorIs(t, err, vault.ErrSpaceNotEmpty)
}

// ---------------------------------------------------------------------------
// SP-I7 — DeleteSpace：默认空间受保护
// ---------------------------------------------------------------------------

func TestSPI7_DeleteSpace_ProtectDefault(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	err := service.DeleteSpace(dir, testPassword, vault.DefaultSpaceID)
	assert.ErrorIs(t, err, vault.ErrSpaceProtected)
}

// ---------------------------------------------------------------------------
// SP-I8 — DeleteSpace：空空间软删除成功，后续不可见
// ---------------------------------------------------------------------------

func TestSPI8_DeleteSpace_EmptySuccess(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	sp, err := service.CreateSpace(dir, testPassword, "临时")
	require.NoError(t, err)

	require.NoError(t, service.DeleteSpace(dir, testPassword, sp.ID))

	spaces, err := service.ListSpaces(dir, testPassword)
	require.NoError(t, err)
	for _, s := range spaces {
		assert.NotEqual(t, sp.ID, s.ID, "已删除空间不应出现在列表中")
	}
}

// ---------------------------------------------------------------------------
// SP-I9 — AddEntry 到指定空间 + ListEntries 按空间隔离
// ---------------------------------------------------------------------------

func TestSPI9_AddAndListBySpace(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	work, err := service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)
	personal, err := service.CreateSpace(dir, testPassword, "个人")
	require.NoError(t, err)

	require.NoError(t, service.AddEntry(dir, testPassword, work.ID, "Jira", "u", "p", "", nil))
	require.NoError(t, service.AddEntry(dir, testPassword, work.ID, "GitLab", "u", "p", "", nil))
	require.NoError(t, service.AddEntry(dir, testPassword, personal.ID, "Email", "u", "p", "", nil))
	require.NoError(t, service.AddEntry(dir, testPassword, "", "Router", "u", "p", "", nil)) // 默认空间

	defEntries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, defEntries, 1)
	assert.Equal(t, "Router", defEntries[0].Name)

	workEntries, err := service.ListEntries(dir, testPassword, work.ID)
	require.NoError(t, err)
	require.Len(t, workEntries, 2)

	personalEntries, err := service.ListEntries(dir, testPassword, personal.ID)
	require.NoError(t, err)
	require.Len(t, personalEntries, 1)
	assert.Equal(t, "Email", personalEntries[0].Name)
}

// ---------------------------------------------------------------------------
// SP-I10 — 空间不存在时 AddEntry / ListEntries 报错
// ---------------------------------------------------------------------------

func TestSPI10_InvalidSpaceRejected(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	err := service.AddEntry(dir, testPassword, "nonexistent", "X", "u", "p", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "空间不存在")

	_, err = service.ListEntries(dir, testPassword, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "空间不存在")
}

// ---------------------------------------------------------------------------
// SP-I11 — UpdateEntry 跨空间移动
// ---------------------------------------------------------------------------

func TestSPI11_UpdateEntry_MoveBetweenSpaces(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	work, err := service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)

	require.NoError(t, service.AddEntry(dir, testPassword, "", "Email", "u", "p", "", nil))

	defEntries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, defEntries, 1)
	e := defEntries[0]

	// 移动到 work 空间
	e.SpaceID = work.ID
	require.NoError(t, service.UpdateEntry(dir, testPassword, e))

	after, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	assert.Empty(t, after)

	workEntries, err := service.ListEntries(dir, testPassword, work.ID)
	require.NoError(t, err)
	require.Len(t, workEntries, 1)
	assert.Equal(t, "Email", workEntries[0].Name)
}

// ---------------------------------------------------------------------------
// SP-I12 — UpdateEntry 目标空间不存在则拒绝
// ---------------------------------------------------------------------------

func TestSPI12_UpdateEntry_InvalidTargetSpace(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))
	require.NoError(t, service.AddEntry(dir, testPassword, "", "Email", "u", "p", "", nil))

	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	e.SpaceID = "nonexistent"
	err = service.UpdateEntry(dir, testPassword, e)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "空间不存在")
}

// ---------------------------------------------------------------------------
// MV-I1 — 批量移动若干条目到另一空间，ListEntries 对应变化
// ---------------------------------------------------------------------------

func TestMVI1_MoveEntries_Batch(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	work, err := service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)

	require.NoError(t, service.AddEntry(dir, testPassword, "", "A", "u", "p", "", nil))
	require.NoError(t, service.AddEntry(dir, testPassword, "", "B", "u", "p", "", nil))
	require.NoError(t, service.AddEntry(dir, testPassword, "", "C", "u", "p", "", nil))

	defEntries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, defEntries, 3)

	ids := []string{defEntries[0].ID, defEntries[1].ID}
	moved, err := service.MoveEntries(dir, testPassword, ids, work.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, moved)

	defAfter, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, defAfter, 1)

	workEntries, err := service.ListEntries(dir, testPassword, work.ID)
	require.NoError(t, err)
	require.Len(t, workEntries, 2)
}

// ---------------------------------------------------------------------------
// MV-I2 — 单条移动（ids 长度为 1）
// ---------------------------------------------------------------------------

func TestMVI2_MoveEntries_Single(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	work, err := service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)

	require.NoError(t, service.AddEntry(dir, testPassword, "", "Solo", "u", "p", "", nil))
	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	moved, err := service.MoveEntries(dir, testPassword, []string{entries[0].ID}, work.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, moved)

	workEntries, err := service.ListEntries(dir, testPassword, work.ID)
	require.NoError(t, err)
	require.Len(t, workEntries, 1)
	assert.Equal(t, "Solo", workEntries[0].Name)
}

// ---------------------------------------------------------------------------
// MV-I3 — 目标空间不存在/已删除时报错
// ---------------------------------------------------------------------------

func TestMVI3_MoveEntries_TargetNotFound(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))
	require.NoError(t, service.AddEntry(dir, testPassword, "", "A", "u", "p", "", nil))

	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)

	moved, err := service.MoveEntries(dir, testPassword, []string{entries[0].ID}, "ghost")
	assert.Error(t, err)
	assert.Zero(t, moved)
	assert.Contains(t, err.Error(), "空间不存在")
}

// ---------------------------------------------------------------------------
// MV-I4 — 空 ID 列表返回 0，不报错
// ---------------------------------------------------------------------------

func TestMVI4_MoveEntries_EmptyIDs(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	moved, err := service.MoveEntries(dir, testPassword, nil, vault.DefaultSpaceID)
	require.NoError(t, err)
	assert.Zero(t, moved)

	moved, err = service.MoveEntries(dir, testPassword, []string{}, vault.DefaultSpaceID)
	require.NoError(t, err)
	assert.Zero(t, moved)
}

// ---------------------------------------------------------------------------
// MV-I5 — 部分 ID 合法，其它被跳过：返回实际移动数，仍能成功保存
// ---------------------------------------------------------------------------

func TestMVI5_MoveEntries_PartialValid(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	work, err := service.CreateSpace(dir, testPassword, "工作")
	require.NoError(t, err)

	require.NoError(t, service.AddEntry(dir, testPassword, "", "A", "u", "p", "", nil))
	defEntries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, defEntries, 1)

	moved, err := service.MoveEntries(dir, testPassword,
		[]string{defEntries[0].ID, "ghost-1", "ghost-2"}, work.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, moved)

	workEntries, err := service.ListEntries(dir, testPassword, work.ID)
	require.NoError(t, err)
	require.Len(t, workEntries, 1)
}

// ---------------------------------------------------------------------------
// SP-I13 — 旧版本 vault.dat（无 Spaces）加载后自动迁移到默认空间
// ---------------------------------------------------------------------------

func TestSPI13_LegacyVaultMigration(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	gitExec(t, dir, "init")
	gitCfg(t, dir)

	// 直接构造一个"旧"格式的 vault：Spaces 为 nil，Entry 无 SpaceID
	legacy := &vault.Vault{
		Version: 1,
		Entries: []vault.Entry{
			{ID: "e1", Name: "OldEmail", Username: "u", Password: "p", UpdatedAt: 100},
		},
	}
	saveLocalVault(t, dir, testPassword, legacy)

	// service.ListEntries 应将其视为默认空间下的条目
	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "OldEmail", entries[0].Name)

	// ListSpaces 也应返回默认空间
	spaces, err := service.ListSpaces(dir, testPassword)
	require.NoError(t, err)
	require.Len(t, spaces, 1)
	assert.Equal(t, vault.DefaultSpaceID, spaces[0].ID)
}
