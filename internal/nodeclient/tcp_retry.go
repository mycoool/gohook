package nodeclient

import (
	"context"
	"log"
	"os"
	"strings"
	"time"
)

func (a *Agent) serveTCPWithRetry(ctx context.Context) {
	endpoint := strings.TrimSpace(a.cfg.Endpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("GOHOOK_SERVER"))
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("SYNC_TCP_ENDPOINT"))
	}
	if endpoint == "" {
		log.Printf("nodeclient: server endpoint not set; TCP sync disabled")
		return
	}

	backoff := 1 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := a.connectAndServeTCP(ctx); err != nil && ctx.Err() == nil {
			log.Printf("nodeclient: tcp sync disconnected: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}
