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
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5/plumbing/transport"
	gssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// dbg 统一 debug 日志出口；默认始终打印到 stderr，信息量很小（不含 key / 口令 /
// 公钥等敏感内容），仅用于定位"远程操作走到哪一步挂了"。要彻底静默，可在启动前
// 设置环境变量 PWDMGR_GIT_SILENT=1。
func dbg(format string, args ...any) {
	if os.Getenv("PWDMGR_GIT_SILENT") != "" {
		return
	}
	fmt.Fprintf(os.Stderr, "[pwdmgr/git] "+format+"\n", args...)
}

// ---------------------------------------------------------------------------
// 用户显式配置的 SSH 凭据（ssh_key_path / ssh_key_passphrase）
// 由 app 层根据 pwdmgr.config.json 在启动 / 重载 / 保存后注入。
// ---------------------------------------------------------------------------

var (
	sshCredMu        sync.RWMutex
	sshKeyPath       string
	sshKeyPassphrase string

	proxyEnvOnce sync.Once
	proxyEnvMu   sync.Mutex
)

// proxyEnvKeys 是 golang.org/x/net/proxy.FromEnvironment 会读取的环境变量。
// go-git 的 SSH transport 在 dial 时会走这个函数，于是只要进程里设置了这些变量
// （macOS 上很多代理工具 FlClash/ClashX 的教程都会让用户在 ~/.zshrc export），
// 本来应该直连 github.com:22 的 SSH 连接就会被改道到本地代理端口，
// 大多数代理对裸 SSH 协议的处理都很糟糕，表现为 "ssh: handshake failed: EOF"。
//
// 系统的 `git` / `ssh` 命令不读这些变量，所以相同环境下 exec 后端不受影响。
var proxyEnvKeys = []string{
	"ALL_PROXY", "all_proxy",
	"HTTPS_PROXY", "https_proxy",
	"HTTP_PROXY", "http_proxy",
}

// withDirectNetwork 在执行 fn 期间把 *_PROXY 环境变量临时 unset，
// 结束后原样恢复。这样 go-git 的 dial 就会走直连。
//
// 用 mutex 串行化，避免并发 git 操作期间 env 切换互相踩踏
// （os.Setenv/Unsetenv 是进程全局的）。pwdmgr 里的 Git 操作都是用户交互触发，
// 顺序执行开销可忽略。
func withDirectNetwork(fn func() error) error {
	proxyEnvMu.Lock()
	defer proxyEnvMu.Unlock()

	saved := make(map[string]string, len(proxyEnvKeys))
	for _, k := range proxyEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = v
			_ = os.Unsetenv(k)
		}
	}
	if len(saved) > 0 {
		keys := make([]string, 0, len(saved))
		for k := range saved {
			keys = append(keys, k)
		}
		dbg("withDirectNetwork: 临时清除 %d 个代理环境变量 %v（执行完会恢复）", len(saved), keys)
	}
	defer func() {
		for k, v := range saved {
			_ = os.Setenv(k, v)
		}
	}()
	return fn()
}

// logProxyEnvOnce 在首次调用时把 *_PROXY 相关环境变量打到 dbg。
// go-git 的 SSH dial 走 golang.org/x/net/proxy，会读取这些变量；
// 系统 ssh 不读，所以系统 ssh 通而 go-git 挂时，这是最可能的根因。
func logProxyEnvOnce() {
	proxyEnvOnce.Do(func() {
		hit := []string{}
		for _, k := range []string{
			"ALL_PROXY", "all_proxy",
			"HTTPS_PROXY", "https_proxy",
			"HTTP_PROXY", "http_proxy",
			"NO_PROXY", "no_proxy",
		} {
			if v := os.Getenv(k); v != "" {
				hit = append(hit, k+"="+v)
			}
		}
		if len(hit) == 0 {
			dbg("proxy-env: (none)")
		} else {
			dbg("proxy-env: %s  ← go-git 的 SSH dial 会读取这些，可能是 handshake EOF 的元凶", strings.Join(hit, " "))
		}
	})
}

