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

// 默认配置文件名，放在「当前工作目录」下（开发时一般为项目根目录）。
const DefaultConfigFileName = "pwdmgr.config.json"

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
	ConfigPath    string `json:"config_path"`
	RepoRoot      string `json:"repo_root"`
	RemoteURL     string `json:"remote_url"`
	VaultFileName string `json:"vault_file_name"`
	LoadError     string `json:"load_error,omitempty"`
}

// ResolveConfigPath 返回将读取的配置文件绝对路径。
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
	wd, err := os.Getwd()
	if err != nil {
		return DefaultConfigFileName
	}
	return filepath.Join(wd, DefaultConfigFileName)
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
		}
	}
	return Snapshot{
		ConfigPath:    c.resolvedPath,
		RepoRoot:      c.RepoRoot,
		RemoteURL:     c.RemoteURL,
		VaultFileName: VaultFileName,
	}
}
