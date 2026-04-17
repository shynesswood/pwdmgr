# 待办事项

## ~~Git 操作：使用 go-git 库重写~~（已完成）

> 已在保留原 `exec` 后端的前提下，新增基于 [go-git/v5](https://github.com/go-git/go-git) 的纯 Go 后端，通过 `internal/git` 的 `Backend` 接口统一分发。`pwdmgr.config.json` 新增 `git_client` 字段（`exec` / `go-git`），未配置时默认 `exec`，`app.Startup / ReloadConfig` 会据此切换后端；对外公共 API 签名保持不变。自动化测试新增 BK1~BK4、GG1~GG10、CFG-GC1~CFG-GC2 共 16 个用例。

## 页面配置：本地仓库与远程仓库

**优先级**：低  
**现状**：本地仓库路径和远程仓库地址需要手动修改配置文件，用户无法在界面上直接配置。

**方案**：
1. 设置页面增加「本地仓库路径」和「远程仓库地址」配置项
2. 后端提供读取/更新配置的接口
3. 配置变更后自动验证路径和远程连通性，给出反馈

**涉及文件**：
- `pwdmgr.config.json` — 配置文件
- `app.go` — 新增配置读写接口
- 前端 `SettingsTab.vue` — 设置页面 UI

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
