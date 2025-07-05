package stream

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocket connection manager
type StreamManager struct {
	clients    map[*websocket.Conn]*ClientInfo
	clientsMux sync.RWMutex
}

// Client connection info - for tracking connection status and heartbeat
type ClientInfo struct {
	ConnectedAt time.Time
	LastPing    time.Time
	UserAgent   string
	RemoteAddr  string
}

// global WebSocket manager instance
var Global = &StreamManager{
	clients: make(map[*websocket.Conn]*ClientInfo),
}

// WebSocket message type
type WsMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// hook triggered message
type HookTriggeredMessage struct {
	HookID     string `json:"hookId"`
	HookName   string `json:"hookName"`
	Method     string `json:"method"`
	RemoteAddr string `json:"remoteAddr"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

// hook manage message
type HookManageMessage struct {
	Action   string `json:"action"`          // "create" | "update_basic" | "update_parameters" | "update_triggers" | "update_response" | "delete" | "update_script"
	HookID   string `json:"hookId"`          // Hook ID
	HookName string `json:"hookName"`        // Hook name
	Success  bool   `json:"success"`         // success or not
	Error    string `json:"error,omitempty"` // error message
}

// version switch message
type VersionSwitchMessage struct {
	ProjectName string `json:"projectName"`
	Action      string `json:"action"` // "switch-branch" | "switch-tag"
	Target      string `json:"target"` // branch name or tag name
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// project manage message
type ProjectManageMessage struct {
	Action      string `json:"action"` // "add" | "delete" | "edit"
	ProjectName string `json:"projectName"`
	ProjectPath string `json:"projectPath,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// git hook triggered message
type GitHookTriggeredMessage struct {
	ProjectName string `json:"projectName"`
	Action      string `json:"action"` // "switch-branch" | "switch-tag" | "delete-tag" | "delete-branch" | "skip-branch-switch" | "skip-mode-mismatch"
	Target      string `json:"target"` // branch name or tag name
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
	Skipped     bool   `json:"skipped,omitempty"` // skip operation
	Message     string `json:"message,omitempty"` // detailed message
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow cross-origin
	},
	Subprotocols: []string{"Authorization"}, // support Authorization subprotocol for authentication
}

// add WebSocket connection with client tracking
func (m *StreamManager) AddClient(conn *websocket.Conn, userAgent, remoteAddr string) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	m.clients[conn] = &ClientInfo{
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
		UserAgent:   userAgent,
		RemoteAddr:  remoteAddr,
	}
}

// remove WebSocket connection safely
func (m *StreamManager) RemoveClient(conn *websocket.Conn) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	if _, exists := m.clients[conn]; exists {
		delete(m.clients, conn)
		log.Printf("WebSocket client removed, remaining clients: %d", len(m.clients))
	}
}

// update client ping time for connection health tracking
func (m *StreamManager) UpdateClientPing(conn *websocket.Conn) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	if client, exists := m.clients[conn]; exists {
		client.LastPing = time.Now()
	}
}

// get stale connections (no ping for more than 5 minutes)
func (m *StreamManager) GetStaleConnections() []*websocket.Conn {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()

	var staleConns []*websocket.Conn
	cutoff := time.Now().Add(-5 * time.Minute)

	for conn, client := range m.clients {
		if client.LastPing.Before(cutoff) {
			staleConns = append(staleConns, conn)
		}
	}

	return staleConns
}

// broadcast message to all connected clients
// fix race condition: collect dead connections first, then batch clean up
func (m *StreamManager) Broadcast(message WsMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal WebSocket message: %v", err)
		return
	}

	// collect connections to delete, avoid modifying map during read lock
	var deadConnections []*websocket.Conn

	// get read lock for iteration
	m.clientsMux.RLock()
	for client := range m.clients {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			// collect connections to delete, not delete immediately
			deadConnections = append(deadConnections, client)
		}
	}
	m.clientsMux.RUnlock()

	// batch clean up dead connections
	if len(deadConnections) > 0 {
		//log.Printf("Cleaning up %d dead WebSocket connections", len(deadConnections))
		for _, conn := range deadConnections {
			m.RemoveClient(conn)
			conn.Close()
		}
	}
}

// get client count
func (m *StreamManager) ClientCount() int {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()
	return len(m.clients)
}

// start cleanup routine for stale connections
// clean up stale connections, prevent connection leak
func (m *StreamManager) StartCleanup() {
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			staleConns := m.GetStaleConnections()
			if len(staleConns) > 0 {
				//log.Printf("Cleaning up %d stale WebSocket connections", len(staleConns))
				for _, conn := range staleConns {
					m.RemoveClient(conn)
					conn.Close()
				}
			}
		}
	}()
}

// handle WebSocket connection
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "WebSocket upgrade failed"})
		return
	}

	// make sure connection is cleaned up
	defer func() {
		Global.RemoveClient(conn)
		conn.Close()
	}()

	// add connection to manager, include client info
	userAgent := c.GetHeader("User-Agent")
	remoteAddr := c.ClientIP()
	Global.AddClient(conn, userAgent, remoteAddr)
	log.Printf("WebSocket client connected from %s, total clients: %d", remoteAddr, Global.ClientCount())

	// set connection timeout, prevent dead connection
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Minute)); err != nil {
		log.Printf("Failed to set read deadline for %s: %v", remoteAddr, err)
		return
	}
	if err := conn.SetWriteDeadline(time.Now().Add(1 * time.Minute)); err != nil {
		log.Printf("Failed to set write deadline for %s: %v", remoteAddr, err)
		return
	}

	// send connected message
	connectedMsg := WsMessage{
		Type:      "connected",
		Timestamp: time.Now(),
		Data:      map[string]string{"message": "WebSocket connected successfully"},
	}
	if connectedData, err := json.Marshal(connectedMsg); err == nil {
		if err := conn.WriteMessage(websocket.TextMessage, connectedData); err != nil {
			log.Printf("Failed to send connected message to %s: %v", remoteAddr, err)
			return
		}
	}

	// main loop: handle messages and heartbeat
	for {
		// reset read timeout
		if err := conn.SetReadDeadline(time.Now().Add(10 * time.Minute)); err != nil {
			log.Printf("Failed to reset read deadline for %s: %v", remoteAddr, err)
			break
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			// only log unexpected close errors
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error from %s: %v", remoteAddr, err)
			}
			break
		}

		// handle client messages (heartbeat, etc.)
		var clientMsg map[string]interface{}
		if err := json.Unmarshal(message, &clientMsg); err == nil {
			if msgType, ok := clientMsg["type"].(string); ok && msgType == "ping" {
				// update heartbeat time
				Global.UpdateClientPing(conn)

				// response heartbeat
				pongMsg := WsMessage{
					Type:      "pong",
					Timestamp: time.Now(),
					Data:      map[string]string{"message": "pong"},
				}

				if pongData, err := json.Marshal(pongMsg); err == nil {
					// reset write timeout
					if err := conn.SetWriteDeadline(time.Now().Add(1 * time.Minute)); err != nil {
						log.Printf("Failed to set write deadline for %s: %v", remoteAddr, err)
						break
					}

					if err := conn.WriteMessage(websocket.TextMessage, pongData); err != nil {
						log.Printf("Failed to send pong to %s: %v", remoteAddr, err)
						break
					}
				}
			}
		}
	}

	//log.Printf("WebSocket client %s disconnected, remaining clients: %d", remoteAddr, Global.ClientCount())
}

// Initialize the stream manager
func init() {
	// start cleanup routine
	Global.StartCleanup()
}
