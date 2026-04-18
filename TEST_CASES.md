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
| `internal/vault` | `vault_test.go` | 模型 CRUD、MergeVault (B5/B6/B7)、标签、软删除 (D1~D4)、多空间 (SP1~SP10)、批量移动 (MV1~MV4) |
| `internal/storage` | `storage_test.go` | 序列化、SaveLoad 往返、X1 密码错误 |
| `internal/git` | `git_test.go` | G1–G11（exec 后端） |
| `internal/git` | `backend_test.go` | BK1–BK4（后端切换/规范化） |
| `internal/git` | `gogit_test.go` | GG1–GG10（go-git 后端） |
| `internal/config` | `config_test.go` | CFG-GC1 / CFG-GC2（git_client 解析）、CFG-SV1~CFG-SV4（Save 写回） |
| `internal/app` | `app_test.go` | APP-UC1~APP-UC5（UpdateAppConfig 校验 + 写盘 + 切后端） |
| `internal/service` | `init_test.go` | I1–I4 |
| `internal/service` | `status_test.go` | R1–R4 |
| `internal/service` | `entries_test.go` | L1–L2、CRUD 集成、软删除 (D5/D6) |
| `internal/service` | `spaces_test.go` | 多空间 CRUD + 按空间 CRUD (SP-I1~SP-I13) + 批量移动 (MV-I1~MV-I5) |
| `internal/service` | `bind_test.go` | B1–B8 |
| `internal/service` | `sync_test.go` | S1–S8、软删除同步 (D7/D8)、空间同步 (SP-S1/SP-S2) |
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

## 一 · A、Git 后端切换与 go-git 实现（14 个用例）

底层 Git 操作抽象出 `Backend` 接口，保留旧的 `exec` 后端（调用本机 `git` 命令），新增 `go-git` 后端（基于 `github.com/go-git/go-git/v5`，纯 Go 实现）。通过 `pwdmgr.config.json` 中的 `git_client` 字段选择后端，缺省/未知值回退为 `"exec"`。

### BK1 — Normalize 规范化后端名 🤖

- **自动化**：`TestBK1_Normalize` @ `internal/git/backend_test.go`
- **覆盖**：空串 / `"exec"` / `"EXEC"` / `"system"` / `"cli"` → `exec`；`"go-git"` / `"gogit"` / `"go_git"` / `"go-git-v5"` → `go-git`；未知值回退 `exec`

### BK2 — 默认后端是 exec 🤖

- **自动化**：`TestBK2_DefaultBackendIsExec` @ `internal/git/backend_test.go`
- **预期**：显式 `SetBackend("")` 后，`CurrentBackend()` 返回 `"exec"`

### BK3 — SetBackend 切换 exec / go-git 🤖

- **自动化**：`TestBK3_SetBackend` @ `internal/git/backend_test.go`
- **覆盖**：`exec ↔ go-git` 正常切换，未知值静默回退 `exec`

### BK4 — SetBackendStrict 严格校验 🤖

- **自动化**：`TestBK4_SetBackendStrict` @ `internal/git/backend_test.go`
- **覆盖**：合法值不报错；空串等价默认；未知值返回携带原值的错误

### CFG-GC1 — config.NormalizeGitClient 规范化 🤖

- **自动化**：`TestCFGGC1_NormalizeGitClient` @ `internal/config/config_test.go`
- **覆盖**：与 `git.Normalize` 一致的回退规则，保证 config 层和 git 层命名一致

### CFG-GC2 — Load 读取 git_client 字段 🤖

- **自动化**：`TestCFGGC2_LoadReadsGitClient` @ `internal/config/config_test.go`
- **覆盖**：json 中缺失字段、显式 `exec` / `go-git`、未知值回退、大小写兼容；并验证 `Snapshot().GitClient` 同步更新

### GG1 — go-git Init + IsGitRepo 🤖

- **自动化**：`TestGG1_InitAndIsGitRepo` @ `internal/git/gogit_test.go`
- **预期**：`PlainInit` 创建 `.git`，`IsGitRepo` 返回 true

### GG2 — go-git AddRemote 首次 + 重复覆盖 🤖

- **自动化**：`TestGG2_AddRemote_FirstThenOverride` @ `internal/git/gogit_test.go`
- **预期**：首次 `AddRemote` 成功；重复添加不报错且 URL 被更新

### GG3 — go-git Commit 🤖

- **自动化**：`TestGG3_Commit` @ `internal/git/gogit_test.go`
- **预期**：提交后 `CurrentBranch` 返回 `main` 或 `master`

### GG4 — go-git HasChanges 🤖

- **自动化**：`TestGG4_HasChanges` @ `internal/git/gogit_test.go`
- **覆盖**：干净仓库、修改已跟踪文件、恢复后新增未跟踪文件

### GG5 — go-git RestoreFile 恢复已跟踪文件 🤖

- **自动化**：`TestGG5_RestoreFile` @ `internal/git/gogit_test.go`
- **预期**：`RestoreFile` 把文件内容还原为 HEAD 版本

