# PwdMgr 测试用例

## 测试类型说明

本文档中的测试用例分为两类：

| 标记 | 含义 | 说明 |
|------|------|------|
| 🤖 **自动化测试** | Go `_test.go` | 通过 `go test` 运行，无需人工干预，CI 可集成 |
| 🖱️ **手工测试** | 在 GUI 界面操作 | 需要人工在 Wails 桌面应用中点击、输入、观察 |

自动化测试文件分布：

| 包 | 测试文件 | 覆盖用例 |
|----|----------|----------|
| `internal/crypto` | `crypto_test.go` | 加解密往返、错误密码、空密码 |
| `internal/vault` | `vault_test.go` | 模型 CRUD、MergeVault (B5/B6/B7)、标签 |
| `internal/storage` | `storage_test.go` | 序列化、SaveLoad 往返、X1 密码错误 |
| `internal/git` | `git_test.go` | G1–G11 全部 |
| `internal/service` | `init_test.go` | I1–I4 |
| `internal/service` | `status_test.go` | R1–R4 |
| `internal/service` | `entries_test.go` | L1–L2、CRUD 集成 |
| `internal/service` | `bind_test.go` | B1–B8 |
| `internal/service` | `sync_test.go` | S1–S8 |
| `internal/service` | `edge_test.go` | X1–X7 及扩展 |

运行方式：

```bash
# 运行全部自动化测试
go test ./internal/... -timeout=600s

# 运行单个包
go test ./internal/git/ -v
go test ./internal/service/ -v -timeout=600s

# 运行单个用例
go test ./internal/service/ -run TestB6 -v
```

---

## 前置准备

所有测试需要一个真实的 Git 远程仓库（用本地 bare repo 模拟）。

```powershell
# 创建临时远程裸仓库
$remote = "$env:TEMP\pwdmgr-test-remote"
Remove-Item -Recurse -Force $remote -ErrorAction SilentlyContinue
git init --bare $remote

# 本地 repo_root 测试目录
$local = "$env:TEMP\pwdmgr-test-local"
Remove-Item -Recurse -Force $local -ErrorAction SilentlyContinue
mkdir $local
```

每个测试场景前清理 `$local` 目录和远程仓库，确保测试隔离。

配置文件 `pwdmgr.config.json` 示例：

```json
{
  "repo_root": "C:\\Users\\<user>\\AppData\\Local\\Temp\\pwdmgr-test-local",
  "remote_url": "C:\\Users\\<user>\\AppData\\Local\\Temp\\pwdmgr-test-remote"
}
```

---

## 一、git 层基础操作（11 个用例）

### G1 — runGitCommand 错误信息可读 🤖

- **自动化**：`TestG1_RunGitCommand_ReadableError` @ `internal/git/git_test.go`
- **操作**：在空目录（非 git repo）执行一个会失败的 git 命令（如 `git status`）
- **预期**：错误信息包含 git 原始输出（如 `"not a git repository"`），不再只是 `"exit status 128"`

### G2 — AddRemote 首次添加 🤖

- **自动化**：`TestG2_AddRemote_FirstTime` @ `internal/git/git_test.go`
- **操作**：`git init` → `AddRemote(path, url)`
- **预期**：成功，`git remote -v` 可见 origin 指向 url

### G3 — AddRemote 重复添加（容错）🤖

- **自动化**：`TestG3_AddRemote_Duplicate` @ `internal/git/git_test.go`
- **前置**：接 G2 已有 origin
- **操作**：再次 `AddRemote(path, newUrl)`
- **预期**：不报错，origin URL 更新为 newUrl（内部 fallback 到 `git remote set-url`）

### G4 — Pull 回退策略：无跟踪分支 🤖

- **自动化**：`TestG4_Pull_FallbackNoTrackingBranch` @ `internal/git/git_test.go`
- **前置**：远程仓库已有至少一个提交
- **操作**：`git init` → `git remote add origin <远程>` → `Pull(path)`
- **预期**：成功拉取远程数据，本地 checkout 到正确分支（main 或 master）

### G5 — Push 首次推送设置 upstream 🤖

