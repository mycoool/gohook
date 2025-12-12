# Sync Node 设计方案

## 背景

当前的 GoHook 只负责本机的代码拉取与脚本执行。为了让同一个仓库或项目可以在多台机器上水平扩容，常见做法是为每个节点单独部署一个 GoHook，并在 Git 仓库中配置多条 webhook，这会带来配置重复、状态不可见与版本漂移的问题。为了解决这些痛点，需要为 GoHook 引入“主节点 + 子节点”模式：主节点统一接收 webhook、执行拉取与构建，并驱动其他节点同步相同的代码/构建产物。

## 目标

- 提供节点管理能力：子节点注册、分组、健康检查、凭证与元信息维护。
- 在项目（或 Hook）级别声明需要同步的目标节点，并配置目标路径、策略、并发和重试行为。
- 在主节点完成代码更新后触发同步工作流，确保子节点的目录状态与主节点保持一致。
- 给 Web UI/REST API 增加可观测性：节点状态、同步任务队列、单节点的执行日志。
- 同步机制可插拔，默认提供具备 `rsync` 语义的 Sync Agent，并保留纯 `ssh + rsync` 驱动作为回退方案。

## 非目标

- 不在第一阶段实现跨主节点集群（多主）或任务自动抢占。
- 不负责变更控制/灰度发布流程，只解决代码同步。
- 不强制约束仓库结构，默认同步到指定路径，由上层脚本负责重启服务等操作。

## 系统角色

| 角色 | 描述 |
| --- | --- |
| 主节点（Primary GoHook） | 当前运行 GoHook Web UI/控制面的实例，负责节点注册、Webhook 执行、同步调度与状态持久化。 |
| 子节点（Sync Node） | 需要保持代码一致的服务器，运行同步客户端（或开放 SSH）。 |
| Sync Agent | 可选组件。部署在子节点，用于和主节点进行双向认证、接收同步任务、回传状态。 |

## 架构概览

1. 主节点保存节点清单（`sync_nodes` 表/配置），维持健康状态。
2. 项目配置里声明要同步的节点（`project_nodes` 表）。
3. 当 Webhook/GitHook 完成拉取或构建后，主节点把成功事件送入 Sync Controller。
4. Sync Controller 根据策略生成 `sync_tasks`，放入任务队列。
5. Executor 根据驱动执行同步：默认通过 Sync Agent 下发任务，也可退化为 `ssh + rsync` 直接复制。
6. 节点/任务状态写回数据库，并通过 WebSocket 通知 UI。

## 当前实现进度（截至 2025-12）

已完成：

1. **节点管理（主节点）**
   - 数据模型：`sync_nodes`/`sync_tasks`/`sync_file_changes` 已加入并自动迁移。
   - API：节点 CRUD、安装触发、心跳、Token 刷新（`POST /api/sync/nodes/:id/rotate-token`）。
   - 鉴权：Agent 心跳接口使用节点 Token 认证；管理接口使用管理员 JWT。
   - UI：节点管理页、创建/编辑弹窗展示 Token、复制/显隐/刷新。

2. **项目级同步配置（版本管理）**
   - 类型定义：`ProjectSyncConfig`/`ProjectSyncNodeConfig` 已支持项目级 ignore 与 `ignore_permissions`。
   - API：项目列表返回 `sync` 配置；编辑项目支持保存 `sync`。
   - UI：版本管理“编辑项目”中新增“同步”区域：开启同步、选择节点、目标目录、忽略规则与忽略权限开关。

3. **变更监听与落库（主节点）**
   - 基于 `fsnotify` 的目录监听与递归 watch。
   - 变更写入 `sync_file_changes`，含 path/hash/mtime/size/type。
   - watcher 仅在项目 `sync.enabled=true` 时启动。

未完成（核心缺口）：

1. **Sync Controller**
   - 未把 GitHook/Watcher 事件转为 `sync_tasks`。
   - 未实现按项目并发/重试策略。

2. **Sync Executor**
   - 未实现 `ssh+rsync` 或 agent 驱动的实际同步执行。
   - 未写回任务状态/日志/错误。

3. **Sync Agent 执行侧**
   - 目前只有心跳上报，没有拉取任务、文件/块传输、结果回传。

4. **自动安装真实流程**
   - 安装流程仍为 stub（记录日志并标记成功）。

## 数据模型（建议）

### sync_nodes

