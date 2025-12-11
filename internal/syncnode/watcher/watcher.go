package watcher

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/syncnode"
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
	project   types.ProjectConfig
	queue     syncnode.ChangeQueue
	nodeID    uint
	nodeName  string
	watcher   *fsnotify.Watcher
	ignoreSet map[string]struct{}
	mux       sync.Mutex
}

// NewScanner creates a scanner for the given project.
func NewScanner(project types.ProjectConfig, node database.SyncNode, queue syncnode.ChangeQueue) (*Scanner, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	s := &Scanner{
		project:   project,
		queue:     queue,
		nodeID:    node.ID,
		nodeName:  node.Name,
		watcher:   watcher,
		ignoreSet: map[string]struct{}{},
	}
	return s, nil
}

// Start begins watching directories.
func (s *Scanner) Start(root string) error {
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
				s.handleEvent(event.Name)
			}
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("watcher error: %v\n", err)
		}
	}
}

func (s *Scanner) walkAndWatch(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return s.watcher.Add(path)
		}
		return nil
	})
}

func (s *Scanner) handleEvent(path string) {
	info, err := os.Stat(path)
	change := database.SyncFileChange{
		Path:        path,
		ProjectName: s.project.Name,
		NodeID:      s.nodeID,
		NodeName:    s.nodeName,
		ModTime:     time.Now(),
	}
	if err == nil {
		if info.IsDir() {
			return
		}
		change.Size = info.Size()
		change.ModTime = info.ModTime()
		change.Type = "modified"
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
