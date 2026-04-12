package git

import "os"

func Clone(repoURL, path string) error {
	_, err := runGitCommand("", "clone", repoURL, path)
	return err
}

func Init(path string) error {
	_, err := runGitCommand(path, "init")
	return err
}

func Pull(path string) error {
	_, err := runGitCommand(path, "pull", "--rebase")
	return err
}

func Push(path string) error {
	_, err := runGitCommand(path, "add", ".")
	if err != nil {
		return err
	}

	_, _ = runGitCommand(path, "commit", "-m", "sync vault")

	_, err = runGitCommand(path, "push")
	return err
}

func HasChanges(path string) bool {
	out, _ := runGitCommand(path, "status", "--porcelain")
	return len(out) > 0
}

func AddRemote(path, repoURL string) error {
	_, err := runGitCommand(path, "remote", "add", "origin", repoURL)
	return err
}

func IsGitRepo(path string) bool {
	_, err := os.Stat(path + "/.git")
	return err == nil
}

func RemoteHasCommit(path string) (bool, error) {
	out, err := runGitCommand(path, "ls-remote", "--heads", "origin")
	if err != nil {
		return false, err
	}

	return len(out) > 0, nil
}

func HasOriginRemote(path string) (bool, error) {
	out, err := runGitCommand(path, "remote", "get-url", "origin")
	if err != nil {
		// 没有 origin 会报错，这里返回 false
		return false, nil
	}

	return len(out) > 0, nil
}
