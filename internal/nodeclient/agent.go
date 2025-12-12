package nodeclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Agent is a lightweight client that talks to the primary GoHook instance.
type Agent struct {
	cfg  Config
	http HTTPClient
}

// Config controls agent behavior.
type Config struct {
	ID       uint
	APIBase  string
	Token    string
	Interval time.Duration
	NodeName string
	Version  string
	WorkDir  string
}

// HTTPClient defines the http.Client subset required by Agent.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// New constructs an Agent with sane defaults.
func New(cfg Config) *Agent {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.APIBase == "" {
		cfg.APIBase = "http://127.0.0.1:9000/api"
	}
	client := &http.Client{Timeout: 10 * time.Second}
	return &Agent{cfg: cfg, http: client}
}

// Run starts the heartbeat loop until the context is cancelled.
func (a *Agent) Run(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.Interval)
	defer ticker.Stop()

	a.sendHeartbeat(ctx)
	// Force TCP transport for sync tasks (no HTTP fallback).
	go func() {
		a.serveTCPWithRetry(ctx)
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.sendHeartbeat(ctx)
		}
	}
}

func (a *Agent) sendHeartbeat(ctx context.Context) {
	payload := map[string]interface{}{
		"token":        a.cfg.Token,
		"status":       "ONLINE",
		"health":       "HEALTHY",
		"agentVersion": a.cfg.Version,
		"hostname":     a.cfg.NodeName,
	}

	body, _ := json.Marshal(payload)
	endpoint := fmt.Sprintf("%s/sync/nodes/%d/heartbeat", strings.TrimRight(a.cfg.APIBase, "/"), a.cfg.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		log.Printf("nodeclient: build heartbeat request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sync-Token", a.cfg.Token)

	resp, err := a.http.Do(req)
	if err != nil {
		log.Printf("nodeclient: heartbeat request failed: %v", err)
		return
	}
	resp.Body.Close()
}
