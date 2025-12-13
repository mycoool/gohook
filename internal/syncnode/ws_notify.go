package syncnode

import (
	"time"

	"github.com/mycoool/gohook/internal/stream"
)

const (
	wsTypeSyncNodeEvent    = "sync_node_event"
	wsTypeSyncTaskEvent    = "sync_task_event"
	wsTypeSyncProjectEvent = "sync_project_event"
)

type syncNodeEvent struct {
	NodeID uint   `json:"nodeId"`
	Event  string `json:"event"` // created|updated|deleted|connected|disconnected|heartbeat
}

type syncTaskEvent struct {
	TaskID      uint   `json:"taskId,omitempty"`
	ProjectName string `json:"projectName,omitempty"`
	NodeID      uint   `json:"nodeId,omitempty"`
	Status      string `json:"status,omitempty"`
	Event       string `json:"event"` // created|running|reported|reaped
}

type syncProjectEvent struct {
	ProjectName string `json:"projectName"`
	Event       string `json:"event"` // changed|tasks
}

func broadcastWS(typ string, data any) {
	stream.Global.Broadcast(stream.WsMessage{
		Type:      typ,
		Timestamp: time.Now(),
		Data:      data,
	})
}

