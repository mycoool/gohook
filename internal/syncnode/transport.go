package syncnode

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/database"
	"gorm.io/gorm"
)

type helloMessage struct {
	Type        string `json:"type"`
	NodeID      uint   `json:"nodeId"`
	Token       string `json:"token"`
	AgentName   string `json:"agentName,omitempty"`
	AgentVersion string `json:"agentVersion,omitempty"`
}

type helloAck struct {
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Server string `json:"server,omitempty"`
}

type taskPush struct {
	Type string `json:"type"`
	Task taskResponse `json:"task"`
}

// StartAgentTCPServer starts a TLS-enabled TCP server for agent long connections.
// Env:
// - SYNC_TCP_ADDR (default ":9001")
// - SYNC_TLS_DIR (default "./sync_tls")
func StartAgentTCPServer(ctx context.Context) error {
	addr := os.Getenv("SYNC_TCP_ADDR")
	if strings.TrimSpace(addr) == "" {
		addr = ":9001"
	}
	tlsDir := os.Getenv("SYNC_TLS_DIR")
	if strings.TrimSpace(tlsDir) == "" {
		tlsDir = "./sync_tls"
	}

	cfg, err := loadOrCreateServerTLS(tlsDir)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	tln := tls.NewListener(ln, cfg)
	log.Printf("syncnode: agent TCP server listening on %s", addr)

	go func() {
		<-ctx.Done()
		_ = tln.Close()
	}()

	for {
		conn, err := tln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				return nil
			}
			log.Printf("syncnode: accept error: %v", err)
			continue
		}
		go handleAgentConn(ctx, conn)
	}
}

func handleAgentConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	tconn, ok := conn.(*tls.Conn)
	if ok {
		_ = tconn.SetDeadline(time.Now().Add(10 * time.Second))
		if err := tconn.Handshake(); err != nil {
			log.Printf("syncnode: tls handshake failed: %v", err)
			return
		}
		_ = tconn.SetDeadline(time.Time{})
	}

	var hello helloMessage
	if err := ReadStreamMessage(conn, &hello); err != nil {
		return
	}
	if hello.Type != "hello" || hello.NodeID == 0 {
		_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: false, Error: "invalid hello"})
		return
	}

	fp, err := peerFingerprint(conn)
	if err != nil {
		_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: false, Error: "missing client cert"})
		return
	}

	svc := NewService()
	node, err := svc.ValidateAgentToken(ctx, hello.NodeID, hello.Token)
	if err != nil {
		_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: false, Error: "invalid token"})
		return
	}

	// Pairing: first connection stores fingerprint; subsequent must match.
	if strings.TrimSpace(node.AgentCertFingerprint) == "" {
		node.AgentCertFingerprint = fp
		if db := database.GetDB(); db != nil {
			_ = db.WithContext(ctx).Save(node).Error
		}
	} else if node.AgentCertFingerprint != fp {
		_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: false, Error: "fingerprint mismatch"})
		return
	}

	_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: true, Server: "gohook"})

	// Push loop: poll pending tasks and send to agent.
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			task, err := defaultTaskService.PullNextTask(ctx, hello.NodeID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					continue
				}
				continue
			}
			_ = WriteStreamMessage(conn, taskPush{Type: "task", Task: mapTask(task)})
		}
	}
}

func peerFingerprint(conn net.Conn) (string, error) {
	tconn, ok := conn.(*tls.Conn)
	if !ok {
		return "", errors.New("not tls")
	}
	state := tconn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return "", errors.New("no peer cert")
	}
	sum := sha256.Sum256(state.PeerCertificates[0].Raw)
	return hex.EncodeToString(sum[:]), nil
}

func loadOrCreateServerTLS(dir string) (*tls.Config, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	certFile := filepath.Join(dir, "server.crt")
	keyFile := filepath.Join(dir, "server.key")
	if _, err := os.Stat(certFile); err != nil {
		if err := generateSelfSignedCert(certFile, keyFile, "gohook-sync-server"); err != nil {
			return nil, err
		}
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAnyClientCert,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			// We verify client identity after hello by pinning fingerprint in DB.
			if len(rawCerts) == 0 {
				return errors.New("missing client certificate")
			}
			return nil
		},
	}
	return cfg, nil
}

// generateSelfSignedCert creates a self-signed certificate and key on disk.
func generateSelfSignedCert(certPath, keyPath, cn string) error {
	certPEM, keyPEM, err := GenerateSelfSignedPEM(cn)
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
