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
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
}

// global WebSocket manager instance
var Global = &StreamManager{
	clients: make(map[*websocket.Conn]bool),
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
	Action      string `json:"action"` // "add" | "delete"
	ProjectName string `json:"projectName"`
	ProjectPath string `json:"projectPath,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow cross-origin
	},
	Subprotocols: []string{"Authorization"}, // 支持Authorization子协议用于认证
}

// add WebSocket connection
func (m *StreamManager) AddClient(conn *websocket.Conn) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	m.clients[conn] = true
}

// remove WebSocket connection
func (m *StreamManager) RemoveClient(conn *websocket.Conn) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	delete(m.clients, conn)
}

// broadcast message to all connected clients
func (m *StreamManager) Broadcast(message WsMessage) {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	for client := range m.clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			// connection disconnected, remove client
			go func(conn *websocket.Conn) {
				m.RemoveClient(conn)
				conn.Close()
			}(client)
		}
	}
}

// get client count
func (m *StreamManager) ClientCount() int {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()
	return len(m.clients)
}

// handle WebSocket connection
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "WebSocket upgrade failed"})
		return
	}
	defer func() {
		// remove connection from manager
		Global.RemoveClient(conn)
		conn.Close()
	}()

	// add connection to global manager
	Global.AddClient(conn)
	log.Printf("WebSocket client connected, total clients: %d", Global.ClientCount())

	// send connected message
	connectedMsg := WsMessage{
		Type:      "connected",
		Timestamp: time.Now(),
		Data:      map[string]string{"message": "WebSocket connected successfully"},
	}
	connectedData, _ := json.Marshal(connectedMsg)
	if err := conn.WriteMessage(websocket.TextMessage, connectedData); err != nil {
		log.Printf("Error writing connected message: %v", err)
		return
	}

	// keep connection, handle heartbeat
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// handle client message (heartbeat, etc.)
		var clientMsg map[string]interface{}
		if json.Unmarshal(message, &clientMsg) == nil {
			if msgType, ok := clientMsg["type"].(string); ok && msgType == "ping" {
				// response heartbeat
				pongMsg := WsMessage{
					Type:      "pong",
					Timestamp: time.Now(),
					Data:      map[string]string{"message": "pong"},
				}
				pongData, _ := json.Marshal(pongMsg)
				if err := conn.WriteMessage(websocket.TextMessage, pongData); err != nil {
					log.Printf("Error writing pong message: %v", err)
					return
				}
			}
		}
	}

	log.Printf("WebSocket client disconnected, remaining clients: %d", Global.ClientCount())
}
