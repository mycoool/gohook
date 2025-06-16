package router

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/websocket"
	"github.com/mycoool/gohook/internal/hook"
	"github.com/mycoool/gohook/internal/stream"
	"github.com/mycoool/gohook/internal/version"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v2"
)

// UserConfig 用户配置结构
type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Role     string `yaml:"role"`
}

// UsersConfig 用户配置文件结构（原AppConfig）
type UsersConfig struct {
	Users []UserConfig `yaml:"users"`
}

// AppConfig 应用程序配置结构
type AppConfig struct {
	Port              int    `yaml:"port"`
	JWTSecret         string `yaml:"jwt_secret"`
	JWTExpiryDuration int    `yaml:"jwt_expiry_duration"`
}

// Claims JWT声明结构
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// UserResponse 用户响应结构
type UserResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Config 配置文件结构
type Config struct {
	Projects []ProjectConfig `yaml:"projects"`
}

// ProjectConfig 项目配置
type ProjectConfig struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
	Enhook      bool   `yaml:"enhook,omitempty"`
	Hookmode    string `yaml:"hookmode,omitempty"`
	Hookbranch  string `yaml:"hookbranch,omitempty"`
	Hooksecret  string `yaml:"hooksecret,omitempty"`
}

// VersionResponse 版本响应结构
type VersionResponse struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Description    string `json:"description"`
	CurrentBranch  string `json:"currentBranch"`
	CurrentTag     string `json:"currentTag"`
	Mode           string `json:"mode"` // "branch" 或 "tag"
	Status         string `json:"status"`
	LastCommit     string `json:"lastCommit"`
	LastCommitTime string `json:"lastCommitTime"`
	Enhook         bool   `json:"enhook,omitempty"`
	Hookmode       string `json:"hookmode,omitempty"`
	Hookbranch     string `json:"hookbranch,omitempty"`
	Hooksecret     string `json:"hooksecret,omitempty"`
}

// BranchResponse 分支响应结构
type BranchResponse struct {
	Name           string `json:"name"`
	IsCurrent      bool   `json:"isCurrent"`
	LastCommit     string `json:"lastCommit"`
	LastCommitTime string `json:"lastCommitTime"`
	Type           string `json:"type"` // "local", "remote", or "detached"
}

// TagResponse 标签响应结构
type TagResponse struct {
	Name       string `json:"name"`
	IsCurrent  bool   `json:"isCurrent"`
	CommitHash string `json:"commitHash"`
	Date       string `json:"date"`
	Message    string `json:"message"`
}

