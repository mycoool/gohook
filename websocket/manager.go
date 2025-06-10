package websocket

import (
	"encoding/json"
	"sync"
	"time"

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
