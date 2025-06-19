package router

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/client"
	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/hook"
	"github.com/mycoool/gohook/internal/stream"
	"github.com/mycoool/gohook/internal/types"
	"github.com/mycoool/gohook/internal/version"
)

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

// noLogMiddleware disable logging for the request
func noLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// set a flag to indicate that this request should not be logged
		c.Set("disable_log", true)
		c.Next()
	}
}

// wsAuthMiddleware WebSocket auth middleware, support query parameter token and Sec-WebSocket-Protocol header
func wsAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// get token from header first
		tokenString := c.GetHeader("X-GoHook-Key")

		// if no token in header, try to get it from Sec-WebSocket-Protocol header
		if tokenString == "" {
			protocols := c.GetHeader("Sec-WebSocket-Protocol")
			if protocols != "" {
				// parse protocols: "Authorization, <token>"
				parts := strings.Split(protocols, ",")
				if len(parts) >= 2 {
					// trim whitespace and check if first part is "Authorization"
					protocol := strings.TrimSpace(parts[0])
					if protocol == "Authorization" {
						tokenString = strings.TrimSpace(parts[1])
					}
				}
			}
		}

		// if still no token, get it from query parameter (fallback)
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

		// update session last used time
		client.UpdateSessionLastUsed(tokenString)

		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("token", tokenString)
		c.Next()
	}
}

// adminMiddleware admin permission middleware
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

