package service

import (
	"os"

	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"
)

func EnsureVault(path string, password []byte) (*vault.Vault, error) {

	if fileExists(path) {
		// 已存在 → 正常加载
		return storage.LoadVault(path, password)
	}

	// 不存在 → 初始化
	v := vault.NewVault()

	if err := storage.SaveVault(path, password, v); err != nil {
		return nil, err
	}

	return v, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
