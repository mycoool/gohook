package nodeclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/syncnode"
)

type indexBeginMsg struct {
	Type      string `json:"type"`
	TaskID    uint   `json:"taskId"`
	Project   string `json:"projectName"`
	BlockHash string `json:"blockHash"`
}

type indexFileMsg struct {
	Type   string                  `json:"type"`
	TaskID uint                    `json:"taskId"`
	File   syncnode.IndexFileEntry `json:"file"`
}

type indexEndMsg struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
}

type blockReqMsg struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
	Path   string `json:"path"`
	Index  int    `json:"index"`
}

type blockRespMsg struct {
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
	ErrorCode string `json:"errorCode,omitempty"`
}

func (a *Agent) runTaskTCP(ctx context.Context, conn io.ReadWriter, task *taskResponse) {
	var payload taskPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		ce := classifyError(err)
		_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
		return
	}

	if payload.TargetPath == "" || payload.TargetPath == "/" {
		_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: "invalid targetPath", ErrorCode: "INVALID_TARGET"})
		return
	}

	if err := ensureTargetWritable(payload.TargetPath); err != nil {
		ce := classifyError(err)
		_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
		return
	}

	if err := syncnode.WriteStreamMessage(conn, map[string]any{"type": "sync_start", "taskId": task.ID}); err != nil {
		return
	}

	var begin indexBeginMsg
	if err := syncnode.ReadStreamMessage(conn, &begin); err != nil || begin.Type != "index_begin" || begin.TaskID != task.ID {
		_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: "missing index_begin", ErrorCode: "PROTO"})
		return
	}

	expected := map[string]syncnode.IndexFileEntry{}
	var files int
	for {
		var envelope map[string]any
		if err := syncnode.ReadStreamMessage(conn, &envelope); err != nil {
			ce := classifyError(err)
			_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
			return
		}
		typ, _ := envelope["type"].(string)
		switch typ {
		case "index_file":
			raw, _ := json.Marshal(envelope)
			var f indexFileMsg
			_ = json.Unmarshal(raw, &f)
			if f.TaskID != task.ID || f.File.Path == "" {
				continue
			}
			expected[f.File.Path] = f.File
			if err := a.applyFileBlocks(ctx, conn, task.ID, payload.TargetPath, payload.IgnorePermissions, f.File); err != nil {
				ce := classifyError(err)
				_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
				return
			}
			files++
		case "index_end":
			raw, _ := json.Marshal(envelope)
			var end indexEndMsg
			_ = json.Unmarshal(raw, &end)
			if end.TaskID != task.ID {
				continue
			}
			goto indexDone
		default:
			continue
		}
	}

indexDone:
	if strings.ToLower(payload.Strategy) == "" || strings.ToLower(payload.Strategy) == "mirror" {
		if err := mirrorDeleteExtras(payload.TargetPath, expected); err != nil {
			ce := classifyError(err)
			_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
			return
		}
	}

	_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "success", Logs: fmt.Sprintf("synced %d files", files)})
}

func (a *Agent) applyFileBlocks(ctx context.Context, conn io.ReadWriter, taskID uint, targetRoot string, ignorePerms bool, file syncnode.IndexFileEntry) error {
	dst := filepath.Join(targetRoot, filepath.FromSlash(file.Path))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	tmp := dst + ".gohook-sync-tmp-" + fmt.Sprint(time.Now().UnixNano())

	src, _ := os.Open(dst)
	if src != nil {
		defer src.Close()
	}

	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("open temp file: %w", err)
	}
	defer out.Close()
	if err := out.Truncate(file.Size); err != nil {
		return fmt.Errorf("truncate temp file: %w", err)
	}

	for i, remoteHash := range file.Blocks {
		blockOffset := int64(i) * file.BlockSize
		blockLen := minInt64(file.BlockSize, file.Size-blockOffset)
		if blockLen <= 0 {
			break
		}

		needFetch := true
		if src != nil {
			buf := make([]byte, blockLen)
			n, rerr := src.ReadAt(buf, blockOffset)
			if rerr == nil || errors.Is(rerr, io.EOF) {
				sum := sha256.Sum256(buf[:n])
				if hex.EncodeToString(sum[:]) == remoteHash {
					needFetch = false
					if _, err := out.WriteAt(buf[:n], blockOffset); err != nil {
						return err
					}
				}
			}
		}

		if needFetch {
			if err := syncnode.WriteStreamMessage(conn, blockReqMsg{Type: "block_request", TaskID: taskID, Path: file.Path, Index: i}); err != nil {
				return err
			}
			var resp blockRespMsg
			if err := syncnode.ReadStreamMessage(conn, &resp); err != nil {
				return fmt.Errorf("read block response: %w", err)
			}
			if resp.Type != "block_response_bin" || resp.TaskID != taskID || resp.Path != file.Path || resp.Index != i {
				return fmt.Errorf("unexpected block response")
			}
			data, err := syncnode.ReadStreamFrame(conn)
			if err != nil {
				return fmt.Errorf("read block frame: %w", err)
			}
			if resp.Size != len(data) {
				return fmt.Errorf("block size mismatch for %s[%d]", file.Path, i)
			}
			sum := sha256.Sum256(data)
			if remoteHash != "" && hex.EncodeToString(sum[:]) != remoteHash {
				return fmt.Errorf("block hash mismatch for %s[%d]", file.Path, i)
			}
			if _, err := out.WriteAt(data, blockOffset); err != nil {
				return fmt.Errorf("write block: %w", err)
			}
		}
	}

	if err := out.Close(); err != nil {
		return err
	}

	// Atomic replace.
	if err := os.Rename(tmp, dst); err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	if !ignorePerms {
		_ = os.Chmod(dst, os.FileMode(file.Mode))
		_ = os.Chtimes(dst, time.Unix(file.MtimeUnix, 0), time.Unix(file.MtimeUnix, 0))
	}
	return nil
}

func mirrorDeleteExtras(targetRoot string, expected map[string]syncnode.IndexFileEntry) error {
	clean := filepath.Clean(targetRoot)
	if clean == "" || clean == "/" || clean == "." {
		return fmt.Errorf("refuse to mirror-delete on unsafe targetPath")
	}
	return filepath.WalkDir(clean, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(clean, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		// skip temp files
		if strings.Contains(rel, ".gohook-sync-tmp-") {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := expected[rel]; ok {
			return nil
		}
		return os.Remove(path)
	})
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
