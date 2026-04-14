package crypto_test

import (
	"testing"

	"pwdmgr/internal/crypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	password := []byte("strong-password-123")
	plaintext := []byte(`{"version":1,"entries":[]}`)

	ciphertext, err := crypto.Encrypt(password, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext, "ciphertext should differ from plaintext")

	decrypted, err := crypto.Decrypt(password, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecrypt_WrongPassword(t *testing.T) {
	password := []byte("correct-password")
	plaintext := []byte("secret data")

	ciphertext, err := crypto.Encrypt(password, plaintext)
	require.NoError(t, err)

	_, err = crypto.Decrypt([]byte("wrong-password"), ciphertext)
	assert.Error(t, err, "decrypting with wrong password should fail")
}

func TestDecrypt_TruncatedData(t *testing.T) {
	_, err := crypto.Decrypt([]byte("pwd"), []byte("short"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too short")
}

func TestEncryptDecrypt_RandomnessEachCall(t *testing.T) {
	password := []byte("pwd")
	plaintext := []byte("same content")

	c1, err := crypto.Encrypt(password, plaintext)
	require.NoError(t, err)
	c2, err := crypto.Encrypt(password, plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, c1, c2, "two encryptions of the same data should produce different ciphertext (random salt/nonce)")
}

// X4 — 空密码功能正常
func TestEncryptDecrypt_EmptyPassword(t *testing.T) {
	password := []byte("")
	plaintext := []byte(`{"version":1,"entries":[{"id":"1"}]}`)

	ciphertext, err := crypto.Encrypt(password, plaintext)
	require.NoError(t, err)

	decrypted, err := crypto.Decrypt(password, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}
