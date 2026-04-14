package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerializeDeserialize_RoundTrip(t *testing.T) {
	v := vault.NewVault()
	v.AddEntry(vault.Entry{ID: "1", Name: "Test", Password: "secret"})

	data, err := storage.Serialize(v)
	require.NoError(t, err)

	var decoded vault.Vault
	err = storage.Deserialize(data, &decoded)
	require.NoError(t, err)
	assert.Len(t, decoded.Entries, 1)
	assert.Equal(t, "secret", decoded.Entries[0].Password)
}

func TestSaveLoadVault_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.dat")
	password := []byte("test-password")

	original := vault.NewVault()
	original.AddEntry(vault.Entry{ID: "e1", Name: "GitHub", Username: "user", Password: "pass123"})
	original.AddEntry(vault.Entry{ID: "e2", Name: "Email", Username: "me@x.com", Password: "p@ss"})

	err := storage.SaveVault(path, password, original)
	require.NoError(t, err)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), "GitHub", "vault file should be encrypted, not plaintext")

	loaded, err := storage.LoadVault(path, password)
	require.NoError(t, err)
	assert.Len(t, loaded.Entries, 2)
	assert.Equal(t, "GitHub", loaded.Entries[0].Name)
	assert.Equal(t, "pass123", loaded.Entries[0].Password)
}

// X1 — 密码错误不损坏文件
func TestLoadVault_WrongPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.dat")

	v := vault.NewVault()
	v.AddEntry(vault.Entry{ID: "1", Name: "A", Password: "secret"})

	err := storage.SaveVault(path, []byte("correct-pwd"), v)
	require.NoError(t, err)

	_, err = storage.LoadVault(path, []byte("wrong-pwd"))
	assert.Error(t, err, "wrong password should cause decryption failure")

	recovered, err := storage.LoadVault(path, []byte("correct-pwd"))
	require.NoError(t, err, "file should not be corrupted after wrong password attempt")
	assert.Len(t, recovered.Entries, 1)
	assert.Equal(t, "secret", recovered.Entries[0].Password)
}

func TestLoadVault_FileNotExist(t *testing.T) {
	_, err := storage.LoadVault(filepath.Join(t.TempDir(), "nonexistent.dat"), []byte("pwd"))
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestSaveVault_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "vault.dat")

	err := storage.SaveVault(path, []byte("pwd"), vault.NewVault())
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.False(t, info.IsDir())
}
