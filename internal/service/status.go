package service

import (
	"fmt"

	"pwdmgr/internal/git"
	"pwdmgr/internal/vault"
)

func GetRepoStatus(path string) (RepoStatus, error) {
	status := RepoStatus{}

	if path == "" {
		return status, fmt.Errorf("仓库路径不能为空")
	}

	// 1. 是否是 git 仓库
	status.IsGitRepo = git.IsGitRepo(path)

	if !status.IsGitRepo {
		return status, nil
	}

	// 2. 是否有 remote
	hasRemote, err := git.HasOriginRemote(path)
	if err != nil {
		return status, err
	}
	status.HasRemote = hasRemote

	// 3. remote 是否有数据
	if hasRemote {
		remoteHas, err := git.RemoteHasCommit(path)
		if err != nil {
			return status, err
		}
		status.RemoteHasData = remoteHas
	}

	// 4. 本地 vault 是否存在
	status.HasLocalVault = vaultExists(path)

	return status, nil
}

// 简单封装（你也可以放 storage）
func vaultExists(path string) bool {
	_, err := vault.LoadVaultPath(path)
	return err == nil
}
