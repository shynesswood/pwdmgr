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

func TestVault_DeleteEntry(t *testing.T) {
	v := vault.NewVault()
	e1 := vault.NewEntry("Keep", "u", "p", "", nil)
	e2 := vault.NewEntry("Delete", "u", "p", "", nil)
	v.AddEntry(e1)
	v.AddEntry(e2)

	v.DeleteEntry(e2.ID)
	assert.Len(t, v.Entries, 1)
	assert.Equal(t, e1.ID, v.Entries[0].ID)
}

func TestVault_DeleteEntry_NonExistentID(t *testing.T) {
	v := vault.NewVault()
	e := vault.NewEntry("A", "u", "p", "", nil)
	v.AddEntry(e)

	v.DeleteEntry("nonexistent-id")
	assert.Len(t, v.Entries, 1, "deleting non-existent ID should be a no-op")
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
