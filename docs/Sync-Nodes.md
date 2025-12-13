# Sync Node 设计方案

## 背景

当前的 GoHook 只负责本机的代码拉取与脚本执行。为了让同一个仓库或项目可以在多台机器上水平扩容，常见做法是为每个节点单独部署一个 GoHook，并在 Git 仓库中配置多条 webhook，这会带来配置重复、状态不可见与版本漂移的问题。为了解决这些痛点，需要为 GoHook 引入“主节点 + 子节点”模式：主节点统一接收 webhook、执行拉取与构建，并驱动其他节点同步相同的代码/构建产物。

## 目标

- 提供节点管理能力：子节点注册、分组、健康检查、凭证与元信息维护。
- 在项目（或 Hook）级别声明需要同步的目标节点，并配置目标路径、策略、并发和重试行为。
- 在主节点完成代码更新后触发同步工作流，确保子节点的目录状态与主节点保持一致。
- 给 Web UI/REST API 增加可观测性：节点状态、同步任务队列、单节点的执行日志。
- 同步机制固定为 Sync Agent：基于 TCP 长连接 + mTLS + 块级传输（不回退到 HTTP 轮询或 `ssh+rsync`）。

## 非目标

- 不在第一阶段实现跨主节点集群（多主）或任务自动抢占。
- 不负责变更控制/灰度发布流程，只解决代码同步。
- 不强制约束仓库结构，默认同步到指定路径，由上层脚本负责重启服务等操作。

## 系统角色

| 角色 | 描述 |
| --- | --- |
| 主节点（Primary GoHook） | 当前运行 GoHook Web UI/控制面的实例，负责节点注册、Webhook 执行、同步调度与状态持久化。 |
| 子节点（Sync Node） | 需要保持代码一致的服务器，运行同步客户端（Sync Agent）。 |
| Sync Agent | 可选组件。部署在子节点，用于和主节点进行双向认证、接收同步任务、回传状态。 |

## 架构概览

1. 主节点保存节点清单（`sync_nodes` 表/配置），维持健康状态。
2. 项目配置里声明要同步的节点（`project_nodes` 表）。
3. 当 Webhook/GitHook 完成拉取或构建后，主节点把成功事件送入 Sync Controller。
4. Sync Controller 根据策略生成 `sync_tasks`，放入任务队列。
5. 主节点通过 TCP 长连接推送任务，并按需提供索引/块数据；Agent 仅拉取缺失块并落地到目标目录。
6. 节点/任务状态写回数据库，并通过 WebSocket 通知 UI。

## 当前实现进度（截至 2025-12）

已完成：

1. **节点管理（主节点）**
   - 数据模型：`sync_nodes`/`sync_tasks`/`sync_file_changes` 已加入并自动迁移。
   - API：节点 CRUD、Token 刷新（`POST /api/sync/nodes/:id/rotate-token`）、配对重置（`POST /api/sync/nodes/:id/reset-pairing`）。
   - 鉴权：Agent 通过 TCP `hello(token)` 鉴权并维持在线状态；管理接口使用管理员 JWT。
   - UI：节点管理页、创建/编辑弹窗展示 Token、复制/显隐/刷新。

2. **项目级同步配置（版本管理）**
   - 类型定义：`ProjectSyncConfig`/`ProjectSyncNodeConfig` 已支持项目级 ignore 与 `ignore_permissions`。
   - API：项目列表返回 `sync` 配置；编辑项目支持保存 `sync`。
   - UI：版本管理“编辑项目”中新增“同步”区域：开启同步、选择节点、目标目录、忽略规则与忽略权限开关。

3. **变更监听与落库（主节点）**
   - 基于 `fsnotify` 的目录监听与递归 watch。
   - 变更写入 `sync_file_changes`，含 path/hash/mtime/size/type。
   - watcher 仅在项目 `sync.enabled=true` 时启动。