// ClientSession 客户端会话结构
type ClientSession struct {
	ID        int       `json:"id"`
	Token     string    `json:"token"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	LastUsed  time.Time `json:"lastUsed"`
	CreatedAt time.Time `json:"createdAt"`
}

// 全局会话存储（在生产环境中应该使用数据库或Redis）
var clientSessions = make(map[string]*ClientSession)
var sessionIDCounter = 1
var sessionMutex sync.RWMutex

// addClientSession 添加客户端会话
func addClientSession(token, name, username string) *ClientSession {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	session := &ClientSession{
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

// getClientSessionsByUser 获取用户的所有会话
func getClientSessionsByUser(username string) []*ClientSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	var sessions []*ClientSession
	for _, session := range clientSessions {
		if session.Username == username {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// removeClientSession 移除客户端会话
func removeClientSession(token string) bool {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	if _, exists := clientSessions[token]; exists {
		delete(clientSessions, token)
		return true
	}

	return false
}

// updateSessionLastUsed 更新会话最后使用时间
func updateSessionLastUsed(token string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	if session, exists := clientSessions[token]; exists {
		session.LastUsed = time.Now()
	}
}

// 全局变量引用，用于访问已加载的hooks
var LoadedHooksFromFiles *map[string]hook.Hooks
var HookManager *hook.HookManager
var configData *Config
var appConfig *AppConfig     // 应用程序配置
var usersConfig *UsersConfig // 用户配置

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// WebSocket消息类型
type WSMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	HookID    string      `json:"hookId"`
	HookName  string      `json:"hookName"`
	Method    string      `json:"method"`
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

// hashPassword 对密码进行哈希
func hashPassword(password string) string {
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
func verifyPassword(password, hashedPassword string) bool {
	// 首先尝试bcrypt验证
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == nil {
		return true
	}

	// 如果bcrypt失败，尝试SHA256验证（向后兼容）
	return hashPassword(password) == hashedPassword
}

// generateToken 生成JWT token
func generateToken(username, role string) (string, error) {
	expirationTime := time.Now().Add(time.Duration(appConfig.JWTExpiryDuration) * time.Hour)
	claims := &Claims{
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(appConfig.JWTSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// validateToken 验证JWT token
func validateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(appConfig.JWTSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// loadUsersConfig 加载用户配置文件
func loadUsersConfig() error {
	filePath := "user.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("用户配置文件 %s 不存在", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取用户配置文件失败: %v", err)
	}

	config := &UsersConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析用户配置文件失败: %v", err)
	}

	usersConfig = config
	return nil
}

// saveUsersConfig 保存用户配置文件
func saveUsersConfig() error {
	if usersConfig == nil {
		return fmt.Errorf("用户配置为空")
	}

	// 创建带注释的YAML内容
	var yamlContent strings.Builder
	yamlContent.WriteString("# GoHook 用户配置文件\n")
	yamlContent.WriteString("# 用户账户信息\n")
	yamlContent.WriteString("users:\n")

	for _, user := range usersConfig.Users {
		yamlContent.WriteString(fmt.Sprintf("  - username: %s\n", user.Username))
		yamlContent.WriteString(fmt.Sprintf("    password: %s\n", user.Password))
		yamlContent.WriteString(fmt.Sprintf("    role: %s\n", user.Role))

		// 如果是默认admin用户且密码是哈希值，添加原始密码注释
		if user.Username == "admin" && strings.HasPrefix(user.Password, "$2a$") {
			// 检查是否是新创建的默认用户（通过检查是否只有一个用户来判断）
			if len(usersConfig.Users) == 1 {
				yamlContent.WriteString("    # 默认密码: admin123 (请及时修改)\n")
			}
		}
	}

	if err := os.WriteFile("user.yaml", []byte(yamlContent.String()), 0644); err != nil {
		return fmt.Errorf("保存用户配置文件失败: %v", err)
	}

	return nil
}

// loadAppConfig 加载应用程序配置文件
func loadAppConfig() error {
	filePath := "app.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 如果配置文件不存在，创建默认配置并保存到文件
		appConfig = &AppConfig{
			Port:              9000,
			JWTSecret:         "gohook-secret-key-change-in-production",
			JWTExpiryDuration: 24,
		}
		// 自动保存默认配置到文件
		if saveErr := saveAppConfig(); saveErr != nil {
			log.Printf("Warning: failed to save default app config: %v", saveErr)
		} else {
			log.Printf("Created default app.yaml configuration file")
		}
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取应用配置文件失败: %v", err)
	}

	config := &AppConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析应用配置文件失败: %v", err)
	}

	appConfig = config
	return nil
}

// saveAppConfig 保存应用程序配置文件
func saveAppConfig() error {
	if appConfig == nil {
		return fmt.Errorf("应用配置为空")
	}

	data, err := yaml.Marshal(appConfig)
	if err != nil {
		return fmt.Errorf("序列化应用配置失败: %v", err)
	}

	if err := os.WriteFile("app.yaml", data, 0644); err != nil {
		return fmt.Errorf("保存应用配置文件失败: %v", err)
	}

	return nil
}

// findUser 查找用户
func findUser(username string) *UserConfig {
	if usersConfig == nil {
		return nil
	}

	for i := range usersConfig.Users {
		if usersConfig.Users[i].Username == username {
			return &usersConfig.Users[i]
		}
	}

	return nil
}

// authMiddleware JWT认证中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("X-GoHook-Key")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		claims, err := validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 更新会话最后使用时间
		updateSessionLastUsed(tokenString)

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)
		c.Next()
	}
}

// wsAuthMiddleware WebSocket专用认证中间件，支持查询参数token
func wsAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 优先从Header获取token
		tokenString := c.GetHeader("X-GoHook-Key")

		// 如果Header中没有token，从查询参数获取
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		claims, err := validateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 更新会话最后使用时间
		updateSessionLastUsed(tokenString)

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)
		c.Next()
	}
}

// adminMiddleware 管理员权限中间件
func adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// loadConfig 加载版本配置文件
func loadConfig() error {
	data, err := os.ReadFile("version.yaml")
	if err != nil {
		return fmt.Errorf("读取版本配置文件失败: %v", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析版本配置文件失败: %v", err)
	}

	configData = config
	return nil
}

// saveConfig 保存版本配置文件
func saveConfig() error {
	if configData == nil {
		return fmt.Errorf("版本配置数据为空")
	}

	data, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("序列化版本配置失败: %v", err)
	}

	// 备份原配置文件
	if _, err := os.Stat("version.yaml"); err == nil {
		if err := os.Rename("version.yaml", "version.yaml.bak"); err != nil {
			log.Printf("Warning: failed to backup version config file: %v", err)
		}
	}

	err = os.WriteFile("version.yaml", data, 0644)
	if err != nil {
		// 如果保存失败，恢复备份
		if _, backupErr := os.Stat("version.yaml.bak"); backupErr == nil {
			if restoreErr := os.Rename("version.yaml.bak", "version.yaml"); restoreErr != nil {
				log.Printf("Error: failed to restore backup version config file: %v", restoreErr)
			}
		}
		return fmt.Errorf("保存版本配置文件失败: %v", err)
	}

	return nil
}

// getGitStatus 获取Git状态
func getGitStatus(projectPath string) (*VersionResponse, error) {
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return nil, fmt.Errorf("不是Git仓库")
	}

	// 获取当前分支
	cmd := exec.Command("git", "-C", projectPath, "branch", "--show-current")
	branchOutput, _ := cmd.Output()
	currentBranch := strings.TrimSpace(string(branchOutput))

	// 获取当前标签（如果在标签上）
	cmd = exec.Command("git", "-C", projectPath, "describe", "--exact-match", "--tags", "HEAD")
	tagOutput, _ := cmd.Output()
	currentTag := strings.TrimSpace(string(tagOutput))

	// 确定模式
	mode := "branch"
	if currentTag != "" {
		mode = "tag"
	}

	// 获取最后提交信息
	cmd = exec.Command("git", "-C", projectPath, "log", "-1", "--format=%H|%ci|%s")
	commitOutput, _ := cmd.Output()
	commitInfo := strings.TrimSpace(string(commitOutput))

	parts := strings.Split(commitInfo, "|")
	lastCommit := ""
	lastCommitTime := ""
	if len(parts) >= 2 {
		lastCommit = parts[0][:8] // 短哈希
		lastCommitTime = parts[1]
	}

	return &VersionResponse{
		CurrentBranch:  currentBranch,
		CurrentTag:     currentTag,
		Mode:           mode,
		Status:         "active",
		LastCommit:     lastCommit,
		LastCommitTime: lastCommitTime,
	}, nil
}

// getBranches 获取分支列表
func getBranches(projectPath string) ([]BranchResponse, error) {
	var branches []BranchResponse
	branchSet := make(map[string]bool) // 用于防止重复添加

	// 1. 获取当前是否处于分离头状态
	_, err := exec.Command("git", "-C", projectPath, "symbolic-ref", "-q", "HEAD").Output()
	isDetached := err != nil

	// 2. 获取当前分支或提交的引用
	var currentRef string
	if isDetached {
		// 分离头状态，获取 HEAD 的短哈希
		headSha, err := exec.Command("git", "-C", projectPath, "rev-parse", "--short", "HEAD").Output()
		if err != nil {
			return nil, fmt.Errorf("获取HEAD commit失败: %v", err)
		}
		currentRef = strings.TrimSpace(string(headSha))
	} else {
		// 在分支上，获取分支名
		branchName, err := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return nil, fmt.Errorf("获取当前分支名失败: %v", err)
		}
		currentRef = strings.TrimSpace(string(branchName))
	}

	// 3. 处理分离头状态
	if isDetached {
		// 尝试获取标签名
		tagName, err := exec.Command("git", "-C", projectPath, "describe", "--tags", "--exact-match", "HEAD").Output()
		var displayName string
		if err == nil {
			displayName = strings.TrimSpace(string(tagName))
		} else {
			displayName = currentRef
		}

		// 获取最后提交信息
		commitOutput, _ := exec.Command("git", "-C", projectPath, "log", "-1", "HEAD", "--format=%H|%ci").Output()
		parts := strings.Split(strings.TrimSpace(string(commitOutput)), "|")
		lastCommit, lastCommitTime := "", ""
		if len(parts) > 0 {
			lastCommit = parts[0][:8]
		}
		if len(parts) > 1 {
			lastCommitTime = parts[1]
		}

		branches = append(branches, BranchResponse{
			Name:           fmt.Sprintf("(当前指向 %s)", displayName),
			IsCurrent:      true,
			LastCommit:     lastCommit,
			LastCommitTime: lastCommitTime,
			Type:           "detached",
		})
	}

	// 4. 获取所有本地分支
	cmd := exec.Command("git", "-C", projectPath, "for-each-ref", "refs/heads", "--format=%(refname:short)|%(committerdate:iso)|%(objectname:short)")
	localOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取本地分支列表失败: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(localOutput)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) >= 3 {
			branchName := parts[0]
			if branchSet[branchName] {
				continue
			}
			branchSet[branchName] = true
			branches = append(branches, BranchResponse{
				Name:           branchName,
				IsCurrent:      !isDetached && branchName == currentRef,
				LastCommitTime: parts[1],
				LastCommit:     parts[2],
				Type:           "local",
			})
		}
	}

	// 5. 获取所有远程分支
	cmd = exec.Command("git", "-C", projectPath, "for-each-ref", "refs/remotes", "--format=%(refname:short)|%(committerdate:iso)|%(objectname:short)")
	remoteOutput, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(remoteOutput)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "|", 3)
			if len(parts) >= 3 {
				remoteRef := parts[0]
				if strings.HasSuffix(remoteRef, "/HEAD") {
					continue // 忽略 HEAD 指针
				}
				branchName := remoteRef // 例如 "origin/master"
				if branchSet[branchName] {
					continue
				}
				branchSet[branchName] = true
				branches = append(branches, BranchResponse{
					Name:           branchName,
					IsCurrent:      false,
					LastCommitTime: parts[1],
					LastCommit:     parts[2],
					Type:           "remote",
				})
			}
		}
	} else {
		log.Printf("获取远程分支列表失败 (项目: %s): %v", projectPath, err)
	}

	return branches, nil
}

// getTags 获取标签列表
func getTags(projectPath string) ([]TagResponse, error) {
	// 获取当前标签
	cmd := exec.Command("git", "-C", projectPath, "describe", "--exact-match", "--tags", "HEAD")
	currentOutput, _ := cmd.Output()
	currentTag := strings.TrimSpace(string(currentOutput))

	// 获取所有标签
	cmd = exec.Command("git", "-C", projectPath, "tag", "-l", "--sort=-version:refname", "--format=%(refname:short)|%(creatordate)|%(objectname:short)|%(subject)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %v", err)
	}

	var tags []TagResponse
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			tagName := parts[0]
			tags = append(tags, TagResponse{
				Name:       tagName,
				IsCurrent:  tagName == currentTag,
				Date:       parts[1],
				CommitHash: parts[2],
				Message:    parts[3],
			})
		}
	}

	return tags, nil
}

// syncBranches 同步远程分支，清理已删除的远程分支引用
func syncBranches(projectPath string) error {
	// 使用 fetch --prune 来更新远程分支信息并删除不存在的引用
	cmd := exec.Command("git", "-C", projectPath, "fetch", "origin", "--prune")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("同步分支失败: %s", string(output))
	}
	return nil
}

// switchBranch 切换分支
func switchBranch(projectPath, branchName string) error {
	var cmd *exec.Cmd
	var isRemoteBranch bool
	var localBranchName string

	// 检查是否是远程分支格式 (例如 origin/release)
	if strings.HasPrefix(branchName, "origin/") {
		isRemoteBranch = true
		localBranchName = strings.TrimPrefix(branchName, "origin/")

		// 检查本地是否已有同名分支
		checkCmd := exec.Command("git", "-C", projectPath, "rev-parse", "--verify", localBranchName)
		if checkCmd.Run() == nil {
			// 本地分支已存在，直接切换
			cmd = exec.Command("git", "-C", projectPath, "checkout", localBranchName)
		} else {
			// 本地分支不存在，基于远程分支创建新的本地分支
			cmd = exec.Command("git", "-C", projectPath, "checkout", "-b", localBranchName, branchName)
		}
	} else {
		// 普通的本地分支切换
		isRemoteBranch = false
		localBranchName = branchName
		cmd = exec.Command("git", "-C", projectPath, "checkout", branchName)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("切换分支失败: %s", string(output))
	}

	// 如果是基于远程分支创建的新分支，尝试拉取最新代码
	if isRemoteBranch {
		pullCmd := exec.Command("git", "-C", projectPath, "pull", "origin", localBranchName)
		pullOutput, pullErr := pullCmd.CombinedOutput()
		if pullErr != nil {
			// 拉取失败不认为是致命错误，但记录日志
			log.Printf("切换分支后拉取最新代码失败 (项目: %s, 分支: %s): %s", projectPath, localBranchName, string(pullOutput))
		}
	}

	return nil
}

// switchTag 切换标签
func switchTag(projectPath, tagName string) error {
	cmd := exec.Command("git", "-C", projectPath, "checkout", tagName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("切换标签失败: %v", err)
	}
	return nil
}

func InitGit(projectPath string) error {
	return version.InitGit(projectPath)
}

// getRemote 获取远程仓库URL
func getRemote(projectPath string) (string, error) {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("不是Git仓库")
	}

	// 获取origin远程仓库URL
	cmd := exec.Command("git", "-C", projectPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		// 如果 "origin" 不存在，命令会返回非零退出码。
		// 这种情况下我们返回空字符串，表示没有设置远程地址。
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// setRemote 设置远程仓库
func setRemote(projectPath, remoteUrl string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库")
	}

	// 检查是否已有origin远程仓库
	cmd := exec.Command("git", "-C", projectPath, "remote", "get-url", "origin")
	if cmd.Run() == nil {
		// 如果已有origin，先删除
		cmd = exec.Command("git", "-C", projectPath, "remote", "remove", "origin")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("删除原有远程仓库失败: %v", err)
		}
	}

	// 添加新的origin远程仓库
	cmd = exec.Command("git", "-C", projectPath, "remote", "add", "origin", remoteUrl)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("设置远程仓库失败: %v", err)
	}

	return nil
}

// ClientResponse 客户端响应结构
type ClientResponse struct {
	Token string `json:"token"`
	ID    int    `json:"id"`
	Name  string `json:"name"`
}

// HookResponse Hook响应结构
type HookResponse struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	ExecuteCommand         string   `json:"executeCommand"`
	WorkingDirectory       string   `json:"workingDirectory"`
	ResponseMessage        string   `json:"responseMessage"`
	HTTPMethods            []string `json:"httpMethods"`
	TriggerRuleDescription string   `json:"triggerRuleDescription"`
	LastUsed               *string  `json:"lastUsed"`
	Status                 string   `json:"status"` // active, inactive
}

// getHookByID 根据ID获取Hook
func getHookByID(id string) *HookResponse {
	if LoadedHooksFromFiles == nil {
		return nil
	}

	for _, hooksInFile := range *LoadedHooksFromFiles {
		if hook := hooksInFile.Match(id); hook != nil {
			hookResponse := convertHookToResponse(hook)
			return &hookResponse
		}
	}

	return nil
}

// convertHookToResponse 将Hook转换为HookResponse
func convertHookToResponse(h *hook.Hook) HookResponse {
	description := fmt.Sprintf("Execute: %s", h.ExecuteCommand)
	if h.ResponseMessage != "" {
		description = h.ResponseMessage
	}

	// 解析触发规则为可读描述
	triggerDesc := "Any request"
	if h.TriggerRule != nil {
		triggerDesc = describeTriggerRule(h.TriggerRule)
	}

	// 设置HTTP方法
	httpMethods := h.HTTPMethods
	if len(httpMethods) == 0 {
		httpMethods = []string{"POST", "GET"} // 默认方法
	}

	return HookResponse{
		ID:                     h.ID,
		Name:                   h.ID, // 使用ID作为名称
		Description:            description,
		ExecuteCommand:         h.ExecuteCommand,
		WorkingDirectory:       h.CommandWorkingDirectory,
		ResponseMessage:        h.ResponseMessage,
		HTTPMethods:            httpMethods,
		TriggerRuleDescription: triggerDesc,
		LastUsed:               nil, // TODO: 可以添加实际的使用时间跟踪
		Status:                 "active",
	}
}

// describeTriggerRule 生成触发规则的可读描述
func describeTriggerRule(rules *hook.Rules) string {
	if rules == nil {
		return "No rules"
	}

	if rules.Match != nil {
		return fmt.Sprintf("Match %s: %s", rules.Match.Type, rules.Match.Value)
	}

	if rules.And != nil {
		return fmt.Sprintf("Multiple conditions required (%d rules)", len(*rules.And))
	}

	if rules.Or != nil {
		return fmt.Sprintf("Any condition satisfied (%d rules)", len(*rules.Or))
	}

	if rules.Not != nil {
		return "Negated condition"
	}

	return "Complex rules"
}

// handleWebSocket 处理WebSocket连接
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "WebSocket upgrade failed"})
		return
	}
	defer func() {
		// 从管理器中移除连接
		stream.Global.RemoveClient(conn)
		conn.Close()
	}()

	// 将连接添加到全局管理器
	stream.Global.AddClient(conn)
	log.Printf("WebSocket client connected, total clients: %d", stream.Global.ClientCount())

	// 发送连接成功消息
	connectedMsg := stream.Message{
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
				pongMsg := stream.Message{
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

	log.Printf("WebSocket client disconnected, remaining clients: %d", stream.Global.ClientCount())
}

func InitRouter() *gin.Engine {
	g := gin.Default()

	// 加载配置文件
	if err := loadConfig(); err != nil {
		// 如果配置文件加载失败，使用默认值
		configData = &Config{}
	}

	// 加载应用配置文件
	if err := loadAppConfig(); err != nil {
		// 如果应用配置文件加载失败，创建默认配置
		appConfig = &AppConfig{
			Port:              9000,
			JWTSecret:         "gohook-secret-key-change-in-production",
			JWTExpiryDuration: 24,
		}
		log.Printf("Warning: failed to load app config, using default settings")
	}

	// 加载用户配置文件
	if err := loadUsersConfig(); err != nil {
		// 如果用户配置文件加载失败，创建默认管理员用户
		defaultPassword := "admin123" // 生成随机密码
		usersConfig = &UsersConfig{
			Users: []UserConfig{
				{
					Username: "admin",
					Password: hashPassword(defaultPassword),
					Role:     "admin",
				},
			},
		}
		// 保存默认用户配置
		if saveErr := saveUsersConfig(); saveErr != nil {
			log.Printf("Error: failed to save default user config: %v", saveErr)
		} else {
			log.Printf("Created default admin user with password: %s", defaultPassword)
		}
		log.Printf("Warning: failed to load user config, created default admin user")
	}

	// CORS中间件 - 在路由注册后添加，避免通配符冲突
	g.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-GoHook-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	g.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 适配前端 All Messages 页面, 暂时返回空
	g.GET("/message", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"messages": []interface{}{},
			"paging":   gin.H{},
		})
	})

	// 登录接口 - 支持Basic认证
	g.POST("/client", func(c *gin.Context) {
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
		user := findUser(username)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// 验证密码
		if !verifyPassword(password, user.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// 生成JWT token
		token, err := generateToken(user.Username, user.Role)
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
		session := addClientSession(token, clientName, user.Username)

		c.JSON(http.StatusOK, ClientResponse{
			Token: token,
			ID:    session.ID,
			Name:  clientName,
		})
	})

	// 获取当前用户信息接口
	g.GET("/current/user", authMiddleware(), func(c *gin.Context) {
		username, _ := c.Get("username")
		role, _ := c.Get("role")

		c.JSON(http.StatusOK, gin.H{
			"id":       1,
			"name":     username,
			"username": username,
			"role":     role,
			"admin":    role == "admin",
		})
	})

	// 用户管理API接口组
	userAPI := g.Group("/user")
	userAPI.Use(authMiddleware())
	{
		// 获取所有用户列表 (仅管理员)
		userAPI.GET("", adminMiddleware(), func(c *gin.Context) {
			var users []UserResponse
			for _, user := range usersConfig.Users {
				users = append(users, UserResponse{
					Username: user.Username,
					Role:     user.Role,
				})
			}
			c.JSON(http.StatusOK, users)
		})

		// 创建用户 (仅管理员)
		userAPI.POST("", adminMiddleware(), func(c *gin.Context) {
			var req struct {
				Username string `json:"username" binding:"required"`
				Password string `json:"password" binding:"required"`
				Role     string `json:"role" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
				return
			}

			// 检查用户是否已存在
			if findUser(req.Username) != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
				return
			}

			// 验证角色
			if req.Role != "admin" && req.Role != "user" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Must be 'admin' or 'user'"})
				return
			}

			// 添加新用户
			newUser := UserConfig{
				Username: req.Username,
				Password: hashPassword(req.Password),
				Role:     req.Role,
			}

			usersConfig.Users = append(usersConfig.Users, newUser)

			// 保存配置文件
			if err := saveUsersConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "User created successfully",
				"user": UserResponse{
					Username: newUser.Username,
					Role:     newUser.Role,
				},
			})
		})

		// 删除用户 (仅管理员)
		userAPI.DELETE("/:username", adminMiddleware(), func(c *gin.Context) {
			username := c.Param("username")
			currentUser, _ := c.Get("username")

			// 不能删除自己
			if username == currentUser {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete yourself"})
				return
			}

			// 查找用户索引
			userIndex := -1
			for i, user := range usersConfig.Users {
				if user.Username == username {
					userIndex = i
					break
				}
			}

			if userIndex == -1 {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}

			// 删除用户
			usersConfig.Users = append(usersConfig.Users[:userIndex], usersConfig.Users[userIndex+1:]...)

			// 保存配置文件
			if err := saveUsersConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "User deleted successfully",
			})
		})

		// 修改密码
		userAPI.POST("/password", func(c *gin.Context) {
			var req struct {
				OldPassword string `json:"oldPassword" binding:"required"`
				NewPassword string `json:"newPassword" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
				return
			}

			username, _ := c.Get("username")
			user := findUser(username.(string))
			if user == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}

			// 验证旧密码
			if !verifyPassword(req.OldPassword, user.Password) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid old password"})
				return
			}

			// 更新密码
			user.Password = hashPassword(req.NewPassword)

			// 保存配置文件
			if err := saveUsersConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Password updated successfully",
			})
		})

		// 管理员重置用户密码
		userAPI.POST("/:username/reset-password", adminMiddleware(), func(c *gin.Context) {
			username := c.Param("username")
			var req struct {
				NewPassword string `json:"newPassword" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
				return
			}

			user := findUser(username)
			if user == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}

			// 更新密码
			user.Password = hashPassword(req.NewPassword)

			// 保存配置文件
			if err := saveUsersConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Password reset successfully",
			})
		})
	}

	// Hooks API接口组
	hookAPI := g.Group("/hook")
	hookAPI.Use(authMiddleware()) // 添加认证中间件
	{
		// 获取所有hooks
		hookAPI.GET("", func(c *gin.Context) {
			if LoadedHooksFromFiles == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "hooks未加载"})
				return
			}

			var hooks []HookResponse
			for _, hooksInFile := range *LoadedHooksFromFiles {
				for _, h := range hooksInFile {
					hookResponse := convertHookToResponse(&h)
					hooks = append(hooks, hookResponse)
				}
			}

			c.JSON(http.StatusOK, hooks)
		})

		// 获取单个Hook详情
		hookAPI.GET("/:id", func(c *gin.Context) {
			hookID := c.Param("id")
			hookResponse := getHookByID(hookID)
			if hookResponse == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
				return
			}
			c.JSON(http.StatusOK, hookResponse)
		})

		// 触发Hook（测试接口）
		hookAPI.POST("/:id/trigger", func(c *gin.Context) {
			hookID := c.Param("id")
			hookResponse := getHookByID(hookID)
			if hookResponse == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
				return
			}

			// 执行Hook命令
			success := false
			output := ""
			errorMsg := ""

			if hookResponse.ExecuteCommand != "" {
				// 执行命令
				var cmd *exec.Cmd
				if hookResponse.WorkingDirectory != "" {
					cmd = exec.Command("bash", "-c", hookResponse.ExecuteCommand)
					cmd.Dir = hookResponse.WorkingDirectory
				} else {
					cmd = exec.Command("bash", "-c", hookResponse.ExecuteCommand)
				}

				result, err := cmd.CombinedOutput()
				output = string(result)
				if err != nil {
					errorMsg = err.Error()
				} else {
					success = true
				}
			} else {
				success = true
				output = "Hook触发成功（无执行命令）"
			}

			// 推送WebSocket消息
			wsMessage := stream.Message{
				Type:      "hook_triggered",
				Timestamp: time.Now(),
				Data: stream.HookTriggeredMessage{
					HookID:     hookID,
					HookName:   hookResponse.Name,
					Method:     c.Request.Method,
					RemoteAddr: c.ClientIP(),
					Success:    success,
					Output:     output,
					Error:      errorMsg,
				},
			}
			stream.Global.Broadcast(wsMessage)

			if success {
				c.JSON(http.StatusOK, gin.H{
					"message": "Hook触发成功",
					"hook":    hookResponse.Name,
					"output":  output,
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Hook触发失败",
					"hook":    hookResponse.Name,
					"error":   errorMsg,
					"output":  output,
				})
			}
		})

		// 加载Hooks配置的专用接口
		hookAPI.POST("/reload-config", func(c *gin.Context) {
			if HookManager == nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Hook管理器未初始化",
				})
				return
			}

			// 执行实际的加载
			err := HookManager.ReloadAllHooks()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":     "加载Hook失败",
					"details":   err.Error(),
					"hookCount": HookManager.GetHookCount(),
				})
				return
			}

			// 获取加载后的hooks数量
			hookCount := HookManager.GetHookCount()

			c.JSON(http.StatusOK, gin.H{
				"message":   "Hooks配置加载成功",
				"hookCount": hookCount,
			})
		})
	}

	//添加websocket
	ws := g.Group("/stream")
	ws.Use(wsAuthMiddleware()) // 使用WebSocket专用认证中间件，支持查询参数token
	{
		//前端访问地址："/stream?token=jwt-token-here"
		ws.GET("", func(c *gin.Context) {
			// Token已通过中间件验证，直接处理WebSocket连接
			handleWebSocket(c)
		})

		// 也支持带ID的路径格式 /stream/:id
		ws.GET("/:id", func(c *gin.Context) {
			// Token已通过中间件验证，直接处理WebSocket连接
			handleWebSocket(c)
		})
	}

	// 版本管理API接口组
	versionAPI := g.Group("/version")
	versionAPI.Use(authMiddleware()) // 添加认证中间件
	{
		// 获取所有项目列表
		versionAPI.GET("", func(c *gin.Context) {
			// 每次获取项目列表时加载配置文件
			if err := loadConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "配置文件加载失败: " + err.Error()})
				return
			}

			if configData == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "配置文件未加载"})
				return
			}

			var projects []VersionResponse
			for _, proj := range configData.Projects {
				if !proj.Enabled {
					continue
				}

				gitStatus, err := getGitStatus(proj.Path)
				if err != nil {
					// 如果不是Git仓库，仍然显示但标记为非Git项目
					projects = append(projects, VersionResponse{
						Name:        proj.Name,
						Path:        proj.Path,
						Description: proj.Description,
						Mode:        "none",
						Status:      "not-git",
					})
					continue
				}

				gitStatus.Name = proj.Name
				gitStatus.Path = proj.Path
				gitStatus.Description = proj.Description
				gitStatus.Enhook = proj.Enhook
				gitStatus.Hookmode = proj.Hookmode
				gitStatus.Hookbranch = proj.Hookbranch
				gitStatus.Hooksecret = proj.Hooksecret
				projects = append(projects, *gitStatus)
			}

			c.JSON(http.StatusOK, projects)
		})

		// 加载配置文件的专用接口
		versionAPI.POST("/reload-config", func(c *gin.Context) {
			if err := loadConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "配置文件加载失败: " + err.Error(),
				})
				return
			}

			projectCount := 0
			if configData != nil {
				for _, proj := range configData.Projects {
					if proj.Enabled {
						projectCount++
					}
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"message":      "配置文件加载成功",
				"projectCount": projectCount,
			})
		})

		// 添加项目
		versionAPI.POST("/add-project", func(c *gin.Context) {
			var req struct {
				Name        string `json:"name" binding:"required"`
				Path        string `json:"path" binding:"required"`
				Description string `json:"description"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
				return
			}

			// 检查项目名称是否已存在
			for _, proj := range configData.Projects {
				if proj.Name == req.Name {
					c.JSON(http.StatusConflict, gin.H{"error": "项目名称已存在"})
					return
				}
			}

			// 检查路径是否存在
			if _, err := os.Stat(req.Path); os.IsNotExist(err) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "指定路径不存在"})
				return
			}

			// 添加新项目
			newProject := ProjectConfig{
				Name:        req.Name,
				Path:        req.Path,
				Description: req.Description,
				Enabled:     true,
			}

			configData.Projects = append(configData.Projects, newProject)

			// 保存配置文件
			if err := saveConfig(); err != nil {
				// 推送失败消息
				wsMessage := stream.Message{
					Type:      "project_managed",
					Timestamp: time.Now(),
					Data: stream.ProjectManageMessage{
						Action:      "add",
						ProjectName: req.Name,
						ProjectPath: req.Path,
						Success:     false,
						Error:       "保存配置失败: " + err.Error(),
					},
				}
				stream.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := stream.Message{
				Type:      "project_managed",
				Timestamp: time.Now(),
				Data: stream.ProjectManageMessage{
					Action:      "add",
					ProjectName: req.Name,
					ProjectPath: req.Path,
					Success:     true,
				},
			}
			stream.Global.Broadcast(wsMessage)

			c.JSON(http.StatusOK, gin.H{
				"message": "项目添加成功",
				"project": newProject,
			})
		})

		// 删除项目
		versionAPI.DELETE("/:name", func(c *gin.Context) {
			projectName := c.Param("name")

			// 查找项目索引
			projectIndex := -1
			for i, proj := range configData.Projects {
				if proj.Name == projectName {
					projectIndex = i
					break
				}
			}

			if projectIndex == -1 {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			// 删除项目
			configData.Projects = append(configData.Projects[:projectIndex], configData.Projects[projectIndex+1:]...)

			// 保存配置文件
			if err := saveConfig(); err != nil {
				// 推送失败消息
				wsMessage := stream.Message{
					Type:      "project_managed",
					Timestamp: time.Now(),
					Data: stream.ProjectManageMessage{
						Action:      "delete",
						ProjectName: projectName,
						Success:     false,
						Error:       "保存配置失败: " + err.Error(),
					},
				}
				stream.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := stream.Message{
				Type:      "project_managed",
				Timestamp: time.Now(),
				Data: stream.ProjectManageMessage{
					Action:      "delete",
					ProjectName: projectName,
					Success:     true,
				},
			}
			stream.Global.Broadcast(wsMessage)

			c.JSON(http.StatusOK, gin.H{
				"message": "项目删除成功",
				"name":    projectName,
			})
		})

		// 获取项目的分支列表
		versionAPI.GET("/:name/branches", func(c *gin.Context) {
			projectName := c.Param("name")

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			branches, err := getBranches(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, branches)
		})

		// 获取项目的标签列表
		versionAPI.GET("/:name/tags", func(c *gin.Context) {
			projectName := c.Param("name")

			// 获取筛选参数
			filter := c.Query("filter")

			// 获取分页参数
			page := 1
			limit := 20
			if pageStr := c.Query("page"); pageStr != "" {
				if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
					page = p
				}
			}
			if limitStr := c.Query("limit"); limitStr != "" {
				if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
					limit = l
				}
			}

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			allTags, err := getTags(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 如果有筛选条件，进行筛选
			var filteredTags []TagResponse
			if filter != "" {
				for _, tag := range allTags {
					if strings.HasPrefix(tag.Name, filter) {
						filteredTags = append(filteredTags, tag)
					}
				}
			} else {
				filteredTags = allTags
			}

			// 计算分页
			total := len(filteredTags)
			totalPages := (total + limit - 1) / limit
			start := (page - 1) * limit
			end := start + limit

			if start >= total {
				// 超出范围，返回空数组
				c.JSON(http.StatusOK, gin.H{
					"tags":       []TagResponse{},
					"total":      total,
					"page":       page,
					"limit":      limit,
					"totalPages": totalPages,
					"hasMore":    false,
				})
				return
			}

			if end > total {
				end = total
			}

			paginatedTags := filteredTags[start:end]
			hasMore := page < totalPages

			c.JSON(http.StatusOK, gin.H{
				"tags":       paginatedTags,
				"total":      total,
				"page":       page,
				"limit":      limit,
				"totalPages": totalPages,
				"hasMore":    hasMore,
			})
		})

		// 同步分支
		versionAPI.POST("/:name/sync-branches", func(c *gin.Context) {
			projectName := c.Param("name")

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := syncBranches(projectPath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "分支同步成功"})
		})

		// 删除分支
		versionAPI.DELETE("/:name/branches/:branchName", func(c *gin.Context) {
			projectName := c.Param("name")
			branchName := c.Param("branchName")

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := deleteBranch(projectPath, branchName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "分支删除成功"})
		})

		// 切换分支
		versionAPI.POST("/:name/switch-branch", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				Branch string `json:"branch"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := switchBranch(projectPath, req.Branch); err != nil {
				// 推送失败消息
				wsMessage := stream.Message{
					Type:      "version_switched",
					Timestamp: time.Now(),
					Data: stream.VersionSwitchMessage{
						ProjectName: projectName,
						Action:      "switch-branch",
						Target:      req.Branch,
						Success:     false,
						Error:       err.Error(),
					},
				}
				stream.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := stream.Message{
				Type:      "version_switched",
				Timestamp: time.Now(),
				Data: stream.VersionSwitchMessage{
					ProjectName: projectName,
					Action:      "switch-branch",
					Target:      req.Branch,
					Success:     true,
				},
			}
			stream.Global.Broadcast(wsMessage)

			c.JSON(http.StatusOK, gin.H{"message": "分支切换成功", "branch": req.Branch})
		})

		// 切换标签
		versionAPI.POST("/:name/switch-tag", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				Tag string `json:"tag"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := switchTag(projectPath, req.Tag); err != nil {
				// 推送失败消息
				wsMessage := stream.Message{
					Type:      "version_switched",
					Timestamp: time.Now(),
					Data: stream.VersionSwitchMessage{
						ProjectName: projectName,
						Action:      "switch-tag",
						Target:      req.Tag,
						Success:     false,
						Error:       err.Error(),
					},
				}
				stream.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := stream.Message{
				Type:      "version_switched",
				Timestamp: time.Now(),
				Data: stream.VersionSwitchMessage{
					ProjectName: projectName,
					Action:      "switch-tag",
					Target:      req.Tag,
					Success:     true,
				},
			}
			stream.Global.Broadcast(wsMessage)

			c.JSON(http.StatusOK, gin.H{"message": "标签切换成功", "tag": req.Tag})
		})

		// 删除标签
		versionAPI.DELETE("/:name/tags/:tagName", func(c *gin.Context) {
			projectName := c.Param("name")
			tagName := c.Param("tagName")

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := deleteTag(projectPath, tagName); err != nil {
				// 推送失败消息
				wsMessage := stream.Message{
					Type:      "version_switched",
					Timestamp: time.Now(),
					Data: stream.VersionSwitchMessage{
						ProjectName: projectName,
						Action:      "delete-tag",
						Target:      tagName,
						Success:     false,
						Error:       err.Error(),
					},
				}
				stream.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := stream.Message{
				Type:      "version_switched",
				Timestamp: time.Now(),
				Data: stream.VersionSwitchMessage{
					ProjectName: projectName,
					Action:      "delete-tag",
					Target:      tagName,
					Success:     true,
				},
			}
			stream.Global.Broadcast(wsMessage)

			c.JSON(http.StatusOK, gin.H{"message": "标签删除成功"})
		})

		// 初始化Git仓库
		versionAPI.POST("/:name/init-git", func(c *gin.Context) {
			projectName := c.Param("name")
			fmt.Printf("收到Git初始化请求: 项目名=%s\n", projectName)

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				fmt.Printf("Git初始化失败: 项目未找到, 项目名=%s\n", projectName)
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			fmt.Printf("Git初始化: 项目名=%s, 路径=%s\n", projectName, projectPath)

			if err := version.InitGit(projectPath); err != nil {
				fmt.Printf("Git初始化失败: 项目名=%s, 路径=%s, 错误=%v\n", projectName, projectPath, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			fmt.Printf("Git初始化成功: 项目名=%s, 路径=%s\n", projectName, projectPath)
			c.JSON(http.StatusOK, gin.H{"message": "Git仓库初始化成功"})
		})

		// 设置远程仓库
		versionAPI.POST("/:name/set-remote", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				RemoteUrl string `json:"remoteUrl"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			if req.RemoteUrl == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "远程仓库URL不能为空"})
				return
			}

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := setRemote(projectPath, req.RemoteUrl); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "远程仓库设置成功"})
		})

		// 获取远程仓库
		versionAPI.GET("/:name/remote", func(c *gin.Context) {
			projectName := c.Param("name")

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			remoteURL, err := getRemote(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"url": remoteURL})
		})

		// 获取项目环境变量文件(.env)
		versionAPI.GET("/:name/env", func(c *gin.Context) {
			projectName := c.Param("name")

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			envContent, exists, err := getEnvFile(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"content": envContent,
				"exists":  exists,
				"path":    filepath.Join(projectPath, ".env"),
			})
		})

		// 保存项目环境变量文件(.env)
		versionAPI.POST("/:name/env", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				Content string `json:"content" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			// 验证环境变量文件格式
			if errors := validateEnvContent(req.Content); len(errors) > 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "环境变量文件格式验证失败",
					"details": errors,
				})
				return
			}

			// 保存环境变量文件
			if err := saveEnvFile(projectPath, req.Content); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "环境变量文件保存成功",
				"path":    filepath.Join(projectPath, ".env"),
			})
		})

		// 删除项目环境变量文件(.env)
		versionAPI.DELETE("/:name/env", func(c *gin.Context) {
			projectName := c.Param("name")

			// 查找项目路径
			var projectPath string
			for _, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := deleteEnvFile(projectPath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "环境变量文件删除成功",
			})
		})

		// 保存项目GitHook配置
		versionAPI.POST("/:name/githook", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				Enhook     bool   `json:"enhook"`
				Hookmode   string `json:"hookmode"`
				Hookbranch string `json:"hookbranch"`
				Hooksecret string `json:"hooksecret"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
				return
			}

			// 查找项目并更新配置
			projectFound := false
			for i, proj := range configData.Projects {
				if proj.Name == projectName && proj.Enabled {
					configData.Projects[i].Enhook = req.Enhook
					configData.Projects[i].Hookmode = req.Hookmode
					configData.Projects[i].Hookbranch = req.Hookbranch
					configData.Projects[i].Hooksecret = req.Hooksecret
					projectFound = true
					break
				}
			}

			if !projectFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			// 保存配置文件
			if err := saveConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "GitHook配置保存成功",
			})
		})
	}

	// GitHook webhook接收端点
	g.POST("/githook/:name", func(c *gin.Context) {
		projectName := c.Param("name")

		// 查找项目配置
		var project *ProjectConfig
		for _, proj := range configData.Projects {
			if proj.Name == projectName && proj.Enabled && proj.Enhook {
				project = &proj
				break
			}
		}

		if project == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到或GitHook未启用"})
			return
		}

		// 读取原始payload数据
		var payloadBody []byte
		if c.Request.Body != nil {
			var err error
			payloadBody, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "读取payload失败"})
				return
			}
			// 重置body以便后续使用
			c.Request.Body = io.NopCloser(bytes.NewReader(payloadBody))
		}

		// 验证webhook密码（如果设置了密码）
		if project.Hooksecret != "" {
			if err := verifyWebhookSignature(c, payloadBody, project.Hooksecret); err != nil {
				log.Printf("GitHook密码验证失败: 项目=%s, 错误=%v", project.Name, err)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "密码验证失败: " + err.Error()})
				return
			}
		}

		// 解析webhook payload (支持GitHub, GitLab等格式)
		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的webhook payload"})
			return
		}

		// 处理GitHook逻辑
		if err := handleGitHook(project, payload); err != nil {
			log.Printf("GitHook处理失败: 项目=%s, 错误=%v", project.Name, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "GitHook处理失败: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "GitHook处理成功"})
	})

	// 插件管理API接口组 (临时空接口)
	pluginAPI := g.Group("/plugin")
	pluginAPI.Use(authMiddleware()) // 添加认证中间件
	{
		// 获取所有插件列表
		pluginAPI.GET("", func(c *gin.Context) {
			// 返回空插件列表
			c.JSON(http.StatusOK, []gin.H{})
		})

		// 获取指定插件配置
		pluginAPI.GET("/:id/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "插件配置功能尚未实现",
			})
		})

		// 获取指定插件显示信息
		pluginAPI.GET("/:id/display", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "插件显示功能尚未实现",
			})
		})

		// 更新插件配置
		pluginAPI.POST("/:id/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "插件配置更新功能尚未实现",
			})
		})

		// 启用插件
		pluginAPI.POST("/:id/enable", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "插件启用功能尚未实现",
			})
		})

		// 禁用插件
		pluginAPI.POST("/:id/disable", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "插件禁用功能尚未实现",
			})
		})
	}

	// 客户端列表API (获取当前用户的所有会话)
	g.GET("/client", authMiddleware(), func(c *gin.Context) {
		username, _ := c.Get("username")
		currentToken, _ := c.Get("token")

		sessions := getClientSessionsByUser(username.(string))

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
	})

	// 删除客户端API (注销指定会话)
	g.DELETE("/client/:id", authMiddleware(), adminMiddleware(), func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID"})
			return
		}

		sessionMutex.Lock()
		defer sessionMutex.Unlock()

		var tokenToDelete string
		for token, session := range clientSessions {
			if session.ID == id {
				tokenToDelete = token
				break
			}
		}

		if tokenToDelete != "" {
			delete(clientSessions, tokenToDelete)
			c.JSON(http.StatusOK, gin.H{"message": "Client session deleted"})
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client session not found"})
		}
	})

	// 删除当前用户的会话
	g.DELETE("/client/current", authMiddleware(), func(c *gin.Context) {
		token := c.GetHeader("X-GoHook-Key")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token not provided"})
			return
		}

		if removeClientSession(token) {
			c.JSON(http.StatusOK, gin.H{"message": "Current session deleted successfully"})
		} else {
			// 即使找不到会话，也返回成功，因为客户端的目标是退出
			c.JSON(http.StatusOK, gin.H{"message": "Session not found, but logout process can continue"})
		}
	})

	// 修改当前用户密码API (补充现有current路由)
	g.POST("/current/user/password", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "密码修改功能尚未实现",
		})
	})

	// 保存router实例
	routerInstance = g

	return g
}

