package client

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mycoool/gohook/internal/types"
	"golang.org/x/crypto/bcrypt"
)

// global session storage (should use database or Redis in production)
var ClientSessions = make(map[string]*types.ClientSession)
var SessionIDCounter = 1
var SessionMutex sync.RWMutex

// add client session
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

// get client sessions by user
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

// remove client session
func RemoveClientSession(token string) bool {
	SessionMutex.Lock()
	defer SessionMutex.Unlock()

	if _, exists := ClientSessions[token]; exists {
		delete(ClientSessions, token)
		return true
	}

	return false
}

// update session last used time
func UpdateSessionLastUsed(token string) {
	SessionMutex.Lock()
	defer SessionMutex.Unlock()

	if session, exists := ClientSessions[token]; exists {
		session.LastUsed = time.Now()
	}
}

// hash password
func HashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		// if bcrypt failed, fallback to SHA256 (not recommended, but ensure system available)
		hash := sha256.Sum256([]byte(password))
		return hex.EncodeToString(hash[:])
	}
	return string(hashedPassword)
}

// verify password
func VerifyPassword(password, hashedPassword string) bool {
	// first try bcrypt
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == nil {
		return true
	}

	// if bcrypt failed, try SHA256 (backward compatible)
	return HashPassword(password) == hashedPassword
}

// generate JWT token
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

// validate JWT token
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

func HandleDeleteClientSession(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
		return
	}

	SessionMutex.Lock()
	defer SessionMutex.Unlock()

	var tokenToDelete string
	for token, session := range ClientSessions {
		if session.ID == id {
			tokenToDelete = token
			break
		}
	}

	if tokenToDelete != "" {
		delete(ClientSessions, tokenToDelete)
		c.JSON(http.StatusOK, gin.H{"message": "Client session deleted"})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client session not found"})
	}
}

func HandleDeleteCurrentClientSession(c *gin.Context) {
	token := c.GetHeader("X-GoHook-Key")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token not provided"})
		return
	}

	if RemoveClientSession(token) {
		c.JSON(http.StatusOK, gin.H{"message": "Current session deleted successfully"})
	} else {
		// even if the session is not found, return success, because the client's goal is to logout
		c.JSON(http.StatusOK, gin.H{"message": "Session not found, but logout process can continue"})
	}
}

func HandleModifyCurrentClientPassword(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Password modification function not implemented",
	})
}

func HandleGetClientSessions(c *gin.Context) {
	username, _ := c.Get("username")
	currentToken, _ := c.Get("token")

	sessions := GetClientSessionsByUser(username.(string))

	// convert to frontend expected format
	var clients []gin.H
	for _, session := range sessions {
		// mark current session
		isCurrent := session.Token == currentToken.(string)

		clients = append(clients, gin.H{
			"id":       session.ID,
			"token":    session.Token,
			"name":     session.Name,
			"lastUsed": session.LastUsed.Format(time.RFC3339),
			"current":  isCurrent,
		})
	}

	c.JSON(http.StatusOK, clients)
}
