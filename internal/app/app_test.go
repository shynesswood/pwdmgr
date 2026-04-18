package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pwdmgr/internal/config"
	"pwdmgr/internal/git"
)

// prepareConfigEnv 指向一个临时目录下的 pwdmgr.config.json，保证测试互不影响。
func prepareConfigEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, config.DefaultConfigFileName)
	t.Setenv(config.EnvConfigPath, path)
	return path
}

// ---------------------------------------------------------------------------
// APP-UC1 — UpdateAppConfig 校验：repo_root 必填
// ---------------------------------------------------------------------------

func TestAPPUC1_UpdateAppConfig_RequiresRepoRoot(t *testing.T) {
	prepareConfigEnv(t)
	a := NewApp()

	_, err := a.UpdateAppConfig("   ", "", "exec")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "仓库路径")
}

// ---------------------------------------------------------------------------
// APP-UC2 — UpdateAppConfig 校验：必须绝对路径
// ---------------------------------------------------------------------------

func TestAPPUC2_UpdateAppConfig_RequiresAbsPath(t *testing.T) {
	prepareConfigEnv(t)
	a := NewApp()

	_, err := a.UpdateAppConfig("relative/path", "", "exec")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "绝对路径")
}

// ---------------------------------------------------------------------------
// APP-UC3 — UpdateAppConfig 校验：repo_root 不能是文件
// ---------------------------------------------------------------------------

func TestAPPUC3_UpdateAppConfig_RejectsFilePath(t *testing.T) {
	prepareConfigEnv(t)
	a := NewApp()

	file := filepath.Join(t.TempDir(), "not_a_dir")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o600))

	_, err := a.UpdateAppConfig(file, "", "exec")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "文件")
}

// ---------------------------------------------------------------------------
// APP-UC4 — UpdateAppConfig 正常流程：写盘、切换后端、返回 Snapshot
// ---------------------------------------------------------------------------

func TestAPPUC4_UpdateAppConfig_Success(t *testing.T) {
	t.Cleanup(func() { git.SetBackend(git.BackendExec) })

	cfgPath := prepareConfigEnv(t)
	a := NewApp()

	repoDir := t.TempDir()
	snap, err := a.UpdateAppConfig(repoDir, "git@example.com:u/r.git", "go-git")
	require.NoError(t, err)

	assert.Equal(t, repoDir, snap.RepoRoot)
	assert.Equal(t, "git@example.com:u/r.git", snap.RemoteURL)
	assert.Equal(t, config.GitClientGoGit, snap.GitClient)
	assert.Equal(t, cfgPath, snap.ConfigPath)

	assert.Equal(t, git.BackendGoGit, git.CurrentBackend(),
		"保存 git_client=go-git 后 git 包应切到 go-git 后端")

	// 文件应当真的落到磁盘
	_, statErr := os.Stat(cfgPath)
	assert.NoError(t, statErr)

	// 再改回 exec 验证回切
	snap, err = a.UpdateAppConfig(repoDir, "", "exec")
	require.NoError(t, err)
	assert.Equal(t, config.GitClientExec, snap.GitClient)
	assert.Equal(t, git.BackendExec, git.CurrentBackend())
}

// ---------------------------------------------------------------------------
// APP-UC5 — 未知 git_client 回退 exec
// ---------------------------------------------------------------------------

func TestAPPUC5_UpdateAppConfig_UnknownGitClientFallsBack(t *testing.T) {
	t.Cleanup(func() { git.SetBackend(git.BackendExec) })
	prepareConfigEnv(t)
	a := NewApp()

	repoDir := t.TempDir()
	snap, err := a.UpdateAppConfig(repoDir, "", "libgit2")
	require.NoError(t, err)
	assert.Equal(t, config.GitClientExec, snap.GitClient)
	assert.Equal(t, git.BackendExec, git.CurrentBackend())
}
