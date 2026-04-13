package service

import (
	"fmt"
	"os"

	"pwdmgr/internal/config"
	"pwdmgr/internal/git"
	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"
)

func SyncVault(repoRoot string, password []byte) error {
	if repoRoot == "" {
		return fmt.Errorf("仓库路径不能为空")
	}

	vaultPath := config.VaultFilePath(repoRoot)

	if !git.HasChanges(repoRoot) {
		return git.Pull(repoRoot)
	}

	// 1. 读取本地 vault（保存在内存中）
	localVault, err := storage.LoadVault(vaultPath, password)
	if err != nil {
		return err
	}

	// 2. 清理工作区，让 pull 不会因未提交变更而失败
	//    先删除文件，再尝试恢复到已提交版本（如果有的话）
	os.Remove(vaultPath)
	git.RestoreFile(repoRoot, config.VaultFileName)

	// 3. pull
	if err := git.Pull(repoRoot); err != nil {
		// pull 失败则恢复本地 vault
		_ = storage.SaveVault(vaultPath, password, localVault)
		return err
	}

	// 4. 读取远程 vault（pull 后磁盘上的版本）
	var remoteVault *vault.Vault
	if fileExists(vaultPath) {
		remoteVault, err = storage.LoadVault(vaultPath, password)
		if err != nil {
			return err
		}
	} else {
		remoteVault = vault.NewVault()
	}

	// 5. 应用层合并
	merged := vault.MergeVault(localVault, remoteVault)

	// 6. 保存
	if err := storage.SaveVault(vaultPath, password, merged); err != nil {
		return err
	}

	// 7. push
	return git.Push(repoRoot)
}

func PullVault(repoRoot string) error {
	if repoRoot == "" {
		return fmt.Errorf("仓库路径不能为空")
	}
	return git.Pull(repoRoot)
}

func PushVault(repoRoot string) error {
	if repoRoot == "" {
		return fmt.Errorf("仓库路径不能为空")
	}
	return git.Push(repoRoot)
}
