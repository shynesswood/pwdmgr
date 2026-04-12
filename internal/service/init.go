package service

import (
	"fmt"
	"os"

	"pwdmgr/internal/config"
	"pwdmgr/internal/git"
	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"
)

// EnsureVault 在仓库根目录下加载或创建 vault.dat。
func EnsureVault(repoRoot string, password []byte) (*vault.Vault, error) {
	vaultPath := config.VaultFilePath(repoRoot)

	if fileExists(vaultPath) {
		return storage.LoadVault(vaultPath, password)
	}

	v := vault.NewVault()
	if err := storage.SaveVault(vaultPath, password, v); err != nil {
		return nil, err
	}

	return v, nil
}

// InitLocalVault 在本地目录初始化 Git 仓库（若尚未初始化）并创建空的加密 vault.dat。
func InitLocalVault(repoRoot string, password []byte) error {
	if repoRoot == "" {
		return fmt.Errorf("仓库路径不能为空")
	}
	if !git.IsGitRepo(repoRoot) {
		if err := git.Init(repoRoot); err != nil {
			return err
		}
	}
	vaultPath := config.VaultFilePath(repoRoot)
	if fileExists(vaultPath) {
		return fmt.Errorf("本地 vault 已存在")
	}
	v := vault.NewVault()
	return storage.SaveVault(vaultPath, password, v)
}

func InitAndPushIfNeeded(repoRoot string, password []byte) error {
	vaultPath := config.VaultFilePath(repoRoot)

	if fileExists(vaultPath) {
		return nil
	}

	v := vault.NewVault()

	if err := storage.SaveVault(vaultPath, password, v); err != nil {
		return err
	}

	return git.Push(repoRoot)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
