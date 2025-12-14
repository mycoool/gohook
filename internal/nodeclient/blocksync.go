package nodeclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/syncnode"
	syncignore "github.com/mycoool/gohook/internal/syncnode/ignore"
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

type indexChunkMsg struct {
	Type   string                    `json:"type"`
	TaskID uint                      `json:"taskId"`
	Files  []syncnode.IndexFileEntry `json:"files"`
}

type indexChunkDoneMsg struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
}

type indexNeedMsg struct {
	Type   string   `json:"type"`
	TaskID uint     `json:"taskId"`
	Paths  []string `json:"paths"`
}

type blockReqMsg struct {
	Type   string `json:"type"`
	TaskID uint   `json:"taskId"`
	Path   string `json:"path"`
	Index  int    `json:"index"`
}

type blockBatchReqMsg struct {
	Type    string `json:"type"`
	TaskID  uint   `json:"taskId"`
	Path    string `json:"path"`
	Indices []int  `json:"indices"`
}

type blockRespMsg struct {
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

type fileResumeMeta struct {
	Version      int      `json:"version"`
	Path         string   `json:"path"`
	Size         int64    `json:"size"`
	BlockSize    int64    `json:"blockSize"`
	BlocksDigest string   `json:"blocksDigest"`
	Done         []uint64 `json:"done"`
	UpdatedUnix  int64    `json:"updatedUnix"`
}

func (a *Agent) runTaskTCP(ctx context.Context, conn io.ReadWriter, task *taskResponse) {
	startedAt := time.Now()
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

	preserveMode := !payload.IgnorePermissions && payload.PreserveMode
	preserveMtime := !payload.IgnorePermissions && payload.PreserveMtime
	symlinkPolicy := strings.ToLower(strings.TrimSpace(payload.SymlinkPolicy))

	// Optional drift healing: request index entries for missing paths from last successful sync.
	if strings.ToLower(payload.Strategy) == "" || strings.ToLower(payload.Strategy) == "overlay" {
		if m, err := readExpectedManifest(payload.TargetPath); err == nil && m != nil && len(m.Paths) > 0 {
			const maxNeed = 512
			need := make([]string, 0, 32)
			for _, rel := range m.Paths {
				if len(need) >= maxNeed {
					break
				}
				full := filepath.Join(payload.TargetPath, filepath.FromSlash(rel))
				if _, err := os.Stat(full); err != nil {
					if os.IsNotExist(err) {
						need = append(need, rel)
					}
				}
			}
			if len(need) > 0 {
				_ = syncnode.WriteStreamMessage(conn, indexNeedMsg{Type: "index_need", TaskID: task.ID, Paths: need})
			}
		}
	}

	if err := syncnode.WriteStreamMessage(conn, map[string]any{"type": "sync_start", "taskId": task.ID}); err != nil {
		return
	}

	var begin indexBeginMsg
	if err := syncnode.ReadStreamMessage(conn, &begin); err != nil || begin.Type != "index_begin" || begin.TaskID != task.ID {
		_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: "missing index_begin", ErrorCode: "PROTO"})
		return
	}

	expectedFiles := map[string]struct{}{}
	expectedDirs := map[string]struct{}{}
	entries := make([]syncnode.IndexFileEntry, 0, 128)
	var blocksFetched int
	var bytesFetched int64
	var filesApplied int
	var linksApplied int
	chunked := false
	for {
		var envelope map[string]any
		if err := syncnode.ReadStreamMessage(conn, &envelope); err != nil {
			ce := classifyError(err)
			_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
			return
		}
		typ, _ := envelope["type"].(string)
		switch typ {
		case "index_chunk":
			chunked = true
			raw, _ := json.Marshal(envelope)
			var c indexChunkMsg
			_ = json.Unmarshal(raw, &c)
			if c.TaskID != task.ID || len(c.Files) == 0 {
				continue
			}
			for i := range c.Files {
				if c.Files[i].Path == "" {
					continue
				}
				switch strings.ToLower(strings.TrimSpace(c.Files[i].Kind)) {
				case "dir":
					if err := applyDirEntry(payload.TargetPath, preserveMode, preserveMtime, c.Files[i]); err != nil {
						ce := classifyError(err)
						_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
						return
					}
					expectedDirs[c.Files[i].Path] = struct{}{}
				case "symlink":
					if symlinkPolicy != "preserve" {
						continue
					}
					if err := applySymlinkEntry(payload.TargetPath, c.Files[i]); err != nil {
						ce := classifyError(err)
						_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
						return
					}
					expectedFiles[c.Files[i].Path] = struct{}{}
					linksApplied++
				default:
					expectedFiles[c.Files[i].Path] = struct{}{}
					bc, by, err := a.applyFileBlocks(ctx, conn, task.ID, payload.TargetPath, preserveMode, preserveMtime, c.Files[i])
					blocksFetched += bc
					bytesFetched += by
					if err != nil {
						ce := classifyError(err)
						_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
						return
					}
					filesApplied++
				}
			}
			_ = syncnode.WriteStreamMessage(conn, indexChunkDoneMsg{Type: "index_chunk_done", TaskID: task.ID})
			continue
		case "index_file":
			raw, _ := json.Marshal(envelope)
			var f indexFileMsg
			_ = json.Unmarshal(raw, &f)
			if f.TaskID != task.ID || f.File.Path == "" {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(f.File.Kind)) {
			case "dir":
				if err := applyDirEntry(payload.TargetPath, preserveMode, preserveMtime, f.File); err != nil {
					ce := classifyError(err)
					_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
					return
				}
				expectedDirs[f.File.Path] = struct{}{}
			case "symlink":
				if symlinkPolicy != "preserve" {
					continue
				}
				if err := applySymlinkEntry(payload.TargetPath, f.File); err != nil {
					ce := classifyError(err)
					_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
					return
				}
				expectedFiles[f.File.Path] = struct{}{}
				linksApplied++
			default:
				expectedFiles[f.File.Path] = struct{}{}
				entries = append(entries, f.File)
			}
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
	// After receiving full index, request missing blocks. This avoids interleaving block responses
	// with the still-streaming index_file messages.
	if !chunked {
		for i := range entries {
			bc, by, err := a.applyFileBlocks(ctx, conn, task.ID, payload.TargetPath, preserveMode, preserveMtime, entries[i])
			blocksFetched += bc
			bytesFetched += by
			if err != nil {
				ce := classifyError(err)
				_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
				return
			}
			filesApplied++
		}
	}

	if strings.ToLower(payload.Strategy) == "" || strings.ToLower(payload.Strategy) == "mirror" {
		ig := syncignore.New(payload.TargetPath, payload.IgnoreDefaults, payload.IgnorePatterns)
		// Keep runtime directory even when ignoring its contents.
		if payload.IgnoreDefaults {
			_ = os.MkdirAll(filepath.Join(payload.TargetPath, "runtime"), 0o755)
		}

		fastDelete := payload.MirrorFastDelete || shouldUseFastMirrorDelete()
		cleanEmpty := payload.MirrorCleanEmptyDirs || shouldCleanEmptyDirs()
		fullEvery := payload.MirrorFastFullscanEvery
		if fullEvery <= 0 {
			fullEvery = mirrorFastFullScanEvery()
		}

		runCount := 0
		if fastDelete {
			manifest, mErr := readMirrorManifest(payload.TargetPath)
			needFull := false
			if mErr != nil || manifest == nil {
				needFull = true
			} else {
				runCount = manifest.SyncCount + 1
				if fullEvery > 0 && runCount%fullEvery == 0 {
					needFull = true
				}
			}

			if needFull {
				if err := mirrorDeleteExtras(payload.TargetPath, expectedFiles, expectedDirs, ig, cleanEmpty); err != nil {
					ce := classifyError(err)
					_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
					return
				}
			} else {
				if _, err := mirrorDeleteFromManifest(payload.TargetPath, expectedFiles, expectedDirs, ig, cleanEmpty); err != nil {
					// Fallback to strict cleanup when manifest is missing/corrupt.
					if err := mirrorDeleteExtras(payload.TargetPath, expectedFiles, expectedDirs, ig, cleanEmpty); err != nil {
						ce := classifyError(err)
						_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
						return
					}
				}
			}
		} else if err := mirrorDeleteExtras(payload.TargetPath, expectedFiles, expectedDirs, ig, cleanEmpty); err != nil {
			ce := classifyError(err)
			_ = syncnode.WriteStreamMessage(conn, taskReportMsg{Type: "task_report", TaskID: task.ID, Status: "failed", LastError: ce.Message, ErrorCode: ce.Code})
			return
		}

		// Best-effort: persist expected file list for future fast mirror deletions.
		_ = writeMirrorManifest(payload.TargetPath, expectedFiles, ig, runCount)
	}

	_ = writeExpectedManifest(payload.TargetPath, expectedFiles)

	_ = syncnode.WriteStreamMessage(conn, taskReportMsg{
		Type:       "task_report",
		TaskID:     task.ID,
		Status:     "success",
		Logs:       fmt.Sprintf("synced %d files (+%d symlinks), fetched %d blocks (%d bytes)", filesApplied, linksApplied, blocksFetched, bytesFetched),
		Files:      filesApplied,
		Blocks:     blocksFetched,
		Bytes:      bytesFetched,
		DurationMs: time.Since(startedAt).Milliseconds(),
	})
}

func (a *Agent) applyFileBlocks(ctx context.Context, conn io.ReadWriter, taskID uint, targetRoot string, preserveMode, preserveMtime bool, file syncnode.IndexFileEntry) (int, int64, error) {
	dst := filepath.Join(targetRoot, filepath.FromSlash(file.Path))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, 0, fmt.Errorf("create parent dir: %w", err)
	}

	partial := dst + ".gohook-sync-tmp-partial"
	metaPath := partial + ".json"
	srcPath := dst
	if st, err := os.Stat(partial); err == nil && st != nil && st.Mode().IsRegular() && st.Size() == file.Size {
		srcPath = partial
	}

	var meta fileResumeMeta
	metaOK := false
	usedMeta := false
	if srcPath == partial {
		if m, ok := loadResumeMeta(metaPath, file); ok {
			meta = m
			metaOK = true
		}
	}

	src, _ := os.Open(srcPath)
	if src != nil {
		defer func() {
			if src != nil {
				_ = src.Close()
			}
		}()
	}

	canReuseLocal := false
	if src != nil {
		if st, err := src.Stat(); err == nil && st != nil && st.Mode().IsRegular() && st.Size() == file.Size {
			canReuseLocal = true
		}
	}

	words := (len(file.Blocks) + 63) / 64
	done := make([]uint64, words)

	missing := make([]int, 0, 64)
	if metaOK && len(meta.Done) == words {
		copy(done, meta.Done)
		usedMeta = true
		for i := range file.Blocks {
			if (done[i/64] & (uint64(1) << uint(i%64))) == 0 {
				missing = append(missing, i)
			}
		}
	} else if !canReuseLocal {
		for i := range file.Blocks {
			missing = append(missing, i)
		}
	} else {
		buf := make([]byte, file.BlockSize)
		for i, remoteHash := range file.Blocks {
			blockOffset := int64(i) * file.BlockSize
			blockLen := minInt64(file.BlockSize, file.Size-blockOffset)
			if blockLen <= 0 {
				break
			}
			n, rerr := io.ReadFull(src, buf[:blockLen])
			if rerr != nil && !errors.Is(rerr, io.EOF) && !errors.Is(rerr, io.ErrUnexpectedEOF) {
				missing = append(missing, i)
				continue
			}
			if int64(n) != blockLen {
				missing = append(missing, i)
				continue
			}
			sum := sha256.Sum256(buf[:n])
			if remoteHash != "" && hex.EncodeToString(sum[:]) == remoteHash {
				done[i/64] |= uint64(1) << uint(i%64)
				continue
			} else {
				missing = append(missing, i)
			}
		}
	}

	if len(missing) == 0 {
		if srcPath == partial {
			// If we skipped verification using a checkpoint, verify before finalizing.
			if usedMeta {
				if err := verifyFileBlocks(partial, file); err != nil {
					_ = os.Remove(metaPath)
					return 0, 0, err
				}
			}
			// If an interrupted run already built a complete temp file, finalize it.
			if src != nil {
				_ = src.Close()
				src = nil
			}
			if err := os.Rename(partial, dst); err != nil {
				return 0, 0, fmt.Errorf("finalize partial file: %w", err)
			}
			_ = os.Remove(metaPath)
		} else {
			// Best-effort cleanup of stale resume files.
			_ = os.Remove(partial)
			_ = os.Remove(metaPath)
		}
		if preserveMode {
			_ = os.Chmod(dst, os.FileMode(file.Mode))
		}
		if preserveMtime {
			_ = os.Chtimes(dst, time.Unix(file.MtimeUnix, 0), time.Unix(file.MtimeUnix, 0))
		}
		return 0, 0, nil
	}

	out, err := os.OpenFile(partial, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return 0, 0, fmt.Errorf("open temp file: %w", err)
	}
	defer out.Close()

	if srcPath != partial && canReuseLocal {
		if err := cloneOrCopyFile(out, src); err != nil {
			return 0, 0, fmt.Errorf("seed temp file: %w", err)
		}
	}
	if err := out.Truncate(file.Size); err != nil {
		return 0, 0, fmt.Errorf("truncate temp file: %w", err)
	}

	// Best-effort: resume metadata is an optimization; sync should still work without it.
	_ = saveResumeMeta(metaPath, file, done)

	batchSize := blockBatchSize(file.BlockSize)
	var blocksFetched int
	var bytesFetched int64
	for i := 0; i < len(missing); i += batchSize {
		end := i + batchSize
		if end > len(missing) {
			end = len(missing)
		}
		chunk := missing[i:end]
		nc, _ := conn.(net.Conn)
		if nc != nil {
			_ = nc.SetWriteDeadline(time.Now().Add(10 * time.Second))
		}
		if err := syncnode.WriteStreamMessage(conn, blockBatchReqMsg{Type: "block_batch_request", TaskID: taskID, Path: file.Path, Indices: chunk}); err != nil {
			if nc != nil {
				_ = nc.SetWriteDeadline(time.Time{})
			}
			return blocksFetched, bytesFetched, err
		}
		if nc != nil {
			_ = nc.SetWriteDeadline(time.Time{})
		}
		for _, idx := range chunk {
			var resp blockRespMsg
			if nc != nil {
				_ = nc.SetReadDeadline(time.Now().Add(2 * time.Minute))
			}
			if err := syncnode.ReadStreamMessage(conn, &resp); err != nil {
				if nc != nil {
					_ = nc.SetReadDeadline(time.Time{})
				}
				return blocksFetched, bytesFetched, fmt.Errorf("read block response: %w", err)
			}
			if resp.Type != "block_response_bin" || resp.TaskID != taskID || resp.Path != file.Path || resp.Index != idx {
				if nc != nil {
					_ = nc.SetReadDeadline(time.Time{})
				}
				return blocksFetched, bytesFetched, fmt.Errorf("PROTO: unexpected block response: type=%s taskId=%d path=%s index=%d", resp.Type, resp.TaskID, resp.Path, resp.Index)
			}
			data, err := syncnode.ReadStreamFrame(conn)
			if err != nil {
				if nc != nil {
					_ = nc.SetReadDeadline(time.Time{})
				}
				return blocksFetched, bytesFetched, fmt.Errorf("read block frame: %w", err)
			}
			if nc != nil {
				_ = nc.SetReadDeadline(time.Time{})
			}
			if strings.TrimSpace(resp.ErrorCode) != "" || strings.TrimSpace(resp.Error) != "" {
				msg := strings.TrimSpace(resp.Error)
				if msg == "" {
					msg = "block fetch failed"
				}
				code := strings.TrimSpace(resp.ErrorCode)
				if code == "" {
					code = "BLOCK_ERROR"
				}
				return blocksFetched, bytesFetched, fmt.Errorf("%s: %s", code, msg)
			}
			if resp.Size <= 0 {
				return blocksFetched, bytesFetched, fmt.Errorf("PROTO: invalid block size for %s[%d]", file.Path, idx)
			}
			if resp.Size != len(data) {
				return blocksFetched, bytesFetched, fmt.Errorf("PROTO: block size mismatch for %s[%d]", file.Path, idx)
			}
			if strings.TrimSpace(resp.Hash) == "" {
				return blocksFetched, bytesFetched, fmt.Errorf("PROTO: missing block hash for %s[%d]", file.Path, idx)
			}
			sum := sha256.Sum256(data)
			if hex.EncodeToString(sum[:]) != strings.TrimSpace(resp.Hash) {
				return blocksFetched, bytesFetched, fmt.Errorf("BLOCK_HASH_MISMATCH: %s[%d]", file.Path, idx)
			}
			blockOffset := int64(idx) * file.BlockSize
			if _, err := out.WriteAt(data, blockOffset); err != nil {
				return blocksFetched, bytesFetched, fmt.Errorf("write block: %w", err)
			}
			blocksFetched++
			bytesFetched += int64(len(data))
			done[idx/64] |= uint64(1) << uint(idx%64)
		}
		_ = saveResumeMeta(metaPath, file, done)
	}

	if err := out.Close(); err != nil {
		return blocksFetched, bytesFetched, err
	}

	if usedMeta {
		if err := verifyFileBlocks(partial, file); err != nil {
			_ = os.Remove(metaPath)
			return blocksFetched, bytesFetched, err
		}
	}

	// Atomic replace.
	if err := os.Rename(partial, dst); err != nil {
		return blocksFetched, bytesFetched, fmt.Errorf("replace file: %w", err)
	}
	_ = os.Remove(metaPath)
	if preserveMode {
		_ = os.Chmod(dst, os.FileMode(file.Mode))
	}
	if preserveMtime {
		_ = os.Chtimes(dst, time.Unix(file.MtimeUnix, 0), time.Unix(file.MtimeUnix, 0))
	}
	return blocksFetched, bytesFetched, nil
}

