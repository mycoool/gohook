package client

import (
	"sync"
	"time"

	"github.com/mycoool/gohook/internal/types"
)

// 全局会话存储（在生产环境中应该使用数据库或Redis）
var clientSessions = make(map[string]*types.ClientSession)
var sessionIDCounter = 1
var sessionMutex sync.RWMutex

// AddClientSession 添加客户端会话
func AddClientSession(token, name, username string) *types.ClientSession {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	session := &types.ClientSession{
		ID:        sessionIDCounter,
		Token:     token,
		Name:      name,
		Username:  username,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}

	clientSessions[token] = session
	sessionIDCounter++

	return session
}

// GetClientSessionsByUser 获取用户的所有会话
func GetClientSessionsByUser(username string) []*types.ClientSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	var sessions []*types.ClientSession
	for _, session := range clientSessions {
		if session.Username == username {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// RemoveClientSession 移除客户端会话
func RemoveClientSession(token string) bool {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	if _, exists := clientSessions[token]; exists {
		delete(clientSessions, token)
		return true
	}

	return false
}

// UpdateSessionLastUsed 更新会话最后使用时间
func UpdateSessionLastUsed(token string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	if session, exists := clientSessions[token]; exists {
		session.LastUsed = time.Now()
	}
}
