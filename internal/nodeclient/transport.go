package nodeclient

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/syncnode"
)

// connectAndServeTCP tries to establish a long-lived mTLS connection for task push.
// If connection fails, caller should fall back to polling.
func (a *Agent) connectAndServeTCP(ctx context.Context) bool {
	endpoint := os.Getenv("SYNC_TCP_ENDPOINT")
	if strings.TrimSpace(endpoint) == "" {
		return false
	}
	tlsDir := os.Getenv("SYNC_AGENT_TLS_DIR")
	if strings.TrimSpace(tlsDir) == "" {
		tlsDir = "./agent_tls"
	}

	cfg, err := loadOrCreateClientTLS(tlsDir)
	if err != nil {
		log.Printf("nodeclient: tls init failed: %v", err)
		return false
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second}
	raw, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		log.Printf("nodeclient: tcp connect failed: %v", err)
		return false
	}
	conn := tls.Client(raw, cfg)
	if err := conn.Handshake(); err != nil {
		log.Printf("nodeclient: tls handshake failed: %v", err)
		conn.Close()
		return false
	}

	hello := map[string]any{
		"type":         "hello",
		"nodeId":       a.cfg.ID,
		"token":        a.cfg.Token,
		"agentName":    a.cfg.NodeName,
		"agentVersion": a.cfg.Version,
	}
	if err := syncnode.WriteStreamMessage(conn, hello); err != nil {
		conn.Close()
		return false
	}

	var ack struct {
		Type  string `json:"type"`
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := syncnode.ReadStreamMessage(conn, &ack); err != nil || !ack.OK {
		log.Printf("nodeclient: hello rejected: %v %s", err, ack.Error)
		conn.Close()
		return false
	}

	log.Printf("nodeclient: tcp connected, waiting for tasks")
	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return true
		default:
			var msg struct {
				Type string        `json:"type"`
				Task taskResponse `json:"task"`
			}
			if err := syncnode.ReadStreamMessage(conn, &msg); err != nil {
				log.Printf("nodeclient: tcp read error: %v", err)
				return true
			}
			if msg.Type == "task" && msg.Task.ID != 0 {
				a.runTask(ctx, &msg.Task)
			}
		}
	}
}

func loadOrCreateClientTLS(dir string) (*tls.Config, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	certFile := filepath.Join(dir, "client.crt")
	keyFile := filepath.Join(dir, "client.key")
	if _, err := os.Stat(certFile); err != nil {
		if err := generateSelfSignedCert(certFile, keyFile, "gohook-sync-agent"); err != nil {
			return nil, err
		}
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	serverFP := strings.TrimSpace(os.Getenv("SYNC_SERVER_FINGERPRINT"))
	fpFile := filepath.Join(dir, "server.fp")
	if serverFP == "" {
		if b, err := os.ReadFile(fpFile); err == nil {
			serverFP = strings.TrimSpace(string(b))
		}
	}

	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("missing server certificate")
			}
			sum := sha256.Sum256(rawCerts[0])
			fp := hex.EncodeToString(sum[:])
			if serverFP != "" {
				if fp != serverFP {
					return fmt.Errorf("server fingerprint mismatch")
				}
				return nil
			}
			// TOFU: save fingerprint for next time.
			_ = os.WriteFile(fpFile, []byte(fp), 0o644)
			log.Printf("nodeclient: trusted new server fingerprint %s", fp)
			return nil
		},
	}
	return cfg, nil
}

// generateSelfSignedCert mirrors server helper for agent TLS.
func generateSelfSignedCert(certPath, keyPath, cn string) error {
	certPEM, keyPEM, err := syncnode.GenerateSelfSignedPEM(cn)
	if err != nil {
		return err
	}
	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return err
	}
	return nil
}