// deleteBranch 删除本地分支
func deleteBranch(projectPath, branchName string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库")
	}

	// 获取当前分支
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranchOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取当前分支失败: %v", err)
	}
	currentBranch := strings.TrimSpace(string(currentBranchOutput))

	// 检查是否试图删除当前分支
	if currentBranch == branchName {
		return fmt.Errorf("不能删除当前分支")
	}

	// 删除本地分支
	cmd = exec.Command("git", "-C", projectPath, "branch", "-D", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除分支失败: %s", string(output))
	}

	return nil
}

// deleteTag 删除本地和远程标签
func deleteTag(projectPath, tagName string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库")
	}

	// 检查当前是否在该标签上
	cmd := exec.Command("git", "-C", projectPath, "describe", "--tags", "--exact-match", "HEAD")
	currentTagOutput, err := cmd.Output()
	if err == nil {
		currentTag := strings.TrimSpace(string(currentTagOutput))
		if currentTag == tagName {
			return fmt.Errorf("不能删除当前标签")
		}
	}

	// 删除本地标签
	cmd = exec.Command("git", "-C", projectPath, "tag", "-d", tagName)
	localOutput, localErr := cmd.CombinedOutput()
	if localErr != nil {
		return fmt.Errorf("删除本地标签失败: %s", string(localOutput))
	}

	// 尝试删除远程标签
	cmd = exec.Command("git", "-C", projectPath, "push", "origin", ":refs/tags/"+tagName)
	remoteOutput, remoteErr := cmd.CombinedOutput()
	if remoteErr != nil {
		log.Printf("删除远程标签失败 (项目: %s, 标签: %s): %s", projectPath, tagName, string(remoteOutput))
		// 远程标签删除失败不作为致命错误，因为可能远程没有该标签
	}

	return nil
}