### GG6 — go-git Push 首次推送到空 bare 远程 🤖

- **自动化**：`TestGG6_Push_FirstTime` @ `internal/git/gogit_test.go`
- **预期**：Push 成功，`RemoteHasCommit` 返回 true（空远程时兼容 `ErrEmptyRemoteRepository`）

### GG7 — go-git Pull 回退：本地无 HEAD 🤖

- **自动化**：`TestGG7_Pull_FallbackNoTrackingBranch` @ `internal/git/gogit_test.go`
- **预期**：`PlainInit + AddRemote` 后 Pull 能正确 checkout 远程默认分支，并拉取文件到工作区

### GG8 — go-git Pull fast-forward 场景 🤖

- **自动化**：`TestGG8_Pull_FastForward` @ `internal/git/gogit_test.go`
- **预期**：两个本地仓库交替 push/pull，Pull 端能 fast-forward 到最新文件

### GG9 — go-git 远程元信息组合 🤖

- **自动化**：`TestGG9_RemoteMetadata` @ `internal/git/gogit_test.go`
- **覆盖**：`HasOriginRemote` / `RemoteURL` / `RemoteHasCommit` 在无 remote / 有 remote / 有远程提交时的返回

### GG10 — 顶层 API 经 SetBackend 切到 go-git 后可用 🤖

- **自动化**：`TestGG10_TopLevelDispatchViaGoGit` @ `internal/git/gogit_test.go`
- **预期**：`SetBackend("go-git")` 后直接调用包级 `Init / AddRemote / Commit / Push / RemoteHasCommit` 仍能完成端到端推送

---

## 一 · B、界面编辑应用配置（9 个用例）

`SettingsTab` 提供编辑/保存表单，后端 `UpdateAppConfig(repoRoot, remoteURL, gitClient) -> Snapshot` 会原子写回配置文件、重新 Load、同步切换 git 后端；`config.Save` 保留未知字段避免吞用户自定义扩展。

### CFG-SV1 — Save 写回三字段并可被 Load 读回 🤖

- **自动化**：`TestCFGSV1_SaveRoundTrip` @ `internal/config/config_test.go`
- **覆盖**：`Save` 成功后 `Load` 能读回同样的 `repo_root / remote_url / git_client`，且 `Path()` 更新为目标路径

### CFG-SV2 — Save 保留未知字段 🤖

- **自动化**：`TestCFGSV2_SavePreservesUnknownFields` @ `internal/config/config_test.go`
- **覆盖**：预先写入含 `theme / extra / custom_flag` 等字段的 json，修改 Config 后 Save，未知字段完整保留

### CFG-SV3 — 新建场景自动创建目录 🤖

- **自动化**：`TestCFGSV3_SaveFirstTime` @ `internal/config/config_test.go`
- **覆盖**：`resolvedPath` 为空时回退 `ResolveConfigPath`，并按需 `MkdirAll`；空 `git_client` 最终落盘为 `exec`

### CFG-SV4 — Save 规范化 git_client 🤖

- **自动化**：`TestCFGSV4_SaveNormalizesGitClient` @ `internal/config/config_test.go`
- **覆盖**：写入 `"GoGit"` 等变体时，磁盘里落地的是规范名 `go-git`

### CFG-CP1 — CandidatePaths 搜索优先级（可执行目录 > wd > 用户配置目录）🤖

- **自动化**：`TestCFGCP1_CandidatePathsOrder` @ `internal/config/config_test.go`
- **覆盖**：返回顺序严格为「可执行文件同级 → 当前工作目录 → 用户配置目录」，三者均按是否可解析裁剪

### CFG-CP2 — 工作目录命中优先于用户配置目录 🤖

- **自动化**：`TestCFGCP2_ResolvePrefersWdOverUserDir` @ `internal/config/config_test.go`
- **覆盖**：当 wd 下存在 `pwdmgr.config.json` 时，`ResolveConfigPath` 返回 wd 路径而不是落到用户配置目录

### APP-UC1 — UpdateAppConfig 要求 repo_root 非空 🤖

- **自动化**：`TestAPPUC1_UpdateAppConfig_RequiresRepoRoot` @ `internal/app/app_test.go`
- **预期**：空白字符串应返回包含"仓库路径"的错误

### APP-UC2 — UpdateAppConfig 要求绝对路径 🤖

- **自动化**：`TestAPPUC2_UpdateAppConfig_RequiresAbsPath` @ `internal/app/app_test.go`
- **预期**：相对路径应返回包含"绝对路径"的错误

### APP-UC3 — UpdateAppConfig 拒绝文件路径 🤖

- **自动化**：`TestAPPUC3_UpdateAppConfig_RejectsFilePath` @ `internal/app/app_test.go`
- **预期**：指向普通文件时应返回包含"文件"的错误，不会覆盖原配置

### APP-UC4 — UpdateAppConfig 正常流程 🤖