func InitRouter() *gin.Engine {
	// 创建不带默认中间件的engine
	g := gin.New()

	// 添加自定义的日志中间件，跳过标记为"no_log"的请求
	g.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 如果上下文中有"no_log"标记，不记录日志
		if param.Keys != nil {
			if noLog, exists := param.Keys["disable_log"]; exists && noLog == true {
				return ""
			}
		}
		// 否则使用默认格式记录日志
		return fmt.Sprintf("[WEB] %v | %3d | %13v | %15s | %-7s %#v\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	}))

	// 添加Recovery中间件
	g.Use(gin.Recovery())

	// load version config file
	if err := config.LoadVersionConfig(); err != nil {
		// if version config file load failed, use default value
		types.GoHookVersionData = &types.VersionConfig{}
	}

	// load app config
	if err := config.LoadAppConfig(); err != nil {
		// if app config file load failed, create default config
		types.GoHookAppConfig = &types.AppConfig{
			Port:              9000,
			JWTSecret:         "gohook-secret-key-change-in-production",
			JWTExpiryDuration: 24,
		}
		log.Printf("Warning: failed to load app config, using default settings")
	}

	// load user config
	if err := client.LoadUsersConfig(); err != nil {
		// if user config file load failed, create default admin user
		defaultPassword := "admin123" // generate random password
		types.GoHookUsersConfig = &types.UsersConfig{
			Users: []types.UserConfig{
				{
					Username: "admin",
					Password: client.HashPassword(defaultPassword),
					Role:     "admin",
				},
			},
		}
		// save default user config
		if saveErr := client.SaveUsersConfig(); saveErr != nil {
			log.Printf("Error: failed to save default user config: %v", saveErr)
		} else {
			log.Printf("Created default admin user with password: %s", defaultPassword)
		}
		log.Printf("Warning: failed to load user config, created default admin user")
	}

	// CORS middleware - add after router registration, avoid wildcard conflict
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

	// adapt frontend All Messages page, return empty for now
	g.GET("/message", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"messages": []interface{}{},
			"paging":   gin.H{},
		})
	})

	// login interface - support Basic authentication
	g.POST("/client", client.Login)

	// get current user info interface
	g.GET("/current/user", noLogMiddleware(), authMiddleware(), client.GetCurrentUser)

	// get application config interface
	g.GET("/app/config", noLogMiddleware(), authMiddleware(), func(c *gin.Context) {
		if types.GoHookAppConfig == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "App config not loaded"})
			return
		}

		// only return safe config fields, not including secrets
		c.JSON(http.StatusOK, gin.H{
			"port": types.GoHookAppConfig.Port,
			"mode": types.GoHookAppConfig.Mode,
		})
	})

	// user management API group
	userAPI := g.Group("/user")
	userAPI.Use(authMiddleware())
	{
		// get all users list (only admin)
		userAPI.GET("", adminMiddleware(), client.GetAllUsers)

		// create user (only admin)
		userAPI.POST("", adminMiddleware(), client.CreateUser)

		// delete user (only admin)
		userAPI.DELETE("/:username", adminMiddleware(), client.DeleteUser)

		// change password
		userAPI.POST("/password", client.ChangePassword)

		// admin reset user password
		userAPI.POST("/:username/reset-password", adminMiddleware(), client.ResetPassword)
	}

	// Hooks API group
	hookAPI := g.Group("/hook")
	hookAPI.Use(authMiddleware()) // add auth middleware
	{
		// get all hooks
		hookAPI.GET("", hook.GetAllHooks)

		// get single hook detail
		hookAPI.GET("/:id", func(c *gin.Context) {
			hookID := c.Param("id")
			hookResponse := hook.GetHookByID(hookID)
			if hookResponse == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
				return
			}
			c.JSON(http.StatusOK, hookResponse)
		})

		// trigger hook (test interface)
		hookAPI.POST("/:id/trigger", func(c *gin.Context) {
			hookID := c.Param("id")
			hookResponse := hook.GetHookByID(hookID)
			if hookResponse == nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
				return
			}

			// execute hook command
			success := false
			output := ""
			errorMsg := ""

			if hookResponse.ExecuteCommand != "" {
				// execute command
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
				output = "Hook triggered successfully (no execute command)"
			}

			// push WebSocket message
			wsMessage := stream.WsMessage{
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
					"message": "Hook triggered successfully",
					"hook":    hookResponse.Name,
					"output":  output,
				})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Hook triggered failed",
					"hook":    hookResponse.Name,
					"error":   errorMsg,
					"output":  output,
				})
			}
		})

		// reload hooks config interface
		hookAPI.POST("/reload-config", func(c *gin.Context) {
			if hook.HookManager == nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Hook manager not initialized",
				})
				return
			}

			// execute actual reload
			err := hook.HookManager.ReloadAllHooks()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":     "Load Hook failed",
					"details":   err.Error(),
					"hookCount": hook.HookManager.GetHookCount(),
				})
				return
			}

			// get loaded hooks count
			hookCount := hook.HookManager.GetHookCount()

			c.JSON(http.StatusOK, gin.H{
				"message":   "Hooks config loaded successfully",
				"hookCount": hookCount,
			})
		})
	}

	// add websocket
	ws := g.Group("/stream")
	ws.Use(wsAuthMiddleware()) // use WebSocket auth middleware, support query parameter token
	{
		// frontend access address: "/stream?token=jwt-token-here"
		ws.GET("", func(c *gin.Context) {
			// Token has been verified by middleware, directly process WebSocket connection
			stream.HandleWebSocket(c)
		})

		// also support path format with ID: /stream/:id
		ws.GET("/:id", func(c *gin.Context) {
			// Token has been verified by middleware, directly process WebSocket connection
			stream.HandleWebSocket(c)
		})
	}

	// version management API group
	versionAPI := g.Group("/version")
	versionAPI.Use(authMiddleware()) // add auth middleware
	{
		// get all projects list
		versionAPI.GET("", version.GetProjects)

		// reload config file interface
		versionAPI.POST("/reload-config", version.ReloadConfig)

		// add project
		versionAPI.POST("/add-project", version.AddProject)

		// delete project
		versionAPI.DELETE("/:name", version.DeleteProject)

		// get project branches list
		versionAPI.GET("/:name/branches", version.GetBranches)

		// get project tags list
		versionAPI.GET("/:name/tags", version.GetTags)

		// sync branches
		versionAPI.POST("/:name/sync-branches", version.SyncBranches)

		// delete branch
		versionAPI.DELETE("/:name/branches/:branchName", version.DeleteBranch)

		// switch branch
		versionAPI.POST("/:name/switch-branch", version.SwitchBranch)

		// switch tag
		versionAPI.POST("/:name/switch-tag", version.SwitchTag)

		// sync tags
		versionAPI.POST("/:name/sync-tags", version.SyncTags)

		// delete tag
		versionAPI.DELETE("/:name/tags/:tagName", version.DeleteTag)

		// delete local tag
		versionAPI.DELETE("/:name/tags/:tagName/local", version.DeleteLocalTag)

		// init git repository
		versionAPI.POST("/:name/init-git", version.InitGitRepository)

		// set remote repository
		versionAPI.POST("/:name/set-remote", version.SetRemote)

		// get remote repository
		versionAPI.GET("/:name/remote", version.GetRemote)

		// get project environment variable file (.env)
		versionAPI.GET("/:name/env", version.GetEnv)

		// save project environment variable file (.env)
		versionAPI.POST("/:name/env", version.SaveEnv)

		// delete project environment variable file (.env)
		versionAPI.DELETE("/:name/env", version.DeleteEnv)

		// save project GitHook configuration
		versionAPI.POST("/:name/githook", version.SaveGitHook)
	}

	// GitHook webhook endpoint
	g.POST("/githook/:name", version.GitHook)

	// plugin management API group (temporary empty interface)
	pluginAPI := g.Group("/plugin")
	pluginAPI.Use(authMiddleware()) // add authentication middleware
	{
		// get all plugins list
		pluginAPI.GET("", func(c *gin.Context) {
			// return empty plugin list
			c.JSON(http.StatusOK, []gin.H{})
		})

		// get specified plugin configuration
		pluginAPI.GET("/:id/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Plugin configuration function not implemented",
			})
		})

		// get specified plugin display information
		pluginAPI.GET("/:id/display", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Plugin display function not implemented",
			})
		})

		// update plugin configuration
		pluginAPI.POST("/:id/config", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Plugin configuration update function not implemented",
			})
		})

		// enable plugin
		pluginAPI.POST("/:id/enable", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Plugin enable function not implemented",
			})
		})

		// disable plugin
		pluginAPI.POST("/:id/disable", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Plugin disable function not implemented",
			})
		})
	}

	// client list API (get all sessions for current user)
	g.GET("/client", authMiddleware(), client.GetClientSessions)

	// delete client API (logout specified session)
	g.DELETE("/client/:id", authMiddleware(), adminMiddleware(), client.DeleteClientSession)

	// delete current user's session
	g.DELETE("/client/current", authMiddleware(), client.DeleteCurrentClientSession)

	// modify current user password API (add to existing current route)
	g.POST("/current/user/password", client.ModifyCurrentClientPassword)

	// save router instance
	routerInstance = g

	return g
}

// GetAppConfig get application configuration
func GetAppConfig() *types.AppConfig {
	return types.GoHookAppConfig
}

// GetUsersConfig get users configuration
func GetUsersConfig() *types.UsersConfig {
	return types.GoHookUsersConfig
}

// GetConfiguredPort get configured port
func GetConfiguredPort() int {
	if types.GoHookAppConfig != nil {
		return types.GoHookAppConfig.Port
	}
	return 9000 // default port
}

// global router instance
var routerInstance *gin.Engine

// GetRouter get current router instance
func GetRouter() *gin.Engine {
	return routerInstance
}
