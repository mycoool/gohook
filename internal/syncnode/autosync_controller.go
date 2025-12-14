package syncnode

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mycoool/gohook/internal/database"
)

var (
	autoSyncOnce sync.Once
	autoSyncCh   chan string
)

// StartAutoSyncController starts a background controller that turns fsnotify events into sync tasks.
// It is designed to behave like Syncthing-style watching: changes are detected via filesystem events
// (not periodic full scans), then debounced into sync runs.
func StartAutoSyncController(ctx context.Context) {
	autoSyncOnce.Do(func() {
		autoSyncCh = make(chan string, 1024)
		go runAutoSyncController(ctx)
	})
}

func notifyAutoSyncProject(projectName string) {
	if strings.TrimSpace(projectName) == "" || autoSyncCh == nil {
		return
	}
	select {
	case autoSyncCh <- projectName:
	default:
		// Drop when busy; subsequent events/tasks will re-notify.
	}
}

func runAutoSyncController(ctx context.Context) {
	fireCh := make(chan string, 256)
	timers := map[string]*time.Timer{}
	mu := sync.Mutex{}

	debounceDefault := 1500 * time.Millisecond
	if raw := strings.TrimSpace(os.Getenv("SYNC_WATCH_DEBOUNCE_MS")); raw != "" {
		if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
			debounceDefault = time.Duration(ms) * time.Millisecond
		}
	}

	// Per-project serialization to avoid concurrent runs for same project.
	var projectLocks sync.Map // map[string]*sync.Mutex
	lockFor := func(projectName string) *sync.Mutex {
		if v, ok := projectLocks.Load(projectName); ok {
			return v.(*sync.Mutex)
		}
		m := &sync.Mutex{}
		if v, loaded := projectLocks.LoadOrStore(projectName, m); loaded {
			return v.(*sync.Mutex)
		}
		return m
	}

	for {
		select {
		case <-ctx.Done():
			mu.Lock()
			for _, t := range timers {
				if t != nil {
					t.Stop()
				}
			}
			timers = map[string]*time.Timer{}
			mu.Unlock()
			return
		case projectName := <-autoSyncCh:
			projectName = strings.TrimSpace(projectName)
			if projectName == "" {
				continue
			}

			delay := debounceDefault
			// Future: allow per-project debounce, for now env-only.

			mu.Lock()
			if t, ok := timers[projectName]; ok && t != nil {
				t.Stop()
			}
			timers[projectName] = time.AfterFunc(delay, func() {
				select {
				case fireCh <- projectName:
				default:
				}
			})
			mu.Unlock()
		case projectName := <-fireCh:
			projectName = strings.TrimSpace(projectName)
			if projectName == "" {
				continue
			}

			// Do not block the controller loop while running sync logic.
			go func(name string) {
				m := lockFor(name)
				m.Lock()
				defer m.Unlock()
				if err := maybeEnqueueAutoSync(ctx, name); err != nil {
					log.Printf("syncnode: auto-sync for %s skipped/failed: %v", name, err)
				}
			}(projectName)
		}
	}
}

func maybeEnqueueAutoSync(ctx context.Context, projectName string) error {
	db := database.GetDB()
	if db == nil {
		return nil
	}

	project := findProject(projectName)
	if project == nil || project.Sync == nil || !project.Sync.Enabled || !watchEnabled(project.Sync) {
		return nil
	}
	if len(project.Sync.Nodes) == 0 {
		return nil
	}

	// If there is already work in-flight/queued for this project, do not enqueue another run.
	var existing database.SyncTask
	if err := db.WithContext(ctx).
		Select("id").
		Where("project_name = ? AND status IN ?", projectName, []string{TaskStatusPending, TaskStatusRunning, TaskStatusRetrying}).
		Order("id DESC").
		First(&existing).Error; err == nil {
		return nil
	}

	// Ensure there is still something to do; the DB enqueue may have been deduplicated away.
	var cnt int64
	if err := db.WithContext(ctx).
		Model(&database.SyncFileChange{}).
		Where("project_name = ? AND processed = ?", projectName, false).
		Count(&cnt).Error; err != nil || cnt == 0 {
		return nil
	}

	_, err := defaultTaskService.CreateProjectTasks(ctx, projectName)
	return err
}
