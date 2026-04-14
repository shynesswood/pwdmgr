package service_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 通用测试 helpers — 供 service 层所有测试文件共享
// ---------------------------------------------------------------------------

var testPassword = []byte("test-password-for-vault")

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping")
	}
}

func gitExec(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, string(out))
	return string(out)
}

func gitCfg(t *testing.T, dir string) {
	t.Helper()
	gitExec(t, dir, "config", "user.email", "test@test.com")
	gitExec(t, dir, "config", "user.name", "Test")
}

// initBareRemote 创建一个空的裸仓库，返回路径。
func initBareRemote(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	gitExec(t, "", "init", "--bare", dir)
	return dir
}

// freshDir 返回一个干净的临时目录（非 git repo）。
func freshDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// pushVaultToRemote 通过临时 clone 把一个 Vault 推送到裸仓库，模拟"另一台设备"。
func pushVaultToRemote(t *testing.T, remote string, password []byte, v *vault.Vault) {
	t.Helper()
	clone := filepath.Join(t.TempDir(), "helper-push")
	gitExec(t, "", "clone", remote, clone)
	gitCfg(t, clone)

	vaultPath := filepath.Join(clone, "vault.dat")
	require.NoError(t, storage.SaveVault(vaultPath, password, v))

	gitExec(t, clone, "add", ".")
	gitExec(t, clone, "commit", "-m", "push vault from helper")
	gitExec(t, clone, "push", "-u", "origin", "HEAD")
}

// updateRemoteVault 模拟"另一台设备"修改远程 vault：clone → 更新 → push。
func updateRemoteVault(t *testing.T, remote string, password []byte, fn func(v *vault.Vault)) {
	t.Helper()
	clone := filepath.Join(t.TempDir(), "helper-update")
	gitExec(t, "", "clone", remote, clone)
	gitCfg(t, clone)

	vaultPath := filepath.Join(clone, "vault.dat")
	v, err := storage.LoadVault(vaultPath, password)
	require.NoError(t, err)

	fn(v)

	require.NoError(t, storage.SaveVault(vaultPath, password, v))
	gitExec(t, clone, "add", ".")
	gitExec(t, clone, "commit", "-m", "update from other device")
	gitExec(t, clone, "push", "origin", "HEAD")
}

// loadLocalVault 从指定目录加载并解密 vault，用于验证测试结果。
func loadLocalVault(t *testing.T, repoRoot string, password []byte) *vault.Vault {
	t.Helper()
	vaultPath := filepath.Join(repoRoot, "vault.dat")
	v, err := storage.LoadVault(vaultPath, password)
	require.NoError(t, err)
	return v
}

// saveLocalVault 直接往目录写加密 vault 文件（不经过 service 层）。
func saveLocalVault(t *testing.T, repoRoot string, password []byte, v *vault.Vault) {
	t.Helper()
	vaultPath := filepath.Join(repoRoot, "vault.dat")
	require.NoError(t, storage.SaveVault(vaultPath, password, v))
}

// vaultFileExists 检查 vault.dat 文件是否存在。
func vaultFileExists(repoRoot string) bool {
	_, err := os.Stat(filepath.Join(repoRoot, "vault.dat"))
	return err == nil
}

// entryByID 在 entries 列表中按 ID 查找。
func entryByID(entries []vault.Entry, id string) *vault.Entry {
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i]
		}
	}
	return nil
}
