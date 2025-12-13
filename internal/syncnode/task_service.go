package syncnode

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/database"
	syncignore "github.com/mycoool/gohook/internal/syncnode/ignore"
	"github.com/mycoool/gohook/internal/types"
	"gorm.io/gorm"
)

const (
	TaskStatusPending  = "pending"
	TaskStatusRunning  = "running"
	TaskStatusSuccess  = "success"
	TaskStatusFailed   = "failed"
	TaskDriverAgent    = "agent"
	TaskDriverRsync    = "rsync"
	defaultBundleLimit = int64(1024 * 1024 * 1024) // 1GiB safety cap
)

type TaskService struct {
	db *gorm.DB
}

func NewTaskService() *TaskService {
	return &TaskService{db: database.GetDB()}
}

type TaskPayload struct {
	ProjectName       string   `json:"projectName"`
	TargetPath        string   `json:"targetPath"`
	Strategy          string   `json:"strategy"` // mirror | overlay
	IgnoreDefaults    bool     `json:"ignoreDefaults"`
	IgnorePatterns    []string `json:"ignorePatterns,omitempty"`
	IgnoreFile        string   `json:"ignoreFile,omitempty"`
	IgnoreFiles       []string `json:"ignoreFiles,omitempty"`
	IgnorePermissions bool     `json:"ignorePermissions"`
}

type IndexFileEntry struct {
	Path      string   `json:"path"`
	Size      int64    `json:"size"`
	Mode      uint32   `json:"mode"`
	MtimeUnix int64    `json:"mtime"`
	BlockSize int64    `json:"blockSize"`
	Blocks    []string `json:"blocks"`
}

