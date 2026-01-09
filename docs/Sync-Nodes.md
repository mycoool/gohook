# Sync Nodes 说明

## 定位与范围

同步节点用于在主节点完成部署后，将代码或产物同步到其他服务器。该功能的目标是简化基础协同与环境一致性维护，不用于复杂的灰度发布、变更控制或多主集群调度。

## 典型场景

- 主节点完成拉取/构建后，将结果同步到 1-2 台从节点
- 需要保持多台服务器目录内容一致，但不希望维护多条 Git 仓库 webhook
- 希望在 Web UI 里查看节点在线状态与同步结果

## 功能说明

- 节点管理：添加、编辑、删除节点，查看连接状态与最后同步信息
- 同步配置入口：版本管理中的“同步配置”用于选择节点与目标目录
- 可视化状态：节点列表提供在线与同步状态，便于快速判断同步是否正常

## 安装说明

### 主节点准备

1. 启动 GoHook 服务（主节点），确保 TCP 端口 `9001` 可达。
2. 如需调整监听地址或证书目录，使用环境变量：
   - `SYNC_TCP_ADDR`：Agent TCP 监听地址（默认 `:9001`）
   - `SYNC_TLS_DIR`：主节点 TLS 证书目录（默认 `./sync_tls`）
3. 在 Web UI 的“节点管理”中创建节点，类型选择 `agent`，复制生成的 Token。

### 子节点安装 Sync Agent

1. 在子节点准备二进制：
   - 源码构建：`go build -o gohook-agent ./cmd/nodeclient`
2. 运行 Agent（最小参数示例）：
   - `GOHOOK_SERVER=主节点IP:9001`
   - `GOHOOK_TOKEN=节点Token`
   - `./gohook-agent`
3. 可选参数：
   - `GOHOOK_NODE_ID`：指定节点 ID（不填则使用 Token 自动登记）
   - `GOHOOK_DATA_DIR`：Agent 状态与证书目录（默认 `~/.gohook-agent`）
   - `GOHOOK_SERVER_FINGERPRINT`：绑定主节点证书指纹（不填则首次连接自动记录）
4. 返回 Web UI，节点列表应显示在线状态与同步状态。

## 使用建议

- 同步范围保持精简，避免传输无关目录（如日志、缓存或构建中间产物）
- 如需复杂发布策略，请在项目脚本中自行编排

## 当前限制

- 不提供多主调度与任务抢占
- 不内置灰度发布或流量切换机制
- 同步失败的重试与告警策略需结合业务脚本处理

## 参数说明

### 节点管理字段（UI / API）

- `name`：节点名称
- `type`：节点类型（`agent` 为当前可用类型）
- `remark`：备注信息
- `address`：节点地址（`agent` 模式下由连接自动上报）
- `tags`：标签列表（用于筛选）
- `authType` / `credentialRef`：预留字段（SSH 方案使用）
- `agentToken`：Agent 连接 Token（创建后生成，可轮换）
- `agentCertFingerprint`：Agent 证书指纹（首次连接自动绑定）

### 项目同步配置（sync）

- `enabled`：是否启用同步
- `driver`：同步驱动（`agent` / `rsync` / `inherit`）
- `maxParallelNodes`：并发节点数
- `ignoreDefaults`：忽略内置默认路径（如 `.git/`、`runtime/`）
- `ignorePatterns`：额外忽略的 glob 列表
- `ignoreFile`：忽略文件路径
- `ignorePermissions`：忽略权限变更
- `watchEnabled`：启用文件变更监听触发同步
- `preserveMode`：保留文件权限
- `preserveMtime`：保留文件修改时间
- `symlinkPolicy`：符号链接策略（`ignore` / `preserve`）
- `deltaIndexOverlay`：开启增量索引（overlay 模式）
- `deltaMaxFiles`：单次增量索引允许的最大文件数
- `overlayFullScanEvery`：每 N 次任务强制全量扫描
- `overlayFullScanInterval`：至少每隔多久进行一次全量扫描（如 `1h`）
- `nodes`：目标节点列表

### 目标节点配置（sync.nodes）

- `nodeId`：节点 ID
- `targetPath`：目标目录
- `strategy`：同步策略（`mirror` / `overlay`）
- `driver`：覆盖项目级驱动
- `include` / `exclude`：白名单 / 黑名单 glob
- `ignoreFile` / `ignorePatterns`：节点级忽略配置
- `mirrorFastDelete`：镜像模式快速删除优化
- `mirrorFastFullscanEvery`：镜像模式强制全量校验频率
- `mirrorCleanEmptyDirs`：镜像模式清理空目录
- `mirrorSyncEmptyDirs`：镜像模式同步空目录

### 运行参数（环境变量）

主节点：

- `SYNC_TCP_ADDR`：Agent TCP 监听地址（默认 `:9001`）
- `SYNC_TLS_DIR`：主节点 TLS 证书目录（默认 `./sync_tls`）
- `SYNC_TASK_TIMEOUT`：任务超时（默认 `30m`）
- `SYNC_TASK_MAX_ATTEMPTS`：失败重试上限（默认 `3`）
- `SYNC_AGENT_PING_INTERVAL`：与 Agent 的空闲 ping 间隔（默认 `2s`）
- `SYNC_WATCH_DEBOUNCE_MS`：文件变更合并窗口（默认 `1500`）
- `SYNC_DELTA_INDEX_OVERLAY`：强制开启/关闭增量索引（`true/false`）
- `SYNC_DELTA_MAX_FILES`：增量索引最大文件数（默认 `5000`）
- `SYNC_OVERLAY_FULLSCAN_EVERY`：每 N 次任务强制全量扫描
- `SYNC_OVERLAY_FULLSCAN_INTERVAL`：全量扫描最小间隔（默认 `1h`）
- `SYNC_INDEX_CHUNK_SIZE`：索引分片大小（默认 `128`，最大 `2000`）
- `SYNC_HASHCACHE_ENTRIES`：哈希缓存条数（默认 `2048`）

Agent：

- `GOHOOK_SERVER` / `SYNC_TCP_ENDPOINT`：主节点 TCP 地址
- `GOHOOK_TOKEN` / `SYNC_NODE_TOKEN`：节点 Token
- `GOHOOK_NODE_ID` / `SYNC_NODE_ID`：节点 ID（可选）
- `GOHOOK_DATA_DIR`：Agent 数据目录（默认 `~/.gohook-agent`）
- `GOHOOK_TLS_DIR` / `SYNC_AGENT_TLS_DIR`：Agent TLS 目录
- `GOHOOK_SERVER_FINGERPRINT` / `SYNC_SERVER_FINGERPRINT`：主节点证书指纹
- `GOHOOK_NAME` / `SYNC_NODE_NAME`：Agent 显示名称
- `GOHOOK_WORK_DIR` / `SYNC_WORK_DIR`：工作目录
- `GOHOOK_AGENT_VERSION` / `SYNC_AGENT_VERSION`：Agent 版本标识
- `SYNC_INDEX_CHUNKED`：启用索引分片特性（`true/false`）
- `SYNC_BLOCK_BATCH_SIZE`：单次批量拉取块数
- `SYNC_BLOCK_BATCH_TARGET_BYTES`：单次批量拉取目标字节数
