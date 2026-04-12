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
			// 保留 os.ErrNotExist 链，供上层区分「尚无库文件」与「解密失败」
			return nil, fmt.Errorf("数据文件不存在: %w", err)
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
