package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// BK1 — Normalize 规范化后端名
// ---------------------------------------------------------------------------

func TestBK1_Normalize(t *testing.T) {
	cases := map[string]string{
		"":            BackendExec,
		"exec":        BackendExec,
		"EXEC":        BackendExec,
		" system ":    BackendExec,
		"cli":         BackendExec,
		"go-git":      BackendGoGit,
		"GoGit":       BackendGoGit,
		"gogit":       BackendGoGit,
		"go_git":      BackendGoGit,
		"go-git-v5":   BackendGoGit,
		"garbage":     BackendExec, // 未知回退到默认
		"libgit2":     BackendExec,
	}
	for in, want := range cases {
		assert.Equal(t, want, Normalize(in), "Normalize(%q)", in)
	}
}

// ---------------------------------------------------------------------------
// BK2 — 默认后端是 exec
// ---------------------------------------------------------------------------

func TestBK2_DefaultBackendIsExec(t *testing.T) {
	// 为避免与其他测试互相影响，先显式重置
	t.Cleanup(func() { SetBackend(BackendExec) })
	SetBackend("")

	assert.Equal(t, BackendExec, CurrentBackend())
}

// ---------------------------------------------------------------------------
// BK3 — SetBackend 切换 exec / go-git
// ---------------------------------------------------------------------------

func TestBK3_SetBackend(t *testing.T) {
	t.Cleanup(func() { SetBackend(BackendExec) })

	SetBackend(BackendGoGit)
	assert.Equal(t, BackendGoGit, CurrentBackend())

	SetBackend(BackendExec)
	assert.Equal(t, BackendExec, CurrentBackend())

	// 未知值 → 回退默认
	SetBackend("bogus")
	assert.Equal(t, BackendExec, CurrentBackend())
}

// ---------------------------------------------------------------------------
// BK4 — SetBackendStrict 对未知值返回错误
// ---------------------------------------------------------------------------

func TestBK4_SetBackendStrict(t *testing.T) {
	t.Cleanup(func() { SetBackend(BackendExec) })

	require.NoError(t, SetBackendStrict("go-git"))
	assert.Equal(t, BackendGoGit, CurrentBackend())

	require.NoError(t, SetBackendStrict("exec"))
	assert.Equal(t, BackendExec, CurrentBackend())

	require.NoError(t, SetBackendStrict(""))
	assert.Equal(t, BackendExec, CurrentBackend())

	err := SetBackendStrict("bogus-backend")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus-backend")
}
