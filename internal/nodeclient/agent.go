package nodeclient

import (
	"context"
	"log"
	"net/http"
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