- **自动化**：`TestAPPUC4_UpdateAppConfig_Success` @ `internal/app/app_test.go`
- **预期**：首次 `UpdateAppConfig(repoDir, url, "go-git")` → 文件落盘 + Snapshot 正确 + `git.CurrentBackend()` 变 `go-git`；再改回 `"exec"` 后 backend 同步切回

### APP-UC5 — UpdateAppConfig 未知 git_client 回退 exec 🤖

- **自动化**：`TestAPPUC5_UpdateAppConfig_UnknownGitClientFallsBack` @ `internal/app/app_test.go`
- **预期**：`"libgit2"` 等未知值保存后 Snapshot 中显示 `exec`，git backend 也保持 `exec`

### CFG-E1 — 设置页编辑 → 保存 → 磁盘同步 🖱️

- **手工**：打开"设置"→ 点"编辑配置"→ 修改 `repo_root` / `remote_url` / 下拉切换 `git_client` → 保存
- **预期**：
  1. 顶栏 Toast 提示"配置已保存"
  2. 退出编辑态，只读视图立即显示最新值
  3. `pwdmgr.config.json` 内容被原子写回，切出再切回"设置"页字段仍正确
  4. 若切换了 `git_client`，后续 Sync/Pull/Push 走新的后端

### CFG-E2 — 编辑校验失败保持编辑态 🖱️

- **手工**：在编辑模式把 `repo_root` 清空后保存，或填入相对路径 `./foo`
- **预期**：Toast 报红提示；页面仍处于编辑态，字段保留用户输入，不会错误退出

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

## 九、软删除（8 个用例）

> 背景：合并逻辑无法区分"本地删除"和"远程新增"，会导致本地删除的条目同步后被远程旧版本"复活"。
> 设计：`Entry` 新增 `DeletedAt int64` 字段；`DeleteEntry` 改为软删除（打标记 + 刷新 `UpdatedAt`）；
> 合并时仍按 `UpdatedAt` 取较新一方，天然保留删除标记；`ListEntries` 在 service 层过滤 `DeletedAt > 0` 的条目。

### D1 — 软删除打标记 🤖

- **自动化**：`TestVault_DeleteEntry_SoftDelete` @ `internal/vault/vault_test.go`
- **操作**：`NewVault` → 添加两个条目 → `DeleteEntry(e2.ID)`
- **预期**：
  - `v.Entries` 长度仍为 2（物理上保留）
  - 被删条目 `DeletedAt > 0` 且与 `UpdatedAt` 相等
  - 另一条目 `IsDeleted()` 为 false

### D1b — ActiveEntries 过滤软删除 🤖

- **自动化**：`TestVault_ActiveEntries_FilterDeleted` @ `internal/vault/vault_test.go`
- **操作**：添加 2 条目 → 删除其中一个 → `v.ActiveEntries()`
- **预期**：返回 1 条（未删除的那条）

### D1c — DeleteEntry 对已删除条目幂等 🤖

- **自动化**：`TestVault_DeleteEntry_Idempotent` @ `internal/vault/vault_test.go`
- **操作**：`DeleteEntry(id)` 两次
- **预期**：第二次调用不覆盖 `DeletedAt`（保持首次删除的时间戳）

### D2 — 合并：本地软删除胜出旧远程 🤖

- **自动化**：`TestMergeVault_LocalDelete_BeatsOlderRemote` @ `internal/vault/vault_test.go`
- **前置**：local 有 X（`UpdatedAt=200, DeletedAt=200`），remote 有 X（`UpdatedAt=100`，无删除标记）
- **操作**：`MergeVault(local, remote)`
- **预期**：合并结果只有 1 条，且保留 `DeletedAt=200`，不会被远程旧版本复活

### D3 — 合并：远程更新胜出本地删除（恢复条目）🤖

- **自动化**：`TestMergeVault_NewerRemoteUpdate_OverridesLocalDelete` @ `internal/vault/vault_test.go`
- **前置**：local 有 X（`UpdatedAt=100, DeletedAt=100`），remote 有 X（`UpdatedAt=200`，无删除标记）
- **操作**：`MergeVault(local, remote)`
- **预期**：合并结果取远程版本，`IsDeleted()=false`，`UpdatedAt=200`

### D4 — 合并：本地软删除，远程从未有过该条目 🤖

- **自动化**：`TestMergeVault_LocalDelete_RemoteMissing` @ `internal/vault/vault_test.go`
- **前置**：local 有 X（`DeletedAt=200`），remote 为空
- **操作**：`MergeVault(local, remote)`
- **预期**：合并结果保留带删除标记的 X，确保下次同步能把删除传播到远程

### D5 — ListEntries 过滤软删除条目 🤖

- **自动化**：`TestD5_ListEntries_FiltersSoftDeleted` @ `internal/service/entries_test.go`
- **操作**：`AddEntry` 两条 → `DeleteEntry` 其中一条 → `ListEntries`
- **预期**：
  - `ListEntries` 仅返回未删除那条（长度 1）
  - 磁盘加密文件中仍保留两条（其中一条带 `DeletedAt`）

