package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 所有 go-git 后端测试直接实例化 backend，不依赖本机 git 命令；
// 远程仓库用 go-git 自身 PlainInit(bare=true) 创建，保证纯 Go 路径。
// ---------------------------------------------------------------------------

func newGoGit() goGitBackend { return goGitBackend{} }

func initBareGoGit(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	_, err := gogit.PlainInit(dir, true)
	require.NoError(t, err)
	return dir
}

func initGoGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, newGoGit().Init(dir))
	return dir
}

func writeGoGitFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
}

// seedRemoteGoGit 在 remote 上推一个包含 readme.txt 的初始提交。
func seedRemoteGoGit(t *testing.T, remote string) {
	t.Helper()
	b := newGoGit()
	dir := filepath.Join(t.TempDir(), "seed")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, b.Init(dir))
	require.NoError(t, b.AddRemote(dir, remote))
	writeGoGitFile(t, dir, "readme.txt", "hello")
	require.NoError(t, b.Commit(dir, "seed"))
	require.NoError(t, b.Push(dir))
}

// ---------------------------------------------------------------------------
// GG1 — Init + IsGitRepo
// ---------------------------------------------------------------------------

func TestGG1_InitAndIsGitRepo(t *testing.T) {
	b := newGoGit()

	assert.False(t, b.IsGitRepo(t.TempDir()), "空目录不应是 git repo")

	repo := initGoGitRepo(t)
	assert.True(t, b.IsGitRepo(repo))
}

// ---------------------------------------------------------------------------
// GG2 — AddRemote 首次 + 重复
// ---------------------------------------------------------------------------

func TestGG2_AddRemote_FirstThenOverride(t *testing.T) {
	b := newGoGit()
	repo := initGoGitRepo(t)

	require.NoError(t, b.AddRemote(repo, "https://old.example.com/repo.git"))
	assert.Equal(t, "https://old.example.com/repo.git", b.RemoteURL(repo))

	// 重复添加 → 更新 URL，不应报错
	newURL := "https://new.example.com/repo.git"
	require.NoError(t, b.AddRemote(repo, newURL))
	assert.Equal(t, newURL, b.RemoteURL(repo))
}

// ---------------------------------------------------------------------------
// GG3 — Commit 产生本地提交，CurrentBranch 可读
// ---------------------------------------------------------------------------

func TestGG3_Commit(t *testing.T) {
	b := newGoGit()
	repo := initGoGitRepo(t)

	writeGoGitFile(t, repo, "hello.txt", "world")
	require.NoError(t, b.Commit(repo, "test commit"))

	branch := b.CurrentBranch(repo)
	assert.Contains(t, []string{"main", "master"}, branch,
		"首次 commit 后当前分支应存在")
}

// ---------------------------------------------------------------------------
// GG4 — HasChanges 检测干净/修改/未跟踪文件
// ---------------------------------------------------------------------------

func TestGG4_HasChanges(t *testing.T) {
	b := newGoGit()
	repo := initGoGitRepo(t)

	writeGoGitFile(t, repo, "f.txt", "v1")
	require.NoError(t, b.Commit(repo, "init"))

	assert.False(t, b.HasChanges(repo), "干净仓库不应报告变更")

	writeGoGitFile(t, repo, "f.txt", "v2")
	assert.True(t, b.HasChanges(repo), "修改文件应报告变更")

	// 恢复后再测新文件
	b.RestoreFile(repo, "f.txt")
	writeGoGitFile(t, repo, "new.txt", "x")
	assert.True(t, b.HasChanges(repo), "新增未跟踪文件应报告变更")
}

// ---------------------------------------------------------------------------
// GG5 — RestoreFile 恢复已跟踪文件
// ---------------------------------------------------------------------------

func TestGG5_RestoreFile(t *testing.T) {
	b := newGoGit()
	repo := initGoGitRepo(t)

	writeGoGitFile(t, repo, "vault.dat", "original")
	require.NoError(t, b.Commit(repo, "add vault"))

	writeGoGitFile(t, repo, "vault.dat", "modified")
	data, _ := os.ReadFile(filepath.Join(repo, "vault.dat"))
	assert.Equal(t, "modified", string(data))

	b.RestoreFile(repo, "vault.dat")

	data, _ = os.ReadFile(filepath.Join(repo, "vault.dat"))
	assert.Equal(t, "original", string(data))
}

// ---------------------------------------------------------------------------
// GG6 — Push 首次推送到空 bare 远程
// ---------------------------------------------------------------------------

