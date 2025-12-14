package syncnode

import (
	"sync"
	"time"

	"github.com/mycoool/gohook/internal/types"
)

var overlayBaselineRegistry = struct {
	mu sync.Mutex
	m  map[string]time.Time
}{
	m: make(map[string]time.Time),
}

func shouldForceOverlayFullScan(projectName string, taskID uint) bool {
	return shouldForceOverlayFullScanWithConfig(projectName, taskID, nil)
}

func shouldForceOverlayFullScanWithConfig(projectName string, taskID uint, cfg *types.ProjectSyncConfig) bool {
	every := overlayFullScanEvery(cfg)
	if every > 0 && taskID > 0 && taskID%uint(every) == 0 {
		return true
	}

	interval := overlayFullScanInterval(cfg)
	if interval <= 0 || projectName == "" {
		return false
	}

	overlayBaselineRegistry.mu.Lock()
	last, ok := overlayBaselineRegistry.m[projectName]
	if !ok {
		// Don't force a baseline full scan immediately on first run; schedule it after interval.
		overlayBaselineRegistry.m[projectName] = time.Now()
		overlayBaselineRegistry.mu.Unlock()
		return false
	}
	if time.Since(last) >= interval {
		overlayBaselineRegistry.mu.Unlock()
		return true
	}
	overlayBaselineRegistry.mu.Unlock()
	return false
}

func markOverlayFullScan(projectName string) {
	if projectName == "" {
		return
	}
	overlayBaselineRegistry.mu.Lock()
	overlayBaselineRegistry.m[projectName] = time.Now()
	overlayBaselineRegistry.mu.Unlock()
}