4. **长连接 + mTLS（主节点/Agent）**
   - 主节点提供 TCP/TLS 监听（默认 `:9001`），Agent 断线自动重连。
   - 节点 token 用于应用层鉴权；mTLS 证书指纹用于节点配对（TOFU + 可选 pin）。

5. **块级同步（主节点/Agent）**
   - 自适应固定块（128KiB 起倍增，最大 4MiB），SHA-256 块哈希。
   - 索引下发 + 按需拉块 + 二进制帧传输 + 原子落盘。

未完成（核心缺口）：

1. **Sync Controller**
   - 未把 GitHook/Watcher 事件转为 `sync_tasks`。
   - 未实现按项目并发/重试策略。

2. **任务可观测性**
   - 缺少任务列表/详情 API 与 UI（目前只能通过数据库/日志观测、以及手动触发接口验证链路）。

3. **自动安装真实流程**
   - 安装流程仍为 stub（记录日志并标记成功）。

## 本次实现落地记录

本次对话中已落地的关键能力（按时间线汇总）：

1. **节点管理与安全**
   - 节点 Token 自动生成、显示/复制/刷新。
   - Agent 通过 TCP `hello(token)` 完成认证并维持在线状态（不再使用 HTTP 心跳）。
   - 新增 mTLS 长连接与证书指纹配对（`agent_cert_fingerprint`），支持 TOFU + pin 校验。

2. **项目级同步配置**
   - 忽略规则与权限忽略从“节点级”迁移到“项目 sync 配置”。
   - 版本管理 UI 支持：开启同步、选择节点与目标目录、ignore/ignore_permissions 配置。
   - 后端项目编辑 API 支持保存/返回 `sync`。

3. **主节点监听与任务基础设施**
   - 基于 `fsnotify` 的项目目录监听，变更落库到 `sync_file_changes`。
   - 任务模型与手动触发接口：`POST /api/sync/projects/:name/run` 生成 pending 任务（后续由 Controller 替换）。

4. **Agent 同步闭环**
   - Agent 增加任务执行逻辑（早期 tar.gz 原型已弃用，现已升级为块级同步）。
   - 任务状态回传并写入 `sync_tasks`。

5. **块级同步（Syncthing 同款自适应固定块）**
   - 自适应块大小（128KiB 起步倍增，最大 4MiB，单文件 ≤ ~256 块）。
   - 主节点通过长连接下发索引与块数据；Agent 仅请求缺失/变化块。
   - 块数据使用二进制帧传输。
   - Agent 强制走 TCP 块传输，不再回退 HTTP 轮询。

6. **工程与测试**
   - 优化测试：构建一次二进制复用，修复 hooks 初始重复 ID 校验与热重载边界。

## 数据模型（建议）

### sync_nodes

| 字段 | 说明 |
| --- | --- |
| id | 节点 ID，UI/配置引用此值 |
| name/address/type | 节点标识与用途（`type`：`ssh|agent|custom`；当前同步链路仅使用 `agent`） |
| status/health/last_seen | 健康与在线信息（由 TCP 长连接维护 last_seen/status） |
| credential_ref/credential_value | 凭证引用与存储值（Agent Token 存在 `credential_value`） |
| agent_cert_fingerprint | Agent 证书指纹（sha256 hex），用于 mTLS 配对与防冒充 |
| install_status/install_log | 自动安装状态（目前为 stub） |

### project_nodes

| 字段 | 说明 |
| --- | --- |
| project_id | 关联项目/Hook |
| node_id | 关联节点 |
| target_path | 子节点上的目标目录 |
| strategy | `mirror` / `overlay` / 自定义策略 |
| include/exclude | 可选的路径过滤器 |

### sync_tasks

| 字段 | 说明 |
| --- | --- |
| id | 任务 ID |
| project_id / hook_id | 触发源 |
| node_id | 目标节点 |
| driver | `agent`（当前强制 Agent 块传输；`rsync` 字段仅为历史保留，不作为回退方案） |
| status | `PENDING/RUNNING/SUCCESS/FAILED/RETRYING` |
| attempt | 重试次数 |
| payload | 任务入参（JSON：项目名/目标路径/忽略配置等） |
| logs | 执行日志摘要 |