| 字段 | 说明 |
| --- | --- |
| id/uuid | 节点 ID，UI/配置引用此值 |
| name | 可读名称 |
| address | SSH 地址或 Agent API 地址 |
| type | `ssh` / `agent` / `custom` |
| tags | JSON/数组，便于按区域/能力过滤 |
| status | `ONLINE/OFFLINE/DEGRADED` |
| last_seen | 最近一次心跳 |
| credential_ref | SSH key、token 等引用 |

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
| driver | `rsync` / `agent` |
| status | `PENDING/RUNNING/SUCCESS/FAILED/RETRYING` |
| attempt | 重试次数 |
| payload | 任务入参（版本号、压缩包路径等） |
| logs | 执行日志摘要 |

## 核心模块

### 节点管理器（Node Manager）

- REST API：`GET/POST/PATCH/DELETE /api/sync/nodes`，支持批量导入、标签过滤。
- 健康检查：`ssh` 类型通过 `ssh -o BatchMode=yes node "echo ok"` 或 `rsync --list-only` 验证；`agent` 类型则由子节点定期 `POST /api/sync/nodes/{id}/heartbeat`。
- 凭证存储：复用 `user.yaml` 或数据库凭证表，提供 `credential_id` 引用，避免在配置里明文写 key。
- UI：新增“节点管理”页，显示状态、项目绑定数、最近同步结果，支持一键测试连通性。
- 自动安装：创建节点时需录入 SSH 信息（用户名、端口、认证方式/密钥），主节点利用该信息无缝推送/更新 Sync Agent、生成配置与注册 token，并在 UI 中回显安装进度和日志。

### 同步控制器（Sync Controller）

- 监听 webhook 执行事件（可通过现有日志/事件总线），仅当任务成功且配置 `sync.enabled=true` 时入队。
- 支持按项目自定义并发上限（例如 `max_parallel_nodes`）与串行策略。
- 统一的任务重试策略：指数退避 + 最大尝试次数，失败后告警。
- 任务状态实时写入数据库，并通过 WebSocket 推送给 UI。

### 执行器（Sync Executor）

两种驱动模式：

1. **SSH + rsync（默认）**
   - 主节点需要能通过 SSH 免密连接子节点。
   - 使用 `rsync -az --delete --exclude-from=... <src> user@node:/target`，提供 include/exclude。
   - 适合已有 SSH 信任、无需额外客户端的场景。

2. **Sync Agent（默认方案）**
	- 子节点运行一个轻量二进制（可复用 GoHook 的 HTTP server，裁剪成 `gohook-sync-agent`），负责拉取任务、执行同步并回传状态。
	- Agent 在启动时向主节点注册，建立长轮询或 WebSocket 通道；若网络受限，可退化为周期性轮询。
	- 主节点通过 `POST /api/sync/tasks/{id}/dispatch` 将任务下发，Agent 下载压缩包/差异包或通过内置传输从主节点拉取最新内容，并定期 `POST /api/sync/nodes/{id}/heartbeat` 上报状态。
   - Agent 内置 `rsync` 同步语义（权限、增量、删除未使用文件），支持项目级 `include/exclude` 配置，默认忽略 `.git/`、`runtime/`、`tmp/` 等目录，并允许声明额外的忽略文件（如 `sync.ignore`）。
   - 方便做校验、钩子、断点续传，以及在受限网络下（只能出方向）运行。

### 项目配置扩展

在现有项目/Hook 配置中新增 `sync` 段。例如（YAML）：

