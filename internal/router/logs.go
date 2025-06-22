package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/database"
)

// LogRouter 日志路由处理器
type LogRouter struct {
	logService *database.LogService
}

// NewLogRouter 创建日志路由处理器
func NewLogRouter() *LogRouter {
	return &LogRouter{
		logService: database.NewLogService(),
	}
}

// RegisterLogRoutes 注册日志相关路由
func (lr *LogRouter) RegisterLogRoutes(rg *gin.RouterGroup) {
	// Webhook日志路由 - 用户手动建立规则和脚本的webhook
	webhookLogsGroup := rg.Group("/logs/webhooks")
	{
		webhookLogsGroup.GET("", lr.GetWebhookLogs)
		webhookLogsGroup.GET("/stats", lr.GetWebhookLogStats)
	}

	// GitHook日志路由 - 简易版本的githook
	githookLogsGroup := rg.Group("/logs/githook")
	{
		githookLogsGroup.GET("", lr.GetGitHookLogs)
		githookLogsGroup.GET("/stats", lr.GetGitHookLogStats)
	}

	// 用户活动日志路由
	userLogsGroup := rg.Group("/logs/users")
	{
		userLogsGroup.GET("", lr.GetUserActivities)
		userLogsGroup.GET("/stats", lr.GetUserActivityStats)
	}

	// 系统日志路由
	systemLogsGroup := rg.Group("/logs/system")
	{
		systemLogsGroup.GET("", lr.GetSystemLogs)
	}

	// 项目活动日志路由
	projectLogsGroup := rg.Group("/logs/projects")
	{
		projectLogsGroup.GET("", lr.GetProjectActivities)
	}

	// 日志管理路由
	logManagementGroup := rg.Group("/logs")
	{
		logManagementGroup.DELETE("/cleanup", lr.CleanupLogs)
	}
}

// GetWebhookLogs 获取Webhook执行日志
func (lr *LogRouter) GetWebhookLogs(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// 解析过滤参数
	hookID := c.Query("hook_id")
	hookName := c.Query("hook_name")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// 解析时间参数
	var startTime, endTime *time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", endStr); err == nil {
			endTime = &t
		}
	}

	// 查询数据 (Webhook类型)
	logs, total, err := lr.logService.GetHookLogs(page, pageSize, hookID, hookName, "webhook", success, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":        logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetWebhookLogStats 获取Webhook日志统计
func (lr *LogRouter) GetWebhookLogStats(c *gin.Context) {
	// 解析时间参数
	var startTime, endTime *time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", endStr); err == nil {
			endTime = &t
		}
	}

	stats, err := lr.logService.GetHookLogStats("webhook", startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetSystemLogs 获取系统日志
func (lr *LogRouter) GetSystemLogs(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// 解析过滤参数
	level := c.Query("level")
	category := c.Query("category")
	userID := c.Query("user_id")

	// 解析时间参数
	var startTime, endTime *time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", endStr); err == nil {
			endTime = &t
		}
	}

	// 查询数据
	logs, total, err := lr.logService.GetSystemLogs(page, pageSize, level, category, userID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":        logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetUserActivities 获取用户活动记录
func (lr *LogRouter) GetUserActivities(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// 解析过滤参数
	username := c.Query("username")
	action := c.Query("action")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// 解析时间参数
	var startTime, endTime *time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", endStr); err == nil {
			endTime = &t
		}
	}

	// 查询数据
	activities, total, err := lr.logService.GetUserActivities(page, pageSize, username, action, success, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities":  activities,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetProjectActivities 获取项目活动记录
func (lr *LogRouter) GetProjectActivities(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// 解析过滤参数
	projectName := c.Query("project_name")
	action := c.Query("action")
	username := c.Query("username")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// 解析时间参数
	var startTime, endTime *time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", endStr); err == nil {
			endTime = &t
		}
	}

	// 查询数据
	activities, total, err := lr.logService.GetProjectActivities(page, pageSize, projectName, action, username, success, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"activities":  activities,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// CleanupLogs 清理旧日志
func (lr *LogRouter) CleanupLogs(c *gin.Context) {
	// 从查询参数获取保留天数，默认30天
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter"})
		return
	}

	err := lr.logService.CleanOldLogs(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Old logs cleaned successfully"})
}

// GetGitHookLogs 获取GitHook执行日志
func (lr *LogRouter) GetGitHookLogs(c *gin.Context) {
	// 解析分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// 解析过滤参数
	hookID := c.Query("hook_id")
	hookName := c.Query("hook_name")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// 解析时间参数
	var startTime, endTime *time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", endStr); err == nil {
			endTime = &t
		}
	}

	// 查询数据 (GitHook类型)
	logs, total, err := lr.logService.GetHookLogs(page, pageSize, hookID, hookName, "githook", success, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":        logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetGitHookLogStats 获取GitHook日志统计
func (lr *LogRouter) GetGitHookLogStats(c *gin.Context) {
	// 解析时间参数
	var startTime, endTime *time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", endStr); err == nil {
			endTime = &t
		}
	}

	stats, err := lr.logService.GetHookLogStats("githook", startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetUserActivityStats 获取用户活动统计
func (lr *LogRouter) GetUserActivityStats(c *gin.Context) {
	// 解析过滤参数
	username := c.Query("username")

	// 解析时间参数
	var startTime, endTime *time.Time
	if startStr := c.Query("start_time"); startStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", startStr); err == nil {
			startTime = &t
		}
	}
	if endStr := c.Query("end_time"); endStr != "" {
		if t, err := time.Parse("2006-01-02T15:04:05Z", endStr); err == nil {
			endTime = &t
		}
	}

	stats, err := lr.logService.GetUserActivityStats(username, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