## 核心模块

### 节点管理器（Node Manager）

- REST API：`GET/POST/GET/PUT/DELETE /api/sync/nodes`，并支持 Token 轮换 `POST /api/sync/nodes/:id/rotate-token`。
- 在线状态：由 TCP 长连接连接/断开与定时 touch 维护（不再提供 HTTP 心跳接口）。
- 凭证存储：Agent Token 以节点维度保存（`credential_value`），避免在配置文件中明文长期存放。
- UI：新增“节点管理”页，显示状态、项目绑定数、最近同步结果，支持一键测试连通性。
- 自动安装：当前为 stub（记录日志并标记成功），后续再补齐实际下发与回滚。

### 同步控制器（Sync Controller）

- 监听 webhook 执行事件（可通过现有日志/事件总线），仅当任务成功且配置 `sync.enabled=true` 时入队。
- 支持按项目自定义并发上限（例如 `max_parallel_nodes`）与串行策略。
- 统一的任务重试策略：指数退避 + 最大尝试次数，失败后告警。
- 任务状态实时写入数据库，并通过 WebSocket 推送给 UI。

### 执行器（Sync Executor）

当前执行链路固定为 **Sync Agent（TCP/mTLS + 块级传输）**：

- 主节点通过 TCP 长连接推送 task，并流式下发索引；Agent 仅请求缺失/变化块。
- 块数据使用二进制帧传输，Agent 原子落盘并回传任务状态。

### 项目配置扩展

在现有项目/Hook 配置中新增 `sync` 段。例如（YAML）：

```yaml
- id: project-a
  name: "Project A"
  repo: "git@github.com:org/project-a.git"
 sync:
    enabled: true
    driver: "agent"          # 当前仅支持 agent（TCP/mTLS + 块级传输）
    max_parallel_nodes: 2
    ignore_defaults: true    # 默认忽略 .git/runtime/tmp
    ignore_patterns:
      - "node_modules/**"
      - "*.log"
    ignore_file: "sync.ignore"        # 可选，额外忽略文件
    ignore_permissions: true          # 忽略 chmod/chown 等权限变更
    nodes:
      - node_id: "1"
        target_path: "/srv/project-a"
        strategy: "mirror"
      - node_id: "2"
        target_path: "/srv/project-a"
        include:
          - "dist/**"
        exclude:
          - ".git/**"
```

对于使用数据库维护项目的环境，可在项目表中保存同样的 JSON 字段。

## 同步流程

1. **Webhook 触发**：Git push 触发 GoHook，对应项目成功执行 `git pull` / 构建脚本。
2. **事件入队**：执行器通过事件总线通知 Sync Controller，附带项目 ID、commit、工作目录等信息。
3. **生成任务**：Controller 查询项目配置，展开节点列表，写入 `sync_tasks`。
4. **任务调度**：根据项目/全局的并发限制，将任务分派给执行器。
5. **执行同步**：
   - **agent**：主节点下发索引（自适应固定块 + SHA-256），Agent 仅拉取缺失/变化块并重组文件。
6. **结果回写**：任务状态落库，失败则记录错误、增加重试计数。
7. **通知**：UI/WebSocket/Gotify 通知任务结果，可在项目页面查看节点同步状态。

## 异常与回滚

- **网络不可达**：任务标记为 `FAILED`，触发告警，可配置自动降级（跳过该节点）或阻塞后续部署。
- **校验失败**：Agent 重新下载并校验；超过阈值后要求人工介入。
- **长时间未成功**：将项目标记为 `SYNC_DEGRADED`，在 UI 上提示。
- **手动回滚**：支持 UI/API 选择节点并回放历史版本（可在 Sync Task 中保留产物引用）。

## 安全设计

