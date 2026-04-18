# 待办事项

## ~~Git 操作：使用 go-git 库重写~~（已完成）

> 已在保留原 `exec` 后端的前提下，新增基于 [go-git/v5](https://github.com/go-git/go-git) 的纯 Go 后端，通过 `internal/git` 的 `Backend` 接口统一分发。`pwdmgr.config.json` 新增 `git_client` 字段（`exec` / `go-git`），未配置时默认 `exec`，`app.Startup / ReloadConfig` 会据此切换后端；对外公共 API 签名保持不变。自动化测试新增 BK1~BK4、GG1~GG10、CFG-GC1~CFG-GC2 共 16 个用例。

## ~~页面配置：本地仓库与远程仓库~~（已完成）

> `internal/config.Config` 新增原子 `Save()`（JSON map 合并 + `tmp + rename`，保留未知字段）；`internal/app.UpdateAppConfig` 对 `repo_root / remote_url / git_client` 做校验后写盘、重新 Load 并同步切换 git 后端。`SettingsTab.vue` 重写为只读/编辑两种模式，`git_client` 下拉选择，保存时自动锁定 vault 避免配置期间误操作。自动化新增 CFG-SV1~CFG-SV4、APP-UC1~APP-UC5 共 9 个用例，手工新增 CFG-E1/CFG-E2。未做远程连通性自动探测（避免阻塞 UI），保留为后续增强。

## 多仓库支持（2.x）

**优先级**：最低（计划 2.x 系列版本）  
**动机**：当前仅支持单一远程仓库，若该仓库数据损坏或丢失，将无法找回数据。支持多个远程仓库可作为数据冗余备份，提升容灾能力。

**方案**：
1. 本地仓库保持一个，配置文件支持定义多个远程仓库地址
2. push 时同时推送到所有远程仓库，pull 时从主远程拉取
3. 当主远程不可用时，支持切换从备用远程拉取恢复

**涉及文件**：
- `pwdmgr.config.json` — 多远程配置结构
- `internal/git/sync.go` — 多远程 push/pull 逻辑
- `internal/service/sync.go` — 同步调度
- `app.go` — 接口适配
- 前端 `SettingsTab.vue` / `SyncTab.vue` — 多远程管理 UI
