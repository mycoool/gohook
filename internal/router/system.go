package router

import (
	"net/http"

	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SystemRouter 系统配置路由
type SystemRouter struct{}

// NewSystemRouter 创建系统配置路由实例
func NewSystemRouter() *SystemRouter {
	return &SystemRouter{}
}

// RegisterSystemRoutes 注册系统配置路由
func (sr *SystemRouter) RegisterSystemRoutes(rg *gin.RouterGroup) {
	systemGroup := rg.Group("/api/v1/system")
	systemGroup.Use(middleware.AuthMiddleware(), middleware.AdminMiddleware(), middleware.DisableLogMiddleware())
	{
		systemGroup.GET("/config", sr.GetSystemConfig)
		systemGroup.PUT("/config", sr.UpdateSystemConfig)
	}
}

// GetSystemConfig 获取系统配置
func (sr *SystemRouter) GetSystemConfig(c *gin.Context) {
	// 检查管理员权限
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	systemConfig, err := config.LoadSystemConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载配置失败: " + err.Error()})
		return
	}

	// 记录日志
	database.LogUserAction(
		username.(string),
		"VIEW_SYSTEM_CONFIG",
		"system_config",
		"查看系统配置",
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		true,
		nil,
	)

	c.JSON(http.StatusOK, systemConfig)
}

// UpdateSystemConfig 更新系统配置
func (sr *SystemRouter) UpdateSystemConfig(c *gin.Context) {
	// 检查管理员权限
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权访问"})
		return
	}

	role, exists := c.Get("role")
	if !exists || role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
		return
	}

	var newConfig config.SystemConfig
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 获取原配置用于记录变更
	oldConfig, err := config.LoadSystemConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载原配置失败: " + err.Error()})
		return
	}

	// 如果JWT密钥为空，表示不修改，使用原配置的JWT密钥
	if newConfig.JWTSecret == "" {
		newConfig.JWTSecret = oldConfig.JWTSecret
	}

	// 保存新配置
	if err := config.SaveSystemConfig(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "保存配置失败: " + err.Error()})
		// 记录失败日志
		database.LogUserAction(
			username.(string),
			"UPDATE_SYSTEM_CONFIG",
			"system_config",
			"更新系统配置失败",
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

	// 记录成功日志
	database.LogUserAction(
		username.(string),
		"UPDATE_SYSTEM_CONFIG",
		"system_config",
		"更新系统配置成功",
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		true,
		gin.H{
			"old_config": oldConfig,
			"new_config": newConfig,
		},
	)

	c.JSON(http.StatusOK, gin.H{"message": "配置更新成功"})
}
