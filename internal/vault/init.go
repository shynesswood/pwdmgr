package vault

func NewVault() *Vault {
	return &Vault{
		Version: 1,
		Entries: []Entry{},
	}
}
