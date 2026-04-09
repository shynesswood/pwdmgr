package vault

// 合并本地和远程
func MergeVault(local, remote *Vault) *Vault {

	result := NewVault()

	m := make(map[string]Entry)

	// 先放本地
	for _, e := range local.Entries {
		m[e.ID] = e
	}

	// 再合并远程
	for _, re := range remote.Entries {
		if le, ok := m[re.ID]; ok {
			if re.UpdatedAt > le.UpdatedAt {
				m[re.ID] = re
			}
		} else {
			m[re.ID] = re
		}
	}

	for _, e := range m {
		result.Entries = append(result.Entries, e)
	}

	return result
}
