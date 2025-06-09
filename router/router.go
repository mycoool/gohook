package router

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/hook"
	"github.com/mycoool/gohook/ui"
)

// 全局变量引用，用于访问已加载的hooks
var LoadedHooksFromFiles *map[string]hook.Hooks

// 版本信息
var vInfo = &ui.VersionInfo{
	Version:   "2.8.2", // 与app.go中的version常量保持一致
	Commit:    "unknown",
	BuildDate: "unknown",
}

// 配置信息
type Config struct {
	Registration bool
}

var conf = &Config{
	Registration: true, // 允许注册
}

// ClientResponse 客户端响应结构
type ClientResponse struct {
	Token string `json:"token"`
	ID    int    `json:"id"`
	Name  string `json:"name"`
}

// ProjectResponse 项目响应结构
type ProjectResponse struct {
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

// getProjectsList 获取所有项目列表
func getProjectsList() []ProjectResponse {
	var projects []ProjectResponse

	if LoadedHooksFromFiles == nil {
		return projects
	}

	for _, hooks := range *LoadedHooksFromFiles {
		for _, h := range hooks {
			project := convertHookToProject(&h)
			projects = append(projects, project)
		}
	}

	return projects
}

// getProjectByID 根据ID获取项目
func getProjectByID(id string) *ProjectResponse {
	if LoadedHooksFromFiles == nil {
		return nil
	}

	for _, hooks := range *LoadedHooksFromFiles {
		if hook := hooks.Match(id); hook != nil {
			project := convertHookToProject(hook)
			return &project
		}
	}

	return nil
}

// convertHookToProject 将Hook转换为ProjectResponse
func convertHookToProject(h *hook.Hook) ProjectResponse {
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

	return ProjectResponse{
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

	// 注册前端UI路由，这将接管根路径 "/"
	ui.Register(g, *vInfo, conf.Registration)

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

	// Projects API 接口组
	projectsAPI := g.Group("/project")
	{
		// 获取所有项目列表
		projectsAPI.GET("", func(c *gin.Context) {
			projects := getProjectsList()
			c.JSON(http.StatusOK, projects)
		})

		// 获取单个项目详情
		projectsAPI.GET("/:id", func(c *gin.Context) {
			projectID := c.Param("id")
			project := getProjectByID(projectID)
			if project == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}
			c.JSON(http.StatusOK, project)
		})

		// 触发项目webhook（测试接口）
		projectsAPI.POST("/:id/trigger", func(c *gin.Context) {
			projectID := c.Param("id")
			project := getProjectByID(projectID)
			if project == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			// 这里可以添加触发webhook的逻辑
			c.JSON(http.StatusOK, gin.H{
				"message": "Project webhook triggered successfully",
				"project": project.Name,
			})
		})
	}

	// 获取用户列表
	g.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"users": []string{"admin", "user"}})
	})
	return g
}
