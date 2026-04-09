package storage

import (
	"fmt"
	"os"

	"pwdmgr/internal/crypto"
	"pwdmgr/internal/vault"
)

func LoadVault(path string, password []byte) (*vault.Vault, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("load failed: data file not found")
		}
		return nil, err
	}

	plain, err := crypto.Decrypt(password, data)
	if err != nil {
		return nil, err
	}

	var v vault.Vault
	if err := Deserialize(plain, &v); err != nil {
		return nil, err
	}

	return &v, nil
}

func SaveVault(path string, password []byte, v *vault.Vault) error {
	data, err := Serialize(v)
	if err != nil {
		return err
	}

	enc, err := crypto.Encrypt(password, data)
	if err != nil {
		return err
	}

	return os.WriteFile(path, enc, 0600)
}