- 强制 HTTPS/TLS，对外 API 使用 JWT + 节点专用 token。
- Agent 与主节点使用双向 TLS（mTLS）+ 节点 token（应用层），并通过证书指纹进行配对（TOFU + 可选 pin）。
- token 轮换使用管理端接口 `POST /api/sync/nodes/:id/rotate-token`；轮换后旧 token 立即失效。

## Web UI/REST 变更

- 复用左侧原有的 “All Projects” 侧边栏空白区域展示“节点管理”入口：点击后列表区域显示节点清单、连接状态（由 TCP 长连接维护）和操作按钮，下方切换到节点详情/最近同步任务等子页，右侧主面板仍用于项目内容。
- 节点管理仅维护节点连通性与认证（SSH/Agent token），不再配置忽略规则。
- 项目编辑表单里添加“同步”区域：开启同步、选择节点、设置目标路径/策略，并配置项目级忽略文件/目录与是否忽略权限变更。
- 新增“同步管理（Sync Projects）”菜单与页面：
  - 列表展示所有 `sync.enabled=true` 的项目（等同 Syncthing 的 Folder 概念）。
  - 展示字段：项目名称、主节点目录（与版本管理展示方式一致）、同步状态、最后同步时间、已绑定节点数。
  - 进入详情可管理：绑定节点及每节点 `target_path` 覆盖、项目级 ignore/ignore_permissions。
- 新增“同步任务”列表页或面板，支持按项目/节点过滤并查看日志与重试。
- API 文档需要新增节点、任务相关的端点说明。

## 同步项目（Folder）管理设计

本节描述“同步管理”能力的目标与落地步骤。GoHook 的同步是**单向**：主节点仅发送、子节点仅接收，不提供方向切换。

### 数据来源与模型

- 同步项目不新增独立表：直接由“版本管理项目”中 `sync.enabled=true` 的项目生成。
- 每个同步项目对应一个 `ProjectSyncConfig`：
  - 项目级 ignore：`ignoreDefaults / ignorePatterns / ignoreFile / ignorePermissions`
  - 绑定节点与覆盖目录：`nodes[{nodeId,targetPath}]`
- 状态/最后同步时间从 `sync_tasks` 汇总：
  - `lastSyncAt`：该项目所有节点最近一次 `success` 的任务时间（取 max）。
  - `status`：按最近任务与节点健康聚合（如有任一节点 `failed` 或 OFFLINE → `DEGRADED`）。

### Syncthing 逻辑对齐与优化建议

- **忽略规则语义**：参考 Syncthing 的 `.stignore` 语法（glob、`!` 反选、`#` 注释）。
  - GoHook 中 ignore 由主节点权威生成索引；Agent 端仅按索引落盘，因此天然保持单向一致性。
  - 当前已支持的语法子集：`#` 注释、`!` 反选、`**` 跨目录匹配、无 `/` 的规则默认匹配任意目录层级。
- **默认目标目录**：新增节点时默认 `target_path = project.path`（主/子节点路径一致的常见场景）；允许每节点手动覆盖为不同目录。
- **只接收模式**：不提供“send/receive only”切换；可在 UI 上显式标注“单向同步”避免误解。

### 实施步骤（建议分两档）

#### Step A（MVP：列表 + 跳转编辑）

1. **后端汇总 API**
   - 新增 `GET /api/sync/projects`（JWT）。
   - 返回：`[{projectName,path,sync,status,lastSyncAt,nodes:[{nodeId,nodeName,targetPath,lastStatus,lastSyncAt}]}]`
   - `sync` 字段直接复用项目的 `ProjectSyncConfig`。

2. **UI：同步管理页面**
   - 新增菜单入口 `nav.syncProjects`，路由 `/sync/projects`。
   - 列表展示同步项目（过滤 `sync.enabled=true`），字段与版本管理一致。
   - 操作：
     - “查看/管理同步配置” → 直接打开现有“版本管理-编辑项目”弹窗并定位到“同步”区域。
     - “立即同步” → 调用临时接口 `POST /api/sync/projects/:name/run`。

> Step A 不需要新增编辑 API，复用版本管理保存逻辑。

#### Step B（增强：独立管理对话框）

