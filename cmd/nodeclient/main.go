package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mycoool/gohook/internal/nodeclient"
)

func main() {
	cfg := loadConfig()
	if cfg.NodeID == 0 || cfg.Token == "" {
		log.Fatalf("SYNC_NODE_ID and SYNC_NODE_TOKEN must be set")
	}

	agent := nodeclient.New(nodeclient.Config{
		ID:       cfg.NodeID,
		APIBase:  cfg.APIBase,
		Token:    cfg.Token,
		Interval: cfg.Interval,
		NodeName: cfg.NodeName,
		Version:  cfg.Version,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("sync agent started for node %d", cfg.NodeID)
	agent.Run(ctx)
}

func loadConfig() nodeclient.RuntimeConfig {
	nodeID, _ := strconv.ParseUint(os.Getenv("SYNC_NODE_ID"), 10, 64)
	interval := 30 * time.Second
	if raw := os.Getenv("SYNC_HEARTBEAT_INTERVAL"); raw != "" {
		if v, err := time.ParseDuration(raw); err == nil {
			interval = v
		}
	}

	version := os.Getenv("SYNC_AGENT_VERSION")
	if version == "" {
		version = "dev"
	}

	return nodeclient.RuntimeConfig{
		NodeID:   uint(nodeID),
		APIBase:  getenvDefault("SYNC_API_BASE", "http://127.0.0.1:9000/api"),
		Token:    os.Getenv("SYNC_NODE_TOKEN"),
		Interval: interval,
		NodeName: getenvDefault("SYNC_NODE_NAME", hostnameFallback()),
		Version:  version,
	}
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func hostnameFallback() string {
	if name, err := os.Hostname(); err == nil {
		return name
	}
	return "unknown"
}
