package service

type RepoStatus struct {
	IsGitRepo     bool `json:"isGitRepo"`
	HasRemote     bool `json:"hasRemote"`
	RemoteHasData bool `json:"remoteHasData"`
	HasLocalVault bool `json:"hasLocalVault"`
}