### D6 — UpdateEntry 拒绝软删除条目（防复活）🤖

- **自动化**：`TestD6_UpdateEntry_SoftDeletedReturnsError` @ `internal/service/entries_test.go`
- **前置**：条目已被软删除
- **操作**：用原 ID 调用 `UpdateEntry`
- **预期**：返回错误 `"条目不存在"`，`DeletedAt` 状态不被前端误传清除

### D7 — SyncVault 软删除传播且不被远程复活 🤖

- **自动化**：`TestD7_SyncVault_SoftDeletePropagates` @ `internal/service/sync_test.go`
- **前置**：远程初始有条目 X（`UpdatedAt=50`），本地绑定拉取
- **操作**：
  1. 本地 `DeleteEntry("x")`（软删除）
  2. `SyncVault`（推送删除标记）
  3. 再次 `SyncVault`
- **预期**：
  - 本地 `ListEntries` 始终为空（不可见）
  - 克隆远程后 vault 内仍有 1 条 X，但 `IsDeleted()=true`（标记已传播）
  - 第二次 Sync 不会把远程旧版本"复活"

### D8 — SyncVault 远程更新更晚则恢复条目 🤖

- **自动化**：`TestD8_SyncVault_NewerRemoteUpdateRestoresDeleted` @ `internal/service/sync_test.go`
- **前置**：
  - 远程原有 X（`UpdatedAt=50`）
  - 本地删除 X（`DeletedAt=100, UpdatedAt=100`）
  - 另一设备在 `UpdatedAt=300` 修改 X 的密码为 `"restored"`
- **操作**：`SyncVault`
- **预期**：合并后本地 `ListEntries` 返回 1 条，`Password="restored"`、`UpdatedAt=300`、`IsDeleted()=false`

---

## 十、多空间支持（25 个用例）

> 背景：Vault 增加 `Spaces` 列表；每个 `Entry` 带 `SpaceID`，前端按空间隔离展示。
> 默认空间 ID 固定为 `default`，不可删除/不可重命名；合并时 `Spaces` 同样按 `UpdatedAt` 取较新一方，支持软删除。
> 旧版本 vault.dat（无 `Spaces` 字段、无 `SpaceID` 的条目）加载时由 `EnsureDefaultSpace` 自动迁移到默认空间。

### 10.1 模型与合并（自动化，vault 层）

#### SP1 — NewVault 自动包含默认空间 🤖
- **自动化**：`TestNewVault_HasDefaultSpace` @ `internal/vault/vault_test.go`
- **预期**：`NewVault().ActiveSpaces()` 长度为 1，ID = `default`，`Name` = `默认空间`

#### SP2 — NewEntry / NewEntryInSpace 正确归属空间 🤖
- **自动化**：`TestNewEntry_DefaultSpaceAssigned` @ `internal/vault/vault_test.go`
- **预期**：`NewEntry` 默认 `SpaceID=default`；`NewEntryInSpace("work", …)` 得 `work`；空字符串回退默认

#### SP3 — AddSpace 名称去空格 / 重名 / 空名校验 🤖
- **自动化**：`TestVault_AddSpace` @ `internal/vault/vault_test.go`
- **预期**：添加重复（活跃）同名 → `ErrSpaceNameDuplicate`；全空白 → `ErrSpaceNameEmpty`

#### SP4 — RenameSpace 规则 🤖
- **自动化**：`TestVault_RenameSpace` @ `internal/vault/vault_test.go`
- **预期**：默认空间 → `ErrSpaceProtected`；同名 → `ErrSpaceNameDuplicate`；空名 → `ErrSpaceNameEmpty`；不存在 → `ErrSpaceNotFound`；正常重命名后 `UpdatedAt` 刷新

#### SP5 — DeleteSpace 规则 🤖
- **自动化**：`TestVault_DeleteSpace` @ `internal/vault/vault_test.go`
- **预期**：默认空间 → `ErrSpaceProtected`；含活跃条目 → `ErrSpaceNotEmpty`；空空间 → 成功软删除；再次删除同 ID → `ErrSpaceNotFound`

#### SP6 — EntriesInSpace 过滤 🤖
- **自动化**：`TestVault_EntriesInSpace` @ `internal/vault/vault_test.go`
- **预期**：返回指定空间下未软删除的条目；空 `spaceID` 视作默认空间

#### SP7 — EnsureDefaultSpace 迁移旧 vault 🤖
- **自动化**：`TestVault_EnsureDefaultSpace_MigratesLegacy` @ `internal/vault/vault_test.go`
- **前置**：手工构造 `Vault{Spaces: nil, Entries: [{SpaceID: ""}, …]}`
- **预期**：调用 `EnsureDefaultSpace` 后 `Spaces` 含默认空间，所有无 `SpaceID` 的条目被归入默认空间

