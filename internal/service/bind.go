package service

import (
	"fmt"

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

	// 1. 初始化 git（如果没有）
	if !git.IsGitRepo(localPath) {
		if err := git.Init(localPath); err != nil {
			return err
		}
	}

	// 2. 添加远程
	if err := git.AddRemote(localPath, repoURL); err != nil {
		return err
	}

	// 3. 判断远程是否有提交（关键！）
	remoteHasData, err := git.RemoteHasCommit(localPath)
	if err != nil {
		return err
	}

	// -------- 分支逻辑 --------

	// 🟢 情况1：远程空，本地有 → push
	if !remoteHasData && localExists {
		return git.Push(localPath)
	}

	// 🟢 情况2：远程有，本地无 → pull
	if remoteHasData && !localExists {
		return git.Pull(localPath)
	}

	// 🟡 情况3：两边都有 → merge（注意顺序！）
	if remoteHasData && localExists {

		// 先读本地（未被破坏）
		localVault, err := storage.LoadVault(vaultPath, password)
		if err != nil {
			return err
		}

		// pull（此时才允许）
		if err := git.Pull(localPath); err != nil {
			return err
		}

		// 再读远程版本（已经更新到本地）
		remoteVault, err := storage.LoadVault(vaultPath, password)
		if err != nil {
			return err
		}

		// merge
		merged := vault.MergeVault(localVault, remoteVault)

		// 保存
		if err := storage.SaveVault(vaultPath, password, merged); err != nil {
			return err
		}

		return git.Push(localPath)
	}

	// 🔵 情况4：两边都没有 → 初始化
	if !remoteHasData && !localExists {
		v := vault.NewVault()
		if err := storage.SaveVault(vaultPath, password, v); err != nil {
			return err
		}
		return git.Push(localPath)
	}

	return nil
}
