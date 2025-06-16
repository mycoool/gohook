package router

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/client"
	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/env"
	"github.com/mycoool/gohook/internal/hook"
	"github.com/mycoool/gohook/internal/stream"
	"github.com/mycoool/gohook/internal/types"
	"github.com/mycoool/gohook/internal/version"
)

// 全局变量引用，用于访问已加载的hooks
var LoadedHooksFromFiles *map[string]hook.Hooks
var HookManager *hook.HookManager

// authMiddleware JWT认证中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("X-GoHook-Key")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		claims, err := client.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 更新会话最后使用时间
		client.UpdateSessionLastUsed(tokenString)

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

		claims, err := client.ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// 更新会话最后使用时间
		client.UpdateSessionLastUsed(tokenString)

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

func InitGit(projectPath string) error {
	return version.InitGit(projectPath)
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

func InitRouter() *gin.Engine {
	g := gin.Default()

	// 加载配置文件
	if err := config.LoadConfig(); err != nil {
		// 如果配置文件加载失败，使用默认值
		types.ConfigData = &types.Config{}
	}

	// 加载应用配置文件
	if err := config.LoadAppConfig(); err != nil {
		// 如果应用配置文件加载失败，创建默认配置
		types.GoHookAppConfig = &types.AppConfig{
			Port:              9000,
			JWTSecret:         "gohook-secret-key-change-in-production",
			JWTExpiryDuration: 24,
		}
		log.Printf("Warning: failed to load app config, using default settings")
	}

	// 加载用户配置文件
	if err := client.LoadUsersConfig(); err != nil {
		// 如果用户配置文件加载失败，创建默认管理员用户
		defaultPassword := "admin123" // 生成随机密码
		types.GoHookUsersConfig = &types.UsersConfig{
			Users: []types.UserConfig{
				{
					Username: "admin",
					Password: client.HashPassword(defaultPassword),
					Role:     "admin",
				},
			},
		}
		// 保存默认用户配置
		if saveErr := client.SaveUsersConfig(); saveErr != nil {
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
		user := client.FindUser(username)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// 验证密码
		if !client.VerifyPassword(password, user.Password) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		// 生成JWT token
		token, err := client.GenerateToken(user.Username, user.Role)
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
		session := client.AddClientSession(token, clientName, user.Username)

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
			var users []types.UserResponse
			for _, user := range types.GoHookUsersConfig.Users {
				users = append(users, types.UserResponse{
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
			if client.FindUser(req.Username) != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
				return
			}

			// 验证角色
			if req.Role != "admin" && req.Role != "user" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Must be 'admin' or 'user'"})
				return
			}

			// 添加新用户
			newUser := types.UserConfig{
				Username: req.Username,
				Password: client.HashPassword(req.Password),
				Role:     req.Role,
			}

			types.GoHookUsersConfig.Users = append(types.GoHookUsersConfig.Users, newUser)

			// 保存配置文件
			if err := client.SaveUsersConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save config: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "User created successfully",
				"user": types.UserResponse{
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
			for i, user := range types.GoHookUsersConfig.Users {
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
			types.GoHookUsersConfig.Users = append(types.GoHookUsersConfig.Users[:userIndex], types.GoHookUsersConfig.Users[userIndex+1:]...)

			// 保存配置文件
			if err := client.SaveUsersConfig(); err != nil {
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
			user := client.FindUser(username.(string))
			if user == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}

			// 验证旧密码
			if !client.VerifyPassword(req.OldPassword, user.Password) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid old password"})
				return
			}

			// 更新密码
			user.Password = client.HashPassword(req.NewPassword)

			// 保存配置文件
			if err := client.SaveUsersConfig(); err != nil {
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

			user := client.FindUser(username)
			if user == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}

			// 更新密码
			user.Password = client.HashPassword(req.NewPassword)

			// 保存配置文件
			if err := client.SaveUsersConfig(); err != nil {
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
			stream.HandleWebSocket(c)
		})

		// 也支持带ID的路径格式 /stream/:id
		ws.GET("/:id", func(c *gin.Context) {
			// Token已通过中间件验证，直接处理WebSocket连接
			stream.HandleWebSocket(c)
		})
	}

	// 版本管理API接口组
	versionAPI := g.Group("/version")
	versionAPI.Use(authMiddleware()) // 添加认证中间件
	{
		// 获取所有项目列表
		versionAPI.GET("", func(c *gin.Context) {
			// 每次获取项目列表时加载配置文件
			if err := config.LoadConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "配置文件加载失败: " + err.Error()})
				return
			}

			if types.ConfigData == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "配置文件未加载"})
				return
			}

			var projects []types.VersionResponse
			for _, proj := range types.ConfigData.Projects {
				if !proj.Enabled {
					continue
				}

				gitStatus, err := version.GetGitStatus(proj.Path)
				if err != nil {
					// 如果不是Git仓库，仍然显示但标记为非Git项目
					projects = append(projects, types.VersionResponse{
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
			if err := config.LoadConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "配置文件加载失败: " + err.Error(),
				})
				return
			}

			projectCount := 0
			if types.ConfigData != nil {
				for _, proj := range types.ConfigData.Projects {
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
			for _, proj := range types.ConfigData.Projects {
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
			newProject := types.ProjectConfig{
				Name:        req.Name,
				Path:        req.Path,
				Description: req.Description,
				Enabled:     true,
			}

			types.ConfigData.Projects = append(types.ConfigData.Projects, newProject)

			// 保存配置文件
			if err := config.SaveConfig(); err != nil {
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
			for i, proj := range types.ConfigData.Projects {
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
			types.ConfigData.Projects = append(types.ConfigData.Projects[:projectIndex], types.ConfigData.Projects[projectIndex+1:]...)

			// 保存配置文件
			if err := config.SaveConfig(); err != nil {
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			branches, err := version.GetBranches(projectPath)
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			allTags, err := version.GetTags(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 如果有筛选条件，进行筛选
			var filteredTags []types.TagResponse
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
					"tags":       []types.TagResponse{},
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := version.SyncBranches(projectPath); err != nil {
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := version.DeleteBranch(projectPath, branchName); err != nil {
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := version.SwitchBranch(projectPath, req.Branch); err != nil {
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := version.SwitchTag(projectPath, req.Tag); err != nil {
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := version.DeleteTag(projectPath, tagName); err != nil {
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
			for _, proj := range types.ConfigData.Projects {
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := version.SetRemote(projectPath, req.RemoteUrl); err != nil {
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			remoteURL, err := version.GetRemote(projectPath)
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			envContent, exists, err := env.GetEnvFile(projectPath)
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
			for _, proj := range types.ConfigData.Projects {
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
			if errors := env.ValidateEnvContent(req.Content); len(errors) > 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "环境变量文件格式验证失败",
					"details": errors,
				})
				return
			}

			// 保存环境变量文件
			if err := env.SaveEnvFile(projectPath, req.Content); err != nil {
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
			for _, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			if err := env.DeleteEnvFile(projectPath); err != nil {
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
			for i, proj := range types.ConfigData.Projects {
				if proj.Name == projectName && proj.Enabled {
					types.ConfigData.Projects[i].Enhook = req.Enhook
					types.ConfigData.Projects[i].Hookmode = req.Hookmode
					types.ConfigData.Projects[i].Hookbranch = req.Hookbranch
					types.ConfigData.Projects[i].Hooksecret = req.Hooksecret
					projectFound = true
					break
				}
			}

			if !projectFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
				return
			}

			// 保存配置文件
			if err := config.SaveConfig(); err != nil {
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
		var project *types.ProjectConfig
		for _, proj := range types.ConfigData.Projects {
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
			if err := version.VerifyWebhookSignature(c, payloadBody, project.Hooksecret); err != nil {
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
		if err := version.HandleGitHook(project, payload); err != nil {
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

		sessions := client.GetClientSessionsByUser(username.(string))

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

		client.SessionMutex.Lock()
		defer client.SessionMutex.Unlock()

		var tokenToDelete string
		for token, session := range client.ClientSessions {
			if session.ID == id {
				tokenToDelete = token
				break
			}
		}

		if tokenToDelete != "" {
			delete(client.ClientSessions, tokenToDelete)
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

		if client.RemoveClientSession(token) {
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

// GetAppConfig 获取应用程序配置
func GetAppConfig() *types.AppConfig {
	return types.GoHookAppConfig
}

// GetUsersConfig 获取用户配置
func GetUsersConfig() *types.UsersConfig {
	return types.GoHookUsersConfig
}

// GetConfiguredPort 获取配置的端口号
func GetConfiguredPort() int {
	if types.GoHookAppConfig != nil {
		return types.GoHookAppConfig.Port
	}
	return 9000 // 默认端口
}

// 全局router实例
var routerInstance *gin.Engine

// GetRouter 获取当前的router实例
func GetRouter() *gin.Engine {
	return routerInstance
}
