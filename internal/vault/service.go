package vault

func (v *Vault) AddEntry(e Entry) {
	v.Entries = append(v.Entries, e)
}

func (v *Vault) UpdateEntry(updated Entry) {
	for i, e := range v.Entries {
		if e.ID == updated.ID {
			updated.UpdatedAt = now()
			v.Entries[i] = updated
			return
		}
	}
}

func (v *Vault) DeleteEntry(id string) {
	var result []Entry
	for _, e := range v.Entries {
		if e.ID != id {
			result = append(result, e)
		}
	}
	v.Entries = result
}

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
