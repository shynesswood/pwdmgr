package service

import (
	"pwdmgr/internal/git"
	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"
)

func SyncVault(path string, password []byte) error {

	if !git.HasChanges(path) {
		// 没有本地修改，只 pull
		return git.Pull(path)
	}

	// 1. 读本地
	localVault, err := storage.LoadVault(path, password)
	if err != nil {
		return err
	}

	// 2. git pull
	if err := git.Pull(path); err != nil {
		return err
	}

	// 3. 再读（远程更新后的）
	remoteVault, err := storage.LoadVault(path, password)
	if err != nil {
		return err
	}

	// 4. merge
	merged := vault.MergeVault(localVault, remoteVault)

	// 5. 保存
	if err := storage.SaveVault(path, password, merged); err != nil {
		return err
	}

	// 6. git push
	if err := git.Push(path); err != nil {
		return err
	}

	return nil
}

func PullVault(path string) error {
	return git.Pull(path)
}

func PushVault(path string) error {
	return git.Push(path)
}