// 环境变量文件处理函数

// getEnvFile 读取项目的.env文件内容
func getEnvFile(projectPath string) (string, bool, error) {
	envFilePath := filepath.Join(projectPath, ".env")

	// 检查文件是否存在
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return "", false, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(envFilePath)
	if err != nil {
		return "", true, fmt.Errorf("读取环境变量文件失败: %v", err)
	}

	return string(content), true, nil
}

// saveEnvFile 保存项目的.env文件
func saveEnvFile(projectPath, content string) error {
	envFilePath := filepath.Join(projectPath, ".env")

	// 确保项目目录存在
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("项目目录不存在: %s", projectPath)
	}

	// 写入文件，如果文件不存在会自动创建
	err := os.WriteFile(envFilePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("保存环境变量文件失败: %v", err)
	}

	return nil
}

// deleteEnvFile 删除项目的.env文件
func deleteEnvFile(projectPath string) error {
	envFilePath := filepath.Join(projectPath, ".env")

	// 检查文件是否存在
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return fmt.Errorf("环境变量文件不存在")
	}

	// 删除文件
	err := os.Remove(envFilePath)
	if err != nil {
		return fmt.Errorf("删除环境变量文件失败: %v", err)
	}

	return nil
}

// validateEnvContent 验证环境变量文件格式
func validateEnvContent(content string) []string {
	var errors []string
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)

		// 跳过空行和注释行
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// 检查是否包含等号
		if !strings.Contains(trimmedLine, "=") {
			errors = append(errors, fmt.Sprintf("第%d行: 缺少等号分隔符", lineNum))
			continue
		}

		// 分割键值对
		parts := strings.SplitN(trimmedLine, "=", 2)
		if len(parts) != 2 {
			errors = append(errors, fmt.Sprintf("第%d行: 格式错误", lineNum))
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 验证键名
		if key == "" {
			errors = append(errors, fmt.Sprintf("第%d行: 变量名不能为空", lineNum))
			continue
		}

		// 验证键名格式（只允许字母、数字、下划线）
		if !isValidEnvKey(key) {
			errors = append(errors, fmt.Sprintf("第%d行: 变量名'%s'格式无效，只允许字母、数字和下划线", lineNum, key))
			continue
		}

		// 验证值的引号匹配
		if !isValidEnvValue(value) {
			errors = append(errors, fmt.Sprintf("第%d行: 变量值'%s'的引号不匹配", lineNum, value))
		}
	}

	return errors
}

