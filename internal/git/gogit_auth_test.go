package git

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
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

// ---------------------------------------------------------------------------
// GGA4 — SetSSHCredentials 设置一把存在的未加密私钥后 buildAuth 可成功返回 AuthMethod
// ---------------------------------------------------------------------------

func TestGGA4_ExplicitKeyIsUsed(t *testing.T) {
	dir := t.TempDir()
	keyPath := writeEd25519PrivateKey(t, dir)

	SetSSHCredentials(keyPath, "")
	t.Cleanup(func() { SetSSHCredentials("", "") })

	auth, err := buildAuth("git@github.com:u/r.git")
	require.NoError(t, err)
	assert.NotNil(t, auth, "配置了 ssh_key_path 后 buildAuth 应返回非 nil AuthMethod")
	assert.Equal(t, "ssh-public-keys", auth.Name())
}

// ---------------------------------------------------------------------------
// GGA5 — 配置了 ssh_key_path 但文件不存在，应返回明确错误
// ---------------------------------------------------------------------------

func TestGGA5_MissingKeyReturnsFriendlyError(t *testing.T) {
	SetSSHCredentials("/no/such/key/file", "")
	t.Cleanup(func() { SetSSHCredentials("", "") })

	auth, err := buildAuth("git@github.com:u/r.git")
	assert.Nil(t, auth)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不存在或不可读")
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// writeEd25519PrivateKey 生成一把未加密的 OpenSSH 格式 ed25519 私钥写到临时目录，
// 返回私钥文件路径。ed25519 是当下推荐算法，go-git 的 NewPublicKeysFromFile 能直接读。
func writeEd25519PrivateKey(t *testing.T, dir string) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	block, err := ssh.MarshalPrivateKey(priv, "pwdmgr-test")
	require.NoError(t, err)
	path := filepath.Join(dir, "id_ed25519")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(block), 0o600))
	return path
}
