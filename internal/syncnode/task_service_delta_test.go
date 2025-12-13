package syncnode

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStreamIndex_OverlayDelta_EmitsOnlyChangedFiles(t *testing.T) {
	t.Setenv("SYNC_DELTA_INDEX_OVERLAY", "1")

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "aaa")
	writeFile(t, filepath.Join(root, "b.txt"), "bbb")

	prev := types.GoHookVersionData
	t.Cleanup(func() { types.GoHookVersionData = prev })
	types.GoHookVersionData = &types.VersionConfig{
		Projects: []types.ProjectConfig{
			{Name: "p", Path: root, Enabled: true, Sync: &types.ProjectSyncConfig{Enabled: true}},
		},
	}

	db := openTestDB(t)
	if err := db.AutoMigrate(&database.SyncFileChange{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&database.SyncFileChange{
		Path:        "a.txt",
		Type:        "modified",
		ProjectName: "p",
		Processed:   false,
		ModTime:     time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}

	payload := TaskPayload{ProjectName: "p", Strategy: "overlay"}
	raw, _ := json.Marshal(payload)
	task := database.SyncTask{ProjectName: "p", Payload: string(raw)}

	svc := &TaskService{db: db}
	var got []IndexFileEntry
	if err := svc.StreamIndex(context.Background(), task, func(e IndexFileEntry) error {
		got = append(got, e)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].Path != "a.txt" {
		t.Fatalf("expected only a.txt, got %#v", got)
	}
}

func TestStreamIndex_OverlayDelta_FallsBackOnRename(t *testing.T) {
	t.Setenv("SYNC_DELTA_INDEX_OVERLAY", "1")

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "aaa")
	writeFile(t, filepath.Join(root, "b.txt"), "bbb")

	prev := types.GoHookVersionData
	t.Cleanup(func() { types.GoHookVersionData = prev })
	types.GoHookVersionData = &types.VersionConfig{
		Projects: []types.ProjectConfig{
			{Name: "p", Path: root, Enabled: true, Sync: &types.ProjectSyncConfig{Enabled: true}},
		},
	}

	db := openTestDB(t)
	if err := db.AutoMigrate(&database.SyncFileChange{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&database.SyncFileChange{
		Path:        "a.txt",
		Type:        "renamed",
		ProjectName: "p",
		Processed:   false,
		ModTime:     time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}

	payload := TaskPayload{ProjectName: "p", Strategy: "overlay"}
	raw, _ := json.Marshal(payload)
	task := database.SyncTask{ProjectName: "p", Payload: string(raw)}

	svc := &TaskService{db: db}
	seen := map[string]struct{}{}
	if err := svc.StreamIndex(context.Background(), task, func(e IndexFileEntry) error {
		seen[e.Path] = struct{}{}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if _, ok := seen["a.txt"]; !ok {
		t.Fatalf("expected a.txt in full-walk fallback, got %v", seen)
	}
	if _, ok := seen["b.txt"]; !ok {
		t.Fatalf("expected b.txt in full-walk fallback, got %v", seen)
	}
}

func TestReportTask_OverlaySuccess_MarksChangesProcessed(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.txt"), "aaa")

	prev := types.GoHookVersionData
	t.Cleanup(func() { types.GoHookVersionData = prev })
	types.GoHookVersionData = &types.VersionConfig{
		Projects: []types.ProjectConfig{
			{Name: "p", Path: root, Enabled: true, Sync: &types.ProjectSyncConfig{Enabled: true}},
		},
	}

	db := openTestDB(t)
	if err := db.AutoMigrate(&database.SyncFileChange{}, &database.SyncTask{}); err != nil {
		t.Fatal(err)
	}

	if err := db.Create(&database.SyncFileChange{
		Path:        "a.txt",
		Type:        "modified",
		ProjectName: "p",
		Processed:   false,
		ModTime:     time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}

	payload := TaskPayload{ProjectName: "p", Strategy: "overlay"}
	raw, _ := json.Marshal(payload)
	task := database.SyncTask{ProjectName: "p", NodeID: 1, Status: TaskStatusRunning, Payload: string(raw)}
	if err := db.Create(&task).Error; err != nil {
		t.Fatal(err)
	}

	svc := &TaskService{db: db}
	if _, err := svc.ReportTask(context.Background(), 1, task.ID, TaskReport{Status: "success"}); err != nil {
		t.Fatal(err)
	}

	var left int64
	if err := db.Model(&database.SyncFileChange{}).
		Where("project_name = ? AND processed = ?", "p", false).
		Count(&left).Error; err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("expected all changes processed, remaining=%d", left)
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
