package service

import (
	"fmt"
	"os"

	"pwdmgr/internal/config"
	"pwdmgr/internal/git"
	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"
)

// InitLocalVault 在本地目录初始化 Git 仓库（若尚未初始化），
// 创建空的加密 vault.dat 并提交初始版本。
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
	if err := storage.SaveVault(vaultPath, password, v); err != nil {
		return err
	}
	return git.Commit(repoRoot, "init vault")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