1. **UI 抽离 SyncConfigDialog**
   - 将 `EditProjectDialog` 中的 Sync 区域抽成 `SyncConfigDialog`（仅展示同步相关字段）。
   - 同步管理页使用该对话框进行编辑；版本管理页也复用同一组件。

2. **专用保存端点（已实现）**
   - `PUT /api/sync/projects/:name/config`：仅更新项目的 `sync` 段（避免覆盖项目其他字段）。

## 实施步骤（当前可用）

1. **主节点：创建 Sync Node**
   - UI：节点管理 → 新建节点（`type=agent`）。
   - 保存后在弹窗中复制 token（后续可刷新）。

2. **主节点：开启 TCP/mTLS**
   - 默认监听 `SYNC_TCP_ADDR=":9001"`，证书目录 `SYNC_TLS_DIR="./sync_tls"`（首次启动自动生成）。

3. **子节点：启动 Agent**
   - 推荐（最简）：仅提供 `server + token`，Agent 会自动 enroll 获取 `nodeId` 并持久化到 `data-dir`。
     - CLI：`./nodeclient --server 10.0.0.10:9001 --token <TOKEN>`
   - 默认持久化目录：`~/.gohook-agent`（可选：`--data-dir /var/lib/gohook-agent`）
   - 可选：`--server-fingerprint <sha256-hex>`（严格校验服务端证书指纹；否则使用 TOFU）
   - 兼容：仍支持传统环境变量 `SYNC_TCP_ENDPOINT` / `SYNC_NODE_TOKEN` / `SYNC_NODE_ID` 等

4. **主节点：项目开启同步**
   - 版本管理 → 项目行“同步配置”按钮：启用、选择节点、设置 `target_path`、配置 ignore 与 `ignore_permissions`。

5. **验证链路**
   - 手动触发：`POST /api/sync/projects/:name/run`（临时入口，后续由 Controller 替换）。
   - 观察：节点状态（TCP 在线/last_seen）、任务状态（数据库/日志），以及子节点目标目录文件变更。

> 说明：Syncthing 的实现采用非 MIT 许可，GoHook 这里将“参考算法与接口设计”自行实现，不直接拷贝其源码到仓库中，避免许可证污染。

## 长连接与 mTLS（已实施）

GoHook 与 Agent 之间新增 TCP/TLS 长连接，用于任务即时推送与后续块级数据面传输。

### 主节点

1. 启动后自动监听 TCP（默认 `:9001`），可通过环境变量修改：
   - `SYNC_TCP_ADDR=":9001"`
2. 首次启动会在 `SYNC_TLS_DIR` 目录生成自签服务端证书：
   - `SYNC_TLS_DIR="./sync_tls"`
   - 生成 `server.crt` / `server.key`
3. 连接建立后，Agent 先发送 `hello(nodeId, token, agentVersion)`；主节点验证 token，
   - 如果该节点 `agent_cert_fingerprint` 为空，则写入本次连接的证书指纹（配对/TOFU）。
   - 否则必须匹配已登记指纹。

### Agent

1. 推荐使用 CLI 参数启动（最少参数）：
   - `./nodeclient --server 10.0.0.10:9001 --token <TOKEN>`
   - 首次连接会发送 `enroll(token)` 获取 `nodeId`，并保存到 `data-dir/state.json`，后续重启无需再手动提供 `nodeId`。
2. 持久化目录（默认 `~/.gohook-agent`）：
   - `state.json`：保存 `nodeId/token/server`
   - `tls/`：保存 `client.crt` / `client.key` 与 `server.fp`（TOFU）
3. 使用 `.env`（可选）：
   - 可通过 `--env-file /path/to/.env` 指定；也会自动尝试加载 `./.env` 与 `<data-dir>/.env`
   - 推荐变量：`GOHOOK_SERVER` / `GOHOOK_TOKEN` / `GOHOOK_DATA_DIR`
   - 兼容变量：`SYNC_TCP_ENDPOINT` / `SYNC_NODE_TOKEN` / `SYNC_AGENT_TLS_DIR` / `SYNC_SERVER_FINGERPRINT`
