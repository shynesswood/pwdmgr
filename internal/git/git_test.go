package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping")
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, Init(dir))
	gitCfg(t, dir)
	return dir
}

func initBareRemote(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	out, err := exec.Command("git", "init", "--bare", dir).CombinedOutput()
	require.NoError(t, err, string(out))
	return dir
}

func gitCfg(t *testing.T, dir string) {
	t.Helper()
	run(t, dir, "config", "user.email", "test@test.com")
	run(t, dir, "config", "user.name", "Test")
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, string(out))
	return string(out)
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
}

// ---------------------------------------------------------------------------
// G1 — runGitCommand 错误信息可读
// ---------------------------------------------------------------------------

func TestG1_RunGitCommand_ReadableError(t *testing.T) {
	requireGit(t)
	dir := t.TempDir() // not a git repo

	_, err := runGitCommand(dir, "status")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository",
		"error should include git's original message, not just exit code")
}

// ---------------------------------------------------------------------------
// G2 — AddRemote 首次添加
// ---------------------------------------------------------------------------

func TestG2_AddRemote_FirstTime(t *testing.T) {
	requireGit(t)
	repo := initTestRepo(t)
	url := "https://example.com/repo.git"

	err := AddRemote(repo, url)
	require.NoError(t, err)

	out := run(t, repo, "remote", "-v")
	assert.Contains(t, out, url)
}

// ---------------------------------------------------------------------------
// G3 — AddRemote 重复添加（容错）
// ---------------------------------------------------------------------------

func TestG3_AddRemote_Duplicate(t *testing.T) {
	requireGit(t)
	repo := initTestRepo(t)

	require.NoError(t, AddRemote(repo, "https://old.example.com/repo.git"))

	newURL := "https://new.example.com/repo.git"
	err := AddRemote(repo, newURL)
	require.NoError(t, err, "duplicate AddRemote should not error")

	got := RemoteURL(repo)
	assert.Equal(t, newURL, got)
}

// ---------------------------------------------------------------------------
// G4 — Pull 回退策略：无跟踪分支
// ---------------------------------------------------------------------------

func TestG4_Pull_FallbackNoTrackingBranch(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	// push a commit to remote via temp clone
	clone := filepath.Join(t.TempDir(), "seed")
	run(t, "", "clone", remote, clone)
	gitCfg(t, clone)
	writeFile(t, clone, "readme.txt", "hello")
	run(t, clone, "add", ".")
	run(t, clone, "commit", "-m", "seed")
	run(t, clone, "push", "-u", "origin", "HEAD")

	// new local repo: init + remote add (no tracking branch yet)
	local := initTestRepo(t)
	require.NoError(t, AddRemote(local, remote))

	err := Pull(local)
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(local, "readme.txt"))
	assert.NoError(t, statErr, "pulled file should exist locally")
}

// ---------------------------------------------------------------------------
// G5 — Push 首次推送设置 upstream
// ---------------------------------------------------------------------------

func TestG5_Push_FirstTimeSetUpstream(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)
	repo := initTestRepo(t)
	require.NoError(t, AddRemote(repo, remote))

	writeFile(t, repo, "data.txt", "content")
	require.NoError(t, Commit(repo, "initial"))
	require.NoError(t, Push(repo))

	// verify remote has the commit
	verify := filepath.Join(t.TempDir(), "verify")
	run(t, "", "clone", remote, verify)
	_, err := os.Stat(filepath.Join(verify, "data.txt"))
	assert.NoError(t, err, "pushed file should be in remote")
}

// ---------------------------------------------------------------------------
// G6 — detectDefaultBranch 检测
// ---------------------------------------------------------------------------

func TestG6_DetectDefaultBranch(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	// push to create a branch
	clone := filepath.Join(t.TempDir(), "seed")
	run(t, "", "clone", remote, clone)
	gitCfg(t, clone)
	writeFile(t, clone, "f.txt", "x")
	run(t, clone, "add", ".")
	run(t, clone, "commit", "-m", "init")
	run(t, clone, "push", "-u", "origin", "HEAD")

	// detect from a local repo that has fetched
	local := initTestRepo(t)
	require.NoError(t, AddRemote(local, remote))
	run(t, local, "fetch", "origin")

	branch := detectDefaultBranch(local)
	assert.Contains(t, []string{"main", "master"}, branch,
		"should detect main or master")

	// no remote branches → fallback "main"
	emptyLocal := initTestRepo(t)
	fb := detectDefaultBranch(emptyLocal)
	assert.Equal(t, "main", fb)
}

// ---------------------------------------------------------------------------
// G7 — Commit 函数
// ---------------------------------------------------------------------------

func TestG7_Commit(t *testing.T) {
	requireGit(t)
	repo := initTestRepo(t)

	writeFile(t, repo, "hello.txt", "world")
	err := Commit(repo, "test commit")
	require.NoError(t, err)

	out := run(t, repo, "log", "--oneline")
	assert.Contains(t, out, "test commit")
}

// ---------------------------------------------------------------------------
// G8 — HasChanges 检测
// ---------------------------------------------------------------------------