func blockBatchSize(blockSize int64) int {
	if blockSize <= 0 {
		return 32
	}
	if raw := strings.TrimSpace(os.Getenv("SYNC_BLOCK_BATCH_SIZE")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			if v > 2048 {
				v = 2048
			}
			return v
		}
	}

	target := int64(32 << 20) // 32MiB
	if raw := strings.TrimSpace(os.Getenv("SYNC_BLOCK_BATCH_TARGET_BYTES")); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v > 0 {
			target = v
		}
	}

	n := int(target / blockSize)
	if n < 8 {
		n = 8
	}
	if n > 256 {
		n = 256
	}
	return n
}

func resumeBlocksDigest(file syncnode.IndexFileEntry) string {
	h := sha256.New()
	_, _ = h.Write([]byte(file.Path))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(fmt.Sprint(file.Size)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(fmt.Sprint(file.BlockSize)))
	_, _ = h.Write([]byte{0})
	for _, b := range file.Blocks {
		_, _ = h.Write([]byte(b))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func loadResumeMeta(path string, file syncnode.IndexFileEntry) (fileResumeMeta, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return fileResumeMeta{}, false
	}
	var m fileResumeMeta
	if err := json.Unmarshal(b, &m); err != nil {
		return fileResumeMeta{}, false
	}
	if m.Version != 1 {
		return fileResumeMeta{}, false
	}
	if m.Path != file.Path || m.Size != file.Size || m.BlockSize != file.BlockSize {
		return fileResumeMeta{}, false
	}
	if strings.TrimSpace(m.BlocksDigest) == "" || m.BlocksDigest != resumeBlocksDigest(file) {
		return fileResumeMeta{}, false
	}
	return m, true
}

func saveResumeMeta(path string, file syncnode.IndexFileEntry, done []uint64) error {
	m := fileResumeMeta{
		Version:      1,
		Path:         file.Path,
		Size:         file.Size,
		BlockSize:    file.BlockSize,
		BlocksDigest: resumeBlocksDigest(file),
		Done:         append([]uint64(nil), done...),
		UpdatedUnix:  time.Now().Unix(),
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func verifyFileBlocks(path string, file syncnode.IndexFileEntry) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("verify open: %w", err)
	}
	defer f.Close()

	buf := make([]byte, file.BlockSize)
	for i, remoteHash := range file.Blocks {
		blockOffset := int64(i) * file.BlockSize
		blockLen := minInt64(file.BlockSize, file.Size-blockOffset)
		if blockLen <= 0 {
			break
		}
		n, err := f.ReadAt(buf[:blockLen], blockOffset)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("verify read: %w", err)
		}
		if int64(n) != blockLen {
			return fmt.Errorf("verify short read: %s[%d]", file.Path, i)
		}
		sum := sha256.Sum256(buf[:blockLen])
		if remoteHash != "" && hex.EncodeToString(sum[:]) != remoteHash {
			return fmt.Errorf("BLOCK_HASH_MISMATCH: %s[%d]", file.Path, i)
		}
	}
	return nil
}

