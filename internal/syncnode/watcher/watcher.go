package watcher

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mycoool/gohook/internal/database"
	syncignore "github.com/mycoool/gohook/internal/syncnode/ignore"
	"github.com/mycoool/gohook/internal/types"
)

// FileSnapshot describes a file entry.
type FileSnapshot struct {
	Path    string
	Size    int64
	ModTime time.Time
	Hash    string
}

// Scanner monitors a project directory and enqueues detected changes.
type Scanner struct {
	project  types.ProjectConfig
	queue    ChangeQueue
	nodeID   uint
	nodeName string
	watcher  *fsnotify.Watcher
	root     string
	ignore   *syncignore.Matcher
}

// ChangeQueue defines storage interface for detected file changes.
type ChangeQueue interface {
	Enqueue(change database.SyncFileChange) error
}

// NewScanner creates a scanner for the given project.
func NewScanner(project types.ProjectConfig, node database.SyncNode, queue ChangeQueue) (*Scanner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	s := &Scanner{
		project:  project,
		queue:    queue,
		nodeID:   node.ID,
		nodeName: node.Name,
		watcher:  watcher,
		ignore:   buildIgnoreMatcher(project),
	}
	return s, nil
}

// Start begins watching directories.
func (s *Scanner) Start(root string) error {
	s.root = root
	if err := s.walkAndWatch(root); err != nil {
		return err
	}
	go s.loop()
	return nil
}

// Close releases watcher resources.
func (s *Scanner) Close() error {
	return s.watcher.Close()
}

func (s *Scanner) loop() {
	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0 {
				s.handleEvent(event)
			}
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("watcher error: %v\n", err)
		}
	}
}

func (s *Scanner) walkAndWatch(start string) error {
	return filepath.WalkDir(start, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := toRel(s.root, path)
		if s.ignore != nil && s.ignore.Match(rel, d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return s.watcher.Add(path)
		}
		return nil
	})
}

func (s *Scanner) handleEvent(event fsnotify.Event) {
	path := event.Name
	rel := toRel(s.root, path)
	if rel == "" {
		return
	}
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		if s.ignore != nil && s.ignore.Match(rel, true) {
			return
		}
	} else {
		if s.ignore != nil && s.ignore.Match(rel, false) {
			return
		}
	}

	change := database.SyncFileChange{
		Path:        rel,
		ProjectName: s.project.Name,
		NodeID:      s.nodeID,
		NodeName:    s.nodeName,
		ModTime:     time.Now(),
	}

	if err == nil && info.IsDir() && event.Op&fsnotify.Create != 0 {
		// newly created directory: add watch and recurse
		_ = s.watcher.Add(path)
		_ = s.walkAndWatch(path)
		return
	}

	if err == nil && info.IsDir() {
		return
	}

	switch {
	case event.Op&fsnotify.Create != 0:
		change.Type = "created"
	case event.Op&fsnotify.Remove != 0:
		change.Type = "deleted"
	case event.Op&fsnotify.Rename != 0:
		change.Type = "renamed"
	default:
		change.Type = "modified"
	}

	if err == nil {
		change.Size = info.Size()
		change.ModTime = info.ModTime()
		if hash, hErr := hashFile(path); hErr == nil {
			change.Hash = hash
		}
	} else if os.IsNotExist(err) {
		change.Type = "deleted"
	} else {
		change.Error = err.Error()
	}

	_ = s.queue.Enqueue(change)
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func buildIgnoreMatcher(project types.ProjectConfig) *syncignore.Matcher {
	if project.Sync == nil {
		return syncignore.New(project.Path, false, nil)
	}
	files := []string{}
	if strings.TrimSpace(project.Sync.IgnoreFile) != "" {
		files = append(files, project.Sync.IgnoreFile)
	}
	return syncignore.New(project.Path, project.Sync.IgnoreDefaults, project.Sync.IgnorePatterns, files...)
}

func toRel(root, path string) string {
	if root == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}
