package client

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mycoool/gohook/internal/types"
	"golang.org/x/crypto/bcrypt"
)

// 全局会话存储（在生产环境中应该使用数据库或Redis）
var ClientSessions = make(map[string]*types.ClientSession)
var SessionIDCounter = 1
var SessionMutex sync.RWMutex

// addClientSession 添加客户端会话
func AddClientSession(token, name, username string) *types.ClientSession {
	SessionMutex.Lock()
	defer SessionMutex.Unlock()

	session := &types.ClientSession{
		ID:        SessionIDCounter,
		Token:     token,
		Name:      name,
		Username:  username,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}

	ClientSessions[token] = session
	SessionIDCounter++

	return session
}

// getClientSessionsByUser 获取用户的所有会话
func GetClientSessionsByUser(username string) []*types.ClientSession {
	SessionMutex.RLock()
	defer SessionMutex.RUnlock()

	var sessions []*types.ClientSession
	for _, session := range ClientSessions {
		if session.Username == username {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// removeClientSession 移除客户端会话
func RemoveClientSession(token string) bool {
	SessionMutex.Lock()
	defer SessionMutex.Unlock()

	if _, exists := ClientSessions[token]; exists {
		delete(ClientSessions, token)
		return true
	}

	return false
}

// updateSessionLastUsed 更新会话最后使用时间
func UpdateSessionLastUsed(token string) {
	SessionMutex.Lock()
	defer SessionMutex.Unlock()

	if session, exists := ClientSessions[token]; exists {
		session.LastUsed = time.Now()
	}
}

// hashPassword 对密码进行哈希
func HashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		// 如果bcrypt失败，回退到SHA256（不推荐，但确保系统可用）
		hash := sha256.Sum256([]byte(password))
		return hex.EncodeToString(hash[:])
	}
	return string(hashedPassword)
}

// verifyPassword 验证密码
func VerifyPassword(password, hashedPassword string) bool {
	// 首先尝试bcrypt验证
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == nil {
		return true
	}

	// 如果bcrypt失败，尝试SHA256验证（向后兼容）
	return HashPassword(password) == hashedPassword
}

// generateToken 生成JWT token
func GenerateToken(username, role string) (string, error) {
	expirationTime := time.Now().Add(time.Duration(types.GoHookAppConfig.JWTExpiryDuration) * time.Hour)
	claims := &types.Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(types.GoHookAppConfig.JWTSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// validateToken 验证JWT token
func ValidateToken(tokenString string) (*types.Claims, error) {
	claims := &types.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(types.GoHookAppConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
