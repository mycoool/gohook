package syncnode

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

type helloMessage struct {
	Type         string `json:"type"`
	NodeID       uint   `json:"nodeId"`
	Token        string `json:"token"`
	AgentName    string `json:"agentName,omitempty"`
	AgentVersion string `json:"agentVersion,omitempty"`
}

type helloAck struct {
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Server string `json:"server,omitempty"`
}

type taskPush struct {
	Type string       `json:"type"`
	Task taskResponse `json:"task"`
}

type syncStart struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
}

type indexBegin struct {
	Type      string `json:"type"`
	TaskID    uint   `json:"taskId"`
	Project   string `json:"projectName"`
	BlockHash string `json:"blockHash"`
}

type indexFile struct {
	Type   string         `json:"type"`
	TaskID uint           `json:"taskId"`
	File   IndexFileEntry `json:"file"`
}

type indexEnd struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
}

type blockRequest struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
	Path   string `json:"path"`
	Index  int    `json:"index"`
}

type blockResponse struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
	Path   string `json:"path"`
	Index  int    `json:"index"`
	Hash   string `json:"hash"`
	Size   int    `json:"size"`
}

type taskReportMsg struct {
	Type      string `json:"type"`
	TaskID    uint   `json:"taskId"`
	Status    string `json:"status"`
	Logs      string `json:"logs,omitempty"`
	LastError string `json:"lastError,omitempty"`
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
		db, dbErr := svc.ensureDB()
		if dbErr != nil {
			log.Printf("syncnode: save pairing fingerprint failed (db): %v", dbErr)
		} else if err := db.WithContext(ctx).Save(node).Error; err != nil {
			log.Printf("syncnode: save pairing fingerprint failed: %v", err)
		}
	} else if node.AgentCertFingerprint != fp {
		_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: false, Error: "fingerprint mismatch"})
		return
	}

	_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: true, Server: "gohook"})

	// Heartbeat via TCP connection: mark online on connect, touch periodically, mark offline on close.
	_ = svc.RecordTCPConnected(ctx, hello.NodeID, hello.AgentName, hello.AgentVersion)
	touchStop := make(chan struct{})
	defer close(touchStop)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-touchStop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				svc.TouchTCPHeartbeat(ctx, hello.NodeID)
			}
		}
	}()
	defer svc.RecordTCPDisconnected(ctx, hello.NodeID)

	// Single-task loop: push next task, then serve index/blocks until report arrives.
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		task, err := defaultTaskService.PullNextTask(ctx, hello.NodeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				time.Sleep(2 * time.Second)
				continue
			}
			time.Sleep(2 * time.Second)
			continue
		}

		if err := WriteStreamMessage(conn, taskPush{Type: "task", Task: mapTask(task)}); err != nil {
			return
		}

		// Expect sync_start
		var start syncStart
		if err := ReadStreamMessage(conn, &start); err != nil || start.Type != "sync_start" || start.TaskID != task.ID {
			_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: "failed", LastError: "missing sync_start"})
			return
		}

		// Stream index.
		_ = WriteStreamMessage(conn, indexBegin{Type: "index_begin", TaskID: task.ID, Project: task.ProjectName, BlockHash: "sha256"})
		indexEntries := map[string]IndexFileEntry{}
		_ = defaultTaskService.StreamIndex(ctx, *task, func(entry IndexFileEntry) error {
			indexEntries[entry.Path] = entry
			return WriteStreamMessage(conn, indexFile{Type: "index_file", TaskID: task.ID, File: entry})
		})
		_ = WriteStreamMessage(conn, indexEnd{Type: "index_end", TaskID: task.ID})

		// Serve block requests until task_report.
		for {
			var envelope map[string]any
			if err := ReadStreamMessage(conn, &envelope); err != nil {
				return
			}
			typ, _ := envelope["type"].(string)
			switch typ {
			case "block_request":
				var req blockRequest
				raw, _ := json.Marshal(envelope)
				_ = json.Unmarshal(raw, &req)
				entry, ok := indexEntries[req.Path]
				if !ok {
					_ = WriteStreamMessage(conn, blockResponse{Type: "block_response_bin", TaskID: task.ID, Path: req.Path, Index: req.Index, Hash: "", Size: 0})
					_ = WriteStreamFrame(conn, []byte{})
					continue
				}
				data, err := defaultTaskService.ReadBlock(*task, entry, req.Index)
				if err != nil {
					_ = WriteStreamMessage(conn, blockResponse{Type: "block_response_bin", TaskID: task.ID, Path: req.Path, Index: req.Index, Hash: "", Size: 0})
					_ = WriteStreamFrame(conn, []byte{})
					continue
				}
				sum := sha256.Sum256(data)
				resp := blockResponse{
					Type:   "block_response_bin",
					TaskID: task.ID,
					Path:   req.Path,
					Index:  req.Index,
					Hash:   hex.EncodeToString(sum[:]),
					Size:   len(data),
				}
				if err := WriteStreamMessage(conn, resp); err != nil {
					return
				}
				if err := WriteStreamFrame(conn, data); err != nil {
					return
				}
			case "task_report":
				var rep taskReportMsg
				raw, _ := json.Marshal(envelope)
				_ = json.Unmarshal(raw, &rep)
				if rep.TaskID != task.ID {
					continue
				}
				status := "failed"
				if strings.ToLower(rep.Status) == "success" {
					status = "success"
				}
				_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: status, Logs: rep.Logs, LastError: rep.LastError})
				goto nextTask
			default:
				continue
			}
		}
	nextTask:
		continue
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