- **自动化**：`TestG5_Push_FirstTimeSetUpstream` @ `internal/git/git_test.go`
- **前置**：空远程仓库
- **操作**：`git init` → `git remote add origin <空远程>` → 创建一个文件并提交 → `Push(path)`
- **预期**：成功推送（`git push -u origin HEAD`），远程仓库可见提交，upstream 已设置

### G6 — detectDefaultBranch 检测 🤖

- **自动化**：`TestG6_DetectDefaultBranch` @ `internal/git/git_test.go`
- **操作**：远程默认分支为 `main` 时调用 / 为 `master` 时调用
- **预期**：分别返回 `"main"` / `"master"`；当远程无分支时 fallback 返回 `"main"`

### G7 — Commit 函数 🤖

- **自动化**：`TestG7_Commit` @ `internal/git/git_test.go`
- **前置**：`git init` → 创建一个文件
- **操作**：`Commit(path, "test commit")`
- **预期**：成功；`git log --oneline` 可见一条提交信息为 `"test commit"` 的记录

### G8 — HasChanges 检测 🤖

- **自动化**：`TestG8_HasChanges` @ `internal/git/git_test.go`
- **前置**：`git init` → 创建一个文件并提交
- **操作 A**：无修改时调用 `HasChanges(path)` → 返回 `false`
- **操作 B**：修改文件后调用 `HasChanges(path)` → 返回 `true`
- **操作 C**：新建未跟踪文件后调用 `HasChanges(path)` → 返回 `true`

### G9 — RestoreFile 恢复已跟踪文件 🤖

- **自动化**：`TestG9_RestoreFile` @ `internal/git/git_test.go`
- **前置**：`git init` → 创建 `vault.dat` 并提交 → 修改 `vault.dat` 内容
- **操作**：`RestoreFile(path, "vault.dat")`
- **预期**：`vault.dat` 恢复为提交时的内容

### G10 — CurrentBranch 获取当前分支 🤖

- **自动化**：`TestG10_CurrentBranch` @ `internal/git/git_test.go`
- **前置**：`git init` → 创建文件并提交
- **操作**：`CurrentBranch(path)`
- **预期**：返回当前分支名（如 `"main"` 或 `"master"`），非空字符串

### G11 — RemoteURL 获取远程地址 🤖

- **自动化**：`TestG11_RemoteURL` @ `internal/git/git_test.go`
- **前置**：`git init` → `AddRemote(path, "https://example.com/repo.git")`
- **操作**：`RemoteURL(path)`
- **预期**：返回 `"https://example.com/repo.git"`
- **无 remote 时**：返回空字符串

---

## 二、BindRemoteRepo 四大场景（8 个用例）

### 场景1：远程空 + 本地有 vault → Push

#### B1 — 全新目录 + 本地 vault → 推送 🤖

- **自动化**：`TestB1_BindRemoteRepo_NewDirLocalVaultPush` @ `internal/service/bind_test.go`
- **前置**：空目录（非 git repo），调用 `InitLocalVault` 创建并提交 vault.dat
- **操作**：`BindRemoteRepo(localPath, remoteUrl, pwd)`
- **预期**：成功；远程仓库包含 vault.dat

#### B2 — 已有 git repo + 本地 vault → 推送 🤖

- **自动化**：`TestB2_BindRemoteRepo_ExistingGitLocalVaultPush` @ `internal/service/bind_test.go`
- **前置**：已 `git init` 且通过 `InitLocalVault` 创建了 vault.dat（`InitLocalVault` 会自动执行 `git commit`，因此 vault.dat 已提交）
- **操作**：`BindRemoteRepo(...)`
- **预期**：成功；vault.dat 被 push 到远程

### 场景2：远程有数据 + 本地无 vault → Pull

#### B3 — 全新空目录拉取远程 🤖

- **自动化**：`TestB3_BindRemoteRepo_EmptyLocalPullFromRemote` @ `internal/service/bind_test.go`
- **前置**：远程已有 vault.dat（由另一个 clone 推送），本地为空目录
- **操作**：`BindRemoteRepo(...)`
- **预期**：成功；本地出现 vault.dat，可用密码解密查看条目

#### B4 — 已有 git repo + 无 vault 拉取 🤖

