package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pwdmgr/internal/config"
	"pwdmgr/internal/git"
	"pwdmgr/internal/service"
	"pwdmgr/internal/vault"
)

// App struct
type App struct {
	ctx context.Context
	cfg *config.Config
	// cfgErr 记录最近一次加载配置的错误（Reload 成功后会清空）。
	cfgErr error
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// Startup 启动时读取工作目录下的 pwdmgr.config.json（或 PWDMGR_CONFIG）。
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg, a.cfgErr = config.Load()
	a.applyGitBackend()
}

// applyGitBackend 根据当前配置切换 git 包的底层后端；配置未加载时回退默认。
func (a *App) applyGitBackend() {
	if a.cfg != nil {
		git.SetBackend(a.cfg.GitClient)
		return
	}
	git.SetBackend(config.DefaultGitClient)
}

func (a *App) activeConfig() (*config.Config, error) {
	if a.cfgErr != nil {
		return nil, fmt.Errorf("配置文件无效: %w", a.cfgErr)
	}
	if a.cfg == nil {
		return nil, fmt.Errorf("配置未加载")
	}
	return a.cfg, nil
}

// GetAppConfig 返回当前配置快照（含配置文件路径与加载错误说明）。
func (a *App) GetAppConfig() config.Snapshot {
	if a.cfgErr != nil {
		return config.Snapshot{
			ConfigPath:    config.ResolveConfigPath(),
			GitClient:     config.DefaultGitClient,
			VaultFileName: config.VaultFileName,
			LoadError:     a.cfgErr.Error(),
			SearchPaths:   config.CandidatePaths(),
		}
	}
	if a.cfg != nil {
		return a.cfg.Snapshot()
	}
	return config.Snapshot{
		ConfigPath:    config.ResolveConfigPath(),
		GitClient:     config.DefaultGitClient,
		VaultFileName: config.VaultFileName,
		SearchPaths:   config.CandidatePaths(),
	}
}

// ReloadConfig 重新从磁盘读取配置（修改 pwdmgr.config.json 后可调用）。
func (a *App) ReloadConfig() error {
	cfg, err := config.Load()
	a.cfg = cfg
	a.cfgErr = err
	a.applyGitBackend()
	return err
}

// UpdateAppConfig 在 UI 中编辑仓库路径 / 远程 URL / git_client 后，把最新值
// 写回磁盘 pwdmgr.config.json，并同步内存 cfg 与 git 后端。
//
// 校验规则：
//   - repo_root：必填、必须是绝对路径；若已存在则必须是目录（不能是普通文件）
//   - remote_url：允许为空（仅本地使用），非空时 TrimSpace
//   - gitClient：Normalize 到 exec / go-git，未知值回退默认
//
// 返回保存后的最新 Snapshot 供前端刷新。
func (a *App) UpdateAppConfig(repoRoot, remoteURL, gitClient string) (config.Snapshot, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	remoteURL = strings.TrimSpace(remoteURL)
	gitClient = config.NormalizeGitClient(gitClient)

	if repoRoot == "" {
		return a.GetAppConfig(), fmt.Errorf("仓库路径不能为空")
	}
	if !filepath.IsAbs(repoRoot) {
		return a.GetAppConfig(), fmt.Errorf("仓库路径必须是绝对路径（当前：%s）", repoRoot)
	}
	if info, err := os.Stat(repoRoot); err == nil && !info.IsDir() {
		return a.GetAppConfig(), fmt.Errorf("仓库路径指向的是文件而不是目录：%s", repoRoot)
	}

	// 以已加载的 cfg 为基础保留 resolvedPath 等内部状态；首次运行时 cfg 可能为 nil。
	cfg := a.cfg
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.RepoRoot = repoRoot
	cfg.RemoteURL = remoteURL
	cfg.GitClient = gitClient

	if err := cfg.Save(); err != nil {
		return a.GetAppConfig(), err
	}

	// 写完重新 Load，拿到规范化的 resolvedPath 并确保文件内容自洽。
	loaded, err := config.Load()
	if err != nil {
		// 兜底：保存成功但重新读取失败（极少见），保持内存 cfg 不丢。
		a.cfg = cfg
		a.cfgErr = err
		a.applyGitBackend()
		return a.GetAppConfig(), err
	}
	a.cfg = loaded
	a.cfgErr = nil
	a.applyGitBackend()
	return a.cfg.Snapshot(), nil
}

func (a *App) GetRepoStatus() (service.RepoStatus, error) {
	c, err := a.activeConfig()
	if err != nil {
		return service.RepoStatus{}, err
	}
	return service.GetRepoStatus(c.RepoRoot)
}

func (a *App) Pull() error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.PullVault(c.RepoRoot)
}

func (a *App) Push() error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.PushVault(c.RepoRoot)
}

func (a *App) Sync(password string) error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.SyncVault(c.RepoRoot, []byte(password))
}

func (a *App) BindRepo(password string) error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	if strings.TrimSpace(c.RemoteURL) == "" {
		return fmt.Errorf("请先在配置文件中填写 remote_url")
	}
	return service.BindRemoteRepo(c.RepoRoot, c.RemoteURL, []byte(password))
}

func (a *App) InitLocalVault(password string) error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.InitLocalVault(c.RepoRoot, []byte(password))
}

func (a *App) ListVaultEntries(password, spaceID string) ([]vault.Entry, error) {
	c, err := a.activeConfig()
	if err != nil {
		return nil, err
	}
	return service.ListEntries(c.RepoRoot, []byte(password), spaceID)
}

func (a *App) AddVaultEntry(password, spaceID, name, username, entryPassword, note string, tags []string) error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.AddEntry(c.RepoRoot, []byte(password), spaceID, name, username, entryPassword, note, tags)
}

func (a *App) UpdateVaultEntry(password string, entry vault.Entry) error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.UpdateEntry(c.RepoRoot, []byte(password), entry)
}

func (a *App) DeleteVaultEntry(password, id string) error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.DeleteEntry(c.RepoRoot, []byte(password), id)
}

// MoveVaultEntries 批量把指定条目迁到目标空间，返回实际移动数量。
// ids 长度为 1 时也可用于单条移动场景。
func (a *App) MoveVaultEntries(password, targetSpaceID string, ids []string) (int, error) {
	c, err := a.activeConfig()
	if err != nil {
		return 0, err
	}
	return service.MoveEntries(c.RepoRoot, []byte(password), ids, targetSpaceID)
}

// ---------------------------------------------------------------------------
// 空间管理
// ---------------------------------------------------------------------------

func (a *App) ListVaultSpaces(password string) ([]vault.Space, error) {
	c, err := a.activeConfig()
	if err != nil {
		return nil, err
	}
	return service.ListSpaces(c.RepoRoot, []byte(password))
}

func (a *App) CreateVaultSpace(password, name string) (vault.Space, error) {
	c, err := a.activeConfig()
	if err != nil {
		return vault.Space{}, err
	}
	return service.CreateSpace(c.RepoRoot, []byte(password), name)
}

func (a *App) RenameVaultSpace(password, id, name string) error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.RenameSpace(c.RepoRoot, []byte(password), id, name)
}

func (a *App) DeleteVaultSpace(password, id string) error {
	c, err := a.activeConfig()
	if err != nil {
		return err
	}
	return service.DeleteSpace(c.RepoRoot, []byte(password), id)
}
