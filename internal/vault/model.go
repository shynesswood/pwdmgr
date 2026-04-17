package vault

// DefaultSpaceID 是默认空间的固定 ID；不可删除/不可重命名，
// 用于承接旧版本（无 Spaces 字段）vault 迁移过来的历史条目。
const DefaultSpaceID = "default"

// DefaultSpaceName 是默认空间的初始展示名称。
const DefaultSpaceName = "默认空间"

// Space 表示保险库中的一个逻辑分区，供用户按场景（工作、个人等）
// 隔离密码条目。所有 Space 与其下 Entry 共享同一加密文件。
type Space struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	// DeletedAt 为软删除时间戳，语义同 Entry.DeletedAt，用于合并时去重。
	DeletedAt int64 `json:"deleted_at,omitempty"`
}

func (s Space) IsDeleted() bool {
	return s.DeletedAt > 0
}

type Entry struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	Note      string   `json:"note"`
	Tags      []string `json:"tags"`
	UpdatedAt int64    `json:"updated_at"`
	// DeletedAt 为软删除时间戳。0 表示未删除；>0 表示已删除，
	// 用于在合并时区分"本地删除"与"远程新增"，避免删除的条目被远程旧版本"复活"。
	DeletedAt int64 `json:"deleted_at,omitempty"`
	// SpaceID 指明条目所属空间；空字符串表示迁移前的旧数据，
	// 会在 LoadVault 后被 EnsureDefaultSpace 自动归入 DefaultSpaceID。
	SpaceID string `json:"space_id,omitempty"`
}

// IsDeleted 返回该条目是否被软删除。
func (e Entry) IsDeleted() bool {
	return e.DeletedAt > 0
}

type Vault struct {
	Version int     `json:"version"`
	Spaces  []Space `json:"spaces"`
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
		SpaceID:   DefaultSpaceID,
	}
}

// NewEntryInSpace 在指定空间下创建条目；spaceID 为空时回退到 DefaultSpaceID。
func NewEntryInSpace(spaceID, name, username, password, note string, tags []string) Entry {
	if spaceID == "" {
		spaceID = DefaultSpaceID
	}
	e := NewEntry(name, username, password, note, tags)
	e.SpaceID = spaceID
	return e
}

// NewSpace 创建一个新的空间，CreatedAt/UpdatedAt 初始化为当前时间戳。
func NewSpace(name string) Space {
	ts := now()
	return Space{
		ID:        generateID(),
		Name:      name,
		CreatedAt: ts,
		UpdatedAt: ts,
	}
}
