package vault

type Entry struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Note      string   `json:"note"`
	Tags      []string `json:"tags"`
	UpdatedAt int64    `json:"updated_at"`
}

type Vault struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// 创建新 Entry
func NewEntry(name, username, password, note string, tags []string) Entry {
	return Entry{
		ID:        generateID(),
		Name:      name,
		Username:  username,
		Password:  password,
		Note:      note,
		Tags:      normalizeTags(tags),
		UpdatedAt: now(),
	}
}