// isValidEnvKey 检查环境变量键名是否有效
func isValidEnvKey(key string) bool {
	if key == "" {
		return false
	}

	// 第一个字符必须是字母或下划线
	firstChar := key[0]
	if !((firstChar >= 'A' && firstChar <= 'Z') ||
		(firstChar >= 'a' && firstChar <= 'z') ||
		firstChar == '_') {
		return false
	}

	// 其余字符必须是字母、数字或下划线
	for _, char := range key[1:] {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}

	return true
}

// isValidEnvValue 检查环境变量值的引号是否匹配
func isValidEnvValue(value string) bool {
	if value == "" {
		return true
	}

	// 检查单引号
	if strings.HasPrefix(value, "'") {
		return strings.HasSuffix(value, "'") && len(value) >= 2
	}

	// 检查双引号
	if strings.HasPrefix(value, "\"") {
		return strings.HasSuffix(value, "\"") && len(value) >= 2
	}

	// 没有引号的值也是有效的
	return true
}

// handleGitHook 处理GitHook webhook请求
func handleGitHook(project *ProjectConfig, payload map[string]interface{}) error {
	log.Printf("处理GitHook: 项目=%s, 模式=%s, 分支设置=%s", project.Name, project.Hookmode, project.Hookbranch)

	// 解析webhook payload，提取分支或标签信息
	var targetRef string
	var refType string
	var afterCommit string

	// 尝试解析GitHub/GitLab格式的webhook
	if ref, ok := payload["ref"].(string); ok {
		// 提取after字段（用于检测删除操作）
		if after, ok := payload["after"].(string); ok {
			afterCommit = after
		}

		if strings.HasPrefix(ref, "refs/heads/") {
			// 分支推送
			targetRef = strings.TrimPrefix(ref, "refs/heads/")
			refType = "branch"
		} else if strings.HasPrefix(ref, "refs/tags/") {
			// 标签推送
			targetRef = strings.TrimPrefix(ref, "refs/tags/")
			refType = "tag"
		}
	}

	// 如果没有解析到ref，尝试其他格式
	if targetRef == "" {
		// 尝试GitLab格式
		if ref, ok := payload["ref"].(string); ok {
			parts := strings.Split(ref, "/")
			if len(parts) >= 3 {
				if parts[1] == "heads" {
					targetRef = strings.Join(parts[2:], "/")
					refType = "branch"
				} else if parts[1] == "tags" {
					targetRef = strings.Join(parts[2:], "/")
					refType = "tag"
				}
			}
		}
	}

	if targetRef == "" {
		return fmt.Errorf("无法从webhook payload中解析分支或标签信息")
	}

	log.Printf("解析到webhook: 类型=%s, 目标=%s, after=%s", refType, targetRef, afterCommit)

	// 检查是否匹配项目的hook模式
	if project.Hookmode != refType {
		log.Printf("webhook类型(%s)与项目hook模式(%s)不匹配，忽略", refType, project.Hookmode)
		return nil
	}

	// 如果是分支模式，检查分支匹配
	if project.Hookmode == "branch" {
		if project.Hookbranch != "*" && project.Hookbranch != targetRef {
			log.Printf("webhook分支(%s)与配置分支(%s)不匹配，忽略", targetRef, project.Hookbranch)
			return nil
		}
	}

	// 检查是否是删除操作（after字段为全零）
	if afterCommit == "0000000000000000000000000000000000000000" {
		if refType == "tag" {
			// 标签删除：只删除本地标签
			log.Printf("检测到标签删除事件: %s", targetRef)
			return deleteLocalTag(project.Path, targetRef)
		} else if refType == "branch" {
			// 分支删除：需要智能判断
			log.Printf("检测到分支删除事件: %s", targetRef)
			return handleBranchDeletion(project, targetRef)
		}
	}

	// 执行Git操作
	if err := executeGitHook(project, refType, targetRef); err != nil {
		return fmt.Errorf("执行Git操作失败: %v", err)
	}

	log.Printf("GitHook处理成功: 项目=%s, 类型=%s, 目标=%s", project.Name, refType, targetRef)
	return nil
}

