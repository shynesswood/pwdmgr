package service

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"pwdmgr/internal/config"
	"pwdmgr/internal/storage"
	"pwdmgr/internal/vault"
)

func loadVault(repoRoot string, password []byte) (*vault.Vault, error) {
	path := config.VaultFilePath(repoRoot)
	return storage.LoadVault(path, password)
}

func saveVault(repoRoot string, password []byte, v *vault.Vault) error {
	path := config.VaultFilePath(repoRoot)
	return storage.SaveVault(path, password, v)
}

func isNoVaultFile(err error) bool {
	return err != nil && (errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "数据文件不存在"))
}

// ListEntries 解密并返回所有条目（按名称排序）。尚无 vault 文件时返回空列表。
func ListEntries(repoRoot string, password []byte) ([]vault.Entry, error) {
	v, err := loadVault(repoRoot, password)
	if err != nil {
		if isNoVaultFile(err) {
			return []vault.Entry{}, nil
		}
		return nil, err
	}
	out := append([]vault.Entry{}, v.Entries...)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// AddEntry 新增一条记录并写回磁盘。若尚无加密库文件则自动创建空库再写入。
func AddEntry(repoRoot string, password []byte, name, username, entryPassword, note string, tags []string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("名称不能为空")
	}
	v, err := loadVault(repoRoot, password)
	if err != nil {
		if isNoVaultFile(err) {
			v = vault.NewVault()
		} else {
			return err
		}
	}
	if tags == nil {
		tags = []string{}
	}
	e := vault.NewEntry(name, username, entryPassword, note, tags)
	v.AddEntry(e)
	return saveVault(repoRoot, password, v)
}

// UpdateEntry 按 ID 更新条目。
func UpdateEntry(repoRoot string, password []byte, e vault.Entry) error {
	if strings.TrimSpace(e.ID) == "" {
		return fmt.Errorf("条目 ID 无效")
	}
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("名称不能为空")
	}
	v, err := loadVault(repoRoot, password)
	if err != nil {
		return err
	}
	found := false
	for _, x := range v.Entries {
		if x.ID == e.ID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("条目不存在")
	}
	v.UpdateEntry(e)
	return saveVault(repoRoot, password, v)
}

// DeleteEntry 按 ID 删除条目。
func DeleteEntry(repoRoot string, password []byte, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("条目 ID 无效")
	}
	v, err := loadVault(repoRoot, password)
	if err != nil {
		return err
	}
	v.DeleteEntry(id)
	return saveVault(repoRoot, password, v)
}
