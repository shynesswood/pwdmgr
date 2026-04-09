package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	saltSize  = 16
	nonceSize = 12
	keySize   = 32
)

func deriveKey(password, salt []byte) []byte {
	return argon2.IDKey(password, salt, 1, 64*1024, 4, keySize)
}

func Encrypt(password, plaintext []byte) ([]byte, error) {
	salt := make([]byte, saltSize)
	io.ReadFull(rand.Reader, salt)

	key := deriveKey(password, salt)

	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)

	nonce := make([]byte, nonceSize)
	io.ReadFull(rand.Reader, nonce)

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	result := append(salt, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

func Decrypt(password, data []byte) ([]byte, error) {
	if len(data) < saltSize+nonceSize {
		return nil, fmt.Errorf("invalid data: too short")
	}

	salt := data[:saltSize]
	nonce := data[saltSize : saltSize+nonceSize]
	ciphertext := data[saltSize+nonceSize:]

	key := deriveKey(password, salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: wrong password or corrupted data")
	}

	return plain, nil
}
