package syncnode

import "github.com/mycoool/gohook/internal/types"

func watchEnabled(cfg *types.ProjectSyncConfig) bool {
	if cfg == nil {
		return false
	}
	if cfg.WatchEnabled == nil {
		return true
	}
	return *cfg.WatchEnabled
}