#### SP8 — Merge 新增/保留空间 🤖
- **自动化**：`TestMergeVault_MergesSpaces` @ `internal/vault/vault_test.go`
- **前置**：本地独有 `work` 空间；远程独有 `personal` 空间
- **预期**：合并结果包含默认 + work + personal 三个空间

#### SP9 — Merge 空间冲突取较新者 🤖
- **自动化**：`TestMergeVault_SpaceConflict_NewerWins` @ `internal/vault/vault_test.go`
- **前置**：同 ID 的 Space 双端 `UpdatedAt` 不同
- **预期**：`UpdatedAt` 更大一方的 `Name` 胜出

#### SP10 — Merge 软删除空间胜出 🤖
- **自动化**：`TestMergeVault_DeletedSpace_BeatsOlderRemote` @ `internal/vault/vault_test.go`
- **前置**：本地 `Space{DeletedAt=200, UpdatedAt=200}` + 远程 `Space{UpdatedAt=100}`
- **预期**：合并结果保留删除标记，与条目软删除逻辑一致

### 10.2 service 层集成（自动化）

#### SP-I1 — 初始化后 ListSpaces 仅含默认空间 🤖
- **自动化**：`TestSPI1_ListSpaces_DefaultOnly` @ `internal/service/spaces_test.go`

#### SP-I2 — CreateSpace + ListSpaces 排序（默认空间置顶）🤖
- **自动化**：`TestSPI2_CreateSpace_ListOrder` @ `internal/service/spaces_test.go`

#### SP-I3 — CreateSpace 重复名称被拒绝 🤖
- **自动化**：`TestSPI3_CreateSpace_DuplicateName` @ `internal/service/spaces_test.go`
- **预期**：返回 `ErrSpaceNameDuplicate`

#### SP-I4 — RenameSpace 默认空间受保护 🤖
- **自动化**：`TestSPI4_RenameSpace_ProtectDefault` @ `internal/service/spaces_test.go`

#### SP-I5 — RenameSpace 正常流程 🤖
- **自动化**：`TestSPI5_RenameSpace_Success` @ `internal/service/spaces_test.go`

#### SP-I6 — DeleteSpace 非空空间拒绝 🤖
- **自动化**：`TestSPI6_DeleteSpace_NotEmptyRejected` @ `internal/service/spaces_test.go`
- **预期**：返回 `ErrSpaceNotEmpty`

#### SP-I7 — DeleteSpace 默认空间受保护 🤖
- **自动化**：`TestSPI7_DeleteSpace_ProtectDefault` @ `internal/service/spaces_test.go`

#### SP-I8 — DeleteSpace 空空间软删除成功 🤖
- **自动化**：`TestSPI8_DeleteSpace_EmptySuccess` @ `internal/service/spaces_test.go`
- **预期**：`ListSpaces` 中不再出现已删除空间

#### SP-I9 — AddEntry 指定空间 + ListEntries 空间隔离 🤖
- **自动化**：`TestSPI9_AddAndListBySpace` @ `internal/service/spaces_test.go`
- **预期**：工作、个人、默认空间下的条目互不混淆

#### SP-I10 — AddEntry / ListEntries 空间不存在时报错 🤖
- **自动化**：`TestSPI10_InvalidSpaceRejected` @ `internal/service/spaces_test.go`
- **预期**：错误信息包含 "空间不存在"

#### SP-I11 — UpdateEntry 跨空间移动条目 🤖
- **自动化**：`TestSPI11_UpdateEntry_MoveBetweenSpaces` @ `internal/service/spaces_test.go`
- **预期**：将条目 `space_id` 改为另一个空间并 `UpdateEntry` 后，源空间 `ListEntries` 为空、目标空间能看到

#### SP-I12 — UpdateEntry 目标空间不存在被拒绝 🤖
- **自动化**：`TestSPI12_UpdateEntry_InvalidTargetSpace` @ `internal/service/spaces_test.go`

#### SP-I13 — 旧 vault.dat 加载后自动迁移 🤖
- **自动化**：`TestSPI13_LegacyVaultMigration` @ `internal/service/spaces_test.go`
- **前置**：直接写入一个不含 `Spaces` 字段的 vault.dat
- **预期**：`ListEntries` / `ListSpaces` 均可用，旧条目归入默认空间

### 10.3 同步场景（自动化）

#### SP-S1 — 不同空间各自合并，互不干扰 🤖
- **自动化**：`TestSPS1_SyncVault_DifferentSpacesMergeIndependently` @ `internal/service/sync_test.go`
- **前置**：本地新增「工作」空间 + 条目；远程（另一设备）新增「个人」空间 + 条目
- **预期**：Sync 后本地 `ListSpaces` 含三空间；工作/个人空间下条目独立

#### SP-S2 — 远程删除的空间同步到本地后被过滤 🤖
- **自动化**：`TestSPS2_SyncVault_RemoteDeletedSpaceHidden` @ `internal/service/sync_test.go`
- **前置**：本地创建并 Sync 了「存档」空间；远程将该空间 `DeletedAt` 设为更晚时间
- **预期**：再次 Sync 后 `ListSpaces` 不再包含该空间

