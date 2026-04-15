# 待办事项

## 合并逻辑：软删除支持

**优先级**：中  
**现状**：`MergeVault` 无法区分"本地删除"和"远程新增"，导致本地删除的条目在同步后被远程重新拉回，永远无法真正删除。目前可通过强制 push/pull 规避。

**方案**：
1. `Entry` 增加 `DeletedAt int64` 字段，删除操作改为软删除（标记时间戳而非移除）
2. `MergeVault` 合并时比较双方 `UpdatedAt`，保留更新的一方（包括删除标记）
3. 前端展示时过滤掉 `DeletedAt > 0` 的条目

**涉及文件**：
- `internal/vault/model.go` — Entry 结构体
- `internal/vault/merge.go` — 合并逻辑
- `internal/vault/vault.go` — 删除方法改为软删除
- `app.go` / 前端 — 列表过滤已删除条目