- **自动化**：`TestB4_BindRemoteRepo_ExistingGitNoVaultPull` @ `internal/service/bind_test.go`
- **前置**：本地已 `git init`，远程有数据
- **操作**：`BindRemoteRepo(...)`
- **预期**：成功拉取，本地出现 vault.dat

### 场景3：两边都有 vault → 合并

#### B5 — 双端相同条目 🤖

- **自动化**：`TestB5_BindRemoteRepo_BothHaveSameEntries` @ `internal/service/bind_test.go`
- **前置**：远程有条目 A，本地也有条目 A（相同 ID，相同内容）
- **操作**：`BindRemoteRepo(...)`
- **预期**：成功合并，结果只有一个条目 A

#### B6 — 双端不同条目 🤖

- **自动化**：`TestB6_BindRemoteRepo_BothHaveDifferentEntries` @ `internal/service/bind_test.go`
- **前置**：远程有条目 A，本地有条目 B（不同 ID）
- **操作**：`BindRemoteRepo(...)`
- **预期**：成功合并，结果包含 A 和 B 两个条目
- **合并机制**：内部先读取本地 vault 到内存 → 删除 vault.dat → pull 远程版本 → 应用层 `MergeVault` → 保存 → push

#### B7 — 双端同 ID 不同时间戳 🤖

- **自动化**：`TestB7_BindRemoteRepo_SameIDNewerTimestampWins` @ `internal/service/bind_test.go`
- **前置**：远程有条目 X（updated_at=100），本地有条目 X（updated_at=200，密码已修改）
- **操作**：`BindRemoteRepo(...)`
- **预期**：合并后条目 X 取本地版本（时间戳较大的胜出）

### 场景4：两边都没有 → 初始化

#### B8 — 双空初始化 🤖

- **自动化**：`TestB8_BindRemoteRepo_BothEmpty` @ `internal/service/bind_test.go`
- **前置**：空目录 + 空远程
- **操作**：`BindRemoteRepo(...)`
- **预期**：成功；本地和远程都有空 vault.dat（entries 为空数组）

---

## 三、SyncVault 同步（8 个用例）

### S1 — 无本地变更，纯 pull 🤖

- **自动化**：`TestS1_SyncVault_NoLocalChanges_PurePull` @ `internal/service/sync_test.go`
- **前置**：已绑定成功的仓库，本地无修改（`HasChanges` 返回 false）；通过另一个 clone 向远程推送了新条目
- **操作**：`SyncVault(root, pwd)`
- **预期**：走纯 `git.Pull` 路径，成功拉取远程变更，本地 vault 包含新条目

### S2 — 有本地变更，远程无变更 🤖

- **自动化**：`TestS2_SyncVault_LocalChanges_NoRemoteChanges` @ `internal/service/sync_test.go`
- **前置**：已绑定成功；本地通过 `AddEntry` 新增条目（磁盘 vault.dat 已改，`HasChanges` 返回 true），远程无新提交
- **操作**：`SyncVault(root, pwd)`
- **预期**：内部执行"读内存 → 清理工作区 → pull → 合并 → 保存 → push"流程，成功推送，远程包含新条目

### S3 — 有本地变更，远程也有变更（不同条目）🤖

- **自动化**：`TestS3_SyncVault_BothChanged_DifferentEntries` @ `internal/service/sync_test.go`
- **前置**：本地新增条目 A；远程（另一设备模拟）新增条目 B
- **操作**：`SyncVault(root, pwd)`
- **预期**：成功合并，最终 vault 包含 A 和 B

### S4 — 有本地变更，远程也有变更（相同条目冲突）🤖

- **自动化**：`TestS4_SyncVault_BothChanged_SameEntryConflict` @ `internal/service/sync_test.go`
- **前置**：本地修改条目 X 的密码（updated_at 较新）；远程修改条目 X 的用户名（updated_at 较旧）
- **操作**：`SyncVault(root, pwd)`
- **预期**：合并后取 updated_at 更大的版本

### S5 — pull 失败时恢复本地 vault 🤖

