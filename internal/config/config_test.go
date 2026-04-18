package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CFG-GC1 — NormalizeGitClient 规范化
// ---------------------------------------------------------------------------

func TestCFGGC1_NormalizeGitClient(t *testing.T) {
	cases := map[string]string{
		"":            GitClientExec,
		"exec":        GitClientExec,
		"EXEC":        GitClientExec,
		" system ":    GitClientExec,
		"cli":         GitClientExec,
		"go-git":      GitClientGoGit,
		"gogit":       GitClientGoGit,
		"Go_Git":      GitClientGoGit,
		"go-git-v5":   GitClientGoGit,
		"unknown-x":   GitClientExec,
	}
	for in, want := range cases {
		assert.Equal(t, want, NormalizeGitClient(in), "NormalizeGitClient(%q)", in)
	}
}

// ---------------------------------------------------------------------------
// CFG-GC2 — Load 读取 git_client 字段，缺失时填充默认
// ---------------------------------------------------------------------------

func TestCFGGC2_LoadReadsGitClient(t *testing.T) {
	type tc struct {
		name string
		json string
		want string
	}
	cases := []tc{
		{"missing", `{"repo_root":"/tmp/x"}`, GitClientExec},
		{"explicit-exec", `{"repo_root":"/tmp/x","git_client":"exec"}`, GitClientExec},
		{"explicit-gogit", `{"repo_root":"/tmp/x","git_client":"go-git"}`, GitClientGoGit},
		{"unknown-fallback", `{"repo_root":"/tmp/x","git_client":"whatever"}`, GitClientExec},
		{"alias-gogit", `{"repo_root":"/tmp/x","git_client":"GoGit"}`, GitClientGoGit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, DefaultConfigFileName)
			require.NoError(t, os.WriteFile(path, []byte(c.json), 0644))

			t.Setenv(EnvConfigPath, path)

			cfg, err := Load()
			require.NoError(t, err)
			assert.Equal(t, c.want, cfg.GitClient)
			assert.Equal(t, c.want, cfg.Snapshot().GitClient)
		})
	}
}

// ---------------------------------------------------------------------------
// CFG-SV1 — Save 写回三字段并可被 Load 读回
// ---------------------------------------------------------------------------

func TestCFGSV1_SaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultConfigFileName)
	t.Setenv(EnvConfigPath, path)

	cfg := &Config{
		RepoRoot:  "/tmp/vault",
		RemoteURL: "git@example.com:u/r.git",
		GitClient: "go-git",
	}
	require.NoError(t, cfg.Save())

	got, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/vault", got.RepoRoot)
	assert.Equal(t, "git@example.com:u/r.git", got.RemoteURL)
	assert.Equal(t, GitClientGoGit, got.GitClient)
	assert.Equal(t, path, got.Path())
}

// ---------------------------------------------------------------------------
// CFG-SV2 — Save 保留未知字段（避免用户自定义扩展被吞掉）
// ---------------------------------------------------------------------------

func TestCFGSV2_SavePreservesUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultConfigFileName)
	t.Setenv(EnvConfigPath, path)

	original := map[string]any{
		"repo_root":   "/old",
		"remote_url":  "",
		"git_client":  "exec",
		"theme":       "dark",
		"extra":       map[string]any{"foo": "bar"},
		"custom_flag": true,
	}
	raw, _ := json.MarshalIndent(original, "", "  ")
	require.NoError(t, os.WriteFile(path, raw, 0o600))

	loaded, err := Load()
	require.NoError(t, err)
	loaded.RepoRoot = "/new"
	loaded.RemoteURL = "https://example.com/r.git"
	loaded.GitClient = "go-git"
	require.NoError(t, loaded.Save())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var back map[string]any
	require.NoError(t, json.Unmarshal(data, &back))

	assert.Equal(t, "/new", back["repo_root"])
	assert.Equal(t, "https://example.com/r.git", back["remote_url"])
	assert.Equal(t, "go-git", back["git_client"])
	assert.Equal(t, "dark", back["theme"])
	assert.Equal(t, true, back["custom_flag"])
	extra, ok := back["extra"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "bar", extra["foo"])
}

// ---------------------------------------------------------------------------
// CFG-SV3 — 新建场景：resolvedPath 为空时回退到 ResolveConfigPath
// ---------------------------------------------------------------------------

func TestCFGSV3_SaveFirstTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", DefaultConfigFileName)
	t.Setenv(EnvConfigPath, path)

	cfg := &Config{
		RepoRoot:  "/absolute",
		RemoteURL: "",
		GitClient: "",
	}
	require.NoError(t, cfg.Save())

	info, err := os.Stat(path)
	require.NoError(t, err, "Save 应当能创建目录与文件")
	assert.Greater(t, info.Size(), int64(0))
	assert.Equal(t, path, cfg.Path(), "Save 成功后 resolvedPath 应被设置")

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "/absolute", loaded.RepoRoot)
	assert.Equal(t, GitClientExec, loaded.GitClient, "空 git_client 应被规范化为 exec")
}

// ---------------------------------------------------------------------------
// CFG-CP1 — CandidatePaths 顺序：executable > wd > user config dir
// ---------------------------------------------------------------------------

func TestCFGCP1_CandidatePathsOrder(t *testing.T) {
	t.Setenv(EnvConfigPath, "") // 清空环境变量，只看候选顺序

	paths := CandidatePaths()
	require.NotEmpty(t, paths)

	exeDir := executableDir()
	wd, _ := os.Getwd()
	userDir := userConfigDir()

	// 按新顺序：可执行目录（若存在）→ 工作目录（若存在）→ 用户配置目录（若存在）
	var want []string
	if exeDir != "" {
		want = append(want, filepath.Join(exeDir, DefaultConfigFileName))
	}
	if wd != "" {
		want = append(want, filepath.Join(wd, DefaultConfigFileName))
	}
	if userDir != "" {
		want = append(want, filepath.Join(userDir, DefaultConfigFileName))
	}
	assert.Equal(t, want, paths)
}

// ---------------------------------------------------------------------------
// CFG-CP2 — ResolveConfigPath 命中 wd 文件时优先于用户配置目录
// ---------------------------------------------------------------------------

func TestCFGCP2_ResolvePrefersWdOverUserDir(t *testing.T) {
	// 切到一个受控的临时目录，在其下放 pwdmgr.config.json
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv(EnvConfigPath, "")

	wdFile := filepath.Join(dir, DefaultConfigFileName)
	require.NoError(t, os.WriteFile(wdFile, []byte(`{"repo_root":"/x"}`), 0o600))

	got := ResolveConfigPath()
	assert.Equal(t, wdFile, got,
		"当 wd 下存在配置文件时应命中 wd，不应回退到用户配置目录")
}

// ---------------------------------------------------------------------------
// CFG-SV4 — Save 规范化 git_client，写入文件落地的是合法值
// ---------------------------------------------------------------------------

func TestCFGSV4_SaveNormalizesGitClient(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultConfigFileName)
	t.Setenv(EnvConfigPath, path)

	cfg := &Config{RepoRoot: "/x", GitClient: "GoGit"}
	require.NoError(t, cfg.Save())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var back map[string]any
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, "go-git", back["git_client"])
}
