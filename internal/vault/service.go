package vault

import (
	"errors"
	"strings"
)

// ---------------------------------------------------------------------------
// Entry 相关
// ---------------------------------------------------------------------------

func (v *Vault) AddEntry(e Entry) {
	if e.SpaceID == "" {
		e.SpaceID = DefaultSpaceID
	}
	v.Entries = append(v.Entries, e)
}

func (v *Vault) UpdateEntry(updated Entry) {
	for i, e := range v.Entries {
		if e.ID == updated.ID {
			updated.UpdatedAt = now()
			if updated.SpaceID == "" {
				updated.SpaceID = e.SpaceID
			}
			v.Entries[i] = updated
			return
		}
	}
}

// DeleteEntry 软删除指定条目：不再从 Entries 中移除，
// 而是在该条目上打上 DeletedAt 时间戳并刷新 UpdatedAt，
// 以便后续 MergeVault 可以正确区分"本地删除"与"远程新增"。
// 对不存在或已删除的 ID 为 no-op。
func (v *Vault) DeleteEntry(id string) {
	for i, e := range v.Entries {
		if e.ID == id && e.DeletedAt == 0 {
			ts := now()
			v.Entries[i].DeletedAt = ts
			v.Entries[i].UpdatedAt = ts
			return
		}
	}
}

// MoveEntries 将一批条目移动到目标空间；返回实际移动数量。
//   - ids 中不存在 / 已软删除 / 已在目标空间的条目会被跳过（不计入返回值）。
//   - 调用方需自行保证 targetSpaceID 合法（存在且未被软删除），本方法不做校验，
//     以便上层 service.MoveEntries 统一处理错误提示。
//   - 每个被移动的条目刷新 UpdatedAt，使合并时能正确传播。
func (v *Vault) MoveEntries(ids []string, targetSpaceID string) int {
	if targetSpaceID == "" {
		targetSpaceID = DefaultSpaceID
	}
	if len(ids) == 0 {
		return 0
	}
	idSet := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	moved := 0
	ts := now()
	for i := range v.Entries {
		if _, ok := idSet[v.Entries[i].ID]; !ok {
			continue
		}
		if v.Entries[i].IsDeleted() {
			continue
		}
		if v.Entries[i].SpaceID == targetSpaceID {
			continue
		}
		v.Entries[i].SpaceID = targetSpaceID
		v.Entries[i].UpdatedAt = ts
		moved++
	}
	return moved
}

// ActiveEntries 返回未被软删除的条目切片（新分配，不修改原数据）。
func (v *Vault) ActiveEntries() []Entry {
	result := make([]Entry, 0, len(v.Entries))
	for _, e := range v.Entries {
		if e.DeletedAt == 0 {
			result = append(result, e)
		}
	}
	return result
}

// EntriesInSpace 返回指定空间下未被软删除的条目。空 spaceID 视为 DefaultSpaceID。
func (v *Vault) EntriesInSpace(spaceID string) []Entry {
	if spaceID == "" {
		spaceID = DefaultSpaceID
	}
	result := make([]Entry, 0)
	for _, e := range v.Entries {
		if e.DeletedAt == 0 && e.SpaceID == spaceID {
			result = append(result, e)
		}
	}
	return result
}

