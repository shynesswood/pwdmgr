package git

// 本文件专门处理 go-git 后端的 SSH 认证。
//
// 背景：
//   exec 后端会 fork 出系统 `git`，由它再启动 `ssh` 进程并自动继承：
//     - ~/.ssh/config
//     - ~/.ssh/known_hosts
//     - ssh-agent（SSH_AUTH_SOCK）
//     - 自动加载常见私钥（id_ed25519 / id_ecdsa / id_rsa 等）
//
//   而 go-git 的 ssh transport 默认只从 SSH_AUTH_SOCK 取 ssh-agent；
//   macOS 从 Finder / Launchpad 启动的 GUI 应用里 SSH_AUTH_SOCK 通常拿不到，
//   且 go-git 不会自动加载 ~/.ssh/id_xxx，因此握手时没有任何有效认证方式，
//   服务端（GitHub 等）直接 EOF 关闭，表现为 "ssh: handshake failed: EOF"。
//
// 修复：显式组装 SSH Auth，按以下顺序选一种：
//   1. ssh-agent（若 SSH_AUTH_SOCK 可连）
//   2. 常见无口令私钥：id_ed25519 > id_ecdsa > id_rsa
//
// HostKeyCallback 策略：
//   - 优先 knownhosts.New("~/.ssh/known_hosts")
//   - 若 known_hosts 不存在或解析失败，回退到 ssh.InsecureIgnoreHostKey()
//     （对齐本机 git 首次连接也不会直接失败的体验）

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	gssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// isSSHURL 判断一个 Git 远程地址是否需要走 SSH。
//
// 接受两种写法：
//   - "ssh://user@host[:port]/path"
//   - SCP 风格 "user@host:path"（Git 最常见的短写法）
//
// 明确排除 http(s):// 与 git://（各自走不同 transport）。
func isSSHURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	lower := strings.ToLower(u)
	switch {
	case strings.HasPrefix(lower, "ssh://"):
		return true
	case strings.HasPrefix(lower, "http://"),
		strings.HasPrefix(lower, "https://"),
		strings.HasPrefix(lower, "git://"),
		strings.HasPrefix(lower, "file://"):
		return false
	}
	// SCP 短写法：必须同时有 `@` 和 `:`，并且 `@` 在 `:` 前面
	at := strings.Index(u, "@")
	colon := strings.Index(u, ":")
	return at > 0 && colon > at
}

// sshUserFromURL 从 URL 中提取用户名；拿不到时默认 "git"。
func sshUserFromURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return "git"
	}
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if at := strings.Index(u, "@"); at > 0 {
		return u[:at]
	}
	return "git"
}

// buildAuth 按优先级返回 go-git 可用的 AuthMethod。
// 非 SSH URL 返回 (nil, nil)；调用方把 nil 直接传给 go-git（等价于默认）。
func buildAuth(remoteURL string) (transport.AuthMethod, error) {
	if !isSSHURL(remoteURL) {
		return nil, nil
	}
	user := sshUserFromURL(remoteURL)
	hostKeyCB := sshHostKeyCallback()

	// 1. 优先 ssh-agent（有 SSH_AUTH_SOCK 才尝试）
	if sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")); sock != "" {
		if auth, err := gssh.NewSSHAgentAuth(user); err == nil && auth != nil {
			if hostKeyCB != nil {
				auth.HostKeyCallback = hostKeyCB
			}
			return auth, nil
		}
	}

	// 2. 常见无口令私钥（按现代算法优先级）
	home, _ := os.UserHomeDir()
	if home != "" {
		for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
			keyPath := filepath.Join(home, ".ssh", name)
			if _, err := os.Stat(keyPath); err != nil {
				continue
			}
			auth, err := gssh.NewPublicKeysFromFile(user, keyPath, "")
			if err != nil {
				// 加密 key（有口令）或格式不兼容时跳过，继续尝试下一把
				continue
			}
			if hostKeyCB != nil {
				auth.HostKeyCallback = hostKeyCB
			}
			return auth, nil
		}
	}

	// 找不到任何凭据：返回 nil，让 go-git 默认行为触发（仍会失败，但错误栈完整）
	return nil, nil
}

// sshHostKeyCallback 返回用于校验远程 host key 的回调。
// 优先读 ~/.ssh/known_hosts；读取失败时回退 InsecureIgnoreHostKey。
func sshHostKeyCallback() ssh.HostKeyCallback {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		path := filepath.Join(home, ".ssh", "known_hosts")
		if _, err := os.Stat(path); err == nil {
			if cb, err := knownhosts.New(path); err == nil {
				return cb
			}
		}
	}
	return ssh.InsecureIgnoreHostKey()
}
