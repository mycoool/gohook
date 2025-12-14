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
	"strconv"
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/database"
	"gorm.io/gorm"
)

type helloMessage struct {
	Type         string   `json:"type"`
	NodeID       uint     `json:"nodeId"`
	Token        string   `json:"token"`
	AgentName    string   `json:"agentName,omitempty"`
	AgentVersion string   `json:"agentVersion,omitempty"`
	Features     []string `json:"features,omitempty"`
}

type helloAck struct {
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Server string `json:"server,omitempty"`
}

type enrollMessage struct {
	Type         string `json:"type"`
	Token        string `json:"token"`
	AgentName    string `json:"agentName,omitempty"`
	AgentVersion string `json:"agentVersion,omitempty"`
}

type enrollAck struct {
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	NodeID uint   `json:"nodeId,omitempty"`
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

type indexNeed struct {
	Type   string   `json:"type"`
	TaskID uint     `json:"taskId"`
	Paths  []string `json:"paths"`
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

type blockBatchRequest struct {
	Type    string `json:"type"`
	TaskID  uint   `json:"taskId"`
	Path    string `json:"path"`
	Indices []int  `json:"indices"`
}

type blockResponse struct {
	Type      string `json:"type"`
	TaskID    uint   `json:"taskId"`
	Path      string `json:"path"`
	Index     int    `json:"index"`
	Hash      string `json:"hash"`
	Size      int    `json:"size"`
	ErrorCode string `json:"errorCode,omitempty"`
	Error     string `json:"error,omitempty"`
}

type taskReportMsg struct {
	Type       string `json:"type"`
	TaskID     uint   `json:"taskId"`
	Status     string `json:"status"`
	Logs       string `json:"logs,omitempty"`
	LastError  string `json:"lastError,omitempty"`
	ErrorCode  string `json:"errorCode,omitempty"`
	Files      int    `json:"files,omitempty"`
	Blocks     int    `json:"blocks,omitempty"`
	Bytes      int64  `json:"bytes,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
}

type nodeStatusMsg struct {
	Type            string  `json:"type"`
	NodeID          uint    `json:"nodeId"`
	UpdatedAt       string  `json:"updatedAt"`
	Hostname        string  `json:"hostname,omitempty"`
	UptimeSec       uint64  `json:"uptimeSec,omitempty"`
	CPUPercent      float64 `json:"cpuPercent,omitempty"`
	MemUsedPercent  float64 `json:"memUsedPercent,omitempty"`
	Load1           float64 `json:"load1,omitempty"`
	DiskUsedPercent float64 `json:"diskUsedPercent,omitempty"`
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

	// Task reaper: prevent RUNNING tasks from being stuck forever.
	go func() {
		timeout := 30 * time.Minute
		if raw := strings.TrimSpace(os.Getenv("SYNC_TASK_TIMEOUT")); raw != "" {
			if d, err := time.ParseDuration(raw); err == nil && d > 0 {
				timeout = d
			} else if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
				timeout = time.Duration(sec) * time.Second
			}
		}
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				defaultTaskService.FailStaleRunningTasks(ctx, timeout)
			}
		}
	}()

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

	firstFrame, err := ReadStreamFrame(conn)
	if err != nil {
		return
	}

	var base streamMessage
	if err := json.Unmarshal(firstFrame, &base); err != nil {
		return
	}

	// Enrollment: allow agent to discover nodeId using only token.
	if base.Type == "enroll" {
		var enroll enrollMessage
		if err := json.Unmarshal(firstFrame, &enroll); err != nil {
			return
		}
		svc := NewService()
		node, err := svc.FindNodeByToken(ctx, enroll.Token)
		if err != nil {
			_ = WriteStreamMessage(conn, enrollAck{Type: "enroll_ack", OK: false, Error: "invalid token"})
			return
		}
		_ = WriteStreamMessage(conn, enrollAck{Type: "enroll_ack", OK: true, NodeID: node.ID, Server: "gohook"})

		// Expect hello next.
		nextFrame, err := ReadStreamFrame(conn)
		if err != nil {
			return
		}
		firstFrame = nextFrame
		if err := json.Unmarshal(firstFrame, &base); err != nil {
			return
		}
	}

	if base.Type != "hello" {
		_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: false, Error: "invalid hello"})
		return
	}

	var hello helloMessage
	if err := json.Unmarshal(firstFrame, &hello); err != nil {
		_ = WriteStreamMessage(conn, helloAck{Type: "hello_ack", OK: false, Error: "invalid hello"})
		return
	}
	if hello.NodeID == 0 {
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

	// Heartbeat via TCP connection: mark online on connect, mark offline on close.
	_ = svc.RecordTCPConnected(ctx, hello.NodeID, hello.AgentName, hello.AgentVersion, conn.RemoteAddr().String())
	markConnConnected(hello.NodeID)
	defer func() {
		svc.RecordTCPDisconnected(ctx, hello.NodeID)
	}()

	// Single-task loop: push next task, then serve index/blocks until report arrives.
	useIndexChunk := hasFeature(hello.Features, "index_chunk_v1")
	idleBackoff := 1 * time.Second
	pingEvery := 2 * time.Second
	if raw := strings.TrimSpace(os.Getenv("SYNC_AGENT_PING_INTERVAL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			// safety clamp: avoid overwhelming the agent/server with too frequent pings
			if d < 500*time.Millisecond {
				d = 500 * time.Millisecond
			}
			pingEvery = d
		}
	}
	nextPing := time.Now()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		task, err := defaultTaskService.PullNextTask(ctx, hello.NodeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				now := time.Now()
				if now.Before(nextPing) {
					time.Sleep(idleBackoff)
					continue
				}
				nextPing = now.Add(pingEvery)

				// Idle liveness check: send a small frame periodically so that server can detect
				// closed sockets (agent stopped) even when no tasks are being dispatched.
				//
				// Important: keep this in the same goroutine to avoid concurrent writes which
				// would corrupt the length-prefixed stream protocol.
				_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
				if err := WriteStreamMessage(conn, streamMessage{Type: "server_ping"}); err != nil {
					_ = conn.SetWriteDeadline(time.Time{})
					return
				}
				_ = conn.SetWriteDeadline(time.Time{})
				touchConn(hello.NodeID)

				// Best-effort: read one status frame back (timeout is expected).
				_ = conn.SetReadDeadline(time.Now().Add(600 * time.Millisecond))
				var reply map[string]any
				if err := ReadStreamMessage(conn, &reply); err == nil {
					if typ, _ := reply["type"].(string); typ == "node_status" {
						raw, _ := json.Marshal(reply)
						var st nodeStatusMsg
						if json.Unmarshal(raw, &st) == nil && st.NodeID != 0 {
							updated := time.Now()
							if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(st.UpdatedAt)); err == nil {
								updated = ts
							}
							rs := NodeRuntimeStatus{
								UpdatedAt:       updated,
								Hostname:        strings.TrimSpace(st.Hostname),
								UptimeSec:       st.UptimeSec,
								CPUPercent:      st.CPUPercent,
								MemUsedPercent:  st.MemUsedPercent,
								Load1:           st.Load1,
								DiskUsedPercent: st.DiskUsedPercent,
							}
							setRuntimeStatus(st.NodeID, rs)
							broadcastWS(wsTypeSyncNodeStatus, map[string]any{"nodeId": st.NodeID, "runtime": rs})
						}
					}
				}
				_ = conn.SetReadDeadline(time.Time{})

				time.Sleep(idleBackoff)
				if idleBackoff < 2*time.Second {
					idleBackoff *= 2
					if idleBackoff > 2*time.Second {
						idleBackoff = 2 * time.Second
					}
				}
				continue
			}
			idleBackoff = 1 * time.Second
			time.Sleep(2 * time.Second)
			continue
		}
		idleBackoff = 1 * time.Second

		if err := WriteStreamMessage(conn, taskPush{Type: "task", Task: mapTask(task)}); err != nil {
			return
		}
		touchConn(hello.NodeID)

		indexEntries := map[string]IndexFileEntry{}

		// Expect sync_start (or an immediate task_report when agent fails preflight).
		_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		forcedIndexPaths := make([]string, 0, 32)
		for {
			var envelope map[string]any
			if err := ReadStreamMessage(conn, &envelope); err != nil {
				_ = conn.SetReadDeadline(time.Time{})
				msg := "sync_start read failed: " + err.Error()
				if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
					msg = "sync_start timeout"
				}
				_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: "failed", LastError: msg, ErrorCode: "PROTO"})
				goto nextTask
			}

			typ, _ := envelope["type"].(string)
			switch typ {
			case "index_need":
				raw, _ := json.Marshal(envelope)
				var need indexNeed
				_ = json.Unmarshal(raw, &need)
				if need.TaskID != task.ID || len(need.Paths) == 0 {
					continue
				}
				for _, p := range need.Paths {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					forcedIndexPaths = append(forcedIndexPaths, p)
				}
				continue
			case "node_status":
				raw, _ := json.Marshal(envelope)
				var st nodeStatusMsg
				if json.Unmarshal(raw, &st) == nil && st.NodeID != 0 {
					updated := time.Now()
					if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(st.UpdatedAt)); err == nil {
						updated = ts
					}
					rs := NodeRuntimeStatus{
						UpdatedAt:       updated,
						Hostname:        strings.TrimSpace(st.Hostname),
						UptimeSec:       st.UptimeSec,
						CPUPercent:      st.CPUPercent,
						MemUsedPercent:  st.MemUsedPercent,
						Load1:           st.Load1,
						DiskUsedPercent: st.DiskUsedPercent,
					}
					setRuntimeStatus(st.NodeID, rs)
					broadcastWS(wsTypeSyncNodeStatus, map[string]any{"nodeId": st.NodeID, "runtime": rs})
				}
				continue
			case "sync_start":
				var start syncStart
				raw, _ := json.Marshal(envelope)
				_ = json.Unmarshal(raw, &start)
				if start.TaskID != task.ID {
					continue
				}
				_ = conn.SetReadDeadline(time.Time{})
				touchConn(hello.NodeID)
				goto started
			case "task_report":
				var rep taskReportMsg
				raw, _ := json.Marshal(envelope)
				_ = json.Unmarshal(raw, &rep)
				if rep.TaskID != task.ID {
					continue
				}
				_ = conn.SetReadDeadline(time.Time{})
				touchConn(hello.NodeID)
				status := "failed"
				if strings.ToLower(rep.Status) == "success" {
					status = "success"
				}
				_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{
					Status:     status,
					Logs:       rep.Logs,
					LastError:  rep.LastError,
					ErrorCode:  rep.ErrorCode,
					Files:      rep.Files,
					Blocks:     rep.Blocks,
					Bytes:      rep.Bytes,
					DurationMs: rep.DurationMs,
				})
				goto nextTask
			default:
				continue
			}
		}
	started:
		_ = conn.SetReadDeadline(time.Time{})

		// Stream index.
		if err := WriteStreamMessage(conn, indexBegin{Type: "index_begin", TaskID: task.ID, Project: task.ProjectName, BlockHash: "sha256"}); err != nil {
			_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: "failed", LastError: "index_begin write failed: " + err.Error(), ErrorCode: "PROTO"})
			return
		}
		touchConn(hello.NodeID)
		if useIndexChunk {
			if err := streamIndexChunks(ctx, conn, hello.NodeID, *task, forcedIndexPaths); err != nil {
				if _, ok := err.(agentTaskReported); ok {
					goto nextTask
				}
				_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: "failed", LastError: "index stream failed: " + err.Error(), ErrorCode: "INDEX"})
				return
			}
		} else {
			if err := defaultTaskService.StreamIndexWithForcedPaths(ctx, *task, forcedIndexPaths, func(entry IndexFileEntry) error {
				indexEntries[entry.Path] = entry
				touchConn(hello.NodeID)
				return WriteStreamMessage(conn, indexFile{Type: "index_file", TaskID: task.ID, File: entry})
			}); err != nil {
				_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: "failed", LastError: "index stream failed: " + err.Error(), ErrorCode: "INDEX"})
				return
			}
		}
		if err := WriteStreamMessage(conn, indexEnd{Type: "index_end", TaskID: task.ID}); err != nil {
			_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: "failed", LastError: "index_end write failed: " + err.Error(), ErrorCode: "PROTO"})
			return
		}
		touchConn(hello.NodeID)

		// Serve block requests until task_report.
		for {
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
			var envelope map[string]any
			if err := ReadStreamMessage(conn, &envelope); err != nil {
				_ = conn.SetReadDeadline(time.Time{})
				_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: "failed", LastError: "connection lost: " + err.Error(), ErrorCode: "PROTO"})
				return
			}
			_ = conn.SetReadDeadline(time.Time{})
			typ, _ := envelope["type"].(string)
			switch typ {
			case "block_request":
				var req blockRequest
				raw, _ := json.Marshal(envelope)
				_ = json.Unmarshal(raw, &req)
				if req.TaskID != task.ID {
					continue
				}
				touchConn(hello.NodeID)
				entry, ok := indexEntries[req.Path]
				if !ok {
					_ = writeBlockResponseBin(conn, blockResponse{
						Type:      "block_response_bin",
						TaskID:    task.ID,
						Path:      req.Path,
						Index:     req.Index,
						Hash:      "",
						Size:      0,
						ErrorCode: "MISSING_ENTRY",
						Error:     "index entry not found",
					}, []byte{})
					continue
				}
				data, err := defaultTaskService.ReadBlock(*task, entry, req.Index)
				if err != nil {
					_ = writeBlockResponseBin(conn, blockResponse{
						Type:      "block_response_bin",
						TaskID:    task.ID,
						Path:      req.Path,
						Index:     req.Index,
						Hash:      "",
						Size:      0,
						ErrorCode: "BLOCK_READ",
						Error:     err.Error(),
					}, []byte{})
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
				if err := writeBlockResponseBin(conn, resp, data); err != nil {
					return
				}
				touchConn(hello.NodeID)
			case "block_batch_request":
				var req blockBatchRequest
				raw, _ := json.Marshal(envelope)
				_ = json.Unmarshal(raw, &req)
				if req.TaskID != task.ID || req.Path == "" || len(req.Indices) == 0 {
					continue
				}
				touchConn(hello.NodeID)
				entry, ok := indexEntries[req.Path]
				if !ok {
					for _, idx := range req.Indices {
						_ = writeBlockResponseBin(conn, blockResponse{
							Type:      "block_response_bin",
							TaskID:    task.ID,
							Path:      req.Path,
							Index:     idx,
							Hash:      "",
							Size:      0,
							ErrorCode: "MISSING_ENTRY",
							Error:     "index entry not found",
						}, []byte{})
					}
					continue
				}
				for _, idx := range req.Indices {
					data, err := defaultTaskService.ReadBlock(*task, entry, idx)
					if err != nil {
						_ = writeBlockResponseBin(conn, blockResponse{
							Type:      "block_response_bin",
							TaskID:    task.ID,
							Path:      req.Path,
							Index:     idx,
							Hash:      "",
							Size:      0,
							ErrorCode: "BLOCK_READ",
							Error:     err.Error(),
						}, []byte{})
						continue
					}
					sum := sha256.Sum256(data)
					resp := blockResponse{
						Type:   "block_response_bin",
						TaskID: task.ID,
						Path:   req.Path,
						Index:  idx,
						Hash:   hex.EncodeToString(sum[:]),
						Size:   len(data),
					}
					if err := writeBlockResponseBin(conn, resp, data); err != nil {
						return
					}
					touchConn(hello.NodeID)
				}
			case "task_report":
				var rep taskReportMsg
				raw, _ := json.Marshal(envelope)
				_ = json.Unmarshal(raw, &rep)
				if rep.TaskID != task.ID {
					continue
				}
				touchConn(hello.NodeID)
				status := "failed"
				if strings.ToLower(rep.Status) == "success" {
					status = "success"
				}
				_, _ = defaultTaskService.ReportTask(ctx, hello.NodeID, task.ID, TaskReport{Status: status, Logs: rep.Logs, LastError: rep.LastError, ErrorCode: rep.ErrorCode, Files: rep.Files, Blocks: rep.Blocks, Bytes: rep.Bytes, DurationMs: rep.DurationMs})
				goto nextTask
			case "node_status":
				raw, _ := json.Marshal(envelope)
				var st nodeStatusMsg
				if json.Unmarshal(raw, &st) == nil && st.NodeID != 0 {
					updated := time.Now()
					if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(st.UpdatedAt)); err == nil {
						updated = ts
					}
					rs := NodeRuntimeStatus{
						UpdatedAt:       updated,
						Hostname:        strings.TrimSpace(st.Hostname),
						UptimeSec:       st.UptimeSec,
						CPUPercent:      st.CPUPercent,
						MemUsedPercent:  st.MemUsedPercent,
						Load1:           st.Load1,
						DiskUsedPercent: st.DiskUsedPercent,
					}
					setRuntimeStatus(st.NodeID, rs)
					broadcastWS(wsTypeSyncNodeStatus, map[string]any{"nodeId": st.NodeID, "runtime": rs})
				}
				continue
			default:
				continue
			}
		}
	nextTask:
		continue
	}
}

func hasFeature(features []string, want string) bool {
	for i := range features {
		if strings.TrimSpace(features[i]) == want {
			return true
		}
	}
	return false
}

type indexChunk struct {
	Type   string           `json:"type"`
	TaskID uint             `json:"taskId"`
	Files  []IndexFileEntry `json:"files"`
}

type indexChunkDone struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
}

type agentTaskReported struct {
	rep taskReportMsg
}

func (e agentTaskReported) Error() string {
	return "agent reported task_report"
}

func indexChunkSize() int {
	size := 128
	if raw := strings.TrimSpace(os.Getenv("SYNC_INDEX_CHUNK_SIZE")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 && v <= 2000 {
			size = v
		}
	}
	return size
}

func streamIndexChunks(ctx context.Context, conn net.Conn, nodeID uint, task database.SyncTask, forcedPaths []string) error {
	chunkSize := indexChunkSize()
	var chunk []IndexFileEntry

	flush := func(files []IndexFileEntry) error {
		if len(files) == 0 {
			return nil
		}
		return serveIndexChunk(ctx, conn, nodeID, task, files)
	}

	if err := defaultTaskService.StreamIndexWithForcedPaths(ctx, task, forcedPaths, func(entry IndexFileEntry) error {
		chunk = append(chunk, entry)
		touchConn(nodeID)
		if len(chunk) >= chunkSize {
			files := append([]IndexFileEntry(nil), chunk...)
			chunk = chunk[:0]
			return flush(files)
		}
		return nil
	}); err != nil {
		return err
	}
	if len(chunk) == 0 {
		return nil
	}
	files := append([]IndexFileEntry(nil), chunk...)
	chunk = chunk[:0]
	return flush(files)
}

func isFrameTooLarge(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "frame too large")
}

func serveIndexChunk(ctx context.Context, conn net.Conn, nodeID uint, task database.SyncTask, files []IndexFileEntry) error {
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Minute))
	err := WriteStreamMessage(conn, indexChunk{Type: "index_chunk", TaskID: task.ID, Files: files})
	_ = conn.SetWriteDeadline(time.Time{})
	if err != nil {
		if isFrameTooLarge(err) && len(files) > 1 {
			mid := len(files) / 2
			if mid <= 0 {
				return err
			}
			if err := serveIndexChunk(ctx, conn, nodeID, task, files[:mid]); err != nil {
				return err
			}
			return serveIndexChunk(ctx, conn, nodeID, task, files[mid:])
		}
		return err
	}

	entries := make(map[string]IndexFileEntry, len(files))
	for i := range files {
		entries[files[i].Path] = files[i]
	}

	// Serve block requests for this chunk until agent acknowledges completion.
	for {
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		var envelope map[string]any
		if err := ReadStreamMessage(conn, &envelope); err != nil {
			_ = conn.SetReadDeadline(time.Time{})
			return err
		}
		_ = conn.SetReadDeadline(time.Time{})

		typ, _ := envelope["type"].(string)
		switch typ {
		case "index_chunk_done":
			raw, _ := json.Marshal(envelope)
			var done indexChunkDone
			_ = json.Unmarshal(raw, &done)
			if done.TaskID == task.ID {
				return nil
			}
		case "task_report":
			raw, _ := json.Marshal(envelope)
			var rep taskReportMsg
			_ = json.Unmarshal(raw, &rep)
			if rep.TaskID != task.ID {
				continue
			}
			status := "failed"
			if strings.ToLower(rep.Status) == "success" {
				status = "success"
			}
			_, _ = defaultTaskService.ReportTask(ctx, nodeID, task.ID, TaskReport{
				Status:     status,
				Logs:       rep.Logs,
				LastError:  rep.LastError,
				ErrorCode:  rep.ErrorCode,
				Files:      rep.Files,
				Blocks:     rep.Blocks,
				Bytes:      rep.Bytes,
				DurationMs: rep.DurationMs,
			})
			return agentTaskReported{rep: rep}
		case "block_request":
			raw, _ := json.Marshal(envelope)
			var req blockRequest
			_ = json.Unmarshal(raw, &req)
			if req.TaskID != task.ID {
				continue
			}
			touchConn(nodeID)
			entry, ok := entries[req.Path]
			if !ok {
				_ = writeBlockResponseBin(conn, blockResponse{
					Type:      "block_response_bin",
					TaskID:    task.ID,
					Path:      req.Path,
					Index:     req.Index,
					Hash:      "",
					Size:      0,
					ErrorCode: "MISSING_ENTRY",
					Error:     "index entry not found",
				}, []byte{})
				continue
			}
			data, err := defaultTaskService.ReadBlock(task, entry, req.Index)
			if err != nil {
				_ = writeBlockResponseBin(conn, blockResponse{
					Type:      "block_response_bin",
					TaskID:    task.ID,
					Path:      req.Path,
					Index:     req.Index,
					Hash:      "",
					Size:      0,
					ErrorCode: "BLOCK_READ",
					Error:     err.Error(),
				}, []byte{})
				continue
			}
			sum := sha256.Sum256(data)
			resp := blockResponse{Type: "block_response_bin", TaskID: task.ID, Path: req.Path, Index: req.Index, Hash: hex.EncodeToString(sum[:]), Size: len(data)}
			if err := writeBlockResponseBin(conn, resp, data); err != nil {
				return err
			}
		case "block_batch_request":
			raw, _ := json.Marshal(envelope)
			var req blockBatchRequest
			_ = json.Unmarshal(raw, &req)
			if req.TaskID != task.ID || req.Path == "" || len(req.Indices) == 0 {
				continue
			}
			touchConn(nodeID)
			entry, ok := entries[req.Path]
			if !ok {
				for _, idx := range req.Indices {
					_ = writeBlockResponseBin(conn, blockResponse{
						Type:      "block_response_bin",
						TaskID:    task.ID,
						Path:      req.Path,
						Index:     idx,
						Hash:      "",
						Size:      0,
						ErrorCode: "MISSING_ENTRY",
						Error:     "index entry not found",
					}, []byte{})
				}
				continue
			}
			for _, idx := range req.Indices {
				data, err := defaultTaskService.ReadBlock(task, entry, idx)
				if err != nil {
					_ = writeBlockResponseBin(conn, blockResponse{
						Type:      "block_response_bin",
						TaskID:    task.ID,
						Path:      req.Path,
						Index:     idx,
						Hash:      "",
						Size:      0,
						ErrorCode: "BLOCK_READ",
						Error:     err.Error(),
					}, []byte{})
					continue
				}
				sum := sha256.Sum256(data)
				resp := blockResponse{Type: "block_response_bin", TaskID: task.ID, Path: req.Path, Index: idx, Hash: hex.EncodeToString(sum[:]), Size: len(data)}
				if err := writeBlockResponseBin(conn, resp, data); err != nil {
					return err
				}
			}
		case "node_status":
			raw, _ := json.Marshal(envelope)
			var st nodeStatusMsg
			if json.Unmarshal(raw, &st) == nil && st.NodeID != 0 {
				updated := time.Now()
				if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(st.UpdatedAt)); err == nil {
					updated = ts
				}
				rs := NodeRuntimeStatus{
					UpdatedAt:       updated,
					Hostname:        strings.TrimSpace(st.Hostname),
					UptimeSec:       st.UptimeSec,
					CPUPercent:      st.CPUPercent,
					MemUsedPercent:  st.MemUsedPercent,
					Load1:           st.Load1,
					DiskUsedPercent: st.DiskUsedPercent,
				}
				setRuntimeStatus(st.NodeID, rs)
				broadcastWS(wsTypeSyncNodeStatus, map[string]any{"nodeId": st.NodeID, "runtime": rs})
			}
		default:
			continue
		}
	}
}

func writeBlockResponseBin(conn net.Conn, resp blockResponse, data []byte) error {
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Minute))
	if err := WriteStreamMessage(conn, resp); err != nil {
		_ = conn.SetWriteDeadline(time.Time{})
		return err
	}
	if err := WriteStreamFrame(conn, data); err != nil {
		_ = conn.SetWriteDeadline(time.Time{})
		return err
	}
	_ = conn.SetWriteDeadline(time.Time{})
	return nil
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
