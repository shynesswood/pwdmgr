package service

import (
	"fmt"

	"pwdmgr/internal/config"
	"pwdmgr/internal/git"
	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"
)

func SyncVault(repoRoot string, password []byte) error {
	if repoRoot == "" {
		return fmt.Errorf("仓库路径不能为空")
	}

	if !git.HasChanges(repoRoot) {
		// 没有本地修改，只 pull
		return git.Pull(repoRoot)
	}

	vaultPath := config.VaultFilePath(repoRoot)

	// 1. 读本地
	localVault, err := storage.LoadVault(vaultPath, password)
	if err != nil {
		return err
	}

	// 2. git pull
	if err := git.Pull(repoRoot); err != nil {
		return err
	}

	// 3. 再读（远程更新后的）
	remoteVault, err := storage.LoadVault(vaultPath, password)
	if err != nil {
		return err
	}

	// 4. merge
	merged := vault.MergeVault(localVault, remoteVault)

	// 5. 保存
	if err := storage.SaveVault(vaultPath, password, merged); err != nil {
		return err
	}

	// 6. git push
	if err := git.Push(repoRoot); err != nil {
		return err
	}

	return nil
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
