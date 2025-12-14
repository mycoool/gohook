#!/usr/bin/env bash
# 设置同步节点客户端需要的环境变量，可通过 `source scripts/agent-env.sh` 在当前 shell 中生效。
# 支持在执行时传入命令：`scripts/agent-env.sh ./build/gohook-agent-linux-amd64`。

set -euo pipefail

export SYNC_NODE_ID="${SYNC_NODE_ID:-1}"
export SYNC_NODE_TOKEN="${SYNC_NODE_TOKEN:-changeme-token}"
export SYNC_API_BASE="${SYNC_API_BASE:-http://127.0.0.1:9000/api}"
export SYNC_NODE_NAME="${SYNC_NODE_NAME:-$(hostname || echo sync-node)}"
export SYNC_HEARTBEAT_INTERVAL="${SYNC_HEARTBEAT_INTERVAL:-30s}"
export SYNC_AGENT_VERSION="${SYNC_AGENT_VERSION:-dev}"

echo "SYNC_NODE_ID=$SYNC_NODE_ID"
echo "SYNC_NODE_TOKEN=$SYNC_NODE_TOKEN"
echo "SYNC_API_BASE=$SYNC_API_BASE"
echo "SYNC_NODE_NAME=$SYNC_NODE_NAME"
echo "SYNC_HEARTBEAT_INTERVAL=$SYNC_HEARTBEAT_INTERVAL"
echo "SYNC_AGENT_VERSION=$SYNC_AGENT_VERSION"

if [[ $# -gt 0 ]]; then
	exec "$@"
else
	echo "环境变量已设置。如需在当前终端中保留，请运行: source scripts/agent-env.sh"
fi