### 10.4 批量移动（9 个用例）

> 能力：单条或一次移动多条条目到另一空间。service 层的 `MoveEntries(ids, targetSpaceID)` 保证：
> 目标空间必须存在且未删除；已软删除、已在目标空间、不存在的 ID 会被静默跳过；返回实际被移动的数量。
> 被移动的条目会刷新 `UpdatedAt`，在后续合并中正确传播到其它设备。

#### MV1 — MoveEntries 基本流程 🤖
- **自动化**：`TestVault_MoveEntries_Basic` @ `internal/vault/vault_test.go`
- **预期**：返回移动数量，被移动的 `SpaceID` 改变且 `UpdatedAt` 刷新；未指定条目不受影响

#### MV2 — 目标空间为空回退默认空间 🤖
- **自动化**：`TestVault_MoveEntries_EmptyTargetFallsBackToDefault` @ `internal/vault/vault_test.go`
- **预期**：`MoveEntries(ids, "")` 相当于移动到 `default`

#### MV3 — 静默跳过无效条目 🤖
- **自动化**：`TestVault_MoveEntries_SkipsInvalidEntries` @ `internal/vault/vault_test.go`
- **预期**：已在目标空间 / 已软删除 / 不存在的 ID 均被跳过，不计入返回值，也不会改动对应条目

#### MV4 — 空 ID 列表为 no-op 🤖
- **自动化**：`TestVault_MoveEntries_EmptyIDs` @ `internal/vault/vault_test.go`
- **预期**：`MoveEntries(nil, target)` 返回 0，不修改任何条目

#### MV-I1 — service 层批量移动 🤖
- **自动化**：`TestMVI1_MoveEntries_Batch` @ `internal/service/spaces_test.go`
- **预期**：`ListEntries(default)` 减少，`ListEntries(work)` 相应增加

#### MV-I2 — service 层单条移动（ids 长度为 1）🤖
- **自动化**：`TestMVI2_MoveEntries_Single` @ `internal/service/spaces_test.go`
- **预期**：前端"单条移动"复用同一 API 即可

#### MV-I3 — 目标空间不存在时报错 🤖
- **自动化**：`TestMVI3_MoveEntries_TargetNotFound` @ `internal/service/spaces_test.go`
- **预期**：错误信息含 "空间不存在"

#### MV-I4 — 空 ID 列表不报错 🤖
- **自动化**：`TestMVI4_MoveEntries_EmptyIDs` @ `internal/service/spaces_test.go`
- **预期**：`MoveEntries(repo, pwd, nil, target)` 与 `[]string{}` 均返回 `(0, nil)`

#### MV-I5 — 部分 ID 合法仍能成功保存 🤖
- **自动化**：`TestMVI5_MoveEntries_PartialValid` @ `internal/service/spaces_test.go`
- **预期**：真实移动数 = 1，ghost ID 被忽略；vault.dat 正常写回

### 10.5 手工测试（🖱️）

#### SP-E1 — 解锁后看到空间切换器 🖱️
- **操作**：解锁保险库
- **预期**：顶部看到"默认空间"chip，旁边有 "+" 按钮
- **验证**：当前空间名称显示在条目数量说明行中

#### SP-E2 — 新建空间 🖱️
- **操作**：点击 "+" → 输入名称"工作" → 保存
- **预期**：tabs 中新增"工作" chip，自动切换到该空间；条目列表为空
- **验证**：`ListVaultSpaces` 下次返回含"工作"

#### SP-E3 — 在空间下新增条目 🖱️
- **操作**：在"工作"空间点击"添加条目"，填写并保存
- **预期**：条目仅在"工作"空间可见；切回"默认空间"后看不到

#### SP-E4 — 重命名空间 🖱️
- **操作**：选中非默认空间 → 点击 "重命名" → 输入新名 → 保存
- **预期**：tab 上显示新名；默认空间不显示"重命名"按钮

#### SP-E5 — 删除空间 🖱️
- **操作 A**：删除非空空间 → 预期错误提示"空间下仍有条目"
- **操作 B**：清空条目后再删除 → 成功，自动切回默认空间
- **操作 C**：默认空间不显示"删除"按钮

#### SP-E6 — 多设备同步空间 🖱️
- **操作**：设备 A 新建"工作"空间并添加条目 → Sync → 设备 B Sync
- **预期**：设备 B 解锁后空间切换器中出现"工作"，切过去能看到条目

#### MV-E1 — 单条"移动"按钮 🖱️
- **操作**：鼠标移入任一条目卡片 → 点击 footer 的"移动"按钮
- **预期**：弹出"移动到空间"对话框，列出除当前空间外的所有空间；选择后条目消失于源空间，出现在目标空间

