package config

import (
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
