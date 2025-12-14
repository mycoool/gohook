package nodeclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type expectedManifest struct {
	Version   int      `json:"version"`
	CreatedAt string   `json:"createdAt"`
	Paths     []string `json:"paths"`
}

func expectedManifestPath(targetRoot string) (string, error) {
	clean := filepath.Clean(targetRoot)
	if clean == "" || clean == "/" || clean == "." {
		return "", fmt.Errorf("refuse to use expected manifest on unsafe targetPath")
	}
	return filepath.Join(clean, ".gohook-sync-expected.json"), nil
}

func readExpectedManifest(targetRoot string) (*expectedManifest, error) {
	path, err := expectedManifestPath(targetRoot)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m expectedManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m.Version != 1 {
		return nil, fmt.Errorf("unsupported manifest version")
	}
	return &m, nil
}

func writeExpectedManifest(targetRoot string, expectedPaths map[string]struct{}) error {
	path, err := expectedManifestPath(targetRoot)
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
		paths = append(paths, rel)
	}
	sort.Strings(paths)

	m := expectedManifest{
		Version:   1,
		CreatedAt: time.Now().Format(time.RFC3339),
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
