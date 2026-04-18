package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// execBackend 通过调用本机 `git` 命令实现所有操作。
// 依赖用户本地已安装 Git。
type execBackend struct{}

func (execBackend) Name() string { return BackendExec }

// runGitCommand 执行 `git <args...>`，保留 stdout/stderr 合并输出以生成可读错误。
func runGitCommand(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return out, nil
}

func (execBackend) Clone(repoURL, path string) error {
	_, err := runGitCommand("", "clone", repoURL, path)
	return err
}

func (execBackend) Init(path string) error {
	_, err := runGitCommand(path, "init")
	return err
}

func (execBackend) Pull(path string) error {
	_, err := runGitCommand(path, "pull", "--rebase")
	if err == nil {
		return nil
	}
	originalErr := err

	// 回退策略：仓库可能刚 init + remote add，没有本地分支/跟踪分支
	if _, ferr := runGitCommand(path, "fetch", "origin"); ferr != nil {
		return originalErr
	}

	// 远程仍是空仓库（没有任何 remote-tracking ref）→ 没东西可拉，不算错
	if out, rerr := runGitCommand(path, "branch", "-r"); rerr == nil && strings.TrimSpace(string(out)) == "" {
		return nil
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
func (execBackend) Commit(path, message string) error {
	if _, err := runGitCommand(path, "add", "."); err != nil {
		return err
	}
	_, err := runGitCommand(path, "commit", "-m", message)
	return err
}

func (execBackend) Push(path string) error {
	if _, err := runGitCommand(path, "add", "."); err != nil {
		return err
	}

	_, _ = runGitCommand(path, "commit", "-m", "sync vault")

	_, err := runGitCommand(path, "push", "-u", "origin", "HEAD")
	return err
}

func (execBackend) HasChanges(path string) bool {
	out, _ := runGitCommand(path, "status", "--porcelain")
	return len(out) > 0
}

func (execBackend) AddRemote(path, repoURL string) error {
	_, err := runGitCommand(path, "remote", "add", "origin", repoURL)
	if err != nil {
		// origin 可能已存在，改为更新 URL
		_, err = runGitCommand(path, "remote", "set-url", "origin", repoURL)
	}
	return err
}

func (execBackend) IsGitRepo(path string) bool {
	_, err := os.Stat(path + "/.git")
	return err == nil
}

func (execBackend) RemoteHasCommit(path string) (bool, error) {
	out, err := runGitCommand(path, "ls-remote", "--heads", "origin")
	if err != nil {
		return false, err
	}
	return len(out) > 0, nil
}

func (execBackend) HasOriginRemote(path string) (bool, error) {
	out, err := runGitCommand(path, "remote", "get-url", "origin")
	if err != nil {
		return false, nil
	}
	return len(out) > 0, nil
}

// RestoreFile 将已跟踪的文件恢复到最后一次提交的版本。
func (execBackend) RestoreFile(repoRoot, fileName string) {
	_, _ = runGitCommand(repoRoot, "checkout", "--", fileName)
}

// CurrentBranch 返回当前所在分支名。
func (execBackend) CurrentBranch(path string) string {
	out, err := runGitCommand(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// RemoteURL 返回 origin 远程的 URL。
func (execBackend) RemoteURL(path string) string {
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
