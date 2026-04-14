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

// Config 对应 pwdmgr.config.json 的字段。
type Config struct {
	RepoRoot  string `json:"repo_root"`
	RemoteURL string `json:"remote_url"`

	resolvedPath string `json:"-"`
}

// Snapshot 供界面展示（不含敏感逻辑，仅路径与元信息）。
type Snapshot struct {
	ConfigPath    string   `json:"config_path"`
	RepoRoot      string   `json:"repo_root"`
	RemoteURL     string   `json:"remote_url"`
	VaultFileName string   `json:"vault_file_name"`
	LoadError     string   `json:"load_error,omitempty"`
	SearchPaths   []string `json:"search_paths,omitempty"`
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

// CandidatePaths 返回按优先级排列的候选配置文件路径列表。
func CandidatePaths() []string {
	var paths []string

	if d := userConfigDir(); d != "" {
		paths = append(paths, filepath.Join(d, DefaultConfigFileName))
	}

	if d := executableDir(); d != "" {
		paths = append(paths, filepath.Join(d, DefaultConfigFileName))
	}

	if wd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(wd, DefaultConfigFileName))
	}

	return paths
}

// ResolveConfigPath 按优先级搜索配置文件并返回其绝对路径。
// 搜索顺序：环境变量 > 用户配置目录 > 可执行文件同级目录 > 当前工作目录。
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

	// 都不存在时，返回用户配置目录路径作为默认位置（方便报错提示用户在此创建）
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
			VaultFileName: VaultFileName,
			SearchPaths:   CandidatePaths(),
		}
	}
	return Snapshot{
		ConfigPath:    c.resolvedPath,
		RepoRoot:      c.RepoRoot,
		RemoteURL:     c.RemoteURL,
		VaultFileName: VaultFileName,
		SearchPaths:   CandidatePaths(),
	}
}