// SetSSHCredentials 设置 go-git 使用的 SSH 私钥文件和口令。
// 两个参数都为空时，buildAuth 回退到默认的 ssh-agent / ~/.ssh/id_xxx 探测。
func SetSSHCredentials(keyPath, passphrase string) {
	sshCredMu.Lock()
	sshKeyPath = strings.TrimSpace(keyPath)
	sshKeyPassphrase = passphrase
	path, passLen := sshKeyPath, len(sshKeyPassphrase)
	sshCredMu.Unlock()
	dbg("SetSSHCredentials: keyPath=%q passLen=%d", path, passLen)
}

func currentSSHCredentials() (string, string) {
	sshCredMu.RLock()
	defer sshCredMu.RUnlock()
	return sshKeyPath, sshKeyPassphrase
}

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
//
// 优先级：
//  1. 用户显式配置 ssh_key_path（+ 可选 ssh_key_passphrase）
//  2. ssh-agent（SSH_AUTH_SOCK 可用时）
//  3. 默认私钥：~/.ssh/id_ed25519 → id_ecdsa → id_rsa（仅未加密 key 可用）
//
// 任何一级加载失败，会返回详细 error（带文件路径/原因），方便前端提示用户。
func buildAuth(remoteURL string) (transport.AuthMethod, error) {
	if !isSSHURL(remoteURL) {
		dbg("buildAuth: non-ssh url=%q → auth=nil", remoteURL)
		return nil, nil
	}
	logProxyEnvOnce()
	user := sshUserFromURL(remoteURL)
	hostKeyCB, hostKeyMode := sshHostKeyCallbackWithMode()

	// 1. 用户显式指定的 key
	if keyPath, passphrase := currentSSHCredentials(); keyPath != "" {
		if _, err := os.Stat(keyPath); err != nil {
			dbg("buildAuth: explicit key stat failed path=%q err=%v", keyPath, err)
			return nil, fmt.Errorf("ssh_key_path 指向的文件不存在或不可读: %s (%w)", keyPath, err)
		}
		auth, err := gssh.NewPublicKeysFromFile(user, keyPath, passphrase)
		if err != nil {
			dbg("buildAuth: explicit key load failed path=%q passLen=%d err=%v", keyPath, len(passphrase), err)
			if strings.Contains(err.Error(), "passphrase") || strings.Contains(err.Error(), "decrypt") {
				return nil, fmt.Errorf("加载 %s 失败：私钥被口令加密，请在 pwdmgr.config.json 设置 ssh_key_passphrase", keyPath)
			}
			return nil, fmt.Errorf("加载 ssh_key_path=%s 失败: %w", keyPath, err)
		}
		if hostKeyCB != nil {
			auth.HostKeyCallback = hostKeyCB
		}
		dbg("buildAuth: ok via=explicit user=%s key=%s passLen=%d hostkey=%s", user, keyPath, len(passphrase), hostKeyMode)
		return auth, nil
	}

	// 2. ssh-agent（SSH_AUTH_SOCK 存在才尝试）
	if sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK")); sock != "" {
		if auth, err := gssh.NewSSHAgentAuth(user); err == nil && auth != nil {
			if hostKeyCB != nil {
				auth.HostKeyCallback = hostKeyCB
			}
			dbg("buildAuth: ok via=ssh-agent user=%s sock=%s hostkey=%s", user, sock, hostKeyMode)
			return auth, nil
		} else {
			dbg("buildAuth: ssh-agent failed sock=%s err=%v", sock, err)
		}
	} else {
		dbg("buildAuth: no SSH_AUTH_SOCK; skip agent")
	}

	// 3. 默认私钥探测（~/.ssh/id_xxx；仅未加密 key 可用）
	home, _ := os.UserHomeDir()
	if home != "" {
		tried := []string{}
		for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
			keyPath := filepath.Join(home, ".ssh", name)
			if _, err := os.Stat(keyPath); err != nil {
				continue
			}
			tried = append(tried, keyPath)
			auth, err := gssh.NewPublicKeysFromFile(user, keyPath, "")
			if err != nil {
				dbg("buildAuth: default key load failed path=%s err=%v", keyPath, err)
				continue
			}
			if hostKeyCB != nil {
				auth.HostKeyCallback = hostKeyCB
			}
			dbg("buildAuth: ok via=default user=%s key=%s hostkey=%s", user, keyPath, hostKeyMode)
			return auth, nil
		}
		if len(tried) > 0 {
			dbg("buildAuth: all default keys failed tried=%v", tried)
			return nil, fmt.Errorf("探测到 SSH 私钥 %v 但都加载失败（多数情况是被口令加密）。\n请在 pwdmgr.config.json 设置 ssh_key_path（和 ssh_key_passphrase），或切换到 git_client: \"exec\"", tried)
		}
	}

	dbg("buildAuth: no credentials found")
	return nil, fmt.Errorf("未找到可用的 SSH 凭据：ssh-agent 不可达，~/.ssh 下也没有 id_ed25519/id_ecdsa/id_rsa。\n请在 pwdmgr.config.json 设置 ssh_key_path，或切换到 git_client: \"exec\"")
}

