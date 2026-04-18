package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// GGA1 — URL 分类：ssh / scp / http(s) / 其他
func TestGGA1_IsSSHURL(t *testing.T) {
	cases := []struct {
		url  string
		want bool
	}{
		{"", false},
		{"   ", false},
		{"git@github.com:u/r.git", true},
		{"user@example.com:~/path/to/repo.git", true},
		{"ssh://git@github.com/u/r.git", true},
		{"SSH://git@github.com:2222/u/r.git", true},
		{"https://github.com/u/r.git", false},
		{"http://example.com/u/r.git", false},
		{"git://github.com/u/r.git", false},
		{"file:///tmp/bare.git", false},
		{"/tmp/bare.git", false},
		{"C:\\path\\to\\repo", false}, // Windows 路径不应被误判
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, isSSHURL(c.url), "isSSHURL(%q)", c.url)
	}
}

// GGA2 — URL 用户名提取
func TestGGA2_SSHUserFromURL(t *testing.T) {
	cases := []struct {
		url  string
		want string
	}{
		{"git@github.com:u/r.git", "git"},
		{"deploy@host:repo", "deploy"},
		{"ssh://gitolite@host:22/path", "gitolite"},
		{"", "git"},
		{"/local/path", "git"},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, sshUserFromURL(c.url), "sshUserFromURL(%q)", c.url)
	}
}

// GGA3 — 非 SSH URL 走 buildAuth 时返回 nil（保持默认行为）
func TestGGA3_BuildAuthSkipsNonSSH(t *testing.T) {
	for _, u := range []string{
		"",
		"https://github.com/u/r.git",
		"http://example.com/u/r.git",
		"file:///tmp/bare.git",
		"/tmp/bare.git",
	} {
		got, err := buildAuth(u)
		assert.NoErrorf(t, err, "buildAuth(%q) 不应返回错误", u)
		assert.Nilf(t, got, "buildAuth(%q) 非 SSH URL 应返回 nil", u)
	}
}
