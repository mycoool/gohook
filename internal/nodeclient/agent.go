package nodeclient

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

// Agent is a lightweight client that talks to the primary GoHook instance.
type Agent struct {
	cfg       Config
	http      HTTPClient
	statePath string
}

// Config controls agent behavior.
type Config struct {
	ID                uint
	APIBase           string
	Token             string
	Interval          time.Duration
	NodeName          string
	Version           string
	WorkDir           string
	Endpoint          string
	DataDir           string
	TLSDir            string
	ServerFingerprint string
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
	if cfg.DataDir != "" {
		cfg.DataDir = filepath.Clean(cfg.DataDir)
	}
	if cfg.TLSDir == "" && cfg.DataDir != "" {
		cfg.TLSDir = filepath.Join(cfg.DataDir, "tls")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	a := &Agent{cfg: cfg, http: client}
	if cfg.DataDir != "" {
		a.statePath = filepath.Join(cfg.DataDir, "state.json")
		if st, err := LoadState(a.statePath); err == nil {
			if a.cfg.ID == 0 && st.NodeID != 0 {
				a.cfg.ID = st.NodeID
			}
			if a.cfg.Token == "" && st.Token != "" {
				a.cfg.Token = st.Token
			}
			if a.cfg.Endpoint == "" && st.Server != "" {
				a.cfg.Endpoint = st.Server
			}
		}
	}
	return a
}

// Run starts the agent loop until the context is cancelled.
// Node heartbeat is now derived from the TCP/mTLS long connection; HTTP heartbeat has been removed.
func (a *Agent) Run(ctx context.Context) {
	// Force TCP transport for sync tasks (no HTTP fallback).
	go func() {
		a.serveTCPWithRetry(ctx)
	}()
	<-ctx.Done()
	log.Printf("nodeclient: stopped")
}