// HasActiveEntriesInSpace 判断某空间下是否仍有未软删除的条目，
// 删除空间前用于阻止"非空删除"。
func (v *Vault) HasActiveEntriesInSpace(spaceID string) bool {
	for _, e := range v.Entries {
		if e.DeletedAt == 0 && e.SpaceID == spaceID {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Space 相关
// ---------------------------------------------------------------------------

// Space CRUD 可能返回的错误。调用方可以用 errors.Is 检查。
var (
	ErrSpaceNotFound      = errors.New("空间不存在")
	ErrSpaceNameEmpty     = errors.New("空间名称不能为空")
	ErrSpaceNameDuplicate = errors.New("空间名称已存在")
	ErrSpaceProtected     = errors.New("默认空间不可修改或删除")
	ErrSpaceNotEmpty      = errors.New("空间下仍有条目，无法删除")
)

// ActiveSpaces 返回未被软删除的空间。
func (v *Vault) ActiveSpaces() []Space {
	result := make([]Space, 0, len(v.Spaces))
	for _, s := range v.Spaces {
		if s.DeletedAt == 0 {
			result = append(result, s)
		}
	}
	return result
}

// FindSpace 按 ID 在所有空间（含已软删除）中查找。
func (v *Vault) FindSpace(id string) *Space {
	for i := range v.Spaces {
		if v.Spaces[i].ID == id {
			return &v.Spaces[i]
		}
	}
	return nil
}

// AddSpace 创建一个新空间。name 非空；同名（活跃状态）返回 ErrSpaceNameDuplicate。
func (v *Vault) AddSpace(name string) (Space, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return Space{}, ErrSpaceNameEmpty
	}
	for _, s := range v.Spaces {
		if s.DeletedAt == 0 && s.Name == trimmed {
			return Space{}, ErrSpaceNameDuplicate
		}
	}
	sp := NewSpace(trimmed)
	v.Spaces = append(v.Spaces, sp)
	return sp, nil
}

// RenameSpace 重命名指定空间。默认空间不可改名；新名不能与其它活跃空间冲突。
func (v *Vault) RenameSpace(id, name string) error {
	if id == DefaultSpaceID {
		return ErrSpaceProtected
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrSpaceNameEmpty
	}
	idx := -1
	for i, s := range v.Spaces {
		if s.ID == id {
			idx = i
		} else if s.DeletedAt == 0 && s.Name == trimmed {
			return ErrSpaceNameDuplicate
		}
	}
	if idx < 0 || v.Spaces[idx].IsDeleted() {
		return ErrSpaceNotFound
	}
	v.Spaces[idx].Name = trimmed
	v.Spaces[idx].UpdatedAt = now()
	return nil
}

// DeleteSpace 软删除指定空间。禁止删除默认空间；空间下若仍有活跃条目则返回错误。
func (v *Vault) DeleteSpace(id string) error {
	if id == DefaultSpaceID {
		return ErrSpaceProtected
	}
	if v.HasActiveEntriesInSpace(id) {
		return ErrSpaceNotEmpty
	}
	for i, s := range v.Spaces {
		if s.ID == id && s.DeletedAt == 0 {
			ts := now()
			v.Spaces[i].DeletedAt = ts
			v.Spaces[i].UpdatedAt = ts
			return nil
		}
	}
	return ErrSpaceNotFound
}

// EnsureDefaultSpace 保证 Vault 中存在默认空间，并把遗留的无 SpaceID 条目归入默认空间。
// 兼容加载旧版本 vault.dat（无 Spaces 字段）。
func (v *Vault) EnsureDefaultSpace() {
	hasDefault := false
	for _, s := range v.Spaces {
		if s.ID == DefaultSpaceID {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		ts := now()
		v.Spaces = append(v.Spaces, Space{
			ID:        DefaultSpaceID,
			Name:      DefaultSpaceName,
			CreatedAt: ts,
			UpdatedAt: ts,
		})
	}
	for i := range v.Entries {
		if v.Entries[i].SpaceID == "" {
			v.Entries[i].SpaceID = DefaultSpaceID
		}
	}
}

// ---------------------------------------------------------------------------
// Tag 相关（保持原有语义，针对当前所有活跃条目）
// ---------------------------------------------------------------------------

func (v *Vault) FilterByTags(tags []string) []Entry {
	if len(tags) == 0 {
		return v.Entries
	}

	var result []Entry

	for _, e := range v.Entries {
		if containsAll(e.Tags, tags) {
			result = append(result, e)
		}
	}

	return result
}

func containsAll(entryTags, filterTags []string) bool {
	m := make(map[string]struct{})
	for _, t := range entryTags {
		m[t] = struct{}{}
	}

	for _, ft := range filterTags {
		if _, ok := m[ft]; !ok {
			return false
		}
	}
	return true
}

func (v *Vault) TagStats() map[string]int {
	stats := make(map[string]int)

	for _, e := range v.Entries {
		for _, t := range e.Tags {
			stats[t]++
		}
	}

	return stats
}