```yaml
- id: project-a
  name: "Project A"
  repo: "git@github.com:org/project-a.git"
  sync:
    enabled: true
    driver: "agent"          # agent / rsync / inherit
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
   - **rsync**：构建 `rsync` 命令，使用 `credential_ref` 对应的 SSH key；执行后记录 stdout/stderr。
   - **agent（默认）**：打包变更（`tar`/`rsync --dry-run` 生成 patch），或直接通过 Agent 的内置增量同步能力抓取最新内容；Agent 执行前会合并项目配置/节点配置中的 `include/exclude` 列表与 `sync.ignore` 文件，确保 `.git/`、`runtime/` 等目录不被同步，同时覆盖权限/删除语义。
6. **结果回写**：任务状态落库，失败则记录错误、增加重试计数。
7. **通知**：UI/WebSocket/Gotify 通知任务结果，可在项目页面查看节点同步状态。

## 异常与回滚

- **网络不可达**：任务标记为 `FAILED`，触发告警，可配置自动降级（跳过该节点）或阻塞后续部署。
- **校验失败**：Agent 重新下载并校验；超过阈值后要求人工介入。
- **长时间未成功**：将项目标记为 `SYNC_DEGRADED`，在 UI 上提示。
- **手动回滚**：支持 UI/API 选择节点并回放历史版本（可在 Sync Task 中保留产物引用）。

## 安全设计

- 强制 HTTPS/TLS，对外 API 使用 JWT + 节点专用 token。
- SSH 凭证单独管理，建议使用机器账户 + 最小权限。
- Agent 与主节点使用双向 TLS 并定期轮换 token。
- 为 `rsync` 命令提供默认的 `--safe-links --perms --chmod` 限制，防止覆盖敏感文件。

## Web UI/REST 变更

- 复用左侧原有的 “All Projects” 侧边栏空白区域展示“节点管理”入口：点击后列表区域显示节点清单、健康状态和操作按钮，下方切换到节点详情/最近同步任务等子页，右侧主面板仍用于项目内容。
- 节点管理仅维护节点连通性与认证（SSH/Agent token），不再配置忽略规则。
- 项目编辑表单里添加“同步”区域：开启同步、选择节点、设置目标路径/策略，并配置项目级忽略文件/目录与是否忽略权限变更。
- 新增“同步任务”列表页或面板，支持按项目/节点过滤并查看日志。
- API 文档需要新增节点、任务相关的端点说明。

## 实施步骤（建议按阶段落地）

### Phase 0：基础配置与节点上线（已完成）

1. 在“节点管理”创建节点（agent 或 ssh），保存后复制 `SYNC_NODE_TOKEN`。
2. 在子节点部署 `gohook-sync-agent`（当前只需心跳），设置：
   - `SYNC_NODE_ID`
   - `SYNC_NODE_TOKEN`
   - `SYNC_API_BASE`
3. 在“版本管理 → 编辑项目”开启同步、选择节点并设置目标目录与 ignore。

### Phase 1：Controller + 任务生成（下一步）

目标：把“变更/Hook 成功事件”转成 `sync_tasks`。

1. 定义任务生成入口：
   - GitHook 成功后触发（现有 hook 执行回调点）。
   - 或 watcher 检测到变更后触发（读取 `sync_file_changes`）。
2. 实现 `SyncController`：
   - 读取项目 `sync` 配置展开节点列表。
   - 生成 `sync_tasks(status=pending)`，写入 payload（项目名、根目录、变更列表/commit）。
3. 增加任务 API：
   - `GET /api/sync/tasks`（分页+过滤）
   - `GET /api/sync/tasks/:id`
4. UI 增加任务列表页（最小可用：按项目/节点过滤、显示状态与日志摘要）。

### Phase 2：Executor（ssh+rsync 优先）

目标：让主节点能真正把目录同步到子节点（不依赖 agent 执行侧）。

1. 实现 rsync 驱动：
   - 根据项目/节点 `include/exclude/ignore` 生成 rsync 参数。
   - 若 `ignore_permissions=true`，关闭 `-p/-o/-g` 或使用 `--no-perms --no-owner --no-group`。
2. Executor 拉取 pending 任务并执行：
   - 更新任务状态 `running→success/failed`。
   - 写入 stdout/stderr 到 `sync_tasks.logs` 与 `last_error`。
3. Controller 按 `max_parallel_nodes` 进行并发控制与失败重试（指数退避）。

### Phase 3：Agent 执行侧（整文件差量）

目标：agent 驱动替代 ssh，同步任务由子节点执行。

1. Agent 增加任务拉取/回传接口：
   - `GET /api/sync/nodes/:id/tasks/pull`
   - `POST /api/sync/nodes/:id/tasks/:taskId/report`
   - `GET /api/sync/nodes/:id/tasks/:taskId/bundle`
2. 主节点实现任务创建与打包：
   - 管理端临时接口：`POST /api/sync/projects/:name/run` 创建 pending 任务（后续由 Controller 替换）。
   - bundle 采用 `tar.gz`（按项目级 ignore 过滤），供 Agent 下载。
3. Agent 侧实现：
   - 心跳 + 轮询拉取任务。
   - 下载 bundle → 解压到临时目录 → 按策略落地：
     - `mirror`：目录整体替换（swap）
     - `overlay`：覆盖写入（不删除旧文件）
   - 成功/失败回传任务状态与错误信息。

### Phase 4：块级传输（Syncthing 关键能力）

目标：只传输缺失/变化的块，降低带宽与时间。

1. 引入分块与块索引：
   - 固定或滚动分块，记录 `block hash list`。
2. 主节点/Agent 交换块索引，计算差集。
3. 仅传输缺失块并重组文件，校验完成后更新快照。

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

1. 设置 TCP 端点（必须）：
   - `SYNC_TCP_ENDPOINT="10.0.0.10:9001"`
2. Agent 首次启动会在 `SYNC_AGENT_TLS_DIR` 生成自签客户端证书：
   - `SYNC_AGENT_TLS_DIR="./agent_tls"`
   - 生成 `client.crt` / `client.key`
3. 服务端指纹校验：
   - 推荐预先设置 `SYNC_SERVER_FINGERPRINT="<sha256-hex>"`。
   - 若未设置，Agent 会在首次连接时信任并保存到 `agent_tls/server.fp`（TOFU），后续必须匹配。

连接建立后，任务通过长连接即时下发；若未配置 `SYNC_TCP_ENDPOINT` 则回退到 HTTP 轮询。

## 部署建议

1. **初始化**：在主节点配置 `sync` 开关，录入 SSH key 或部署 Sync Agent。
2. **节点上线**：通过节点管理 UI 输入 SSH 主机信息并触发“自动安装 Sync Agent”，主节点会上传二进制、生成配置、推送 ignore 列表并自动注册；若环境禁止 SSH 入站，可手动部署 Agent 并提供注册 token。
3. **项目接入**：在 UI/配置中勾选需要同步的节点，设置路径。
4. **灰度试跑**：先对某个项目开启同步，观察任务队列与日志。
5. **全面启用**：结合监控/告警（Prometheus、Grafana 或现有日志系统）观察节点健康。

## 实施计划（建议分阶段）

1. **Phase 1 - Sync Agent 与自动安装**
   - UI/API 支持节点 CRUD，并在创建节点时采集 SSH 信息完成 Agent 自动部署、注册和基础健康检查。
   - 项目配置可声明节点及 `include/exclude` 规则，Agent 默认加载 `.git/`、`runtime/` 等忽略目录并允许覆盖。
   - 完成 Agent 驱动与任务队列，具备日志、重试、忽略策略同步与 UI 可视化。
2. **Phase 2 - 扩展 rsync/自定义驱动**
   - 为极简场景维持 `ssh + rsync` 驱动，沿用统一的 ignore 配置格式与 UI。
   - Agent/主节点新增差分包、断点续传以及并行多进程优化。
3. **Phase 3 - 高级特性**
   - 增量同步、断点续传、差分压缩。
   - 基于标签/区域的调度、版本回放。
   - 指标/告警打通 Observability。

## 差异化同步落地计划

为满足“非 Git 文件实时同步 + 只传输变更块”的需求，计划在现有 Sync Agent 之上分阶段实现：

1. **文件监听与快照（迭代 1）**
   - 在主节点和 Agent 端实现目录快照（`path + size + mtime + hash`）。
   - 集成 `fsnotify` 监听，监听事件触发轻量霍希校验，生成“待同步文件列表”。
   - 将差异条目写入数据库，提供 API/UI 展示；同步控制器读取这些条目生成任务，不再依赖 GitHook。
2. **整文件差量传输（迭代 2）**
   - 在主节点和 Agent 之间新增文件传输 API（HTTP/WebSocket），仅上传变动文件，保留 ACL/mtime 信息。
   - 在任务生命周期中记录断点信息，确保失败可重试；默认 gzip 压缩以降低带宽。
3. **块级增量与优化（迭代 3）**
   - 引入固定/滚动分块（参考 Syncthing/rsync 算法），维护块索引并只传输缺失块。
    - 支持多通道并发、限速、校验回写；任务完成后更新快照，使下一次扫描增量更小。

上述阶段每完成一步都同步更新 UI/API：先展示差异列表，再提供整文件传输状态，最后细化为块级指标。必要时可参考 Syncthing 的 `lib/scanner`/`lib/protocol` 包实现块索引，但仍保持 GoHook 现有控制面不变。

通过上述设计，GoHook 可以在保持主节点现有能力的同时，为需要多节点部署的场景提供统一的节点管理与代码同步体验，显著降低多环境同步成本。
