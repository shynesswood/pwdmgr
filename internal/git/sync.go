package git

import (
	"os"
	"strings"
)

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
	if err == nil {
		return nil
	}
	originalErr := err

	// 回退策略：仓库可能刚 init + remote add，没有本地分支/跟踪分支
	if _, ferr := runGitCommand(path, "fetch", "origin"); ferr != nil {
		return originalErr
	}

	branch := detectDefaultBranch(path)

	// 检查是否有本地提交
	if _, herr := runGitCommand(path, "rev-parse", "HEAD"); herr != nil {
		// 没有本地提交 → 直接 checkout 远程分支
		_, err = runGitCommand(path, "checkout", "-b", branch, "--track", "origin/"+branch)
		return err
	}

	// 有本地提交但没有跟踪分支 → 指定远程和分支拉取
	_, err = runGitCommand(path, "pull", "--rebase", "origin", branch)
	return err
}

// Commit 执行 git add + commit（不 push）。
func Commit(path, message string) error {
	if _, err := runGitCommand(path, "add", "."); err != nil {
		return err
	}
	_, err := runGitCommand(path, "commit", "-m", message)
	return err
}

func Push(path string) error {
	if _, err := runGitCommand(path, "add", "."); err != nil {
		return err
	}

	_, _ = runGitCommand(path, "commit", "-m", "sync vault")

	_, err := runGitCommand(path, "push", "-u", "origin", "HEAD")
	return err
}

func HasChanges(path string) bool {
	out, _ := runGitCommand(path, "status", "--porcelain")
	return len(out) > 0
}

func AddRemote(path, repoURL string) error {
	_, err := runGitCommand(path, "remote", "add", "origin", repoURL)
	if err != nil {
		// origin 可能已存在，改为更新 URL
		_, err = runGitCommand(path, "remote", "set-url", "origin", repoURL)
	}
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
		return false, nil
	}

	return len(out) > 0, nil
}

// RestoreFile 将已跟踪的文件恢复到最后一次提交的版本。
func RestoreFile(repoRoot, fileName string) {
	_, _ = runGitCommand(repoRoot, "checkout", "--", fileName)
}

// CurrentBranch 返回当前所在分支名。
func CurrentBranch(path string) string {
	out, err := runGitCommand(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RemoteURL 返回 origin 远程的 URL。
func RemoteURL(path string) string {
	out, err := runGitCommand(path, "remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func detectDefaultBranch(path string) string {
	out, err := runGitCommand(path, "branch", "-r")
	if err == nil {
		s := string(out)
		if strings.Contains(s, "origin/main") {
			return "main"
		}
		if strings.Contains(s, "origin/master") {
			return "master"
		}
	}
	return "main"
}
