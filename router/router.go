package router

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mycoool/gohook/internal/hook"
	"github.com/mycoool/gohook/ui"
	"github.com/mycoool/gohook/version"
	wsmanager "github.com/mycoool/gohook/websocket"
	"gopkg.in/yaml.v2"
)

// Config 配置文件结构
type Config struct {
	Auth struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"auth"`
	Projects []ProjectConfig `yaml:"projects"`
}

// ProjectConfig 项目配置
type ProjectConfig struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
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
}

// BranchResponse 分支响应结构
type BranchResponse struct {
	Name           string `json:"name"`
	IsCurrent      bool   `json:"isCurrent"`
	LastCommit     string `json:"lastCommit"`
	LastCommitTime string `json:"lastCommitTime"`
}

// TagResponse 标签响应结构
type TagResponse struct {
	Name       string `json:"name"`
	IsCurrent  bool   `json:"isCurrent"`
	CommitHash string `json:"commitHash"`
	Date       string `json:"date"`
	Message    string `json:"message"`
}

// 全局变量引用，用于访问已加载的hooks
var LoadedHooksFromFiles *map[string]hook.Hooks
var configData *Config

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// WebSocket连接管理器
type WSManager struct {
	clients    map[*websocket.Conn]bool
	clientsMux sync.RWMutex
}

// 全局WebSocket管理器
var wsManager = &WSManager{
	clients: make(map[*websocket.Conn]bool),
}

// WebSocket消息类型
type WSMessage struct {
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

// 版本信息
var vInfo = &ui.VersionInfo{
	Version:   version.Version,
	Commit:    version.Commit,
	BuildDate: version.BuildDate,
}

// loadConfig 加载配置文件
func loadConfig() error {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	configData = config
	return nil
}

// saveConfig 保存配置文件
func saveConfig() error {
	if configData == nil {
		return fmt.Errorf("配置数据为空")
	}

	data, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	// 备份原配置文件
	if _, err := os.Stat("config.yaml"); err == nil {
		os.Rename("config.yaml", "config.yaml.bak")
	}

	err = os.WriteFile("config.yaml", data, 0644)
	if err != nil {
		// 如果保存失败，恢复备份
		if _, backupErr := os.Stat("config.yaml.bak"); backupErr == nil {
			os.Rename("config.yaml.bak", "config.yaml")
		}
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	// 删除备份文件
	os.Remove("config.yaml.bak")
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
	// 获取当前分支
	cmd := exec.Command("git", "-C", projectPath, "branch", "--show-current")
	currentOutput, _ := cmd.Output()
	currentBranch := strings.TrimSpace(string(currentOutput))

	// 获取所有分支
	cmd = exec.Command("git", "-C", projectPath, "branch", "-a", "--format=%(refname:short)|%(committerdate)|%(objectname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取分支列表失败: %v", err)
	}

	var branches []BranchResponse
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			branchName := parts[0]
			// 跳过远程分支的本地引用
			if strings.HasPrefix(branchName, "origin/") && !strings.Contains(branchName, "HEAD") {
				branchName = strings.TrimPrefix(branchName, "origin/")
				// 检查是否已有本地分支
				hasLocal := false
				for _, existing := range branches {
					if existing.Name == branchName {
						hasLocal = true
						break
					}
				}
				if hasLocal {
					continue
				}
			} else if strings.Contains(branchName, "/") {
				continue // 跳过其他远程分支
			}

			branches = append(branches, BranchResponse{
				Name:           branchName,
				IsCurrent:      branchName == currentBranch,
				LastCommitTime: parts[1],
				LastCommit:     parts[2],
			})
		}
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

// switchBranch 切换分支
func switchBranch(projectPath, branchName string) error {
	// 检查是否是远程分支
	cmd := exec.Command("git", "-C", projectPath, "branch", "-r")
	remoteOutput, _ := cmd.Output()
	isRemote := strings.Contains(string(remoteOutput), "origin/"+branchName)

	if isRemote {
		// 创建并切换到本地分支
		cmd = exec.Command("git", "-C", projectPath, "checkout", "-b", branchName, "origin/"+branchName)
	} else {
		// 切换到现有本地分支
		cmd = exec.Command("git", "-C", projectPath, "checkout", branchName)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("切换分支失败: %v", err)
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

// initGit 初始化Git仓库
func initGit(projectPath string) error {
	// 检查项目路径是否存在
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", projectPath)
	}

	// 检查项目路径是否为目录
	if info, err := os.Stat(projectPath); err != nil {
		return fmt.Errorf("无法访问项目路径: %s, 错误: %v", projectPath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("项目路径不是目录: %s", projectPath)
	}

	// 检查是否已经是Git仓库
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return fmt.Errorf("目录已经是Git仓库")
	}

	// 尝试创建一个临时文件来测试写权限
	testFile := filepath.Join(projectPath, ".gohook-permission-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("项目路径没有写权限: %s，请检查目录权限。建议运行: sudo chown -R %s:%s %s",
			projectPath, os.Getenv("USER"), os.Getenv("USER"), projectPath)
	}
	// 清理测试文件
	os.Remove(testFile)

	// 执行git init命令
	cmd := exec.Command("git", "-C", projectPath, "init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Git仓库初始化失败: %v, 输出: %s", err, string(output))
	}

	// 验证Git仓库是否成功创建
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("Git仓库初始化后验证失败: .git目录未创建")
	}

	return nil
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

// getHooksList 获取所有Hook列表
func getHooksList() []HookResponse {
	var hooks []HookResponse

	if LoadedHooksFromFiles == nil {
		return hooks
	}

	for _, hooksInFile := range *LoadedHooksFromFiles {
		for _, h := range hooksInFile {
			hookResponse := convertHookToResponse(&h)
			hooks = append(hooks, hookResponse)
		}
	}

	return hooks
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
		wsmanager.Global.RemoveClient(conn)
		conn.Close()
	}()

	// 添加到连接管理器
	wsmanager.Global.AddClient(conn)

	// 发送欢迎消息
	welcomeMsg := wsmanager.Message{
		Type:      "connected",
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "WebSocket connected",
			"server":  "gohook",
		},
	}
	welcomeData, _ := json.Marshal(welcomeMsg)
	conn.WriteMessage(websocket.TextMessage, welcomeData)

	// 保持连接，处理心跳
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// 处理客户端消息（心跳等）
		var clientMsg map[string]interface{}
		if json.Unmarshal(message, &clientMsg) == nil {
			if msgType, ok := clientMsg["type"].(string); ok && msgType == "ping" {
				// 响应心跳
				pongMsg := wsmanager.Message{
					Type:      "pong",
					Timestamp: time.Now(),
					Data:      map[string]string{"message": "pong"},
				}
				pongData, _ := json.Marshal(pongMsg)
				conn.WriteMessage(websocket.TextMessage, pongData)
			}
		}
	}
}

func InitRouter() *gin.Engine {
	g := gin.Default()

	// 加载配置文件
	if err := loadConfig(); err != nil {
		// 如果配置文件加载失败，使用默认值
		configData = &Config{}
	}

	// 注册前端UI路由，这将接管根路径 "/"
	ui.Register(g, *vInfo, true)

	// CORS中间件 - 在路由注册后添加，避免通配符冲突
	g.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Gotify-Key")

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

		// 验证用户名和密码
		if username == "admin" && password == "123456" {
			// 生成一个简单的token（实际应用中应该使用更安全的token生成方式）
			token := "gohook-token-" + username + "-12345"

			// 获取客户端名称（从请求体中的name字段）
			var requestBody struct {
				Name string `json:"name"`
			}
			c.BindJSON(&requestBody)

			clientName := requestBody.Name
			if clientName == "" {
				clientName = "unknown client"
			}

			c.JSON(http.StatusOK, ClientResponse{
				Token: token,
				ID:    1,
				Name:  clientName,
			})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		}
	})

	// 获取当前用户信息接口
	g.GET("/current/user", func(c *gin.Context) {
		token := c.GetHeader("X-Gotify-Key")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			return
		}

		// 简单的token验证（实际应用中应该有更复杂的验证逻辑）
		if strings.HasPrefix(token, "gohook-token-") {
			c.JSON(http.StatusOK, gin.H{
				"id":    1,
				"name":  "admin",
				"admin": true,
			})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		}
	})

	// Hooks API接口组
	hookAPI := g.Group("/hook")
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

		// 重新加载hooks配置文件的专用接口
		hookAPI.POST("/reload-config", func(c *gin.Context) {
			if LoadedHooksFromFiles == nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "hooks未初始化",
				})
				return
			}

			// 注意：由于架构限制，这里我们只是返回当前状态
			// 实际的热重载功能是通过文件监控实现的
			hookCount := 0
			for _, hooksInFile := range *LoadedHooksFromFiles {
				hookCount += len(hooksInFile)
			}

			c.JSON(http.StatusOK, gin.H{
				"message":   "hooks配置获取成功（文件监控自动重载）",
				"hookCount": hookCount,
			})
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
			wsMessage := wsmanager.Message{
				Type:      "hook_triggered",
				Timestamp: time.Now(),
				Data: wsmanager.HookTriggeredMessage{
					HookID:     hookID,
					HookName:   hookResponse.Name,
					Method:     c.Request.Method,
					RemoteAddr: c.ClientIP(),
					Success:    success,
					Output:     output,
					Error:      errorMsg,
				},
			}
			wsmanager.Global.Broadcast(wsMessage)

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
	}

	//添加websocket
	ws := g.Group("/stream")
	{
		//前端访问地址："/stream?token=gohook-token-admin-12345"
		ws.GET("", func(c *gin.Context) {
			token := c.Query("token")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
				return
			}

			// 简单的token验证
			if !strings.HasPrefix(token, "gohook-token-") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				return
			}

			handleWebSocket(c)
		})

		// 也支持带ID的路径格式 /stream/:id?token=...
		ws.GET("/:id", func(c *gin.Context) {
			token := c.Query("token")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
				return
			}

			// 简单的token验证
			if !strings.HasPrefix(token, "gohook-token-") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
				return
			}

			handleWebSocket(c)
		})
	}

	// 版本管理API接口组
	versionAPI := g.Group("/version")
	{
		// 获取所有项目列表
		versionAPI.GET("", func(c *gin.Context) {
			// 每次获取项目列表时重新加载配置文件
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
				projects = append(projects, *gitStatus)
			}

			c.JSON(http.StatusOK, projects)
		})

		// 重新加载配置文件的专用接口
		versionAPI.POST("/reload-config", func(c *gin.Context) {
			if err := loadConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "配置文件重新加载失败: " + err.Error(),
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
				"message":      "配置文件重新加载成功",
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
				wsMessage := wsmanager.Message{
					Type:      "project_managed",
					Timestamp: time.Now(),
					Data: wsmanager.ProjectManageMessage{
						Action:      "add",
						ProjectName: req.Name,
						ProjectPath: req.Path,
						Success:     false,
						Error:       "保存配置失败: " + err.Error(),
					},
				}
				wsmanager.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := wsmanager.Message{
				Type:      "project_managed",
				Timestamp: time.Now(),
				Data: wsmanager.ProjectManageMessage{
					Action:      "add",
					ProjectName: req.Name,
					ProjectPath: req.Path,
					Success:     true,
				},
			}
			wsmanager.Global.Broadcast(wsMessage)

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
				wsMessage := wsmanager.Message{
					Type:      "project_managed",
					Timestamp: time.Now(),
					Data: wsmanager.ProjectManageMessage{
						Action:      "delete",
						ProjectName: projectName,
						Success:     false,
						Error:       "保存配置失败: " + err.Error(),
					},
				}
				wsmanager.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := wsmanager.Message{
				Type:      "project_managed",
				Timestamp: time.Now(),
				Data: wsmanager.ProjectManageMessage{
					Action:      "delete",
					ProjectName: projectName,
					Success:     true,
				},
			}
			wsmanager.Global.Broadcast(wsMessage)

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

			tags, err := getTags(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, tags)
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
				wsMessage := wsmanager.Message{
					Type:      "version_switched",
					Timestamp: time.Now(),
					Data: wsmanager.VersionSwitchMessage{
						ProjectName: projectName,
						Action:      "switch-branch",
						Target:      req.Branch,
						Success:     false,
						Error:       err.Error(),
					},
				}
				wsmanager.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := wsmanager.Message{
				Type:      "version_switched",
				Timestamp: time.Now(),
				Data: wsmanager.VersionSwitchMessage{
					ProjectName: projectName,
					Action:      "switch-branch",
					Target:      req.Branch,
					Success:     true,
				},
			}
			wsmanager.Global.Broadcast(wsMessage)

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
				wsMessage := wsmanager.Message{
					Type:      "version_switched",
					Timestamp: time.Now(),
					Data: wsmanager.VersionSwitchMessage{
						ProjectName: projectName,
						Action:      "switch-tag",
						Target:      req.Tag,
						Success:     false,
						Error:       err.Error(),
					},
				}
				wsmanager.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 推送成功消息
			wsMessage := wsmanager.Message{
				Type:      "version_switched",
				Timestamp: time.Now(),
				Data: wsmanager.VersionSwitchMessage{
					ProjectName: projectName,
					Action:      "switch-tag",
					Target:      req.Tag,
					Success:     true,
				},
			}
			wsmanager.Global.Broadcast(wsMessage)

			c.JSON(http.StatusOK, gin.H{"message": "标签切换成功", "tag": req.Tag})
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

			if err := initGit(projectPath); err != nil {
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
	}

	// 获取用户列表
	g.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{"admin", "user"}})
	})
	return g
}
