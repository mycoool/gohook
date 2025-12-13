package syncnode

import (
	"sync"
	"time"
)

type NodeRuntimeStatus struct {
	UpdatedAt       time.Time `json:"updatedAt"`
	Hostname        string    `json:"hostname,omitempty"`
	UptimeSec       uint64    `json:"uptimeSec,omitempty"`
	CPUPercent      float64   `json:"cpuPercent,omitempty"`
	MemUsedPercent  float64   `json:"memUsedPercent,omitempty"`
	Load1           float64   `json:"load1,omitempty"`
	DiskUsedPercent float64   `json:"diskUsedPercent,omitempty"`
}

var runtimeRegistry = struct {
	mu sync.RWMutex
	m  map[uint]NodeRuntimeStatus
}{
	m: make(map[uint]NodeRuntimeStatus),
}

func setRuntimeStatus(nodeID uint, st NodeRuntimeStatus) {
	if nodeID == 0 {
		return
	}
	runtimeRegistry.mu.Lock()
	runtimeRegistry.m[nodeID] = st
	runtimeRegistry.mu.Unlock()
}

func getRuntimeStatus(nodeID uint) (NodeRuntimeStatus, bool) {
	runtimeRegistry.mu.RLock()
	st, ok := runtimeRegistry.m[nodeID]
	runtimeRegistry.mu.RUnlock()
	return st, ok
}
