package router

import (
	"bytes"
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
	g := gin.Default()

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
	g.POST("/client", func(c *gin.Context) {
		client.Login(c)
	})

	// get current user info interface
	g.GET("/current/user", authMiddleware(), func(c *gin.Context) {
		client.GetCurrentUser(c)
	})

	// user management API group
	userAPI := g.Group("/user")
	userAPI.Use(authMiddleware())
	{
		// get all users list (only admin)
		userAPI.GET("", adminMiddleware(), func(c *gin.Context) {
			client.GetAllUsers(c)
		})

		// create user (only admin)
		userAPI.POST("", adminMiddleware(), func(c *gin.Context) {
			client.CreateUser(c)
		})

		// delete user (only admin)
		userAPI.DELETE("/:username", adminMiddleware(), func(c *gin.Context) {
			client.DeleteUser(c)
		})

		// change password
		userAPI.POST("/password", func(c *gin.Context) {
			client.ChangePassword(c)
		})

		// admin reset user password
		userAPI.POST("/:username/reset-password", adminMiddleware(), func(c *gin.Context) {
			client.ResetPassword(c)
		})
	}

	// Hooks API group
	hookAPI := g.Group("/hook")
	hookAPI.Use(authMiddleware()) // add auth middleware
	{
		// get all hooks
		hookAPI.GET("", func(c *gin.Context) {
			hook.GetAllHooks(c)
		})

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
		versionAPI.GET("", func(c *gin.Context) {
			// load config file every time get projects list
			if err := config.LoadVersionConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Load version config failed: " + err.Error()})
				return
			}

			if types.GoHookVersionData == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Version config not loaded"})
				return
			}

			var projects []types.VersionResponse
			for _, proj := range types.GoHookVersionData.Projects {
				if !proj.Enabled {
					continue
				}

				gitStatus, err := version.GetGitStatus(proj.Path)
				if err != nil {
					// if not Git repository, still display but mark as non-Git project
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

		// reload config file interface
		versionAPI.POST("/reload-config", func(c *gin.Context) {
			if err := config.LoadVersionConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Load version config failed: " + err.Error(),
				})
				return
			}

			projectCount := 0
			if types.GoHookVersionData != nil {
				for _, proj := range types.GoHookVersionData.Projects {
					if proj.Enabled {
						projectCount++
					}
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"message":      "Version config loaded successfully",
				"projectCount": projectCount,
			})
		})

		// add project
		versionAPI.POST("/add-project", func(c *gin.Context) {
			var req struct {
				Name        string `json:"name" binding:"required"`
				Path        string `json:"path" binding:"required"`
				Description string `json:"description"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
				return
			}

			// check if project name already exists
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == req.Name {
					c.JSON(http.StatusConflict, gin.H{"error": "Project name already exists"})
					return
				}
			}

			// check if path exists
			if _, err := os.Stat(req.Path); os.IsNotExist(err) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Specified path does not exist"})
				return
			}

			// add new project
			newProject := types.ProjectConfig{
				Name:        req.Name,
				Path:        req.Path,
				Description: req.Description,
				Enabled:     true,
			}

			types.GoHookVersionData.Projects = append(types.GoHookVersionData.Projects, newProject)

			// save config file
			if err := config.SaveVersionConfig(); err != nil {
				// push failed message
				wsMessage := stream.Message{
					Type:      "project_managed",
					Timestamp: time.Now(),
					Data: stream.ProjectManageMessage{
						Action:      "add",
						ProjectName: req.Name,
						ProjectPath: req.Path,
						Success:     false,
						Error:       "Save config failed: " + err.Error(),
					},
				}
				stream.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": "Save config failed: " + err.Error()})
				return
			}

			// push success message
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
				"message": "Project added successfully",
				"project": newProject,
			})
		})

		// delete project
		versionAPI.DELETE("/:name", func(c *gin.Context) {
			projectName := c.Param("name")

			// find project index
			projectIndex := -1
			for i, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName {
					projectIndex = i
					break
				}
			}

			if projectIndex == -1 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			// delete project
			types.GoHookVersionData.Projects = append(types.GoHookVersionData.Projects[:projectIndex], types.GoHookVersionData.Projects[projectIndex+1:]...)

			// save config file
			if err := config.SaveVersionConfig(); err != nil {
				// push failed message
				wsMessage := stream.Message{
					Type:      "project_managed",
					Timestamp: time.Now(),
					Data: stream.ProjectManageMessage{
						Action:      "delete",
						ProjectName: projectName,
						Success:     false,
						Error:       "Save config failed: " + err.Error(),
					},
				}
				stream.Global.Broadcast(wsMessage)

				c.JSON(http.StatusInternalServerError, gin.H{"error": "Save config failed: " + err.Error()})
				return
			}

			// push success message
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
				"message": "Project deleted successfully",
				"name":    projectName,
			})
		})

		// get project branches list
		versionAPI.GET("/:name/branches", func(c *gin.Context) {
			projectName := c.Param("name")

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			branches, err := version.GetBranches(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, branches)
		})

		// get project tags list
		versionAPI.GET("/:name/tags", func(c *gin.Context) {
			projectName := c.Param("name")

			// get filter parameter
			filter := c.Query("filter")

			// get pagination parameter
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

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			allTags, err := version.GetTags(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// if there is filter condition, filter tags
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

			// calculate pagination
			total := len(filteredTags)
			totalPages := (total + limit - 1) / limit
			start := (page - 1) * limit
			end := start + limit

			if start >= total {
				// out of range, return empty array
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

		// sync branches
		versionAPI.POST("/:name/sync-branches", func(c *gin.Context) {
			projectName := c.Param("name")

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			if err := version.SyncBranches(projectPath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Branches synced successfully"})
		})

		// delete branch
		versionAPI.DELETE("/:name/branches/:branchName", func(c *gin.Context) {
			projectName := c.Param("name")
			branchName := c.Param("branchName")

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			if err := version.DeleteBranch(projectPath, branchName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Branch deleted successfully"})
		})

		// switch branch
		versionAPI.POST("/:name/switch-branch", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				Branch string `json:"branch"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
				return
			}

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			if err := version.SwitchBranch(projectPath, req.Branch); err != nil {
				// push failed message
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

			// push success message
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

			c.JSON(http.StatusOK, gin.H{"message": "Branch switched successfully", "branch": req.Branch})
		})

		// switch tag
		versionAPI.POST("/:name/switch-tag", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				Tag string `json:"tag"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
				return
			}

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			if err := version.SwitchTag(projectPath, req.Tag); err != nil {
				// push failed message
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

			// push success message
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

			c.JSON(http.StatusOK, gin.H{"message": "Tag switched successfully", "tag": req.Tag})
		})

		// delete tag
		versionAPI.DELETE("/:name/tags/:tagName", func(c *gin.Context) {
			projectName := c.Param("name")
			tagName := c.Param("tagName")

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			if err := version.DeleteTag(projectPath, tagName); err != nil {
				// push failed message
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

			// push success message
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

			c.JSON(http.StatusOK, gin.H{"message": "Tag deleted successfully"})
		})

		// init git repository
		versionAPI.POST("/:name/init-git", func(c *gin.Context) {
			projectName := c.Param("name")
			fmt.Printf("Received Git initialization request: project name=%s\n", projectName)

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				fmt.Printf("Git initialization failed: project not found, project name=%s\n", projectName)
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			fmt.Printf("Git initialization: project name=%s, path=%s\n", projectName, projectPath)

			if err := version.InitGit(projectPath); err != nil {
				fmt.Printf("Git initialization failed: project name=%s, path=%s, error=%v\n", projectName, projectPath, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			fmt.Printf("Git initialization successful: project name=%s, path=%s\n", projectName, projectPath)
			c.JSON(http.StatusOK, gin.H{"message": "Git repository initialized successfully"})
		})

		// set remote repository
		versionAPI.POST("/:name/set-remote", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				RemoteUrl string `json:"remoteUrl"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
				return
			}

			if req.RemoteUrl == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Remote repository URL cannot be empty"})
				return
			}

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			if err := version.SetRemote(projectPath, req.RemoteUrl); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Remote repository set successfully"})
		})

		// get remote repository
		versionAPI.GET("/:name/remote", func(c *gin.Context) {
			projectName := c.Param("name")

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			remoteURL, err := version.GetRemote(projectPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"url": remoteURL})
		})

		// get project environment variable file (.env)
		versionAPI.GET("/:name/env", func(c *gin.Context) {
			projectName := c.Param("name")

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
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

		// save project environment variable file (.env)
		versionAPI.POST("/:name/env", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				Content string `json:"content" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
				return
			}

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			// validate environment variable file format
			if errors := env.ValidateEnvContent(req.Content); len(errors) > 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Environment variable file format validation failed",
					"details": errors,
				})
				return
			}

			// save environment variable file
			if err := env.SaveEnvFile(projectPath, req.Content); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Environment variable file saved successfully",
				"path":    filepath.Join(projectPath, ".env"),
			})
		})

		// delete project environment variable file (.env)
		versionAPI.DELETE("/:name/env", func(c *gin.Context) {
			projectName := c.Param("name")

			// find project path
			var projectPath string
			for _, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					projectPath = proj.Path
					break
				}
			}

			if projectPath == "" {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			if err := env.DeleteEnvFile(projectPath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Environment variable file deleted successfully",
			})
		})

		// save project GitHook configuration
		versionAPI.POST("/:name/githook", func(c *gin.Context) {
			projectName := c.Param("name")

			var req struct {
				Enhook     bool   `json:"enhook"`
				Hookmode   string `json:"hookmode"`
				Hookbranch string `json:"hookbranch"`
				Hooksecret string `json:"hooksecret"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
				return
			}

			// find project and update configuration
			projectFound := false
			for i, proj := range types.GoHookVersionData.Projects {
				if proj.Name == projectName && proj.Enabled {
					types.GoHookVersionData.Projects[i].Enhook = req.Enhook
					types.GoHookVersionData.Projects[i].Hookmode = req.Hookmode
					types.GoHookVersionData.Projects[i].Hookbranch = req.Hookbranch
					types.GoHookVersionData.Projects[i].Hooksecret = req.Hooksecret
					projectFound = true
					break
				}
			}

			if !projectFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
				return
			}

			// save configuration file
			if err := config.SaveVersionConfig(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Save configuration failed: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "GitHook configuration saved successfully",
			})
		})
	}

	// GitHook webhook endpoint
	g.POST("/githook/:name", func(c *gin.Context) {
		projectName := c.Param("name")

		// find project configuration
		var project *types.ProjectConfig
		for _, proj := range types.GoHookVersionData.Projects {
			if proj.Name == projectName && proj.Enabled && proj.Enhook {
				project = &proj
				break
			}
		}

		if project == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or GitHook not enabled"})
			return
		}

		// read original payload data
		var payloadBody []byte
		if c.Request.Body != nil {
			var err error
			payloadBody, err = io.ReadAll(c.Request.Body)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Read payload failed"})
				return
			}
			// reset body for subsequent use
			c.Request.Body = io.NopCloser(bytes.NewReader(payloadBody))
		}

		// verify webhook password (if set)
		if project.Hooksecret != "" {
			if err := version.VerifyWebhookSignature(c, payloadBody, project.Hooksecret); err != nil {
				log.Printf("GitHook password verification failed: project=%s, error=%v", project.Name, err)
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Password verification failed: " + err.Error()})
				return
			}
		}

		// parse webhook payload (support GitHub, GitLab, etc.)
		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
			return
		}

		// handle GitHook logic
		if err := version.HandleGitHook(project, payload); err != nil {
			log.Printf("GitHook processing failed: project=%s, error=%v", project.Name, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "GitHook processing failed: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "GitHook processing successfully"})
	})

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
	g.GET("/client", authMiddleware(), func(c *gin.Context) {
		username, _ := c.Get("username")
		currentToken, _ := c.Get("token")

		sessions := client.GetClientSessionsByUser(username.(string))

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
	})

	// delete client API (logout specified session)
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

	// delete current user's session
	g.DELETE("/client/current", authMiddleware(), func(c *gin.Context) {
		token := c.GetHeader("X-GoHook-Key")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token not provided"})
			return
		}

		if client.RemoveClientSession(token) {
			c.JSON(http.StatusOK, gin.H{"message": "Current session deleted successfully"})
		} else {
			// even if the session is not found, return success, because the client's goal is to logout
			c.JSON(http.StatusOK, gin.H{"message": "Session not found, but logout process can continue"})
		}
	})

	// modify current user password API (add to existing current route)
	g.POST("/current/user/password", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Password modification function not implemented",
		})
	})

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