#### MV-E2 — 进入/退出批量选择模式 🖱️
- **操作**：点击工具栏的"批量选择"图标按钮
- **预期**：搜索栏被选择工具栏替代，显示"已选 0 条 / 全选当前 / 移动到… / 完成"；条目卡片出现复选框，hover 行为变为"整张卡片可点击切换选中"；条目 footer 隐藏；点击"完成"或切换空间退出选择模式并清空已选

#### MV-E3 — 全选/反选当前视图 🖱️
- **操作**：进入选择模式，先用搜索或标签筛选出部分条目 → 点击"全选当前"
- **预期**：选中数等于筛选后数量；再次点击变为"取消全选"并清空已选（仅限当前视图内）

#### MV-E4 — 批量移动 🖱️
- **操作**：选中若干条目 → 点击"移动到…" → 选择目标空间
- **预期**：提示"已移动 N 条"，条目从当前空间列表消失；自动退出选择模式；切到目标空间能看到它们；工作流对只选 1 条时同样生效

#### MV-E5 — 源空间唯一时的提示 🖱️
- **前置**：只有"默认空间"（未创建其它空间）
- **操作**：对任意条目点击"移动"或批量"移动到…"
- **预期**：提示"没有其它空间可用，请先新建一个空间"（不会打开空对话框）

#### MV-E6 — 跨设备验证移动传播 🖱️
- **操作**：设备 A 将条目从"默认空间"移动到"工作"空间 → Sync；设备 B Sync
- **预期**：设备 B 在"工作"空间能看到这些条目，"默认空间"下看不到（`UpdatedAt` 刷新保证合并中胜出）

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
 9. BK1~BK4      → git 后端规范化与切换
10. GG1~GG10     → go-git 后端功能对等验证
11. CFG-GC1~CFG-GC2 → git_client 配置解析
    CFG-SV1~CFG-SV4   → config.Save 原子写回
    APP-UC1~APP-UC5   → UpdateAppConfig 校验与后端切换
    CFG-E1 / CFG-E2   → 🖱️ 设置页编辑流程