- **自动化**：`TestS5_SyncVault_PullFail_RecoverLocalVault` @ `internal/service/sync_test.go`
- **前置**：本地有未提交的 vault 变更，将远程 URL 改为不可达地址
- **操作**：`SyncVault(root, pwd)`
- **预期**：返回错误；本地 vault.dat 恢复为操作前的内容（内部执行 `SaveVault` 回写），不丢数据

### S6 — BindRepo 后首次 Sync 🤖

- **自动化**：`TestS6_SyncVault_FirstSyncAfterBind` @ `internal/service/sync_test.go`
- **前置**：刚执行完 `BindRemoteRepo` 成功
- **操作**：立即 `SyncVault(root, pwd)`
- **预期**：成功（git pull --rebase 正常工作，跟踪分支已建立）

### S7 — 工作区清理策略验证 🤖

- **自动化**：`TestS7_SyncVault_WorkspaceCleanupStrategy` @ `internal/service/sync_test.go`
- **前置**：已绑定成功，本地 vault.dat 有未提交变更
- **操作**：`SyncVault(root, pwd)`
- **验证**：
  1. 内部先将 vault 读入内存
  2. `os.Remove(vaultPath)` 删除磁盘文件
  3. `git.RestoreFile` 恢复为已提交版本
  4. `git.Pull` 不会因脏工作区失败
  5. 最终 vault 是本地内存版本与远程版本的合并结果

### S8 — SyncVault 空路径校验 🤖

- **自动化**：`TestS8_SyncVault_EmptyPath` @ `internal/service/sync_test.go`
- **操作**：`SyncVault("", pwd)`
- **预期**：返回错误 `"仓库路径不能为空"`

---

## 四、InitLocalVault 初始化（4 个用例）

### I1 — 正常创建 🤖

- **自动化**：`TestI1_InitLocalVault_Normal` @ `internal/service/init_test.go`
- **前置**：空目录，无 `.git`
- **操作**：`InitLocalVault(root, pwd)`
- **预期**：成功；目录下有 `.git` 和 `vault.dat`，vault 可解密且 entries 为空
- **新增验证**：`git log --oneline` 可见一条 `"init vault"` 的提交，vault.dat 已被 git 跟踪

### I2 — 已有 vault 报错 🤖

- **自动化**：`TestI2_InitLocalVault_AlreadyExists` @ `internal/service/init_test.go`
- **前置**：vault.dat 已存在
- **操作**：`InitLocalVault(root, pwd)`
- **预期**：返回错误 `"本地 vault 已存在"`

### I3 — 已有 git repo 🤖

- **自动化**：`TestI3_InitLocalVault_ExistingGitRepo` @ `internal/service/init_test.go`
- **前置**：目录已 `git init` 但无 vault
- **操作**：`InitLocalVault(root, pwd)`
- **预期**：成功；跳过 git init，仅创建 vault.dat 并自动提交（`git log` 可见 `"init vault"` 提交）

### I4 — 空路径校验 🤖

- **自动化**：`TestI4_InitLocalVault_EmptyPath` @ `internal/service/init_test.go`
- **操作**：`InitLocalVault("", pwd)`
- **预期**：返回错误 `"仓库路径不能为空"`

---

## 五、RepoStatus 状态查询（4 个用例）

### R1 — 空目录状态 🤖

- **自动化**：`TestR1_RepoStatus_EmptyDir` @ `internal/service/status_test.go`
- **前置**：空目录（非 git repo）
- **操作**：`GetRepoStatus()`
- **预期**：`isGitRepo=false`，其他字段为空/false

### R2 — 已初始化 + 已绑定远程状态 🤖

- **自动化**：`TestR2_RepoStatus_InitializedWithRemote` @ `internal/service/status_test.go`
- **前置**：`InitLocalVault` → `AddRemote`
- **操作**：`GetRepoStatus()`
- **预期**：
  - `isGitRepo=true`
  - `hasRemote=true`
  - `hasLocalVault=true`
  - `hasUncommitted=false`（InitLocalVault 已自动提交）
  - `currentBranch` 非空（如 `"main"` 或 `"master"`）
  - `remoteURL` 等于配置的远程地址

### R3 — 有未提交变更 🤖

