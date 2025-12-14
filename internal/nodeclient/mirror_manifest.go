package nodeclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	syncignore "github.com/mycoool/gohook/internal/syncnode/ignore"
)

type mirrorManifest struct {
	Version   int      `json:"version"`
	CreatedAt string   `json:"createdAt"`
	SyncCount int      `json:"syncCount,omitempty"`
	Paths     []string `json:"paths"`
}

func mirrorManifestPath(targetRoot string) (string, error) {
	clean := filepath.Clean(targetRoot)
	if clean == "" || clean == "/" || clean == "." {
		return "", fmt.Errorf("refuse to use manifest on unsafe targetPath")
	}
	return filepath.Join(clean, ".gohook-sync-manifest.json"), nil
}

func writeMirrorManifest(targetRoot string, expectedPaths map[string]struct{}, ig *syncignore.Matcher, syncCount int) error {
	path, err := mirrorManifestPath(targetRoot)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	paths := make([]string, 0, len(expectedPaths))
	for rel := range expectedPaths {
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		if ig != nil && ig.Match(rel, false) {
			continue
		}
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	m := mirrorManifest{
		Version:   1,
		CreatedAt: time.Now().Format(time.RFC3339),
		SyncCount: syncCount,
		Paths:     paths,
	}
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	tmp := path + ".tmp-" + fmt.Sprint(time.Now().UnixNano())
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func readMirrorManifest(targetRoot string) (*mirrorManifest, error) {
	path, err := mirrorManifestPath(targetRoot)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m mirrorManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.Version != 1 {
		return nil, errors.New("unsupported manifest version")
	}
	return &m, nil
}

func shouldUseFastMirrorDelete() bool {
	raw := strings.TrimSpace(os.Getenv("SYNC_MIRROR_FAST_DELETE"))
	return strings.EqualFold(raw, "1") || strings.EqualFold(raw, "true")
}

func mirrorFastFullScanEvery() int {
	raw := strings.TrimSpace(os.Getenv("SYNC_MIRROR_FAST_FULLSCAN_EVERY"))
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func shouldCleanEmptyDirs() bool {
	raw := strings.TrimSpace(os.Getenv("SYNC_MIRROR_CLEAN_EMPTY_DIRS"))
	return strings.EqualFold(raw, "1") || strings.EqualFold(raw, "true")
}

func cleanupEmptyParents(targetRoot, rel string, enabled bool, keepDirs map[string]struct{}) {
	if !enabled {
		return
	}
	cleanRoot := filepath.Clean(targetRoot)
	if cleanRoot == "" || cleanRoot == "/" || cleanRoot == "." {
		return
	}
	dir := filepath.Dir(filepath.Join(cleanRoot, filepath.FromSlash(rel)))
	for {
		if dir == cleanRoot || dir == "." || dir == "" || dir == string(filepath.Separator) {
			return
		}
		if keepDirs != nil {
			if drel, err := filepath.Rel(cleanRoot, dir); err == nil {
				drel = filepath.ToSlash(drel)
				if _, ok := keepDirs[drel]; ok {
					return
				}
			}
		}
		err := os.Remove(dir)
		if err != nil {
			// Stop if directory isn't empty or can't be removed.
			return
		}
		dir = filepath.Dir(dir)
	}
}

// mirrorDeleteFromManifest deletes paths that were previously synced but no longer expected.
// This is an opt-in optimization and does not guarantee strict "delete all extras" semantics
// if users create new files locally on the target.
func mirrorDeleteFromManifest(targetRoot string, expectedPaths, expectedDirs map[string]struct{}, ig *syncignore.Matcher, cleanEmptyDirs bool) (int, error) {
	m, err := readMirrorManifest(targetRoot)
	if err != nil {
		return 0, err
	}
	clean := filepath.Clean(targetRoot)
	deleted := 0
	for _, rel := range m.Paths {
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
			continue
		}
		if _, ok := expectedPaths[rel]; ok {
			continue
		}
		if ig != nil && ig.Match(rel, false) {
			continue
		}
		full := filepath.Join(clean, filepath.FromSlash(rel))
		if err := os.Remove(full); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return deleted, err
		}
		deleted++
		cleanupEmptyParents(clean, rel, cleanEmptyDirs, expectedDirs)
	}
	return deleted, nil
}
