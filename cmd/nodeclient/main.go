package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/mycoool/gohook/internal/nodeclient"
)

func main() {
	cfg := loadConfig()
	if cfg.Token == "" {
		log.Fatalf("GOHOOK_TOKEN/SYNC_NODE_TOKEN must be set (or present in state)")
	}
	if cfg.Server == "" {
		log.Fatalf("GOHOOK_SERVER/SYNC_TCP_ENDPOINT must be set (or present in state)")
	}

	agent := nodeclient.New(nodeclient.Config{
		ID:                cfg.NodeID,
		APIBase:           cfg.APIBase,
		Token:             cfg.Token,
		Interval:          cfg.Interval,
		NodeName:          cfg.NodeName,
		Version:           cfg.Version,
		WorkDir:           cfg.WorkDir,
		Endpoint:          cfg.Server,
		DataDir:           cfg.DataDir,
		TLSDir:            cfg.TLSDir,
		ServerFingerprint: cfg.ServerFingerprint,
	})

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if cfg.NodeID == 0 {
		log.Printf("sync agent started (enroll mode) server=%s dataDir=%s", cfg.Server, cfg.DataDir)
	} else {
		log.Printf("sync agent started for node %d server=%s dataDir=%s", cfg.NodeID, cfg.Server, cfg.DataDir)
	}
	agent.Run(ctx)
}

type runtimeConfig struct {
	NodeID            uint
	APIBase           string
	Token             string
	Interval          time.Duration
	NodeName          string
	Version           string
	WorkDir           string
	Server            string
	DataDir           string
	TLSDir            string
	ServerFingerprint string
}

func loadConfig() runtimeConfig {
	defaultDir := defaultDataDir()

	var (
		flagServer = flag.String("server", "", "Primary sync TCP endpoint, e.g. 10.0.0.10:9001")
		flagToken  = flag.String("token", "", "Sync agent token from node management")
		flagNodeID = flag.Uint("node-id", 0, "Node id (optional; if omitted agent will enroll by token)")

		flagDataDir = flag.String("data-dir", defaultDir, "Persistent directory for TLS + state (default: ~/.gohook-agent)")
		flagTLSDir  = flag.String("tls-dir", "", "TLS material directory (default: <data-dir>/tls)")
		flagEnvFile = flag.String("env-file", "", "Load env vars from a .env file (optional)")

		flagName     = flag.String("name", "", "Agent display name (default: hostname)")
		flagWorkDir  = flag.String("work-dir", "", "Agent work directory (optional)")
		flagVersion  = flag.String("version", "", "Agent version string (optional)")
		flagInterval = flag.Duration("interval", 30*time.Second, "Reconnect/heartbeat interval (deprecated, reserved)")
		flagFP       = flag.String("server-fingerprint", "", "Expected server certificate sha256 hex (optional; overrides TOFU)")
	)
	flag.Parse()

	if err := loadDotEnvFiles(*flagEnvFile, *flagDataDir); err != nil {
		log.Printf("nodeclient: failed to load .env: %v", err)
	}

	nodeID := uint(*flagNodeID)
	if nodeID == 0 {
		if raw := firstNonEmpty(os.Getenv("GOHOOK_NODE_ID"), os.Getenv("SYNC_NODE_ID")); raw != "" {
			if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
				nodeID = uint(v)
			}
		}
	}

	version := firstNonEmpty(*flagVersion, os.Getenv("GOHOOK_AGENT_VERSION"), os.Getenv("SYNC_AGENT_VERSION"))
	if version == "" {
		version = "dev"
	}

	dataDir := firstNonEmpty(*flagDataDir, os.Getenv("GOHOOK_DATA_DIR"))
	if dataDir == "" {
		dataDir = defaultDir
	}
	tlsDir := firstNonEmpty(*flagTLSDir, os.Getenv("GOHOOK_TLS_DIR"), os.Getenv("SYNC_AGENT_TLS_DIR"))
	if tlsDir == "" {
		tlsDir = filepath.Join(dataDir, "tls")
	}

	return runtimeConfig{
		NodeID:   nodeID,
		APIBase:  getenvDefault("SYNC_API_BASE", "http://127.0.0.1:9000/api"),
		Token:    firstNonEmpty(*flagToken, os.Getenv("GOHOOK_TOKEN"), os.Getenv("SYNC_NODE_TOKEN")),
		Interval: *flagInterval,
		NodeName: firstNonEmpty(*flagName, os.Getenv("GOHOOK_NAME"), os.Getenv("SYNC_NODE_NAME"), hostnameFallback()),
		Version:  version,
		WorkDir:  firstNonEmpty(*flagWorkDir, os.Getenv("GOHOOK_WORK_DIR"), os.Getenv("SYNC_WORK_DIR")),
		Server:   firstNonEmpty(*flagServer, os.Getenv("GOHOOK_SERVER"), os.Getenv("SYNC_TCP_ENDPOINT")),
		DataDir:  dataDir,
		TLSDir:   tlsDir,
		ServerFingerprint: firstNonEmpty(
			*flagFP,
			os.Getenv("GOHOOK_SERVER_FINGERPRINT"),
			os.Getenv("SYNC_SERVER_FINGERPRINT"),
		),
	}
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func hostnameFallback() string {
	if name, err := os.Hostname(); err == nil {
		return name
	}
	return "unknown"
}

func defaultDataDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".gohook-agent")
	}
	return "./agent_data"
}