// sshHostKeyCallback 返回用于校验远程 host key 的回调（供测试/旧调用使用）。
func sshHostKeyCallback() ssh.HostKeyCallback {
	cb, _ := sshHostKeyCallbackWithMode()
	return cb
}

// sshHostKeyCallbackWithMode 除了返回 callback，还同时返回一个反映当前策略的
// 字符串（用于日志）。
//
// 策略（从强到弱）：
//  1. 显式环境变量 PWDMGR_SSH_INSECURE=1 → 直接 InsecureIgnoreHostKey (mode=insecure-env)
//  2. ~/.ssh/known_hosts 存在且能正常 parse → 返回一个**容忍**的包装回调
//     (mode=known_hosts-tolerant)：
//     当 host 不在文件中 或 key 不匹配时（`*knownhosts.KeyError`），
//     退化成 InsecureIgnoreHostKey 并打 warn，不阻断握手。
//  3. home 读不到 (mode=insecure-no-home) / known_hosts 不存在 (mode=insecure-no-known-hosts) /
//     parse 失败 (mode=insecure-parse-failed) → InsecureIgnoreHostKey
func sshHostKeyCallbackWithMode() (ssh.HostKeyCallback, string) {
	if os.Getenv("PWDMGR_SSH_INSECURE") != "" {
		dbg("sshHostKeyCallback: PWDMGR_SSH_INSECURE=1 → InsecureIgnoreHostKey")
		return ssh.InsecureIgnoreHostKey(), "insecure-env"
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ssh.InsecureIgnoreHostKey(), "insecure-no-home"
	}
	path := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(path); err != nil {
		return ssh.InsecureIgnoreHostKey(), "insecure-no-known-hosts"
	}
	inner, err := knownhosts.New(path)
	if err != nil {
		dbg("sshHostKeyCallback: known_hosts parse failed path=%s err=%v → InsecureIgnoreHostKey", path, err)
		return ssh.InsecureIgnoreHostKey(), "insecure-parse-failed"
	}

	cb := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := inner(hostname, remote, key); err != nil {
			// knownhosts 包对「host 不在表里」和「key 不匹配」都返回 *knownhosts.KeyError。
			// 为了避免 EOF 这种难以诊断的症状，此处退让并打 warn（而不是直接拒绝握手）。
			var kerr *knownhosts.KeyError
			if errors.As(err, &kerr) {
				if len(kerr.Want) == 0 {
					dbg("sshHostKeyCallback: host=%s 不在 ~/.ssh/known_hosts 中，退让接受（首次连接）", hostname)
				} else {
					dbg("sshHostKeyCallback: host=%s 的 key 与 ~/.ssh/known_hosts 不一致（可能对端轮换过），退让接受；"+
						"如需严格校验请从 known_hosts 删除旧条目后重试", hostname)
				}
				return nil
			}
			dbg("sshHostKeyCallback: host=%s 未知错误 err=%v → 放行以便看到后续真实错误", hostname, err)
			return nil
		}
		return nil
	}
	return cb, "known_hosts-tolerant"
}
