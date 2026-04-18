package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 与仓库根目录同级的加密文件名（可在此统一修改）。
const VaultFileName = "vault.dat"

// 默认配置文件名。
const DefaultConfigFileName = "pwdmgr.config.json"

// AppName 用于拼接 OS 标准配置目录。
const AppName = "kPass"

// EnvConfigPath 环境变量名：若设置则指向任意路径的配置文件，便于多环境切换。
const EnvConfigPath = "PWDMGR_CONFIG"

// Git 后端可选值。与 internal/git 包内常量一致；此处单独列出避免 config 依赖 git 包。
const (
	GitClientExec    = "exec"
	GitClientGoGit   = "go-git"
	DefaultGitClient = GitClientExec
)

// Config 对应 pwdmgr.config.json 的字段。
type Config struct {
	RepoRoot  string `json:"repo_root"`
	RemoteURL string `json:"remote_url"`
	// GitClient 选择底层 Git 实现：
	//   - "exec"   (默认) 调用本机安装的 git 命令
	//   - "go-git" 使用 go-git 纯 Go 实现，不依赖本地 git
	// 配置缺省或取值未知时回退为 "exec"。
	GitClient string `json:"git_client,omitempty"`

	resolvedPath string `json:"-"`
}

// Snapshot 供界面展示（不含敏感逻辑，仅路径与元信息）。
type Snapshot struct {
	ConfigPath    string   `json:"config_path"`
	RepoRoot      string   `json:"repo_root"`
	RemoteURL     string   `json:"remote_url"`
	GitClient     string   `json:"git_client"`
	VaultFileName string   `json:"vault_file_name"`
	LoadError     string   `json:"load_error,omitempty"`
	SearchPaths   []string `json:"search_paths,omitempty"`
}

// NormalizeGitClient 将任意外部输入规范化为合法的后端名。
// 空串、未知值、大小写/下划线差异都会回退到默认值 "exec"。
func NormalizeGitClient(v string) string {
	s := strings.ToLower(strings.TrimSpace(v))
	s = strings.ReplaceAll(s, "_", "-")
	switch s {
	case "", GitClientExec, "system", "cli":
		return GitClientExec
	case GitClientGoGit, "gogit", "go-git-v5":
		return GitClientGoGit
	default:
		return DefaultGitClient
	}
}

// userConfigDir 返回 OS 标准用户配置目录下的应用子目录。
//   - macOS:   ~/Library/Application Support/kPass
//   - Windows: %AppData%/kPass
//   - Linux:   ~/.config/kPass
func userConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, AppName)
}

// executableDir 返回当前可执行文件所在的目录。
// macOS .app bundle 中可执行文件位于 Foo.app/Contents/MacOS/，
// 此时向上三层找到 .app 的父目录。
func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)

	// 检测 macOS .app bundle 结构：.../Foo.app/Contents/MacOS/Foo
	if filepath.Base(dir) == "MacOS" {
		contents := filepath.Dir(dir)
		if filepath.Base(contents) == "Contents" {
			appBundle := filepath.Dir(contents)
			if strings.HasSuffix(appBundle, ".app") {
				return filepath.Dir(appBundle)
			}
		}
	}
	return dir
}

// CandidatePaths 返回按搜索优先级排列的候选配置文件路径列表。
//
// 顺序（从高到低）：
//  1. 可执行文件同级目录 — 便携式部署 / Windows 放一起即可
//  2. 当前工作目录        — 开发模式 `wails dev` 自动命中项目根
//  3. 用户配置目录        — 标准安装位置（macOS/Windows/Linux）
//
// 环境变量 `PWDMGR_CONFIG` 优先级高于以上所有项，由 ResolveConfigPath 单独处理。
func CandidatePaths() []string {
	var paths []string

	if d := executableDir(); d != "" {
		paths = append(paths, filepath.Join(d, DefaultConfigFileName))
	}

	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, DefaultConfigFileName))
	}

	if d := userConfigDir(); d != "" {
		paths = append(paths, filepath.Join(d, DefaultConfigFileName))
	}

	return paths
}