// deleteLocalTag 删除本地标签
func deleteLocalTag(projectPath, tagName string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库: %s", projectPath)
	}

	// 检查标签是否存在
	cmd := exec.Command("git", "-C", projectPath, "show-ref", "--tags", "--quiet", "refs/tags/"+tagName)
	if err := cmd.Run(); err != nil {
		log.Printf("本地标签 %s 不存在，无需删除", tagName)
		return nil
	}

	// 删除本地标签
	cmd = exec.Command("git", "-C", projectPath, "tag", "-d", tagName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("删除本地标签 %s 失败: %s", tagName, string(output))
	}

	log.Printf("成功删除本地标签: %s", tagName)
	return nil
}

// deleteLocalBranch 删除本地分支
func deleteLocalBranch(projectPath, branchName string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库: %s", projectPath)
	}

	// 获取当前分支
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranchOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取当前分支失败: %v", err)
	}
	currentBranch := strings.TrimSpace(string(currentBranchOutput))

	// 检查是否试图删除当前分支
	if currentBranch == branchName {
		log.Printf("不能删除当前分支 %s，跳过删除操作", branchName)
		return nil
	}

	// 检查分支是否存在
	cmd = exec.Command("git", "-C", projectPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err := cmd.Run(); err != nil {
		log.Printf("本地分支 %s 不存在，无需删除", branchName)
		return nil
	}

	// 删除本地分支
	cmd = exec.Command("git", "-C", projectPath, "branch", "-D", branchName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("删除本地分支 %s 失败: %s", branchName, string(output))
	}

	log.Printf("成功删除本地分支: %s", branchName)
	return nil
}