func TestG8_HasChanges(t *testing.T) {
	requireGit(t)
	repo := initTestRepo(t)

	writeFile(t, repo, "f.txt", "v1")
	require.NoError(t, Commit(repo, "init"))

	// A: no changes
	assert.False(t, HasChanges(repo), "clean repo should have no changes")

	// B: modified tracked file
	writeFile(t, repo, "f.txt", "v2")
	assert.True(t, HasChanges(repo), "modified file should report changes")

	// reset
	run(t, repo, "checkout", "--", "f.txt")

	// C: new untracked file
	writeFile(t, repo, "new.txt", "untracked")
	assert.True(t, HasChanges(repo), "untracked file should report changes")
}

// ---------------------------------------------------------------------------
// G9 — RestoreFile 恢复已跟踪文件
// ---------------------------------------------------------------------------

func TestG9_RestoreFile(t *testing.T) {
	requireGit(t)
	repo := initTestRepo(t)

	writeFile(t, repo, "vault.dat", "original-content")
	require.NoError(t, Commit(repo, "add vault"))

	writeFile(t, repo, "vault.dat", "modified-content")
	data, _ := os.ReadFile(filepath.Join(repo, "vault.dat"))
	assert.Equal(t, "modified-content", string(data))

	RestoreFile(repo, "vault.dat")

	data, _ = os.ReadFile(filepath.Join(repo, "vault.dat"))
	assert.Equal(t, "original-content", string(data))
}

// ---------------------------------------------------------------------------
// G10 — CurrentBranch 获取当前分支
// ---------------------------------------------------------------------------

func TestG10_CurrentBranch(t *testing.T) {
	requireGit(t)
	repo := initTestRepo(t)

	writeFile(t, repo, "f.txt", "x")
	require.NoError(t, Commit(repo, "init"))

	branch := CurrentBranch(repo)
	assert.NotEmpty(t, branch)
	assert.True(t, branch == "main" || branch == "master",
		"expected main or master, got: %s", branch)
}

// ---------------------------------------------------------------------------
// G11 — RemoteURL 获取远程地址
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// G12 — exec 后端的 Pull 对空远程不报错
// ---------------------------------------------------------------------------

func TestG12_ExecPullToleratesEmptyRemote(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	repo := initTestRepo(t)
	require.NoError(t, AddRemote(repo, remote))

	// 本地无提交 + 空远程
	require.NoError(t, Pull(repo), "空仓库 + 空远程的 Pull 应成功")

	// 本地有提交 + 空远程
	writeFile(t, repo, "local.txt", "only local")
	run(t, repo, "add", ".")
	run(t, repo, "commit", "-m", "local")
	require.NoError(t, Pull(repo), "本地有提交 + 空远程的 Pull 应成功")
}

func TestG11_RemoteURL(t *testing.T) {
	requireGit(t)
	repo := initTestRepo(t)

	assert.Empty(t, RemoteURL(repo), "no remote should return empty")

	url := "https://example.com/repo.git"
	require.NoError(t, AddRemote(repo, url))

	got := RemoteURL(repo)
	assert.Equal(t, url, got)
}

// ---------------------------------------------------------------------------
// IsGitRepo 辅助验证
// ---------------------------------------------------------------------------

func TestIsGitRepo(t *testing.T) {
	requireGit(t)

	assert.False(t, IsGitRepo(t.TempDir()), "empty dir is not a git repo")

	repo := initTestRepo(t)
	assert.True(t, IsGitRepo(repo))
}

// ---------------------------------------------------------------------------
// RemoteHasCommit
// ---------------------------------------------------------------------------

func TestRemoteHasCommit(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	local := initTestRepo(t)
	require.NoError(t, AddRemote(local, remote))

	has, err := RemoteHasCommit(local)
	require.NoError(t, err)
	assert.False(t, has, "empty remote should have no commits")

	writeFile(t, local, "f.txt", "x")
	require.NoError(t, Commit(local, "first"))
	require.NoError(t, Push(local))

	has, err = RemoteHasCommit(local)
	require.NoError(t, err)
	assert.True(t, has)
}

// ---------------------------------------------------------------------------
// HasOriginRemote
// ---------------------------------------------------------------------------

func TestHasOriginRemote(t *testing.T) {
	requireGit(t)
	repo := initTestRepo(t)

	has, err := HasOriginRemote(repo)
	require.NoError(t, err)
	assert.False(t, has)

	require.NoError(t, AddRemote(repo, "https://example.com/repo.git"))
	has, err = HasOriginRemote(repo)
	require.NoError(t, err)
	assert.True(t, has)
}

// ---------------------------------------------------------------------------
// 验证 Pull 的完整 rebase 场景
// ---------------------------------------------------------------------------

func TestPull_NormalRebase(t *testing.T) {
	requireGit(t)
	remote := initBareRemote(t)

	// push initial commit from clone A
	a := filepath.Join(t.TempDir(), "a")
	run(t, "", "clone", remote, a)
	gitCfg(t, a)
	writeFile(t, a, "f.txt", "v1")
	run(t, a, "add", ".")
	run(t, a, "commit", "-m", "v1")
	run(t, a, "push", "-u", "origin", "HEAD")

	// clone B pulls, then A pushes a new commit
	b := filepath.Join(t.TempDir(), "b")
	run(t, "", "clone", remote, b)
	gitCfg(t, b)

	writeFile(t, a, "f.txt", "v2")
	run(t, a, "add", ".")
	run(t, a, "commit", "-m", "v2")
	run(t, a, "push")

	// B pulls
	err := Pull(b)
	require.NoError(t, err)

	content, _ := os.ReadFile(filepath.Join(b, "f.txt"))
	assert.Equal(t, "v2", strings.TrimSpace(string(content)))
}
