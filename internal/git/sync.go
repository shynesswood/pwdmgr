package git

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
