package nodeclient

import (
	"context"
	"log"
	"os"
	"strings"
	"time"
)

func (a *Agent) serveTCPWithRetry(ctx context.Context) {
	endpoint := os.Getenv("SYNC_TCP_ENDPOINT")
	if strings.TrimSpace(endpoint) == "" {
		log.Printf("nodeclient: SYNC_TCP_ENDPOINT not set; TCP sync disabled")
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