- **自动化**：`TestR3_RepoStatus_HasUncommitted` @ `internal/service/status_test.go`
- **前置**：已绑定成功，通过 `AddEntry` 新增条目（vault.dat 被修改但未 git commit）
- **操作**：`GetRepoStatus()`
- **预期**：`hasUncommitted=true`

### R4 — 远程有数据 🤖

- **自动化**：`TestR4_RepoStatus_RemoteHasData` @ `internal/service/status_test.go`
- **前置**：远程仓库已有提交
- **操作**：`GetRepoStatus()`
- **预期**：`remoteHasData=true`

---

## 六、ListEntries 条目列表（2 个用例）

### L1 — 空库返回空数组（不是 null）🤖

- **自动化**：`TestL1_ListEntries_EmptyVault` @ `internal/service/entries_test.go`
- **前置**：`InitLocalVault` 创建空库
- **操作**：`ListEntries(root, pwd)`
- **预期**：返回空数组 `[]`（JSON 序列化为 `[]`，而非 `null`），前端不会因 `null` 崩溃

### L2 — vault 文件不存在时返回空数组 🤖

- **自动化**：`TestL2_ListEntries_NoVaultFile` @ `internal/service/entries_test.go`
- **前置**：已初始化 git repo 但尚无 vault.dat（如手动删除了文件）
- **操作**：`ListEntries(root, pwd)`
- **预期**：返回空数组 `[]`，不报错

---

## 七、端到端完整流程（4 个用例）🖱️

> 以下用例需要在 Wails 桌面应用的 GUI 中手工执行和验证，涉及界面交互、页面跳转、视觉反馈。

### E1 — 新用户首次使用（第一台电脑）🖱️

1. 配置 `repo_root` 和 `remote_url`
2. 点击 **"创建本地库"** → `InitLocalVault`
3. **验证**：`git log` 可见 `"init vault"` 提交
4. 解锁保险库，添加 3 个条目 → `AddEntry` ×3
5. 点击 **"绑定远程并同步"** → `BindRemoteRepo`
6. **验证**：远程仓库包含 vault.dat；克隆远程后用相同密码解密可见 3 个条目

### E2 — 第二台电脑同步 🖱️

- **前置**：E1 完成，远程已有数据
1. 新建空 `repo_root` 目录，配置相同 `remote_url`
2. 点击 **"绑定远程并同步"** → `BindRemoteRepo`
3. 点击 **"解锁保险库"** → `ListVaultEntries`
4. **验证**：看到 E1 推送的所有 3 个条目

### E3 — 双设备交替使用 🖱️

- **前置**：两台设备均已通过 E1/E2 完成绑定
1. 设备 A：新增条目 D → `AddEntry`
2. 设备 A：**Sync** → `SyncVault`
3. 设备 B：**Sync** → `SyncVault`
4. 设备 B：新增条目 E → `AddEntry`
5. 设备 B：**Sync** → `SyncVault`
6. 设备 A：**Sync** → `SyncVault`
7. **验证**：两台设备最终条目完全一致（原 3 个 + D + E = 5 个）

### E4 — 双设备同时修改（冲突合并）🖱️

- **前置**：两台设备都有条目 X
1. 设备 A：修改条目 X 的密码 → `UpdateEntry`（此时 updated_at = T1）
2. 设备 B：修改条目 X 的用户名 → `UpdateEntry`（此时 updated_at = T2，T2 > T1）
3. 设备 A：**Sync** → 推送成功
4. 设备 B：**Sync** → 拉取 A 的变更并合并
5. 设备 A：**Sync** → 拉取 B 的合并结果
6. **验证**：最终条目 X 取设备 B 的版本（updated_at 更大）

---

## 八、边界与异常（7 个用例）

### X1 — 密码错误 🤖

- **自动化**：`TestX1_WrongPassword` @ `internal/service/edge_test.go`
- **操作**：用密码 A 创建 vault，用密码 B 调用 `SyncVault`
- **预期**：返回解密错误，vault.dat 文件内容不被损坏

### X2 — 远程 URL 不可达 🤖

- **自动化**：`TestX2_UnreachableRemoteURL` @ `internal/service/edge_test.go`
- **操作**：配置一个不存在的 remote URL → `BindRemoteRepo`
- **预期**：返回可读的 git 错误信息（包含连接失败原因，不是 "exit status 128"）

