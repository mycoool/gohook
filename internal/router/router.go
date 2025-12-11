package router

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/client"
	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/middleware"
	"github.com/mycoool/gohook/internal/stream"
	"github.com/mycoool/gohook/internal/syncnode"
	"github.com/mycoool/gohook/internal/types"
	"github.com/mycoool/gohook/internal/version"
	"github.com/mycoool/gohook/internal/webhook"
)

func InitRouter() *gin.Engine {
	// create engine without default middleware
	g := gin.New()

	// use custom logger middleware, skip requests with "disable_log" tag
	g.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// if context has "disable_log" tag, skip logging
		if param.Keys != nil {
			if noLog, exists := param.Keys["disable_log"]; exists && noLog == true {
				return ""
			}
		}
		// otherwise use default format to record log
		return fmt.Sprintf("[GoHook] %v | %3d | %13v | %15s | %-7s %#v\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
			param.ErrorMessage,
		)
	}))

	// use Recovery middleware
	g.Use(gin.Recovery())

	// use IP middleware, support real IP in proxy environment
	g.Use(middleware.IPMiddleware())

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
	g.GET("/message", middleware.DisableLogMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"messages": []interface{}{},
			"paging":   gin.H{},
		})
	})

	// login interface - support Basic authentication
	g.POST("/client", client.Login)

	// token renew interface
	g.POST("/client/renew", middleware.AuthMiddleware(), client.HandleRenewToken)

	// get current user info interface
	g.GET("/current/user", middleware.DisableLogMiddleware(), middleware.AuthMiddleware(), client.GetCurrentUser)

	// get application config interface (public, no auth required for panel_alias)
	g.GET("/app/config", middleware.DisableLogMiddleware(), config.HandleGetAppConfig)

	// user management API group
	userAPI := g.Group("/user")
	userAPI.Use(middleware.AuthMiddleware(), middleware.DisableLogMiddleware())
	{
		// get all users list (only admin)
		userAPI.GET("", middleware.AdminMiddleware(), client.GetAllUsers)

		// create user (only admin)
		userAPI.POST("", middleware.AdminMiddleware(), client.CreateUser)

		// delete user (only admin)
		userAPI.DELETE("/:username", middleware.AdminMiddleware(), client.DeleteUser)

		// change password
		userAPI.POST("/password", client.ChangePassword)

		// admin reset user password
		userAPI.POST("/:username/reset-password", middleware.AdminMiddleware(), client.ResetPassword)
	}

	// Hooks API group
	hookAPI := g.Group("/hook")
	hookAPI.Use(middleware.AuthMiddleware(), middleware.DisableLogMiddleware()) // add auth middleware
	{
		// get all hooks
		hookAPI.GET("", webhook.HandleGetAllHooks)

		// get single hook details (for editing)
		hookAPI.GET("/:id", webhook.HandleGetHook)

		// trigger hook (test interface)
		hookAPI.POST("/:id/trigger", webhook.HandleTriggerHook)

		// reload hooks config interface
		hookAPI.POST("/reload-config", webhook.HandleReloadHooksConfig)

		// hook configuration management - split into multiple endpoints
		hookAPI.POST("", webhook.HandleCreateHook)                         // create new hook
		hookAPI.PUT("/:id/basic", webhook.HandleUpdateHookBasic)           // update basic info
		hookAPI.PUT("/:id/parameters", webhook.HandleUpdateHookParameters) // update parameters
		hookAPI.PUT("/:id/triggers", webhook.HandleUpdateHookTriggers)     // update trigger rules
		hookAPI.PUT("/:id/response", webhook.HandleUpdateHookResponse)     // update response config

		// script management
		hookAPI.GET("/:id/script", webhook.HandleGetHookScript)
		hookAPI.POST("/:id/script", webhook.HandleSaveHookScript)
		hookAPI.PUT("/:id/execute-command", webhook.HandleUpdateHookExecuteCommand)

		// delete hook
		hookAPI.DELETE("/:id", webhook.HandleDeleteHook)
	}

	// add websocket
	ws := g.Group("/stream")
	ws.Use(middleware.WsAuthMiddleware(), middleware.DisableLogMiddleware()) // use WebSocket auth middleware, support query parameter token
	{
		// frontend access address: "/stream?token=jwt-token-here"
		ws.GET("", stream.HandleWebSocket)

		// also support path format with ID: /stream/:id
		ws.GET("/:id", stream.HandleWebSocket)
	}

	// version management API group
	versionAPI := g.Group("/version")
	versionAPI.Use(middleware.AuthMiddleware(), middleware.DisableLogMiddleware()) // add auth middleware
	{
		// get all projects list
		versionAPI.GET("", version.HandleGetProjects)

		// reload config file interface
		versionAPI.POST("/reload-config", version.HandleReloadConfig)

		// add project
		versionAPI.POST("/add-project", version.HandleAddProject)

		// project-specific routes (more specific paths first to avoid conflicts)
		// get project branches list
		versionAPI.GET("/:name/branches", version.HandleGetBranches)

		// get project tags list
		versionAPI.GET("/:name/tags", version.HandleGetTags)

		// sync branches
		versionAPI.POST("/:name/sync-branches", version.HandleSyncBranches)

		// delete branch
		versionAPI.DELETE("/:name/branches/:branchName", version.HandleDeleteBranch)

		// delete local branch
		versionAPI.DELETE("/:name/branches/:branchName/local", version.HandleDeleteLocalBranch)

		// switch branch
		versionAPI.POST("/:name/switch-branch", version.HandleSwitchBranch)

		// switch tag
		versionAPI.POST("/:name/switch-tag", version.HandleSwitchTag)

		// sync tags
		versionAPI.POST("/:name/sync-tags", version.HandleSyncTags)

		// delete tag
		versionAPI.DELETE("/:name/tags/:tagName", version.HandleDeleteTag)

		// delete local tag
		versionAPI.DELETE("/:name/tags/:tagName/local", version.HandleDeleteLocalTag)

		// init git repository
		versionAPI.POST("/:name/init-git", version.HandleInitGitRepository)

		// set remote repository
		versionAPI.POST("/:name/set-remote", version.HandleSetRemote)

		// get remote repository
		versionAPI.GET("/:name/remote", version.HandleGetRemote)

		// get project environment variable file (.env)
		versionAPI.GET("/:name/env", version.HandleGetEnv)

		// save project environment variable file (.env)
		versionAPI.POST("/:name/env", version.HandleSaveEnv)

		// delete project environment variable file (.env)
		versionAPI.DELETE("/:name/env", version.HandleDeleteEnv)

		// save project GitHook configuration
		versionAPI.POST("/:name/githook", version.HandleSaveGitHook)

		// project management routes (less specific paths last)
		// edit project
		versionAPI.PUT("/:name", version.HandleEditProject)

		// delete project
		versionAPI.DELETE("/:name", version.HandleDeleteProject)
	}

	// sync node management API
	syncAPI := g.Group("/api/sync")
	syncAPI.Use(middleware.AuthMiddleware(), middleware.DisableLogMiddleware())
	{
		nodeAPI := syncAPI.Group("/nodes")
		nodeAPI.GET("", syncnode.HandleListNodes)
		nodeAPI.POST("", syncnode.HandleCreateNode)
		nodeAPI.GET("/:id", syncnode.HandleGetNode)
		nodeAPI.PUT("/:id", syncnode.HandleUpdateNode)
		nodeAPI.DELETE("/:id", syncnode.HandleDeleteNode)
		nodeAPI.POST("/:id/install", syncnode.HandleInstallNode)
	}

	// GitHook webhook endpoint
	g.POST("/githook/:name", version.HandleGitHook)

	// plugin management API group (temporary empty interface)
	pluginAPI := g.Group("/plugin")
	pluginAPI.Use(middleware.AuthMiddleware(), middleware.DisableLogMiddleware()) // add authentication middleware
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

	// log management API group
	logAPI := g.Group("/api/logs")
	logAPI.Use(middleware.AuthMiddleware(), middleware.DisableLogMiddleware()) // add authentication middleware
	{
		// get log list
		logAPI.GET("", HandleGetLogs)

		// export log
		logAPI.GET("/export", HandleExportLogs)

		// clean old logs
		logAPI.DELETE("/cleanup", HandleCleanupLogs)
	}

	// system configuration management API group
	systemRouter := NewSystemRouter()
	systemRouter.RegisterSystemRoutes(&g.RouterGroup)

	// client list API (get all sessions for current user)
	g.GET("/client", middleware.AuthMiddleware(), client.HandleGetClientSessions)

	// delete client API (logout specified session)
	g.DELETE("/client/:id", middleware.AuthMiddleware(), middleware.AdminMiddleware(), client.HandleDeleteClientSession)

	// delete current user's session
	g.DELETE("/client/current", middleware.AuthMiddleware(), client.HandleDeleteCurrentClientSession)

	// modify current user password API (add to existing current route)
	g.POST("/current/user/password", client.HandleModifyCurrentClientPassword)

	// save router instance
	routerInstance = g

	return g
}

// global router instance
var routerInstance *gin.Engine

// GetRouter get current router instance
func GetRouter() *gin.Engine {
	return routerInstance
}
