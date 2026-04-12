package vault

import (
	"os"

	"pwdmgr/internal/config"
)

func LoadVaultPath(repoRoot string) (string, error) {
	vaultPath := config.VaultFilePath(repoRoot)

	if _, err := os.Stat(vaultPath); err != nil {
		return "", err
	}

	return vaultPath, nil
}