4. 服务端指纹校验（可选）：
   - `--server-fingerprint <sha256-hex>` 或设置 `GOHOOK_SERVER_FINGERPRINT`
   - 若未设置，Agent 会在首次连接时信任并保存到 `tls/server.fp`（TOFU），后续必须匹配。

连接建立后，任务通过长连接即时下发；**Agent 不再回退到 HTTP 轮询**，必须配置 `SYNC_TCP_ENDPOINT` 才能执行同步任务。
当连接中断时，Agent 会自动按指数退避重连（最大 30s 间隔）。

## 状态与错误诊断（新增）

当同步失败（例如目标目录无写入权限）时：

- **节点管理**：
  - “同步状态 = FAILED” 支持悬浮查看最近一次失败的 `project/targetPath/lastError/errorCode`；
  - 可打开“任务详情”查看该节点最近任务列表与日志。
- **同步管理**：
  - 项目状态为 `DEGRADED` 时，可悬浮查看失败节点的错误摘要；
  - 可打开“任务详情”查看该项目最近任务列表与日志。

### 常见 errorCode

- `EACCES/EPERM/EROFS`：目标目录无写入权限或只读文件系统（修复权限或更换 `targetPath`）
- `ENOENT`：目标路径不存在或无权创建
- `ENOSPC`：磁盘空间不足
- `INVALID_TARGET`：`targetPath` 配置不合法（不能为空或 `/`）
- `PROTO`：连接/协议异常（检查主节点与 Agent 版本一致性）
- `TIMEOUT`：任务超时/网络读写超时（默认 30 分钟，可通过 `SYNC_TASK_TIMEOUT` 调整）
- `BLOCK_HASH_MISMATCH`：块数据校验失败（强一致性校验；通常意味着传输异常或对端读块错误）
- `BLOCK_READ`：主节点读取源文件块失败（路径不存在/权限/越界等）

### 传输优化（新增）

- 默认模式：Agent 先完整接收索引（`index_end`）再发起块请求，避免索引与块响应交错导致协议错误。
- 可选模式（Chunked Index）：主节点按批发送 `index_chunk`，Agent 每批同步完成后回 `index_chunk_done`；可降低双方峰值内存并更早开始落盘。
- 块传输支持批量请求（`block_batch_request`），减少消息往返与 CPU 解析开销；Agent 会校验 `sha256` 哈希后再写入目标文件。
- 任务会记录传输统计：`files/blocks/bytes/durationMs`，可在“任务详情”中查看。

### 高性能开关（可选）

以下为“默认关闭/不改变语义”的可选加速点；建议按需灰度开启：

- `SYNC_TASK_MAX_ATTEMPTS`：可重试错误（如 `TIMEOUT/PROTO/DISCONNECTED`）的最大尝试次数（默认 3）。
- `SYNC_HASHCACHE_ENTRIES`：主节点块哈希缓存条数上限（默认 2048），减少重复 hash（大项目多节点更明显）。
- `SYNC_DELTA_INDEX_OVERLAY=1`：Overlay 策略启用增量索引（从 `sync_file_changes` 拉取未处理变更，仅下发变更文件的索引）。
  - `SYNC_DELTA_MAX_FILES`：单次增量索引最大文件数（默认 5000；超过则自动回退全量 Walk 以保证正确性）。
- `SYNC_MIRROR_FAST_DELETE=1`：Mirror 删除启用 manifest 快速路径（不全量 Walk）。
  - 注意：该模式不会删除“目标端用户新创建、且不在 manifest 中”的额外文件；如需周期性恢复完整 mirror 语义，设置：
  - `SYNC_MIRROR_FAST_FULLSCAN_EVERY=N`：每 N 次强制执行一次严格全量扫描删除。
  - `SYNC_MIRROR_CLEAN_EMPTY_DIRS=1`：删除文件后尝试清理空父目录（不会越过 `targetRoot`）。
