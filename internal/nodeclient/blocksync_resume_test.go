package nodeclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/mycoool/gohook/internal/syncnode"
)

type fakeBlockConn struct {
	mu sync.Mutex

	in  bytes.Buffer
	out bytes.Buffer

	taskID uint
	path   string
	blocks map[int][]byte

	requested []int
}

func (c *fakeBlockConn) Read(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.in.Len() == 0 {
		return 0, io.EOF
	}
	return c.in.Read(p)
}

func (c *fakeBlockConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, _ = c.out.Write(p)
	for {
		frame, ok := tryReadFrame(&c.out)
		if !ok {
			break
		}
		var env map[string]any
		if err := json.Unmarshal(frame, &env); err != nil {
			continue
		}
		typ, _ := env["type"].(string)
		if typ != "block_batch_request" {
			continue
		}
		raw, _ := json.Marshal(env)
		var req blockBatchReqMsg
		_ = json.Unmarshal(raw, &req)
		if req.TaskID != c.taskID || req.Path != c.path {
			continue
		}
		for _, idx := range req.Indices {
			c.requested = append(c.requested, idx)
			data, ok := c.blocks[idx]
			if !ok {
				_ = syncnode.WriteStreamMessage(&c.in, blockRespMsg{
					Type:      "block_response_bin",
					TaskID:    c.taskID,
					Path:      c.path,
					Index:     idx,
					Hash:      "",
					Size:      1,
					ErrorCode: "MISSING_BLOCK",
					Error:     "missing block",
				})
				_ = syncnode.WriteStreamFrame(&c.in, []byte{})
				continue
			}
			sum := sha256.Sum256(data)
			_ = syncnode.WriteStreamMessage(&c.in, blockRespMsg{
				Type:   "block_response_bin",
				TaskID: c.taskID,
				Path:   c.path,
				Index:  idx,
				Hash:   hex.EncodeToString(sum[:]),
				Size:   len(data),
			})
			_ = syncnode.WriteStreamFrame(&c.in, data)
		}
	}

	return len(p), nil
}

func tryReadFrame(buf *bytes.Buffer) ([]byte, bool) {
	if buf.Len() < 4 {
		return nil, false
	}
	hdr := buf.Bytes()[:4]
	n := int(binary.BigEndian.Uint32(hdr))
	if n < 0 || n > 16<<20 {
		return nil, false
	}
	if buf.Len() < 4+n {
		return nil, false
	}
	_ = buf.Next(4)
	return buf.Next(n), true
}

func TestApplyFileBlocks_ResumesFromPartialWithMeta(t *testing.T) {
	ctx := context.Background()
	target := t.TempDir()
	rel := "a.txt"
	dst := filepath.Join(target, rel)
	partial := dst + ".gohook-sync-tmp-partial"
	metaPath := partial + ".json"

	b0 := []byte("hello")
	b1 := []byte("world")
	h0 := sha256.Sum256(b0)
	h1 := sha256.Sum256(b1)
	entry := syncnode.IndexFileEntry{
		Path:      rel,
		Size:      int64(len(b0) + len(b1)),
		Mode:      0o644,
		MtimeUnix: 1,
		BlockSize: int64(len(b0)),
		Blocks:    []string{hex.EncodeToString(h0[:]), hex.EncodeToString(h1[:])},
	}

	// Old dst exists, but resume should finalize from the partial file.
	write := func(p string, b []byte) {
		t.Helper()
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(dst, []byte("xxxxxxxxxx"))
	write(partial, append(append([]byte{}, b0...), []byte("xxxxx")...))

	meta := fileResumeMeta{
		Version:      1,
		Path:         entry.Path,
		Size:         entry.Size,
		BlockSize:    entry.BlockSize,
		BlocksDigest: resumeBlocksDigest(entry),
		Done:         []uint64{1 << 0},
	}
	raw, _ := json.MarshalIndent(meta, "", "  ")
	write(metaPath, raw)

	conn := &fakeBlockConn{
		taskID: 1,
		path:   rel,
		blocks: map[int][]byte{1: b1},
	}

	a := &Agent{}
	if _, _, err := a.applyFileBlocks(ctx, conn, 1, target, true, true, entry); err != nil {
		t.Fatalf("applyFileBlocks: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "helloworld" {
		t.Fatalf("dst content mismatch: %q", string(got))
	}

	if _, err := os.Stat(partial); err == nil {
		t.Fatalf("expected partial to be finalized")
	}
	if _, err := os.Stat(metaPath); err == nil {
		t.Fatalf("expected meta to be removed")
	}

	slices.Sort(conn.requested)
	if !slices.Equal(conn.requested, []int{1}) {
		t.Fatalf("unexpected requested blocks: %v", conn.requested)
	}
}
