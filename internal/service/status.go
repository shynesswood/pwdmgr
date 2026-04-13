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

	status.IsGitRepo = git.IsGitRepo(path)
	if !status.IsGitRepo {
		return status, nil
	}

	status.CurrentBranch = git.CurrentBranch(path)
	status.HasUncommitted = git.HasChanges(path)

	hasRemote, err := git.HasOriginRemote(path)
	if err != nil {
		return status, err
	}
	status.HasRemote = hasRemote

	if hasRemote {
		status.RemoteURL = git.RemoteURL(path)
		remoteHas, err := git.RemoteHasCommit(path)
		if err != nil {
			return status, err
		}
		status.RemoteHasData = remoteHas
	}

	status.HasLocalVault = vaultExists(path)

	return status, nil
}

// 简单封装（你也可以放 storage）
func vaultExists(path string) bool {
	_, err := vault.LoadVaultPath(path)
	return err == nil
}