// ResolveConfigPath 按优先级搜索配置文件并返回其绝对路径。
// 搜索顺序：环境变量 > 可执行文件同级目录 > 当前工作目录 > 用户配置目录。
//
// 当所有候选均不存在时，返回 **用户配置目录** 下的默认路径作为新建位置 ——
// 搜索优先级偏向就近查找已有文件，但新建时仍倾向标准可写位置（避免把
// macOS `.app` 内部或 `/` 当成首次创建目录），这能保证首次 `Save()` 不失败。
func ResolveConfigPath() string {
	if p := strings.TrimSpace(os.Getenv(EnvConfigPath)); p != "" {
		if filepath.IsAbs(p) {
			return filepath.Clean(p)
		}
		wd, err := os.Getwd()
		if err != nil {
			return filepath.Clean(p)
		}
		return filepath.Clean(filepath.Join(wd, p))
	}

	for _, p := range CandidatePaths() {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	if d := userConfigDir(); d != "" {
		return filepath.Join(d, DefaultConfigFileName)
	}
	return DefaultConfigFileName
}

// VaultFilePath 返回加密库文件在磁盘上的完整路径。
func VaultFilePath(repoRoot string) string {
	return filepath.Join(repoRoot, VaultFileName)
}

// Load 读取并解析配置文件；repo_root 必填，remote_url 可留空（仅本地时）。
func Load() (*Config, error) {
	path := ResolveConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w", err)
	}
	c.RepoRoot = strings.TrimSpace(c.RepoRoot)
	c.RemoteURL = strings.TrimSpace(c.RemoteURL)
	c.GitClient = NormalizeGitClient(c.GitClient)
	if c.RepoRoot == "" {
		return nil, fmt.Errorf("repo_root 不能为空")
	}
	c.resolvedPath = path
	return &c, nil
}

// Snapshot 生成当前配置的只读视图。
func (c *Config) Snapshot() Snapshot {
	if c == nil {
		return Snapshot{
			ConfigPath:    ResolveConfigPath(),
			GitClient:     DefaultGitClient,
			VaultFileName: VaultFileName,
			SearchPaths:   CandidatePaths(),
		}
	}
	return Snapshot{
		ConfigPath:    c.resolvedPath,
		RepoRoot:      c.RepoRoot,
		RemoteURL:     c.RemoteURL,
		GitClient:     NormalizeGitClient(c.GitClient),
		VaultFileName: VaultFileName,
		SearchPaths:   CandidatePaths(),
	}
}

// Path 返回本次 Load 时解析到的配置文件路径；未 Load 过时为空串。
func (c *Config) Path() string {
	if c == nil {
		return ""
	}
	return c.resolvedPath
}

// ResolvedOrCandidatePath 返回将要用于 Save 的目标路径：
// 优先使用 Load 时解析的路径，否则走候选优先级（用户配置目录）。
func (c *Config) ResolvedOrCandidatePath() string {
	if c != nil && c.resolvedPath != "" {
		return c.resolvedPath
	}
	return ResolveConfigPath()
}

// Save 把 `repo_root / remote_url / git_client` 三个字段写回磁盘。
//
// 为了不丢失未来新增的字段，或用户在 json 中保留的自定义字段，Save 会：
//  1. 先把现有文件读成通用 map
//  2. 覆盖这三个字段
//  3. 以 2 空格缩进的 JSON 原子写回（先写 .tmp 再 rename）
//
// 目标路径：若曾通过 Load 成功解析过则沿用该路径；否则按 CandidatePaths 优先级，
// 通常落到用户配置目录（macOS: ~/Library/Application Support/kPass/...）。
func (c *Config) Save() error {
	if c == nil {
		return fmt.Errorf("config 未初始化")
	}
	path := c.ResolvedOrCandidatePath()
	if path == "" {
		return fmt.Errorf("无法确定配置文件写入路径")
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("创建配置目录失败: %w", err)
		}
	}

	payload := map[string]any{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &payload) // 尽力保留已有字段；解析失败则退化为空 map
	}
	payload["repo_root"] = strings.TrimSpace(c.RepoRoot)
	payload["remote_url"] = strings.TrimSpace(c.RemoteURL)
	payload["git_client"] = NormalizeGitClient(c.GitClient)

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	out = append(out, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("替换配置文件失败: %w", err)
	}
	c.resolvedPath = path
	return nil
}
