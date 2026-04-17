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
	v, err := storage.LoadVault(path, password)
	if err != nil {
		return nil, err
	}
	// 兼容旧版本 vault.dat（无 Spaces 字段 / 无 SpaceID 条目）
	v.EnsureDefaultSpace()
	return v, nil
}

func saveVault(repoRoot string, password []byte, v *vault.Vault) error {
	path := config.VaultFilePath(repoRoot)
	return storage.SaveVault(path, password, v)
}

func isNoVaultFile(err error) bool {
	return err != nil && (errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "数据文件不存在"))
}

// resolveSpaceID 校验调用方传入的空间 ID：
//   - 空字符串回退到默认空间
//   - 必须存在且未被软删除，否则返回错误
func resolveSpaceID(v *vault.Vault, spaceID string) (string, error) {
	if strings.TrimSpace(spaceID) == "" {
		spaceID = vault.DefaultSpaceID
	}
	sp := v.FindSpace(spaceID)
	if sp == nil || sp.IsDeleted() {
		return "", fmt.Errorf("空间不存在或已删除")
	}
	return spaceID, nil
}

// ListEntries 解密并返回指定空间下的条目（按名称排序）。
// 软删除条目（DeletedAt > 0）会被过滤；尚无 vault 文件时返回空列表。
// spaceID 为空字符串时等价于默认空间。
func ListEntries(repoRoot string, password []byte, spaceID string) ([]vault.Entry, error) {
	v, err := loadVault(repoRoot, password)
	if err != nil {
		if isNoVaultFile(err) {
			return []vault.Entry{}, nil
		}
		return nil, err
	}
	resolved, err := resolveSpaceID(v, spaceID)
	if err != nil {
		return nil, err
	}
	out := v.EntriesInSpace(resolved)
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// AddEntry 向指定空间新增一条记录并写回磁盘。
// 若尚无加密库文件则自动创建空库再写入；spaceID 为空则归入默认空间。
func AddEntry(repoRoot string, password []byte, spaceID, name, username, entryPassword, note string, tags []string) error {
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
	resolved, err := resolveSpaceID(v, spaceID)
	if err != nil {
		return err
	}
	if tags == nil {
		tags = []string{}
	}
	e := vault.NewEntryInSpace(resolved, name, username, entryPassword, note, tags)
	v.AddEntry(e)
	return saveVault(repoRoot, password, v)
}

// UpdateEntry 按 ID 更新条目。若 entry 带 SpaceID 且与原空间不同，则视为跨空间移动。
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
	var existing *vault.Entry
	for i := range v.Entries {
		if v.Entries[i].ID == e.ID && !v.Entries[i].IsDeleted() {
			existing = &v.Entries[i]
			break
		}
	}
	if existing == nil {
		return fmt.Errorf("条目不存在")
	}
	// 校验目标空间：若调用方未指定 SpaceID 则保持原空间
	if strings.TrimSpace(e.SpaceID) == "" {
		e.SpaceID = existing.SpaceID
	}
	resolved, err := resolveSpaceID(v, e.SpaceID)
	if err != nil {
		return err
	}
	e.SpaceID = resolved
	// 保险：更新时始终清除 DeletedAt，避免前端误传导致重新标记为已删除。
	e.DeletedAt = 0
	v.UpdateEntry(e)
	return saveVault(repoRoot, password, v)
}

// MoveEntries 将一批条目移动到指定空间。
//   - targetSpaceID 为空时等价于默认空间；目标空间必须存在且未被软删除。
//   - ids 中不存在 / 已软删除 / 已在目标空间的条目会被静默跳过。
//   - 返回实际移动的条目数量（>=0）。
func MoveEntries(repoRoot string, password []byte, ids []string, targetSpaceID string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	v, err := loadVault(repoRoot, password)
	if err != nil {
		return 0, err
	}
	resolved, err := resolveSpaceID(v, targetSpaceID)
	if err != nil {
		return 0, err
	}
	moved := v.MoveEntries(ids, resolved)
	if moved == 0 {
		return 0, nil
	}
	if err := saveVault(repoRoot, password, v); err != nil {
		return 0, err
	}
	return moved, nil
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
