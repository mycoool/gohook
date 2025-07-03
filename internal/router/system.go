package router

import (
	"net/http"

	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/middleware"
	"github.com/mycoool/gohook/internal/types"

	"github.com/gin-gonic/gin"
)

// SystemRouter system config router
type SystemRouter struct{}

// NewSystemRouter create system config router instance
func NewSystemRouter() *SystemRouter {
	return &SystemRouter{}
}

// RegisterSystemRoutes register system config router
func (sr *SystemRouter) RegisterSystemRoutes(rg *gin.RouterGroup) {
	systemGroup := rg.Group("/system")
	systemGroup.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware(), middleware.DisableLogMiddleware())
	{
		systemGroup.GET("/config", sr.GetSystemConfig)
		systemGroup.PUT("/config", sr.UpdateSystemConfig)
	}
}

// GetSystemConfig get system config
func (sr *SystemRouter) GetSystemConfig(c *gin.Context) {
	// check admin permission
	_, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized access"})
		return
	}

	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "need admin permission"})
		return
	}

	systemConfig, err := config.LoadSystemConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load config failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, systemConfig)
}

// UpdateSystemConfig update system config
func (sr *SystemRouter) UpdateSystemConfig(c *gin.Context) {
	// check admin permission
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized access"})
		return
	}

	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "need admin permission"})
		return
	}

	var newConfig config.SystemConfig
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request data: " + err.Error()})
		return
	}

	// load original config for record change
	oldConfig, err := config.LoadSystemConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load original config failed: " + err.Error()})
		return
	}

	// if JWT secret is empty, use original config's JWT secret
	if newConfig.JWTSecret == "" {
		newConfig.JWTSecret = oldConfig.JWTSecret
	}

	// save new config
	if err := config.SaveSystemConfig(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "save config failed: " + err.Error()})
		// record failed log
		database.LogUserAction(
			username.(string),
			"UPDATE_SYSTEM_CONFIG",
			"system_config",
			"update system config failed",
			c.ClientIP(),
			c.GetHeader("User-Agent"),
			false,
			gin.H{
				"error":      err.Error(),
				"new_config": newConfig,
			},
		)
		return
	}

	// record success log
	database.LogUserAction(
		username.(string),
		"UPDATE_SYSTEM_CONFIG",
		"system_config",
		"update system config success",
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		true,
		gin.H{
			"old_config": oldConfig,
			"new_config": newConfig,
		},
	)

	// update types.GoHookAppConfig in memory
	types.UpdateAppConfig(newConfig)

	c.JSON(http.StatusOK, gin.H{"message": "config updated successfully"})
}