### X3 — repo_root 目录不存在 🤖

- **自动化**：`TestX3_NonExistentRepoRoot` @ `internal/service/edge_test.go`
- **操作**：配置一个不存在的目录路径
- **预期**：返回明确错误（git init 或文件操作失败）

### X4 — 空密码 🤖

- **自动化**：`TestX4_EmptyPassword` @ `internal/service/edge_test.go`
- **操作**：用空字符串 `""` 作为密码创建和解密 vault
- **预期**：功能正常（Argon2 + AES-GCM 技术上允许空密码），确认行为一致

### X5 — 重复 BindRemoteRepo 🤖

- **自动化**：`TestX5_DuplicateBindRemoteRepo` @ `internal/service/edge_test.go`
- **前置**：已成功绑定一次
- **操作**：再次点击 **"绑定远程并同步"**
- **预期**：不报 `"remote already exists"` 错误（AddRemote 容错生效），正常执行同步逻辑

### X6 — BindRemoteRepo 空参数 🤖

- **自动化**：`TestX6_BindRemoteRepo_EmptyParams` @ `internal/service/edge_test.go`
- **操作 A**：`BindRemoteRepo("", url, pwd)` → 预期错误 `"仓库路径不能为空"`
- **操作 B**：`BindRemoteRepo(path, "", pwd)` → 预期错误 `"远程仓库地址不能为空"`

### X7 — Pull/Push 空路径 🤖

- **自动化**：`TestX7_PullPush_EmptyPath` @ `internal/service/edge_test.go`
- **操作 A**：`PullVault("")` → 预期错误 `"仓库路径不能为空"`
- **操作 B**：`PushVault("")` → 预期错误 `"仓库路径不能为空"`

---

## 推荐执行顺序

按依赖关系排列，前面的用例是后面的前置条件：

```
 1. G1~G3        → 验证 git 基础层修复
 2. G7~G11       → 验证新增 git 工具函数
 3. I1, I3, I4   → 验证本地初始化（含自动提交）
 4. R1~R4        → 验证状态查询
 5. L1, L2       → 验证空库返回空数组
 6. B8           → 双空初始化（最简单路径）
 7. B1, B2       → 本地有数据推送到远程
 8. B3, B4       → 远程有数据拉取到本地
 9. B5~B7        → 合并场景
10. S1~S4        → 日常同步
11. S5, S7       → 错误恢复与工作区清理
12. S6, S8       → 首次同步与边界
13. E1~E4        → 端到端完整流程（🖱️ 手工）
14. X1~X7        → 边界异常
```

**核心验证点**：每次操作后检查 vault.dat 能用正确密码解密，且条目数据完整不丢失。

---

## 测试覆盖统计

| 类别 | 总数 | 🤖 自动化 | 🖱️ 手工 |
|------|------|-----------|----------|
| git 层 (G) | 11 | 11 | 0 |
| BindRemoteRepo (B) | 8 | 8 | 0 |
| SyncVault (S) | 8 | 8 | 0 |
| InitLocalVault (I) | 4 | 4 | 0 |
| RepoStatus (R) | 4 | 4 | 0 |
| ListEntries (L) | 2 | 2 | 0 |
| 端到端 (E) | 4 | 0 | 4 |
| 边界异常 (X) | 7 | 7 | 0 |
| **合计** | **48** | **44** | **4** |

---

## 变更记录

| 日期 | 变更内容 |
|------|---------|
| 初版 | 32 个用例覆盖 git 层、BindRepo、SyncVault、InitLocalVault、端到端、异常 |
| 更新 | 新增 G7~G11（git 工具函数）、I4（空路径）、S7~S8（工作区清理/空路径）、R1~R4（RepoStatus）、L1~L2（空数组）、X6~X7（空参数）；修正 I1/I3 补充自动提交验证、B2 描述更正为已提交、S1/S2 补充 HasChanges 分支说明、B6 补充合并机制说明 |
| 自动化 | 为 44 个用例编写 Go 自动化测试代码（11 个 `_test.go` 文件），标注 4 个端到端用例为手工 GUI 测试；修复 BindRemoteRepo 重复绑定时工作区未清理的 bug |
