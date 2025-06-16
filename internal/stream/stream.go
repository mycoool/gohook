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

// WebSocket连接管理器
type Manager struct {
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
}

// 全局WebSocket管理器实例
var Global = &Manager{
	clients: make(map[*websocket.Conn]bool),
}

// WebSocket消息类型
type Message struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// Hook触发消息
type HookTriggeredMessage struct {
	HookID     string `json:"hookId"`
	HookName   string `json:"hookName"`
	Method     string `json:"method"`
	RemoteAddr string `json:"remoteAddr"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

// 版本切换消息
type VersionSwitchMessage struct {
	ProjectName string `json:"projectName"`
	Action      string `json:"action"` // "switch-branch" | "switch-tag"
	Target      string `json:"target"` // 分支名或标签名
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// 项目管理消息
type ProjectManageMessage struct {
	Action      string `json:"action"` // "add" | "delete"
	ProjectName string `json:"projectName"`
	ProjectPath string `json:"projectPath,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// 添加WebSocket连接
func (m *Manager) AddClient(conn *websocket.Conn) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	m.clients[conn] = true
}

// 移除WebSocket连接
func (m *Manager) RemoveClient(conn *websocket.Conn) {
	m.clientsMux.Lock()
	defer m.clientsMux.Unlock()
	delete(m.clients, conn)
}

// 广播消息到所有连接的客户端
func (m *Manager) Broadcast(message Message) {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	for client := range m.clients {
		err := client.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			// 连接已断开，移除客户端
			go func(conn *websocket.Conn) {
				m.RemoveClient(conn)
				conn.Close()
			}(client)
		}
	}
}

// 获取连接数量
func (m *Manager) ClientCount() int {
	m.clientsMux.RLock()
	defer m.clientsMux.RUnlock()
	return len(m.clients)
}

// handleWebSocket 处理WebSocket连接
func HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "WebSocket upgrade failed"})
		return
	}
	defer func() {
		// 从管理器中移除连接
		Global.RemoveClient(conn)
		conn.Close()
	}()

	// 将连接添加到全局管理器
	Global.AddClient(conn)
	log.Printf("WebSocket client connected, total clients: %d", Global.ClientCount())

	// 发送连接成功消息
	connectedMsg := Message{
		Type:      "connected",
		Timestamp: time.Now(),
		Data:      map[string]string{"message": "WebSocket connected successfully"},
	}
	connectedData, _ := json.Marshal(connectedMsg)
	if err := conn.WriteMessage(websocket.TextMessage, connectedData); err != nil {
		log.Printf("Error writing connected message: %v", err)
		return
	}

	// 保持连接，处理心跳
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// 处理客户端消息（心跳等）
		var clientMsg map[string]interface{}
		if json.Unmarshal(message, &clientMsg) == nil {
			if msgType, ok := clientMsg["type"].(string); ok && msgType == "ping" {
				// 响应心跳
				pongMsg := Message{
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
