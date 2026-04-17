package vault

// MergeVault 合并本地与远程保险库。
// 对 Spaces 和 Entries 均按 "同 ID 取 UpdatedAt 较新一方" 的规则合并，
// 从而天然支持软删除（DeletedAt 字段作为普通修改传播）。
// 合并后强制保证默认空间存在，以防双方都缺失导致后续写入失败。
func MergeVault(local, remote *Vault) *Vault {
	result := NewVault()
	result.Spaces = result.Spaces[:0]
	result.Entries = result.Entries[:0]

	// Entries 合并
	em := make(map[string]Entry)
	for _, e := range local.Entries {
		em[e.ID] = e
	}
	for _, re := range remote.Entries {
		if le, ok := em[re.ID]; ok {
			if re.UpdatedAt > le.UpdatedAt {
				em[re.ID] = re
			}
		} else {
			em[re.ID] = re
		}
	}
	for _, e := range em {
		result.Entries = append(result.Entries, e)
	}

	// Spaces 合并
	sm := make(map[string]Space)
	for _, s := range local.Spaces {
		sm[s.ID] = s
	}
	for _, rs := range remote.Spaces {
		if ls, ok := sm[rs.ID]; ok {
			if rs.UpdatedAt > ls.UpdatedAt {
				sm[rs.ID] = rs
			}
		} else {
			sm[rs.ID] = rs
		}
	}
	for _, s := range sm {
		result.Spaces = append(result.Spaces, s)
	}

	// 兜底：双方均为空 / 双方都误删 default 的极端情况下，保证默认空间存在
	result.EnsureDefaultSpace()
	return result
}