func TestGG6_Push_FirstTime(t *testing.T) {
	b := newGoGit()
	remote := initBareGoGit(t)
	repo := initGoGitRepo(t)

	require.NoError(t, b.AddRemote(repo, remote))

	writeGoGitFile(t, repo, "data.txt", "content")
	require.NoError(t, b.Commit(repo, "initial"))
	require.NoError(t, b.Push(repo))

	has, err := b.RemoteHasCommit(repo)
	require.NoError(t, err)
	assert.True(t, has, "push 后远程应至少存在一个 branch ref")
}

// ---------------------------------------------------------------------------
// GG7 — Pull 回退策略：本地刚 init + AddRemote，无 HEAD
// ---------------------------------------------------------------------------

func TestGG7_Pull_FallbackNoTrackingBranch(t *testing.T) {
	b := newGoGit()
	remote := initBareGoGit(t)
	seedRemoteGoGit(t, remote)

	local := initGoGitRepo(t)
	require.NoError(t, b.AddRemote(local, remote))

	require.NoError(t, b.Pull(local))

	_, statErr := os.Stat(filepath.Join(local, "readme.txt"))
	assert.NoError(t, statErr, "应拉取到远程文件")
}

// ---------------------------------------------------------------------------
// GG8 — Pull 正常 fast-forward 场景
// ---------------------------------------------------------------------------

func TestGG8_Pull_FastForward(t *testing.T) {
	b := newGoGit()
	remote := initBareGoGit(t)
	seedRemoteGoGit(t, remote)

	// 本地仓库 A：先通过 Pull 建立跟踪分支
	localA := initGoGitRepo(t)
	require.NoError(t, b.AddRemote(localA, remote))
	require.NoError(t, b.Pull(localA))

	// 本地仓库 B：做一次新提交并 push
	localB := initGoGitRepo(t)
	require.NoError(t, b.AddRemote(localB, remote))
	require.NoError(t, b.Pull(localB))
	writeGoGitFile(t, localB, "delta.txt", "new")
	require.NoError(t, b.Commit(localB, "delta"))
	require.NoError(t, b.Push(localB))

	// A 再 pull，拿到 delta
	require.NoError(t, b.Pull(localA))
	_, err := os.Stat(filepath.Join(localA, "delta.txt"))
	assert.NoError(t, err, "fast-forward pull 后应看到新文件")
}

// ---------------------------------------------------------------------------
// GG9 — HasOriginRemote / RemoteURL / RemoteHasCommit 组合
// ---------------------------------------------------------------------------

func TestGG9_RemoteMetadata(t *testing.T) {
	b := newGoGit()
	remote := initBareGoGit(t)
	repo := initGoGitRepo(t)

	has, err := b.HasOriginRemote(repo)
	require.NoError(t, err)
	assert.False(t, has)
	assert.Empty(t, b.RemoteURL(repo))

	require.NoError(t, b.AddRemote(repo, remote))

	has, err = b.HasOriginRemote(repo)
	require.NoError(t, err)
	assert.True(t, has)
	assert.Equal(t, remote, b.RemoteURL(repo))

	hasCommit, err := b.RemoteHasCommit(repo)
	require.NoError(t, err)
	assert.False(t, hasCommit, "空 bare 仓库无提交")

	writeGoGitFile(t, repo, "f.txt", "x")
	require.NoError(t, b.Commit(repo, "first"))
	require.NoError(t, b.Push(repo))

	hasCommit, err = b.RemoteHasCommit(repo)
	require.NoError(t, err)
	assert.True(t, hasCommit)
}

// ---------------------------------------------------------------------------
// GG10 — Dispatcher 顶层 API 经 SetBackend 切到 go-git 后可用
// ---------------------------------------------------------------------------

func TestGG10_TopLevelDispatchViaGoGit(t *testing.T) {
	t.Cleanup(func() { SetBackend(BackendExec) })

	SetBackend(BackendGoGit)
	require.Equal(t, BackendGoGit, CurrentBackend())

	remote := initBareGoGit(t)
	repo := t.TempDir()
	require.NoError(t, Init(repo))
	require.True(t, IsGitRepo(repo))

	require.NoError(t, AddRemote(repo, remote))
	writeGoGitFile(t, repo, "a.txt", "alpha")
	require.NoError(t, Commit(repo, "init"))
	require.NoError(t, Push(repo))

	hasCommit, err := RemoteHasCommit(repo)
	require.NoError(t, err)
	assert.True(t, hasCommit, "顶层 Push 经 go-git 后远程应有提交")
}