func applyDirEntry(targetRoot string, preserveMode, preserveMtime bool, entry syncnode.IndexFileEntry) error {
	dst := filepath.Join(targetRoot, filepath.FromSlash(entry.Path))
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if preserveMode {
		_ = os.Chmod(dst, os.FileMode(entry.Mode))
	}
	if preserveMtime && entry.MtimeUnix > 0 {
		_ = os.Chtimes(dst, time.Unix(entry.MtimeUnix, 0), time.Unix(entry.MtimeUnix, 0))
	}
	return nil
}

func applySymlinkEntry(targetRoot string, entry syncnode.IndexFileEntry) error {
	if strings.TrimSpace(entry.LinkTarget) == "" {
		return fmt.Errorf("SYMLINK: missing linkTarget")
	}
	dst := filepath.Join(targetRoot, filepath.FromSlash(entry.Path))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if st, err := os.Lstat(dst); err == nil && st != nil {
		if st.IsDir() {
			return fmt.Errorf("SYMLINK: destination exists as directory")
		}
		_ = os.Remove(dst)
	}
	if err := os.Symlink(entry.LinkTarget, dst); err != nil {
		return fmt.Errorf("SYMLINK: %w", err)
	}
	return nil
}

func mirrorDeleteExtras(targetRoot string, expectedFiles, expectedDirs map[string]struct{}, ig *syncignore.Matcher, cleanEmptyDirs bool) error {
	clean := filepath.Clean(targetRoot)
	if clean == "" || clean == "/" || clean == "." {
		return fmt.Errorf("refuse to mirror-delete on unsafe targetPath")
	}
	if err := filepath.WalkDir(clean, func(path string, d os.DirEntry, err error) error {
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
		if ig != nil && ig.Match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if _, ok := expectedFiles[rel]; ok {
			return nil
		}
		return os.Remove(path)
	}); err != nil {
		return err
	}
	if cleanEmptyDirs {
		return mirrorDeleteEmptyDirs(clean, expectedDirs, ig)
	}
	return nil
}

func mirrorDeleteEmptyDirs(targetRoot string, expectedDirs map[string]struct{}, ig *syncignore.Matcher) error {
	clean := filepath.Clean(targetRoot)
	if clean == "" || clean == "/" || clean == "." {
		return fmt.Errorf("refuse to mirror-delete on unsafe targetPath")
	}
	var dirs []string
	if err := filepath.WalkDir(clean, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(clean, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if ig != nil && ig.Match(rel, true) {
			return filepath.SkipDir
		}
		dirs = append(dirs, path)
		return nil
	}); err != nil {
		return err
	}
	// Bottom-up: remove deepest directories first.
	slices.SortFunc(dirs, func(a, b string) int {
		if len(a) == len(b) {
			switch {
			case a > b:
				return -1
			case a < b:
				return 1
			default:
				return 0
			}
		}
		if len(a) > len(b) {
			return -1
		}
		return 1
	})
	for _, dir := range dirs {
		rel, err := filepath.Rel(clean, dir)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if expectedDirs != nil {
			if _, ok := expectedDirs[rel]; ok {
				continue
			}
		}
		_ = os.Remove(dir)
	}
	return nil
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
