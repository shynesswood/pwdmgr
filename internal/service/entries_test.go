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

	entries, err := service.ListEntries(dir, testPassword)
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

	entries, err := service.ListEntries(dir, testPassword)
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
	require.NoError(t, service.AddEntry(dir, testPassword, "GitHub", "user", "pass", "note", []string{"dev"}))
	require.NoError(t, service.AddEntry(dir, testPassword, "Email", "me", "secret", "", nil))

	entries, err := service.ListEntries(dir, testPassword)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// ListEntries sorts by name, so Email < GitHub
	assert.Equal(t, "Email", entries[0].Name)
	assert.Equal(t, "GitHub", entries[1].Name)

	// Update
	toUpdate := entries[1] // GitHub
	toUpdate.Password = "new-pass"
	require.NoError(t, service.UpdateEntry(dir, testPassword, toUpdate))

	entries, _ = service.ListEntries(dir, testPassword)
	gh := entryByID(entries, toUpdate.ID)
	require.NotNil(t, gh)
	assert.Equal(t, "new-pass", gh.Password)

	// Delete
	require.NoError(t, service.DeleteEntry(dir, testPassword, entries[0].ID))

	entries, _ = service.ListEntries(dir, testPassword)
	assert.Len(t, entries, 1)
	assert.Equal(t, "GitHub", entries[0].Name)
}

func TestAddEntry_EmptyName(t *testing.T) {
	requireGit(t)
	dir := freshDir(t)
	require.NoError(t, service.InitLocalVault(dir, testPassword))

	err := service.AddEntry(dir, testPassword, "", "u", "p", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "名称不能为空")
}
