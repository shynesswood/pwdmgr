# PwdMgr

基于 Git 同步的本地密码管理器。数据用 AES-256-GCM 加密存储，通过 Git 仓库在多设备间同步，无需依赖第三方云服务。

## 特性

- **端到端加密** — Argon2id 派生密钥 + AES-256-GCM 加密，密码库文件 (`vault.dat`) 以二进制密文形式存储
- **Git 同步** — 利用任意 Git 远程仓库（GitHub、GitLab、自建 Gitea 等）在多台设备间同步
- **可切换 Git 后端** — 默认走本机 `git` 命令；也可在设置页或配置文件中切到 `go-git` 纯 Go 实现，完全免安装 Git
- **界面管理配置** — 设置页支持直接编辑本地仓库路径 / 远程地址 / Git 客户端，保存后写回 `pwdmgr.config.json` 并热切换后端，保留未知字段不丢
- **多空间隔离** — 支持创建多个逻辑空间（如"个人 / 工作"），条目按空间隔离展示，可单条或批量在空间之间移动
- **软删除与合并** — 删除记 `deleted_at` 不立即清除，多设备并发修改时按 `updated_at` 合并条目/空间，无需手动处理冲突
- **跨平台桌面应用** — 基于 [Wails](https://wails.io) 构建，支持 Windows / macOS / Linux
- **零外部依赖** — 不依赖数据库或后台服务；使用 go-git 后端时也不再依赖本机 Git

## 技术栈

| 层 | 技术 |
|---|------|
| 后端 | Go 1.26+、Wails v2 |
| 前端 | Vue 3、Vite |
| 加密 | Argon2id (golang.org/x/crypto) + AES-256-GCM (crypto/aes) |
| 同步 | 可切换：系统 `git` 命令（`os/exec`） / [go-git v5](https://github.com/go-git/go-git)（纯 Go） |
| 存储 | 单文件 `vault.dat`（JSON → 加密 → 二进制） |

## 快速开始

### 环境要求

- Go 1.26+
- Node.js 16+
- Git（**可选**）— 仅 `git_client: "exec"`（默认）时需要；切到 `"go-git"` 后可免安装本机 Git
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### 构建 & 运行

```bash
# 克隆项目
git clone <repo-url>
cd pwdmgr

# 开发模式
wails dev

# 生产构建
wails build
```

### 安装（macOS）

```bash
# 1. 构建（通用二进制，同时支持 Intel 和 Apple Silicon）
wails build -platform darwin/universal

# 2. 创建配置目录并放入配置文件
mkdir -p ~/Library/Application\ Support/kPass
cp pwdmgr.config.example.json ~/Library/Application\ Support/kPass/pwdmgr.config.json
# 编辑配置文件，填入实际的 repo_root 和 remote_url

# 3. 将应用拖入 Applications（可选，放在任意位置均可）
cp -r build/bin/kPass.app /Applications/
```

> 配置文件路径由用户目录决定，与 `.app` 的安装位置无关，两步操作没有先后顺序要求。

### 安装（Windows）

```bash
wails build -platform windows/amd64
```

将 `build/bin/` 下的 `kPass.exe` 放到任意目录，在同目录下创建 `pwdmgr.config.json` 即可；或将配置文件放到 `%AppData%\kPass\pwdmgr.config.json`。

### 配置

创建 `pwdmgr.config.json`：

```json
{
  "repo_root": "/绝对路径/到你的密码库-git仓库根目录",
  "remote_url": "git@github.com:你的用户/远程仓库.git",
  "git_client": "exec",
  "ssh_key_path": "",
  "ssh_key_passphrase": ""
}
```

| 字段 | 必填 | 默认 | 说明 |
|------|:----:|:----:|------|
| `repo_root` | 是 | — | 本地 Git 仓库的绝对路径，密码库文件 `vault.dat` 存放于此 |
| `remote_url` | 否 | 空 | Git 远程仓库地址，留空则仅本地使用，不同步 |
| `git_client` | 否 | `exec` | Git 底层实现：`exec` 调用本机 `git` 命令；`go-git` 使用纯 Go 实现（不依赖本机 Git）。留空或取值未知时回退为 `exec` |
| `ssh_key_path` | 否 | 空 | **仅 `go-git` 模式生效**。指定 SSH 私钥的绝对路径；留空时自动按 `ssh-agent → ~/.ssh/id_ed25519 → id_ecdsa → id_rsa` 顺序探测 |
| `ssh_key_passphrase` | 否 | 空 | **仅 `go-git` 模式生效**。当 `ssh_key_path` 指向的私钥被口令加密时填写；留空表示私钥未加密 |

> **go-git 模式注意**：
>
> - `Pull` 仅支持 fast-forward / 合并（不支持 `--rebase`）。在我们同步时"工作区已清空再 pull"的前提下足够使用；若你习惯频繁手动改动本地仓库文件，推荐继续保持默认的 `exec` 后端。
> - **SSH 认证链**与系统 `git` 不同：go-git 不读 `~/.ssh/config`、不自动调用 ssh-agent。认证优先级为 `ssh_key_path(+passphrase)` → `ssh-agent` → `~/.ssh/id_ed25519 / id_ecdsa / id_rsa`（最后一档仅支持**未加密**私钥）。
> - `HostKeyCallback` 优先读 `~/.ssh/known_hosts`；读不到时回退"忽略主机指纹"（与系统 `git` 的 `StrictHostKeyChecking=ask` 体验一致），避免首次跑 go-git 就报 EOF。
>
> **macOS 用户常见坑** — "`ssh: handshake failed: EOF`"：
>
> - Finder / Launchpad 启动的 `.app` 拿不到 `SSH_AUTH_SOCK`，走不了 `ssh-agent`
> - `~/.ssh/id_rsa` 常被加 passphrase 托管到 Keychain，未加密分支自动跳过 → 最终用 nil 认证方式去连 GitHub，服务端直接关连接
> - 解决方案任选其一：
>   1. **为应用单独做一把未加密 SSH key**（推荐）：`ssh-keygen -t ed25519 -f ~/.ssh/id_pwdmgr -N ""`，把公钥加到 GitHub，然后在 `pwdmgr.config.json` 里 `"ssh_key_path": "/Users/<你>/.ssh/id_pwdmgr"`
>   2. **沿用已有的加密 key**：填 `ssh_key_path` 指向加密私钥，同时填 `ssh_key_passphrase`（明文，安全模型依赖文件系统可信，pwdmgr.config.json 权限应保持 0600）
>   3. **直接切回系统 git**：把 `"git_client": "exec"`，由系统 `git` + Keychain / ssh-agent 管理凭据
> - 若应用仍报 `ssh_key_path 指向的文件不存在或不可读` / `私钥被口令加密，请设置 ssh_key_passphrase`，按错误提示修正 `pwdmgr.config.json` 后在设置页点 **重新加载配置** 或重启即可。
>
> **特别注意：代理工具（FlClash / ClashX / V2Ray 等）导致的 EOF**：如果你在 `~/.zshrc` / `~/.bashrc` 里 `export` 了 `ALL_PROXY` / `HTTPS_PROXY` / `HTTP_PROXY`，系统 `git`/`ssh` **不读这些变量**，但 go-git 内部通过 `golang.org/x/net/proxy.FromEnvironment()` 会自动走代理，多数代理对裸 SSH 协议处理不好，直接表现为 `ssh: handshake failed: EOF`。从 v-next 起，pwdmgr 在 go-git 模式执行每次远程操作前会**临时 unset** 这几个变量（操作结束恢复），所以即使 shell 里长期开着代理，`go-git` 也能直连 `github.com:22`。如果你在更早版本碰到此问题，可以在启动前 `unset ALL_PROXY all_proxy HTTPS_PROXY https_proxy HTTP_PROXY http_proxy` 手工绕过，或关闭代理工具的系统代理。

> **在界面修改配置**：打开应用 → **设置** 页 → **编辑配置**，可直接改写上述三项，点击 **保存** 即可：
>
> - 前端调用 `UpdateAppConfig` 把值写回当前生效的 `pwdmgr.config.json`（原子替换，保留其他未知字段）
> - 后端立即重新加载配置并按 `git_client` 热切换 Git 后端，无需重启应用
> - 保存前会自动锁定保险库，避免正在修改路径/后端期间误操作
> - 校验失败（例如 `repo_root` 为空或不是绝对路径）会停留在编辑态并提示具体原因

#### 配置文件位置

应用按以下优先级自动搜索配置文件，找到第一个即使用：

| 优先级 | 位置 | 说明 |
|:------:|------|------|
| 1 | 环境变量 `PWDMGR_CONFIG` 指定的路径 | 适用于开发调试 / 多环境切换 |
| 2 | 可执行文件同级目录 | 便携式部署：将配置与 exe / 二进制放在一起 |
| 3 | 当前工作目录 | `wails dev` 开发时自动命中项目根目录 |
| 4 | **用户配置目录** | 推荐的标准安装位置（见下表） |

> **首次新建的默认位置**：当以上候选全都不存在时，应用会把 **用户配置目录** 作为首次 `Save()` 的落点（通过设置页保存配置），以保证所有平台都能稳定写入 —— 比如 macOS 的 `.app` 内部和 `/` 通常不可写，而用户配置目录永远可写。

各平台的用户配置目录：

| 平台 | 路径 |
|------|------|
| macOS | `~/Library/Application Support/kPass/pwdmgr.config.json` |
| Windows | `%AppData%\kPass\pwdmgr.config.json` |
| Linux | `~/.config/kPass/pwdmgr.config.json` |

> **macOS 用户注意**：从 Finder / Launchpad 启动的 `.app` 应用，工作目录为 `/` 且不继承 shell 环境变量。请将配置文件放到上述用户配置目录，或使用应用内 **设置 → 编辑配置** 保存一次即可自动落到该目录。

#### 开发时指定配置

开发模式下可通过环境变量覆盖：

```bash
PWDMGR_CONFIG=/path/to/my-config.json wails dev
```

## 使用流程

### 首次使用（第一台设备）

1. 填写 `pwdmgr.config.json` 中的 `repo_root` 和 `remote_url`（或打开应用后进入 **设置** 页直接编辑并保存）
2. 打开应用 → **同步** 页 → 点击 **创建本地库**，设置主密码
3. 切换到 **保险库** 页 → 解锁 → 添加密码条目
4. 回到 **同步** 页 → 点击 **绑定远程并同步**，将密码库推送到远程仓库

### 第二台设备同步

1. 填写相同的 `remote_url`，`repo_root` 指向本机一个空目录
2. 打开应用 → **同步** 页 → 点击 **绑定远程并同步**
3. 切换到 **保险库** 页 → 用相同主密码解锁 → 看到所有条目

### 日常同步

点击 **同步** 页的 **同步远程仓库** 按钮，应用会自动执行：

1. 读取本地密码库到内存
2. 清理 Git 工作区 → 拉取远程最新变更（`exec` 后端走 `git pull --rebase`，`go-git` 后端走 `fetch` + fast-forward）
3. 将本地版本与远程版本按条目/空间 ID + `updated_at` 合并，软删除墓碑一并传播
4. 保存合并结果 → 推送到远程

## 项目结构

```
pwdmgr/
├── main.go                    # 入口，Wails 启动配置
├── pwdmgr.config.json         # 运行时配置（不入库）
├── wails.json                 # Wails 项目配置
│
├── internal/
│   ├── app/app.go             # Wails 绑定层，暴露方法给前端（含 UpdateAppConfig）
│   ├── config/config.go       # 配置加载/解析 + Save（原子写回，保留未知字段）
│   ├── crypto/crypto.go       # Argon2id + AES-GCM 加解密
│   ├── git/
│   │   ├── backend.go         # Backend 接口 + SetBackend / Normalize 分发
│   │   ├── exec_backend.go    # 基于系统 `git` 命令的 exec 后端
│   │   └── gogit_backend.go   # 基于 go-git v5 的纯 Go 后端
│   ├── service/
│   │   ├── bind.go            # BindRemoteRepo — 绑定远程仓库
│   │   ├── sync.go            # SyncVault — 日常同步
│   │   ├── init.go            # InitLocalVault — 创建本地库
│   │   ├── entries.go         # 条目 CRUD + 批量移动（MoveEntries）
│   │   ├── spaces.go          # 空间 CRUD（默认空间受保护、同名检测、软删除）
│   │   ├── repo_status.go     # 仓库状态查询结构
│   │   └── status.go          # GetRepoStatus 实现
│   ├── storage/
│   │   ├── storage.go         # JSON 序列化/反序列化
│   │   └── vault_file.go      # LoadVault / SaveVault
│   └── vault/
│       ├── model.go           # Entry / Space / Vault 数据结构
│       ├── merge.go           # 应用层合并（条目/空间，软删除传播）
│       ├── init.go            # NewVault / NewEntry / EnsureDefaultSpace
│       ├── service.go         # 条目 / 空间的增删改 + MoveEntries
│       ├── helper.go          # UUID 生成、时间戳
│       └── utils.go           # 标签规范化等工具
│
├── frontend/
│   ├── src/
│   │   ├── App.vue            # 主布局（顶栏、Tab 切换、弹窗管理、provide askSpace）
│   │   ├── main.js            # Vue 入口
│   │   ├── style.css          # 全局样式
│   │   └── components/
│   │       ├── VaultTab.vue       # 保险库页（解锁/空间切换/搜索/批量选择/移动）
│   │       ├── EntryCard.vue      # 单条密码卡片（含选择 + 移动按钮）
│   │       ├── EntryFormPanel.vue # 新增 / 编辑条目表单
│   │       ├── SpacePickerDialog.vue # 选择目标空间的通用弹窗
│   │       ├── SyncTab.vue        # 同步页（仓库初始化/状态/操作）
│   │       ├── SettingsTab.vue    # 设置页（只读/编辑双模式，写回 pwdmgr.config.json）
│   │       ├── PasswordDialog.vue # 密码输入弹窗
│   │       ├── ConfirmDialog.vue  # 确认弹窗
│   │       └── ToastNotification.vue # 通知提示
│   └── wailsjs/               # Wails 自动生成的 JS 绑定
│
├── build/                     # 平台打包资源
├── TEST_CASES.md              # 测试用例文档
└── LICENSE                    # Apache 2.0
```

## 安全模型

### 加密流程

```
主密码 ──→ Argon2id(salt=16B, time=1, mem=64MB, threads=4)
         ──→ 256-bit key
         ──→ AES-256-GCM(nonce=12B) 加密 vault JSON
         ──→ [salt | nonce | ciphertext] 写入 vault.dat
```

### 存储格式

`vault.dat` 二进制布局：

| 偏移 | 长度 | 内容 |
|------|------|------|
| 0 | 16 字节 | Argon2id salt |
| 16 | 12 字节 | AES-GCM nonce |
| 28 | 剩余 | 密文 + GCM tag |

解密后的明文是 JSON：

```json
{
  "version": 1,
  "spaces": [
    { "id": "default", "name": "默认空间", "updated_at": 1713000000 },
    { "id": "uuid-1",  "name": "工作",     "updated_at": 1713000100 }
  ],
  "entries": [
    {
      "id": "uuid",
      "space_id": "default",
      "name": "GitHub",
      "username": "user@example.com",
      "password": "secret",
      "note": "",
      "tags": ["工作", "开发"],
      "updated_at": 1713000000,
      "deleted_at": 0
    }
  ]
}
```

> 旧版 `vault.dat`（无 `spaces` 字段、条目无 `space_id`）在加载时会由 `EnsureDefaultSpace` 自动迁移到默认空间，无需手工处理。

### 合并策略

多设备冲突时按条目/空间粒度合并：

- **不同 ID** → 两条都保留
- **相同 ID** → 取 `updated_at` 更大的版本胜出
- **软删除（`deleted_at > 0`）** → 与普通更新一样走 `updated_at` 比较。删除胜出时条目会保留墓碑；若远程有更晚的更新则"复活"
- **空间移动** → `MoveEntries` 同时刷新条目的 `updated_at`，保证移动操作在同步中胜过旧版"留在原空间"的副本

### 安全须知

- 主密码是唯一的加密凭据，**丢失无法恢复**
- 解锁后密码明文保存在进程内存中，锁定后清除
- `vault.dat` 推送到 Git 远程时是密文，远程仓库管理员无法读取内容
- 建议使用 SSH key 或 token 认证 Git 远程仓库，避免在配置中存储明文凭据

## 仓库状态机

应用通过检测以下状态决定可用操作：

| IsGitRepo | HasRemote | RemoteHasData | HasLocalVault | 状态说明 | 可用操作 |
|:---------:|:---------:|:-------------:|:------------:|---------|---------|
| - | - | - | - | 纯空目录 | 创建本地库 |
| Y | - | - | - | 已 init 未绑定 | 创建本地库 / 绑定远程 |
| Y | Y | - | Y | 本地有库，远程空 | 绑定远程并同步 |
| Y | Y | Y | - | 远程有数据，本地无 | 绑定远程并同步 |
| Y | Y | Y | Y | 正常同步状态 | 同步 / Pull / Push |

## 开发

```bash
# 启动开发服务器（前端热更新）
wails dev

# 仅构建前端
cd frontend && npm run build

# 生产构建
wails build
```

## 许可证

[Apache License 2.0](LICENSE)
