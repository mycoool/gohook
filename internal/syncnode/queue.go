package syncnode

import (
	"errors"

	"github.com/mycoool/gohook/internal/database"
	"gorm.io/gorm"
)

// ChangeQueue defines storage interface for detected file changes.
type ChangeQueue interface {
	Enqueue(change database.SyncFileChange) error
}

// DBChangeQueue persists change entries into the database.
type DBChangeQueue struct {
	db *gorm.DB
}

// NewDBChangeQueue returns a queue backed by GORM.
func NewDBChangeQueue(db *gorm.DB) *DBChangeQueue {
	return &DBChangeQueue{db: db}
}

// Enqueue stores the change, deduplicating on (path, project) while the entry is unprocessed.
func (q *DBChangeQueue) Enqueue(change database.SyncFileChange) error {
	if q == nil || q.db == nil {
		return errors.New("change queue database not initialized")
	}

	var existing database.SyncFileChange
	err := q.db.Where("path = ? AND project_name = ? AND processed = ?", change.Path, change.ProjectName, false).
		First(&existing).Error
	if err == nil {
		existing.Type = change.Type
		existing.Size = change.Size
		existing.Hash = change.Hash
		existing.ModTime = change.ModTime
		existing.NodeID = change.NodeID
		existing.NodeName = change.NodeName
		existing.Error = ""
		return q.db.Save(&existing).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return q.db.Create(&change).Error
	}
	return err
}
