package syncnode

import (
	"log"
	"os"
	"sync"

	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/syncnode/watcher"
	"github.com/mycoool/gohook/internal/types"
)

// WatchManager manages file scanners for synced projects on the primary node.
type WatchManager struct {
	mu       sync.Mutex
	scanners map[string]*watcher.Scanner
	queue    ChangeQueue
}

var defaultWatchManager = &WatchManager{
	scanners: map[string]*watcher.Scanner{},
}

// StartProjectWatchers starts scanners for all enabled sync projects.
func StartProjectWatchers() {
	defaultWatchManager.Start()
}

// RefreshProjectWatchers reloads watchers based on current project sync config.
// This is used after config changes so ignore rules and enabled state take effect.
func RefreshProjectWatchers() {
	defaultWatchManager.Refresh()
}

// StopProjectWatchers stops all scanners.
func StopProjectWatchers() {
	defaultWatchManager.Stop()
}

func (m *WatchManager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.queue == nil {
		db := database.GetDB()
		if db == nil {
			log.Printf("syncnode: database not initialized, skipping project watchers")
			return
		}
		m.queue = NewNotifyingChangeQueue(db, notifyAutoSyncProject)
	}

	versionData := types.GoHookVersionData
	if versionData == nil || len(versionData.Projects) == 0 {
		return
	}

	for _, project := range versionData.Projects {
		if !project.Enabled || project.Sync == nil || !project.Sync.Enabled || !watchEnabled(project.Sync) {
			continue
		}
		if project.Path == "" {
			continue
		}
		if _, exists := m.scanners[project.Name]; exists {
			continue
		}
		if _, err := os.Stat(project.Path); err != nil {
			log.Printf("syncnode: project %s path not accessible: %v", project.Name, err)
			continue
		}

		node := database.SyncNode{
			Name: "primary",
		}
		scanner, err := watcher.NewScanner(project, node, m.queue)
		if err != nil {
			log.Printf("syncnode: create watcher for %s failed: %v", project.Name, err)
			continue
		}
		if err := scanner.Start(project.Path); err != nil {
			log.Printf("syncnode: start watcher for %s failed: %v", project.Name, err)
			_ = scanner.Close()
			continue
		}

		m.scanners[project.Name] = scanner
		log.Printf("syncnode: watching project %s at %s", project.Name, project.Path)
	}
}

func (m *WatchManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, scanner := range m.scanners {
		if scanner != nil {
			_ = scanner.Close()
		}
		delete(m.scanners, name)
	}
}

func (m *WatchManager) Refresh() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simplicity over churn minimization: restart all scanners so ignore rules take effect.
	for name, scanner := range m.scanners {
		if scanner != nil {
			_ = scanner.Close()
		}
		delete(m.scanners, name)
	}

	// Recreate based on the latest config.
	// Reuse Start() logic by inlining here to keep single lock.
	if m.queue == nil {
		db := database.GetDB()
		if db == nil {
			log.Printf("syncnode: database not initialized, skipping project watchers")
			return
		}
		m.queue = NewNotifyingChangeQueue(db, notifyAutoSyncProject)
	}

	versionData := types.GoHookVersionData
	if versionData == nil || len(versionData.Projects) == 0 {
		return
	}

	for _, project := range versionData.Projects {
		if !project.Enabled || project.Sync == nil || !project.Sync.Enabled || !watchEnabled(project.Sync) {
			continue
		}
		if project.Path == "" {
			continue
		}
		if _, err := os.Stat(project.Path); err != nil {
			log.Printf("syncnode: project %s path not accessible: %v", project.Name, err)
			continue
		}

		node := database.SyncNode{Name: "primary"}
		scanner, err := watcher.NewScanner(project, node, m.queue)
		if err != nil {
			log.Printf("syncnode: create watcher for %s failed: %v", project.Name, err)
			continue
		}
		if err := scanner.Start(project.Path); err != nil {
			log.Printf("syncnode: start watcher for %s failed: %v", project.Name, err)
			_ = scanner.Close()
			continue
		}

		m.scanners[project.Name] = scanner
		log.Printf("syncnode: watching project %s at %s", project.Name, project.Path)
	}
}