- `SYNC_INDEX_CHUNKED=1`（Agent 侧）：启用 Chunked Index 协议（Agent 会在 hello 中声明 `index_chunk_v1`）。
  - `SYNC_INDEX_CHUNK_SIZE`（主节点侧）：Chunk 大小（默认 128）；服务端会在帧过大时自动拆分。

## 块级同步（自适应固定块，已接入长连接）

GoHook 参考 Syncthing 的“自适应固定块 + SHA-256”策略：

- 最小块：128KiB
- 最大块：4MiB
- 通过倍增块大小使单文件块数不超过 ~256

### TCP 消息流（简化版）

在 `hello_ack` 之后：

1. 主节点推送任务：`task`
2. Agent 开始同步：`sync_start`
3. 主节点下发索引：
   - `index_begin`
   - **默认模式**：多条 `index_file` → `index_end`
   - **Chunked Index**：多条 `index_chunk`（files[] 批量）↔ `index_chunk_done`（逐批确认）→ `index_end`
4. Agent 按需拉取缺块：
   - `block_request`（path + block index）
   - `block_response_bin`（JSON 头 + 二进制块帧）
5. Agent 完成后回传：`task_report`

`block_response_bin` 的数据体采用**二进制帧**传输：
- 先发送 JSON 帧：`block_response_bin`（包含 `size` / `hash`）
- 再发送一个 raw bytes 帧（长度前缀），内容为该块的原始字节

## 后续任务（Roadmap vNext）

按优先级排序，建议分 3 个迭代完成：

### Iteration 1：Controller + 执行链路闭环

1. **Sync Controller**
   - 监听 GitHook 成功事件与/或 `sync_file_changes`。
   - 生成 `sync_tasks`（替换临时 `projects/:name/run`）。
   - 支持 `max_parallel_nodes` 并发控制与失败重试（指数退避、最大次数）。

2. **任务管理 API + UI**
   - `GET /api/sync/tasks`/`GET /api/sync/tasks/:id`。
   - UI 增加任务列表页：状态、节点、日志、失败原因、重试按钮。

3. **清理遗留回退接口（不影响现有链路）**
   - 将 `GET /api/sync/nodes/:id/tasks/pull` 与 `GET /api/sync/nodes/:id/tasks/:taskId/bundle` 标记为 deprecated，并在实现稳定后移除。

### Iteration 2：块传输性能与可靠性

1. **并发拉块**
   - Agent 并行请求多个块（带窗口/限速），提高吞吐。
2. **块缓存与去重**
   - Agent 侧 LRU 缓存近期块。
   - 可选：主节点按 hash 提供跨文件去重（保持协议兼容）。
3. **断点续传**
   - Agent 在 task 失败/重连后继续从缺块列表恢复。

### Iteration 3：文件语义完善与运维

1. **完整 mirror 语义**
   - 支持目录/空目录、符号链接、删除策略更精确。
2. **权限/时间/所有者**
   - 在 `ignore_permissions=false` 时补齐 owner/group（Linux only）与更严格的 mtime 对齐。
3. **证书轮换与撤销**
   - 管理端提供“重置 agent 证书指纹/重新配对”能力。
4. **安全与可观测**
   - 任务/块级指标（速率、命中率、失败率）+ 告警。
   - 长连接心跳/keepalive 与 idle 超时配置。

## 部署建议

1. **初始化**：在主节点开启项目同步配置，并确保 `SYNC_TCP_ADDR` 可被子节点访问。
2. **节点上线**：通过节点管理 UI 创建节点并复制 token，在子节点配置并启动 Agent；如需更强安全，设置 `SYNC_SERVER_FINGERPRINT` 做 pin。
3. **项目接入**：在 UI/配置中勾选需要同步的节点，设置路径。
4. **灰度试跑**：先对某个项目开启同步，观察任务队列与日志。
5. **全面启用**：结合监控/告警（Prometheus、Grafana 或现有日志系统）观察节点健康。
