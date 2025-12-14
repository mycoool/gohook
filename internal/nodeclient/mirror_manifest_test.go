package nodeclient

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMirrorManifest_ReadWriteSorted(t *testing.T) {
	root := t.TempDir()
	expected := map[string]struct{}{
		"b.txt":     {},
		"a.txt":     {},
		"sub/c.txt": {},
	}

	if err := writeMirrorManifest(root, expected, nil, 0); err != nil {
		t.Fatal(err)
	}

	m, err := readMirrorManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != 1 {
		t.Fatalf("expected version 1, got %d", m.Version)
	}
	want := []string{"a.txt", "b.txt", "sub/c.txt"}
	if len(m.Paths) != len(want) {
		t.Fatalf("expected %d paths, got %d", len(want), len(m.Paths))
	}
	for i := range want {
		if m.Paths[i] != want[i] {
			t.Fatalf("expected %q at %d, got %q", want[i], i, m.Paths[i])
		}
	}
}

func TestMirrorDeleteFromManifest_RemovesStalePaths(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "keep.txt"), "k")
	mustWriteFile(t, filepath.Join(root, "old.txt"), "o")

	expectedOld := map[string]struct{}{
		"keep.txt": {},
		"old.txt":  {},
	}
	if err := writeMirrorManifest(root, expectedOld, nil, 0); err != nil {
		t.Fatal(err)
	}

	expectedNew := map[string]struct{}{
		"keep.txt": {},
	}
	if _, err := mirrorDeleteFromManifest(root, expectedNew, nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected old.txt removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "keep.txt")); err != nil {
		t.Fatalf("expected keep.txt present, stat err=%v", err)
	}
}

func TestMirrorDeleteFromManifest_CleansEmptyDirs(t *testing.T) {
	t.Setenv("SYNC_MIRROR_CLEAN_EMPTY_DIRS", "1")

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "sub", "old.txt"), "o")
	mustWriteFile(t, filepath.Join(root, "keep.txt"), "k")

	expectedOld := map[string]struct{}{
		"keep.txt":    {},
		"sub/old.txt": {},
	}
	if err := writeMirrorManifest(root, expectedOld, nil, 0); err != nil {
		t.Fatal(err)
	}

	expectedNew := map[string]struct{}{
		"keep.txt": {},
	}
	if _, err := mirrorDeleteFromManifest(root, expectedNew, nil, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "sub")); !os.IsNotExist(err) {
		t.Fatalf("expected sub/ removed (empty), stat err=%v", err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