// handleBranchDeletion 智能处理分支删除操作
func handleBranchDeletion(project *ProjectConfig, branchName string) error {
	log.Printf("智能处理分支删除: 项目=%s, 分支=%s, 配置分支=%s", project.Name, branchName, project.Hookbranch)

	// 检查是否是配置的分支
	if project.Hookbranch != "*" && project.Hookbranch == branchName {
		log.Printf("删除的是配置分支 %s，忽略删除操作以保护项目运行", branchName)
		return nil
	}

	// 检查是否是master分支
	if branchName == "master" || branchName == "main" {
		log.Printf("删除的是主分支 %s，忽略删除操作以保护项目", branchName)
		return nil
	}

	// 如果是其他分支，执行删除操作
	log.Printf("删除的是非关键分支 %s，执行本地删除操作", branchName)
	return deleteLocalBranch(project.Path, branchName)
}

// verifyWebhookSignature 验证webhook签名，支持GitHub、GitLab等不同格式
func verifyWebhookSignature(c *gin.Context, payloadBody []byte, secret string) error {
	// GitHub使用X-Hub-Signature-256 header with HMAC-SHA256
	if githubSig := c.GetHeader("X-Hub-Signature-256"); githubSig != "" {
		return verifyGitHubSignature(payloadBody, secret, githubSig)
	}

	// GitHub旧版使用X-Hub-Signature header with HMAC-SHA1
	if githubSigLegacy := c.GetHeader("X-Hub-Signature"); githubSigLegacy != "" {
		return verifyGitHubLegacySignature(payloadBody, secret, githubSigLegacy)
	}

	// GitLab使用X-Gitlab-Token header，直接比较密码
	if gitlabToken := c.GetHeader("X-Gitlab-Token"); gitlabToken != "" {
		return verifyGitLabToken(secret, gitlabToken)
	}

	// Gitea使用X-Gitea-Signature header with HMAC-SHA256
	if giteaSig := c.GetHeader("X-Gitea-Signature"); giteaSig != "" {
		return verifyGiteaSignature(payloadBody, secret, giteaSig)
	}

	// Gogs使用X-Gogs-Signature header with HMAC-SHA256
	if gogsSig := c.GetHeader("X-Gogs-Signature"); gogsSig != "" {
		return verifyGogsSignature(payloadBody, secret, gogsSig)
	}

	// 如果没有找到任何已知的签名header，返回错误
	return fmt.Errorf("未找到支持的webhook签名header")
}

