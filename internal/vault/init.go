package vault

// NewVault 返回一个新的空保险库，并预置默认空间，
// 确保任何后续 AddEntry 都能直接命中一个有效的 SpaceID。
func NewVault() *Vault {
	ts := now()
	return &Vault{
		Version: 1,
		Spaces: []Space{
			{ID: DefaultSpaceID, Name: DefaultSpaceName, CreatedAt: ts, UpdatedAt: ts},
		},
		Entries: []Entry{},
	}
}
