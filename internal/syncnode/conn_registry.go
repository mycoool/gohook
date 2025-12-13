package syncnode

import (
	"sync"
	"time"
)

type connState struct {
	Connected bool
	LastSeen  time.Time
}

var connRegistry = struct {
	mu sync.RWMutex
	m  map[uint]connState
}{
	m: make(map[uint]connState),
}

func markConnConnected(nodeID uint) {
	if nodeID == 0 {
		return
	}
	now := time.Now()
	connRegistry.mu.Lock()
	connRegistry.m[nodeID] = connState{Connected: true, LastSeen: now}
	connRegistry.mu.Unlock()
}

func touchConn(nodeID uint) {
	if nodeID == 0 {
		return
	}
	now := time.Now()
	connRegistry.mu.Lock()
	st, ok := connRegistry.m[nodeID]
	if !ok {
		connRegistry.m[nodeID] = connState{Connected: true, LastSeen: now}
		connRegistry.mu.Unlock()
		return
	}
	st.LastSeen = now
	connRegistry.m[nodeID] = st
	connRegistry.mu.Unlock()
}

func markConnDisconnected(nodeID uint) {
	if nodeID == 0 {
		return
	}
	connRegistry.mu.Lock()
	st := connRegistry.m[nodeID]
	st.Connected = false
	connRegistry.m[nodeID] = st
	connRegistry.mu.Unlock()
}

func getConnState(nodeID uint) (connState, bool) {
	connRegistry.mu.RLock()
	st, ok := connRegistry.m[nodeID]
	connRegistry.mu.RUnlock()
	return st, ok
}

