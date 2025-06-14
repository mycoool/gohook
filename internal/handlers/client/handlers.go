package client

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/types"
)

// 声明外部函数类型，避免循环导入
var FindUserFunc func(string) *types.UserConfig
var VerifyPasswordFunc func(string, string) bool
var GenerateTokenFunc func(string, string) (string, error)

// SetAuthFunctions 设置认证相关函数
func SetAuthFunctions(
	findUser func(string) *types.UserConfig,
	verifyPassword func(string, string) bool,
	generateToken func(string, string) (string, error),
) {
	FindUserFunc = findUser
	VerifyPasswordFunc = verifyPassword
	GenerateTokenFunc = generateToken
}

// LoginHandler 登录接口 - 支持Basic认证
func LoginHandler(c *gin.Context) {
	// 从Authorization头中获取Basic认证信息
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
		return
	}

	// 检查是否是Basic认证
	if !strings.HasPrefix(authHeader, "Basic ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization type"})
		return
	}

	// 解码Base64编码的用户名:密码
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization encoding"})
		return
	}

	// 分割用户名和密码
	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials format"})
		return
	}

	username := credentials[0]
	password := credentials[1]

	// 查找用户
	if FindUserFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication service not initialized"})
		return
	}
	user := FindUserFunc(username)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// 验证密码
	if VerifyPasswordFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication service not initialized"})
		return
	}
	if !VerifyPasswordFunc(password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// 生成JWT token
	if GenerateTokenFunc == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Authentication service not initialized"})
		return
	}
	token, err := GenerateTokenFunc(user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// 获取客户端名称（从请求体中的name字段）
	var requestBody struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&requestBody); err != nil {
		log.Printf("Warning: failed to parse request body: %v", err)
	}

	clientName := requestBody.Name
	if clientName == "" {
		clientName = "unknown client"
	}

	// 创建客户端会话记录
	session := AddClientSession(token, clientName, user.Username)

	c.JSON(http.StatusOK, types.ClientResponse{
		Token: token,
		ID:    session.ID,
		Name:  clientName,
	})
}

// GetCurrentUserHandler 获取当前用户信息接口
func GetCurrentUserHandler(c *gin.Context) {
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, gin.H{
		"id":       1,
		"name":     username,
		"username": username,
		"role":     role,
		"admin":    role == "admin",
	})
}

// GetClientListHandler 客户端列表API (获取当前用户的所有会话)
func GetClientListHandler(c *gin.Context) {
	username, _ := c.Get("username")
	currentToken, _ := c.Get("token")

	sessions := GetClientSessionsByUser(username.(string))

	// 转换为前端期望的格式
	var clients []gin.H
	for _, session := range sessions {
		// 标记当前会话
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

// DeleteClientHandler 删除客户端API (注销指定会话)
func DeleteClientHandler(c *gin.Context) {
	id := c.Param("id")
	username, _ := c.Get("username")
	currentToken, _ := c.Get("token")

	// 查找要删除的会话
	sessions := GetClientSessionsByUser(username.(string))
	var targetToken string
	var targetSession *types.ClientSession

	for _, session := range sessions {
		if fmt.Sprintf("%d", session.ID) == id {
			targetToken = session.Token
			targetSession = session
			break
		}
	}

	if targetSession == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话未找到"})
		return
	}

	// 不能删除当前会话
	if targetToken == currentToken.(string) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除当前会话，请使用注销功能"})
		return
	}

	// 删除会话
	if RemoveClientSession(targetToken) {
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("会话 '%s' 已被注销", targetSession.Name),
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除会话失败"})
	}
}
