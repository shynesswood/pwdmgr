package service_test

import (
	"testing"

	"pwdmgr/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// L1 — 空库返回空数组（不是 null）
// ---------------------------------------------------------------------------

func TestL1_ListEntries_EmptyVault(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.NotNil(t, entries, "should return non-nil slice")
	assert.Empty(t, entries)
}

// ---------------------------------------------------------------------------
// L2 — vault 文件不存在时返回空数组
// ---------------------------------------------------------------------------

func TestL2_ListEntries_NoVaultFile(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	gitExec(t, dir, "init")

	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err, "should not error when vault.dat is missing")
	require.NotNil(t, entries)
	assert.Empty(t, entries)
}

// ---------------------------------------------------------------------------
// AddEntry / UpdateEntry / DeleteEntry 集成
// ---------------------------------------------------------------------------

func TestEntries_CRUD(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	// Add
	require.NoError(t, service.AddEntry(dir, testPassword, "", "GitHub", "user", "pass", "note", []string{"dev"}))
	require.NoError(t, service.AddEntry(dir, testPassword, "", "Email", "me", "secret", "", nil))

	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// ListEntries sorts by name, so Email < GitHub
	assert.Equal(t, "Email", entries[0].Name)
	assert.Equal(t, "GitHub", entries[1].Name)

	// Update
	toUpdate := entries[1] // GitHub
	toUpdate.Password = "new-pass"
	require.NoError(t, service.UpdateEntry(dir, testPassword, toUpdate))

	entries, _ = service.ListEntries(dir, testPassword, "")
	gh := entryByID(entries, toUpdate.ID)
	require.NotNil(t, gh)
	assert.Equal(t, "new-pass", gh.Password)

	// Delete
	require.NoError(t, service.DeleteEntry(dir, testPassword, entries[0].ID))

	entries, _ = service.ListEntries(dir, testPassword, "")
	assert.Len(t, entries, 1)
	assert.Equal(t, "GitHub", entries[0].Name)
}

func TestAddEntry_EmptyName(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	err := service.AddEntry(dir, testPassword, "", "", "u", "p", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "名称不能为空")
}

// ---------------------------------------------------------------------------
// D5 — ListEntries 过滤软删除条目；磁盘上 vault 仍保留原始条目（带 DeletedAt）
// ---------------------------------------------------------------------------

func TestD5_ListEntries_FiltersSoftDeleted(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	require.NoError(t, service.AddEntry(dir, testPassword, "", "Keep", "u", "p", "", nil))
	require.NoError(t, service.AddEntry(dir, testPassword, "", "Delete", "u", "p", "", nil))

	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, entries, 2)

	toDelete := entryByID(entries, entries[0].ID)
	if toDelete.Name != "Delete" {
		toDelete = entryByID(entries, entries[1].ID)
	}
	require.NotNil(t, toDelete)
	require.NoError(t, service.DeleteEntry(dir, testPassword, toDelete.ID))

	afterDelete, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	assert.Len(t, afterDelete, 1, "软删除后对用户不可见")
	assert.Equal(t, "Keep", afterDelete[0].Name)

	// 但磁盘加密文件中仍保留了已删除条目（带 DeletedAt 标记）
	raw := loadLocalVault(t, dir, testPassword)
	assert.Len(t, raw.Entries, 2, "物理上仍存在用于合并")

	foundDeleted := false
	for _, e := range raw.Entries {
		if e.ID == toDelete.ID {
			foundDeleted = true
			assert.True(t, e.IsDeleted(), "被删除条目应带 DeletedAt 标记")
		}
	}
	assert.True(t, foundDeleted)
}

// ---------------------------------------------------------------------------
// D6 — UpdateEntry 对软删除条目返回"条目不存在"，防止前端误操作复活
// ---------------------------------------------------------------------------

func TestD6_UpdateEntry_SoftDeletedReturnsError(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))
	require.NoError(t, service.AddEntry(dir, testPassword, "", "Target", "u", "p", "", nil))

	entries, err := service.ListEntries(dir, testPassword, "")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	target := entries[0]

	require.NoError(t, service.DeleteEntry(dir, testPassword, target.ID))

	target.Password = "trying-to-resurrect"
	err = service.UpdateEntry(dir, testPassword, target)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "条目不存在")
}