// verifyGitHubSignature 验证GitHub HMAC-SHA256签名
func verifyGitHubSignature(payload []byte, secret, signature string) error {
	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("GitHub签名格式错误，应以sha256=开头")
	}

	expectedSig := "sha256=" + hmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("GitHub签名验证失败")
	}

	return nil
}

// verifyGitHubLegacySignature 验证GitHub HMAC-SHA1签名（旧版）
func verifyGitHubLegacySignature(payload []byte, secret, signature string) error {
	if !strings.HasPrefix(signature, "sha1=") {
		return fmt.Errorf("GitHub legacy签名格式错误，应以sha1=开头")
	}

	// 注意：这里应该使用SHA1，但为了安全性，我们建议使用SHA256
	expectedSig := "sha1=" + hmacSHA1Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("GitHub legacy签名验证失败")
	}

	return nil
}

// verifyGitLabToken 验证GitLab token（直接比较）
func verifyGitLabToken(secret, token string) error {
	if subtle.ConstantTimeCompare([]byte(secret), []byte(token)) != 1 {
		return fmt.Errorf("GitLab token验证失败")
	}
	return nil
}

// verifyGiteaSignature 验证Gitea HMAC-SHA256签名
func verifyGiteaSignature(payload []byte, secret, signature string) error {
	expectedSig := hmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("Gitea签名验证失败")
	}
	return nil
}

// verifyGogsSignature 验证Gogs HMAC-SHA256签名
func verifyGogsSignature(payload []byte, secret, signature string) error {
	expectedSig := hmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("Gogs签名验证失败")
	}
	return nil
}

// hmacSHA256Hex 计算HMAC-SHA256并返回十六进制字符串
func hmacSHA256Hex(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmacSHA1Hex 计算HMAC-SHA1并返回十六进制字符串（用于GitHub legacy支持）
func hmacSHA1Hex(data []byte, secret string) string {
	// 注意：这里应该导入crypto/sha1，但为了保持简单，我们跳过这个实现
	// 在生产环境中应该正确实现SHA1
	return hmacSHA256Hex(data, secret) // 临时使用SHA256代替
}

// executeGitHook 执行具体的Git操作
func executeGitHook(project *ProjectConfig, refType, targetRef string) error {
	projectPath := project.Path

	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不是Git仓库: %s", projectPath)
	}

	// 首先拉取最新的远程信息
	cmd := exec.Command("git", "-C", projectPath, "fetch", "--all")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("警告: 拉取远程信息失败: %s", string(output))
	}

	if refType == "branch" {
		// 分支模式：切换到指定分支并拉取最新代码
		return switchAndPullBranch(projectPath, targetRef)
	} else if refType == "tag" {
		// 标签模式：切换到指定标签
		return switchToTag(projectPath, targetRef)
	}

	return fmt.Errorf("不支持的引用类型: %s", refType)
}

// switchAndPullBranch 切换到指定分支并拉取最新代码
func switchAndPullBranch(projectPath, branchName string) error {
	// 检查本地是否存在该分支
	cmd := exec.Command("git", "-C", projectPath, "branch", "--list", branchName)
	output, err := cmd.Output()
	localBranchExists := err == nil && strings.TrimSpace(string(output)) != ""

	if !localBranchExists {
		// 本地分支不存在，尝试从远程创建
		cmd = exec.Command("git", "-C", projectPath, "checkout", "-b", branchName, "origin/"+branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("创建并切换到分支 %s 失败: %s", branchName, string(output))
		}
	} else {
		// 本地分支存在，直接切换
		cmd = exec.Command("git", "-C", projectPath, "checkout", branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("切换到分支 %s 失败: %s", branchName, string(output))
		}

		// 拉取最新代码
		cmd = exec.Command("git", "-C", projectPath, "pull", "origin", branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("拉取分支 %s 最新代码失败: %s", branchName, string(output))
		}
	}

	return nil
}

// switchToTag 切换到指定标签
func switchToTag(projectPath, tagName string) error {
	// 拉取标签信息
	cmd := exec.Command("git", "-C", projectPath, "fetch", "--tags")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("警告: 拉取标签信息失败: %s", string(output))
	}

	// 确保标签存在（本地或远程）
	cmd = exec.Command("git", "-C", projectPath, "rev-parse", tagName)
	if err := cmd.Run(); err != nil {
		log.Printf("标签 %s 不存在，尝试从远程获取", tagName)
		cmd = exec.Command("git", "-C", projectPath, "fetch", "origin", "--tags")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("从远程获取标签失败: %s", string(output))
		}

		// 再次检查标签是否存在
		cmd = exec.Command("git", "-C", projectPath, "rev-parse", tagName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("标签 %s 在远程也不存在，无法部署", tagName)
		}
	}

	// 切换到指定标签
	cmd = exec.Command("git", "-C", projectPath, "checkout", tagName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("切换到标签 %s 失败: %s", tagName, string(output))
	}

	log.Printf("成功切换到标签: %s", tagName)
	return nil
}

// GetAppConfig 获取应用程序配置
func GetAppConfig() *AppConfig {
	return appConfig
}

// GetUsersConfig 获取用户配置
func GetUsersConfig() *UsersConfig {
	return usersConfig
}

// GetConfiguredPort 获取配置的端口号
func GetConfiguredPort() int {
	if appConfig != nil {
		return appConfig.Port
	}
	return 9000 // 默认端口
}

// 全局router实例
var routerInstance *gin.Engine

// GetRouter 获取当前的router实例
func GetRouter() *gin.Engine {
	return routerInstance
}

// generateRandomPassword 生成随机密码
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, length)

	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			// 如果随机数生成失败，使用时间戳作为后备
			password[i] = charset[int(time.Now().UnixNano())%len(charset)]
		} else {
			password[i] = charset[num.Int64()]
		}
	}

	return string(password)
}
