package syncnode

import (
	"github.com/mycoool/gohook/internal/database"
	"gorm.io/gorm"
)

type NotifyingChangeQueue struct {
	db     *gorm.DB
	notify func(projectName string)
}

func NewNotifyingChangeQueue(db *gorm.DB, notify func(projectName string)) *NotifyingChangeQueue {
	return &NotifyingChangeQueue{db: db, notify: notify}
}

func (q *NotifyingChangeQueue) Enqueue(change database.SyncFileChange) error {
	base := NewDBChangeQueue(q.db)
	if err := base.Enqueue(change); err != nil {
		return err
	}
	if q.notify != nil && change.ProjectName != "" {
		q.notify(change.ProjectName)
	}
	return nil
}