12. B5~B7        → 合并场景
13. S1~S4        → 日常同步
14. S5, S7       → 错误恢复与工作区清理
15. S6, S8       → 首次同步与边界
16. D1~D4        → 软删除模型与合并
17. D5~D8        → 软删除 service/sync 集成
18. SP1~SP10     → 多空间模型与合并
19. SP-I1~SP-I13 → 多空间 service 层集成
20. SP-S1/SP-S2  → 多空间同步场景
21. MV1~MV4      → 批量移动模型
22. MV-I1~MV-I5  → 批量移动 service 集成
23. E1~E4        → 端到端完整流程（🖱️ 手工）
24. SP-E1~SP-E6  → 多空间前端端到端（🖱️ 手工）
25. MV-E1~MV-E6  → 批量移动前端（🖱️ 手工）
26. X1~X7        → 边界异常
```

**核心验证点**：每次操作后检查 vault.dat 能用正确密码解密，且条目数据完整不丢失。

---

## 测试覆盖统计

| 类别 | 总数 | 🤖 自动化 | 🖱️ 手工 |
|------|------|-----------|----------|
| git 层 (G) | 11 | 11 | 0 |
| git 后端切换 (BK) | 4 | 4 | 0 |
| go-git 后端 (GG) | 10 | 10 | 0 |
| config git_client (CFG-GC) | 2 | 2 | 0 |
| config.Save (CFG-SV) | 4 | 4 | 0 |
| 配置搜索路径 (CFG-CP) | 2 | 2 | 0 |
| UpdateAppConfig (APP-UC) | 5 | 5 | 0 |
| 设置页编辑 (CFG-E) | 2 | 0 | 2 |
| BindRemoteRepo (B) | 8 | 8 | 0 |
| SyncVault (S) | 8 | 8 | 0 |
| InitLocalVault (I) | 4 | 4 | 0 |
| RepoStatus (R) | 4 | 4 | 0 |
| ListEntries (L) | 2 | 2 | 0 |
| 软删除 (D) | 8 | 8 | 0 |
| 多空间 模型 (SP) | 10 | 10 | 0 |
| 多空间 service (SP-I) | 13 | 13 | 0 |
| 多空间 同步 (SP-S) | 2 | 2 | 0 |
| 多空间 前端 (SP-E) | 6 | 0 | 6 |
| 批量移动 模型 (MV) | 4 | 4 | 0 |
| 批量移动 service (MV-I) | 5 | 5 | 0 |
| 批量移动 前端 (MV-E) | 6 | 0 | 6 |
| 端到端 (E) | 4 | 0 | 4 |
| 边界异常 (X) | 7 | 7 | 0 |
| **合计** | **131** | **113** | **18** |

---

## 变更记录

| 日期 | 变更内容 |
|------|---------|
| 初版 | 32 个用例覆盖 git 层、BindRepo、SyncVault、InitLocalVault、端到端、异常 |
| 更新 | 新增 G7~G11（git 工具函数）、I4（空路径）、S7~S8（工作区清理/空路径）、R1~R4（RepoStatus）、L1~L2（空数组）、X6~X7（空参数）；修正 I1/I3 补充自动提交验证、B2 描述更正为已提交、S1/S2 补充 HasChanges 分支说明、B6 补充合并机制说明 |
| 自动化 | 为 44 个用例编写 Go 自动化测试代码（11 个 `_test.go` 文件），标注 4 个端到端用例为手工 GUI 测试；修复 BindRemoteRepo 重复绑定时工作区未清理的 bug |
| 软删除 | Entry 增加 `DeletedAt`，`DeleteEntry` 改为软删除，`ListEntries` 过滤删除条目，`UpdateEntry` 拒绝已软删除 ID；合并逻辑保持不变（仍按 `UpdatedAt` 比较），解决"本地删除同步后被远程复活"问题；新增 D1~D8 共 8 个自动化用例（模型 + 合并 + service 集成 + sync 端到端） |
| 多空间 | Vault 增加 `Spaces` 列表、Entry 增加 `SpaceID`；新增 `Space` CRUD（默认空间受保护、非空空间禁止删除、同名检测、软删除）；`MergeVault` 按同一 `UpdatedAt` 规则合并空间，支持软删除传播；`service.ListEntries` 增加 `spaceID` 参数并过滤；新增 `ListSpaces / CreateSpace / RenameSpace / DeleteSpace` service 与 Wails API；旧版本 vault.dat 由 `EnsureDefaultSpace` 自动迁移到默认空间；前端 VaultTab 新增空间切换器与管理 UI。自动化新增 SP1~SP10（vault 模型/合并）、SP-I1~SP-I13（service）、SP-S1/SP-S2（同步），以及 SP-E1~SP-E6 手工用例，共 31 个新用例 |
| Git 后端抽象 | `internal/git` 抽出 `Backend` 接口，保留原有基于 `os/exec` 的 `execBackend`，新增基于 `github.com/go-git/go-git/v5` 的 `goGitBackend`（纯 Go 实现，不依赖本机 git）；`SetBackend / CurrentBackend` 允许运行时切换；`internal/config.Config` 增加 `git_client` 字段（`exec` / `go-git`，未配置或取值未知时回退 `exec`），`app.Startup / ReloadConfig` 加载配置后自动调用 `git.SetBackend` 同步后端；对外公共 API（`Clone / Pull / Push / Commit / ...`）签名不变，全部通过 dispatcher 调度到当前后端。自动化新增 BK1~BK4（后端切换/规范化）、GG1~GG10（go-git 功能对等）、CFG-GC1~CFG-GC2（配置解析）共 16 个用例 |
| 设置页编辑配置 | `internal/config.Config` 新增 `Save()`（JSON map 合并 → `tmp + rename` 原子写回，保留未知字段）和 `Path()` / `ResolvedOrCandidatePath()`；`internal/app` 暴露 `UpdateAppConfig(repoRoot, remoteURL, gitClient) -> Snapshot`，做绝对路径 / 非文件 / 空值校验后写盘、重新 Load 并同步切换 git 后端；手工维护的 `frontend/wailsjs/go/app/App.{d.ts,js}` 同步补上 `UpdateAppConfig` 绑定；`SettingsTab.vue` 重写为只读 / 编辑两种模式，`git_client` 使用下拉选择，`App.vue` 提供 `doSaveAppConfig`（保存前自动锁定 vault）。自动化新增 CFG-SV1~CFG-SV4（Save）、APP-UC1~APP-UC5（UpdateAppConfig）共 9 个用例，手工新增 CFG-E1 / CFG-E2 共 2 个 |
| 配置搜索优先级 | `CandidatePaths()` 搜索顺序调整为「可执行文件同级 → 当前工作目录 → 用户配置目录」（环境变量 `PWDMGR_CONFIG` 仍然最优先），便于便携式部署与 `wails dev` 开发；`ResolveConfigPath` 在所有候选都不存在时仍回退到用户配置目录，保证首次 `Save()` 在 macOS `.app` 场景下可写。README 表格与提示同步更新。自动化新增 CFG-CP1 / CFG-CP2 共 2 个用例 |
| 批量移动 | Vault 新增 `MoveEntries(ids, targetSpaceID)`，静默跳过无效 ID、对被移动条目刷新 `UpdatedAt` 保证同步胜出；service 层 `MoveEntries` 校验目标空间合法性后写回 vault.dat；Wails 绑定 `MoveVaultEntries(password, targetSpaceID, ids) -> (moved int)`；前端新增 `SpacePickerDialog` 通过 `provide('askSpace')` 提供选择空间的通用能力，EntryCard 增加"移动"按钮、选择模式下的复选框，VaultTab 新增批量选择工具栏（全选当前 / 移动到… / 完成）。自动化新增 MV1~MV4（vault）+ MV-I1~MV-I5（service）共 9 个用例，手工新增 MV-E1~MV-E6 共 6 个用例 |
