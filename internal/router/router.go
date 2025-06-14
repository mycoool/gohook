package router

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/mycoool/gohook/internal/handlers/auth"
	"github.com/mycoool/gohook/internal/handlers/client"
	hookhandler "github.com/mycoool/gohook/internal/handlers/hook"
	"github.com/mycoool/gohook/internal/handlers/user"
	"github.com/mycoool/gohook/internal/handlers/version"
	"github.com/mycoool/gohook/internal/hook"
	wsmanager "github.com/mycoool/gohook/websocket"
)

// WebSocket升级器
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境中应该更严格
	},
}

// SetupRouter 设置路由
func SetupRouter(loadedHooks *map[string]hook.Hooks, hookManager *hook.HookManager) *gin.Engine {
	// 初始化各模块的依赖注入
	setupDependencies(loadedHooks, hookManager)

	// 加载配置
	if err := auth.LoadAppConfig(); err != nil {
		log.Printf("Warning: failed to load user config: %v", err)
		auth.InitDefaultConfig()
	}

	if err := version.LoadConfig(); err != nil {
		log.Printf("Warning: failed to load version config: %v", err)
	}

	r := gin.Default()

	// 静态文件服务
	r.Static("/static", "./ui/build/static")
	r.StaticFile("/", "./ui/build/index.html")
	r.StaticFile("/favicon.ico", "./ui/build/favicon.ico")
	r.StaticFile("/manifest.json", "./ui/build/manifest.json")

	// 处理前端路由
	r.NoRoute(func(c *gin.Context) {
		c.File("./ui/build/index.html")
	})

	// API路由组
	api := r.Group("/api")
	{
		// 认证相关路由
		setupAuthRoutes(api)

		// 需要认证的路由
		authenticated := api.Group("")
		authenticated.Use(auth.AuthMiddleware())
		{
			// 用户管理路由
			setupUserRoutes(authenticated)

			// 版本管理路由
			setupVersionRoutes(authenticated)

			// Hook管理路由
			setupHookRoutes(authenticated)
		}
	}

	// WebSocket路由
	r.GET("/ws", auth.WSAuthMiddleware(), handleWebSocket)

	// 客户端登录路由（不需要认证）
	r.POST("/client", client.LoginHandler)

	// 需要认证的客户端路由
	clientAPI := r.Group("/client")
	clientAPI.Use(auth.AuthMiddleware())
	{
		clientAPI.GET("/me", client.GetCurrentUserHandler)
		clientAPI.GET("/list", client.GetClientListHandler)
		clientAPI.DELETE("/:id", client.DeleteClientHandler)
	}

	return r
}

// setupDependencies 设置依赖注入
func setupDependencies(loadedHooks *map[string]hook.Hooks, hookManager *hook.HookManager) {
	// 设置认证中间件的会话更新函数
	auth.SetUpdateSessionLastUsedFunc(client.UpdateSessionLastUsed)

	// 设置客户端处理器的认证函数
	client.SetAuthFunctions(
		auth.FindUser,
		auth.VerifyPassword,
		auth.GenerateToken,
	)

	// 设置用户管理的函数
	user.SetUserFunctions(
		auth.GetAppConfig,
		auth.SaveAppConfig,
		auth.FindUser,
		auth.HashPassword,
		auth.VerifyPassword,
	)

	// 设置Hook处理器的引用
	hookhandler.SetHookReferences(loadedHooks, hookManager)
}

// setupAuthRoutes 设置认证路由
func setupAuthRoutes(api *gin.RouterGroup) {
	// 这里可以添加其他认证相关的路由，如果需要的话
}

// setupUserRoutes 设置用户管理路由
func setupUserRoutes(api *gin.RouterGroup) {
	userAPI := api.Group("/users")
	userAPI.Use(auth.AdminMiddleware()) // 用户管理需要管理员权限
	{
		userAPI.GET("", user.GetUsersHandler)
		userAPI.POST("", user.CreateUserHandler)
		userAPI.DELETE("/:username", user.DeleteUserHandler)
		userAPI.POST("/:username/reset-password", user.ResetPasswordHandler)
	}

	// 修改密码不需要管理员权限
	api.POST("/change-password", user.ChangePasswordHandler)
}

// setupVersionRoutes 设置版本管理路由
func setupVersionRoutes(api *gin.RouterGroup) {
	versionAPI := api.Group("/version")
	{
		// 获取所有项目版本信息
		versionAPI.GET("", version.GetVersionsHandler)

		// 项目管理（需要管理员权限）
		versionAPI.POST("", auth.AdminMiddleware(), version.AddProjectHandler)
		versionAPI.DELETE("/:name", auth.AdminMiddleware(), version.DeleteProjectHandler)

		// 分支管理
		versionAPI.GET("/:name/branches", version.GetProjectBranchesHandler)
		versionAPI.POST("/:name/branches/:branchName", version.SwitchBranchHandler)

		// 标签管理
		versionAPI.GET("/:name/tags", version.GetProjectTagsHandler)
		versionAPI.POST("/:name/tags/:tagName", version.SwitchTagHandler)
		versionAPI.DELETE("/:name/tags/:tagName", version.DeleteTagHandler)

		// 环境文件管理
		versionAPI.GET("/:name/env", version.GetEnvFileHandler)
		versionAPI.POST("/:name/env", version.SaveEnvFileHandler)
		versionAPI.DELETE("/:name/env", version.DeleteEnvFileHandler)
	}
}

// setupHookRoutes 设置Hook管理路由
func setupHookRoutes(api *gin.RouterGroup) {
	hookAPI := api.Group("/hooks")
	{
		hookAPI.GET("", hookhandler.GetHooksHandler)
		hookAPI.GET("/:id", hookhandler.GetHookHandler)
		hookAPI.POST("/:id/trigger", hookhandler.TriggerHookHandler)
		hookAPI.POST("/reload", auth.AdminMiddleware(), hookhandler.ReloadConfigHandler)
	}
}

// handleWebSocket 处理WebSocket连接
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "WebSocket upgrade failed"})
		return
	}

	// 将连接添加到全局管理器
	wsmanager.Global.AddClient(conn)
	log.Printf("WebSocket client connected, total clients: %d", wsmanager.Global.ClientCount())

	defer func() {
		wsmanager.Global.RemoveClient(conn)
		conn.Close()
		log.Printf("WebSocket client disconnected, total clients: %d", wsmanager.Global.ClientCount())
	}()

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
				if err := conn.WriteMessage(websocket.TextMessage, pongData); err != nil {
					log.Printf("Failed to send pong message: %v", err)
					break
				}
			}
		}
	}
}