type TaskReport struct {
	Status     string `json:"status" binding:"required"` // success | failed
	Logs       string `json:"logs"`
	LastError  string `json:"lastError"`
	ErrorCode  string `json:"errorCode"`
	Files      int    `json:"files,omitempty"`
	Blocks     int    `json:"blocks,omitempty"`
	Bytes      int64  `json:"bytes,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
}

func (s *TaskService) ensureDB() (*gorm.DB, error) {
	if s.db == nil {
		s.db = database.GetDB()
	}
	if s.db == nil {
		return nil, errors.New("database not initialized")
	}
	return s.db, nil
}

func (s *TaskService) CreateProjectTasks(ctx context.Context, projectName string) ([]database.SyncTask, error) {
	if types.GoHookVersionData == nil {
		return nil, errors.New("version config not loaded")
	}
	var project *types.ProjectConfig
	for i := range types.GoHookVersionData.Projects {
		if types.GoHookVersionData.Projects[i].Name == projectName {
			project = &types.GoHookVersionData.Projects[i]
			break
		}
	}
	if project == nil {
		return nil, fmt.Errorf("project not found: %s", projectName)
	}
	if project.Sync == nil || !project.Sync.Enabled {
		return nil, fmt.Errorf("project sync not enabled: %s", projectName)
	}
	if len(project.Sync.Nodes) == 0 {
		return nil, fmt.Errorf("project has no sync nodes: %s", projectName)
	}

	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}

	var created []database.SyncTask
	for _, nodeCfg := range project.Sync.Nodes {
		id64, parseErr := strconv.ParseUint(strings.TrimSpace(nodeCfg.NodeID), 10, 64)
		if parseErr != nil || id64 == 0 {
			return nil, fmt.Errorf("invalid node_id: %s", nodeCfg.NodeID)
		}
		nodeID := uint(id64)

		var node database.SyncNode
		if err := db.WithContext(ctx).First(&node, nodeID).Error; err != nil {
			return nil, err
		}

		payload := TaskPayload{
			ProjectName:       projectName,
			TargetPath:        nodeCfg.TargetPath,
			Strategy:          defaultStrategy(nodeCfg.Strategy),
			IgnoreDefaults:    project.Sync.IgnoreDefaults,
			IgnorePatterns:    append(append([]string{}, project.Sync.IgnorePatterns...), nodeCfg.IgnorePatterns...),
			IgnoreFiles:       []string{project.Sync.IgnoreFile, nodeCfg.IgnoreFile},
			IgnorePermissions: project.Sync.IgnorePermissions,
		}
		raw, _ := json.Marshal(payload)

		task := database.SyncTask{
			ProjectName: projectName,
			NodeID:      nodeID,
			NodeName:    node.Name,
			Driver:      TaskDriverAgent,
			Status:      TaskStatusPending,
			Payload:     string(raw),
		}
		if err := db.WithContext(ctx).Create(&task).Error; err != nil {
			return nil, err
		}
		created = append(created, task)
	}

	return created, nil
}

func defaultStrategy(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	switch v {
	case "overlay":
		return "overlay"
	default:
		return "mirror"
	}
}

func (s *TaskService) PullNextTask(ctx context.Context, nodeID uint) (*database.SyncTask, error) {
	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}

	var task database.SyncTask
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		q := tx.Where("node_id = ? AND status = ?", nodeID, TaskStatusPending).
			Order("created_at ASC").
			First(&task)
		if q.Error != nil {
			return q.Error
		}
		task.Status = TaskStatusRunning
		task.Attempt += 1
		return tx.Save(&task).Error
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *TaskService) ReportTask(ctx context.Context, nodeID, taskID uint, report TaskReport) (*database.SyncTask, error) {
	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}

	var task database.SyncTask
	if err := db.WithContext(ctx).First(&task, taskID).Error; err != nil {
		return nil, err
	}
	if task.NodeID != nodeID {
		return nil, errors.New("task does not belong to node")
	}

	status := strings.ToLower(strings.TrimSpace(report.Status))
	switch status {
	case "success":
		task.Status = TaskStatusSuccess
	case "failed":
		task.Status = TaskStatusFailed
	default:
		return nil, fmt.Errorf("invalid status: %s", report.Status)
	}
	if report.Logs != "" {
		task.Logs = appendLogLine(task.Logs, report.Logs)
	}
	task.LastError = report.LastError
	task.ErrorCode = report.ErrorCode
	if report.Files > 0 {
		task.FilesTotal = report.Files
	}
	if report.Blocks > 0 {
		task.BlocksTotal = report.Blocks
	}
	if report.Bytes > 0 {
		task.BytesTotal = report.Bytes
	}
	if report.DurationMs > 0 {
		task.DurationMs = report.DurationMs
	}
	if err := db.WithContext(ctx).Save(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// FailStaleRunningTasks marks long-running tasks as failed to avoid "RUNNING forever" when a connection drops.
func (s *TaskService) FailStaleRunningTasks(ctx context.Context, maxAge time.Duration) {
	if maxAge <= 0 {
		maxAge = 30 * time.Minute
	}
	db, err := s.ensureDB()
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	_ = db.WithContext(ctx).
		Model(&database.SyncTask{}).
		Where("status = ? AND updated_at < ?", TaskStatusRunning, cutoff).
		Updates(map[string]any{
			"status":     TaskStatusFailed,
			"last_error": "task timeout (connection lost or agent stuck)",
			"error_code": "TIMEOUT",
		}).Error
}

func (s *TaskService) StreamBundle(ctx context.Context, w io.Writer, task database.SyncTask) error {
	var payload TaskPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("invalid task payload: %w", err)
	}
	project := findProject(payload.ProjectName)
	if project == nil {
		return fmt.Errorf("project not found: %s", payload.ProjectName)
	}
	root := project.Path
	if root == "" {
		return fmt.Errorf("project path is empty: %s", payload.ProjectName)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("project path invalid: %s", root)
	}

	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	ig := newIgnoreMatcher(payload, root)
	var bytesWritten int64

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if ig.Match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		fi, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		hdr.Name = rel

		// hard cap to avoid streaming huge bundles accidentally
		if fi.Mode().IsRegular() {
			bytesWritten += fi.Size()
			if bytesWritten > defaultBundleLimit {
				return fmt.Errorf("bundle size exceeds limit")
			}
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}

// StreamIndex streams file index with per-block hashes to the writer via callback.
func (s *TaskService) StreamIndex(ctx context.Context, task database.SyncTask, emit func(IndexFileEntry) error) error {
	var payload TaskPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("invalid task payload: %w", err)
	}
	project := findProject(payload.ProjectName)
	if project == nil {
		return fmt.Errorf("project not found: %s", payload.ProjectName)
	}
	root := project.Path
	if root == "" {
		return fmt.Errorf("project path is empty: %s", payload.ProjectName)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("project path invalid: %s", root)
	}

	ig := newIgnoreMatcher(payload, root)

	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if ig.Match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		if !fi.Mode().IsRegular() {
			return nil
		}

		blockSize := adaptiveBlockSize(fi.Size())
		hashes, err := hashFileBlocks(path, blockSize)
		if err != nil {
			return err
		}

		entry := IndexFileEntry{
			Path:      rel,
			Size:      fi.Size(),
			Mode:      uint32(fi.Mode().Perm()),
			MtimeUnix: fi.ModTime().Unix(),
			BlockSize: blockSize,
			Blocks:    hashes,
		}
		return emit(entry)
	})
}

func adaptiveBlockSize(size int64) int64 {
	const min = 128 * 1024
	const max = 4 * 1024 * 1024
	block := int64(min)
	// Keep blocks per file <= 256, doubling as needed.
	for block < max && size/block > 256 {
		block *= 2
	}
	if block > max {
		block = max
	}
	return block
}

func hashFileBlocks(path string, blockSize int64) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var hashes []string
	buf := make([]byte, blockSize)
	for {
		n, err := io.ReadFull(f, buf)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				if n > 0 {
					sum := sha256.Sum256(buf[:n])
					hashes = append(hashes, hex.EncodeToString(sum[:]))
				}
				break
			}
			return nil, err
		}
		sum := sha256.Sum256(buf[:n])
		hashes = append(hashes, hex.EncodeToString(sum[:]))
	}
	return hashes, nil
}

func (s *TaskService) ReadBlock(task database.SyncTask, entry IndexFileEntry, index int) ([]byte, error) {
	var payload TaskPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return nil, err
	}
	proj := findProject(payload.ProjectName)
	if proj == nil {
		return nil, fmt.Errorf("project not found")
	}
	root := proj.Path
	full := filepath.Join(root, filepath.FromSlash(entry.Path))
	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	offset := int64(index) * entry.BlockSize
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	buf := make([]byte, entry.BlockSize)
	n, err := f.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return buf[:n], nil
}

func findProject(name string) *types.ProjectConfig {
	if types.GoHookVersionData == nil {
		return nil
	}
	for i := range types.GoHookVersionData.Projects {
		if types.GoHookVersionData.Projects[i].Name == name {
			return &types.GoHookVersionData.Projects[i]
		}
	}
	return nil
}

type ignoreMatcher struct {
	m *syncignore.Matcher
}

func newIgnoreMatcher(payload TaskPayload, root string) *ignoreMatcher {
	files := append([]string{}, payload.IgnoreFiles...)
	if strings.TrimSpace(payload.IgnoreFile) != "" {
		files = append(files, payload.IgnoreFile)
	}
	return &ignoreMatcher{
		m: syncignore.New(root, payload.IgnoreDefaults, payload.IgnorePatterns, files...),
	}
}

func (m *ignoreMatcher) Match(rel string, isDir bool) bool {
	if m == nil || m.m == nil {
		return false
	}
	return m.m.Match(rel, isDir)
}
