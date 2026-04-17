package git

import (
	"fmt"
	"strings"
	"sync"
)

// Backend 定义 pwdmgr 依赖的 Git 操作抽象。
// 目前有两个实现：
//   - execBackend：基于系统 `git` 命令（依赖本地安装 Git）
//   - goGitBackend：基于 github.com/go-git/go-git/v5 纯 Go 实现
type Backend interface {
	Clone(repoURL, path string) error
	Init(path string) error
	Pull(path string) error
	Commit(path, message string) error
	Push(path string) error
	HasChanges(path string) bool
	AddRemote(path, repoURL string) error
	IsGitRepo(path string) bool
	RemoteHasCommit(path string) (bool, error)
	HasOriginRemote(path string) (bool, error)
	RestoreFile(repoRoot, fileName string)
	CurrentBranch(path string) string
	RemoteURL(path string) string
	Name() string
}

const (
	BackendExec  = "exec"
	BackendGoGit = "go-git"
)

var (
	backendMu      sync.RWMutex
	activeBackend  Backend = &execBackend{}
	defaultBackend         = BackendExec
)

// Normalize 将外部传入的字符串规范化为受支持的后端名。
// 空值、未知值回退到默认后端（exec）。
func Normalize(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "_", "-")
	switch s {
	case "", BackendExec, "system", "cli":
		return BackendExec
	case BackendGoGit, "gogit", "go-git-v5":
		return BackendGoGit
	default:
		return defaultBackend
	}
}

// SetBackend 按名称切换当前生效的 Git 后端。
// 空/未知名称会回退到默认值，不返回错误。
func SetBackend(name string) {
	backendMu.Lock()
	defer backendMu.Unlock()
	switch Normalize(name) {
	case BackendGoGit:
		activeBackend = &goGitBackend{}
	default:
		activeBackend = &execBackend{}
	}
}

// SetBackendStrict 与 SetBackend 类似，但当 name 指向未知后端时返回错误。
func SetBackendStrict(name string) error {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "_", "-")
	switch s {
	case "", BackendExec, "system", "cli":
		SetBackend(BackendExec)
		return nil
	case BackendGoGit, "gogit", "go-git-v5":
		SetBackend(BackendGoGit)
		return nil
	default:
		return fmt.Errorf("未知 git 后端: %q（可选值: %s / %s）", name, BackendExec, BackendGoGit)
	}
}

// CurrentBackend 返回当前生效的后端名。
func CurrentBackend() string {
	backendMu.RLock()
	defer backendMu.RUnlock()
	return activeBackend.Name()
}

func current() Backend {
	backendMu.RLock()
	defer backendMu.RUnlock()
	return activeBackend
}

// ---------------------------------------------------------------------------
// 顶层公共 API：保持与旧版一致的签名，内部分发到当前后端。
// ---------------------------------------------------------------------------

func Clone(repoURL, path string) error  { return current().Clone(repoURL, path) }
func Init(path string) error            { return current().Init(path) }
func Pull(path string) error            { return current().Pull(path) }
func Commit(path, message string) error { return current().Commit(path, message) }
func Push(path string) error            { return current().Push(path) }
func HasChanges(path string) bool       { return current().HasChanges(path) }
func AddRemote(path, repoURL string) error {
	return current().AddRemote(path, repoURL)
}
func IsGitRepo(path string) bool                { return current().IsGitRepo(path) }
func RemoteHasCommit(path string) (bool, error) { return current().RemoteHasCommit(path) }
func HasOriginRemote(path string) (bool, error) { return current().HasOriginRemote(path) }
func RestoreFile(repoRoot, fileName string)     { current().RestoreFile(repoRoot, fileName) }
func CurrentBranch(path string) string          { return current().CurrentBranch(path) }
func RemoteURL(path string) string              { return current().RemoteURL(path) }
