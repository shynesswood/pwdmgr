package service

import (
	"sort"
	"strings"

	"pwdmgr/internal/vault"
)

// ListSpaces 返回所有未软删除的空间，按名称排序。
// 默认空间始终置顶，便于前端展示。
func ListSpaces(repoRoot string, password []byte) ([]vault.Space, error) {
	v, err := loadVault(repoRoot, password)
	if err != nil {
		if isNoVaultFile(err) {
			// 尚无 vault 时，也返回一个逻辑上的默认空间，避免 UI 空态
			tmp := vault.NewVault()
			return tmp.ActiveSpaces(), nil
		}
		return nil, err
	}
	spaces := v.ActiveSpaces()
	sort.Slice(spaces, func(i, j int) bool {
		if spaces[i].ID == vault.DefaultSpaceID {
			return true
		}
		if spaces[j].ID == vault.DefaultSpaceID {
			return false
		}
		return strings.ToLower(spaces[i].Name) < strings.ToLower(spaces[j].Name)
	})
	return spaces, nil
}

// CreateSpace 新建一个空间，返回创建后的空间（含自动生成的 ID）。
func CreateSpace(repoRoot string, password []byte, name string) (vault.Space, error) {
	v, err := loadVault(repoRoot, password)
	if err != nil {
		if isNoVaultFile(err) {
			v = vault.NewVault()
		} else {
			return vault.Space{}, err
		}
	}
	sp, err := v.AddSpace(name)
	if err != nil {
		return vault.Space{}, err
	}
	if err := saveVault(repoRoot, password, v); err != nil {
		return vault.Space{}, err
	}
	return sp, nil
}

// RenameSpace 重命名指定空间。
func RenameSpace(repoRoot string, password []byte, id, name string) error {
	v, err := loadVault(repoRoot, password)
	if err != nil {
		return err
	}
	if err := v.RenameSpace(id, name); err != nil {
		return err
	}
	return saveVault(repoRoot, password, v)
}

// DeleteSpace 软删除指定空间。空间下仍有活跃条目时会拒绝删除。
func DeleteSpace(repoRoot string, password []byte, id string) error {
	v, err := loadVault(repoRoot, password)
	if err != nil {
		return err
	}
	if err := v.DeleteSpace(id); err != nil {
		return err
	}
	return saveVault(repoRoot, password, v)
}
