package service

import (
	"fmt"
	"os"

	"pwdmgr/internal/config"
	"pwdmgr/internal/git"
	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"
)

func BindRemoteRepo(localPath, repoURL string, password []byte) error {
	if localPath == "" {
		return fmt.Errorf("仓库路径不能为空")
	}
	if repoURL == "" {
		return fmt.Errorf("远程仓库地址不能为空")
	}

	vaultPath := config.VaultFilePath(localPath)
	localExists := fileExists(vaultPath)

	if !git.IsGitRepo(localPath) {
		if err := git.Init(localPath); err != nil {
			return err
		}
	}

	if err := git.AddRemote(localPath, repoURL); err != nil {
		return err
	}

	remoteHasData, err := git.RemoteHasCommit(localPath)
	if err != nil {
		return err
	}

	// 情况1：远程空，本地有 → push
	if !remoteHasData && localExists {
		return git.Push(localPath)
	}

	// 情况2：远程有，本地无 → pull
	if remoteHasData && !localExists {
		return git.Pull(localPath)
	}

	// 情况3：两边都有 → 先读本地，清理工作区后 pull，再合并
	if remoteHasData && localExists {
		localVault, err := storage.LoadVault(vaultPath, password)
		if err != nil {
			return err
		}

		// 移除本地 vault 文件，让 pull 不会因未提交变更而冲突
		os.Remove(vaultPath)

		if err := git.Pull(localPath); err != nil {
			// pull 失败则恢复本地 vault
			_ = storage.SaveVault(vaultPath, password, localVault)
			return err
		}

		remoteVault, err := storage.LoadVault(vaultPath, password)
		if err != nil {
			return err
		}

		merged := vault.MergeVault(localVault, remoteVault)

		if err := storage.SaveVault(vaultPath, password, merged); err != nil {
			return err
		}

		return git.Push(localPath)
	}

	// 情况4：两边都没有 → 初始化空库
	if !remoteHasData && !localExists {
		v := vault.NewVault()
		if err := storage.SaveVault(vaultPath, password, v); err != nil {
			return err
		}
		return git.Push(localPath)
	}

	return nil
}
