package router

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/database"
)

// LogRouter log router handler
type LogRouter struct {
	logService *database.LogService
}

// NewLogRouter create log router handler
func NewLogRouter() *LogRouter {
	return &LogRouter{
		logService: database.NewLogService(),
	}
}

// RegisterLogRoutes register log related routes
func (lr *LogRouter) RegisterLogRoutes(rg *gin.RouterGroup) {
	// Webhook log route - user-defined webhook
	webhookLogsGroup := rg.Group("/logs/webhooks")
	{
		webhookLogsGroup.GET("", lr.GetWebhookLogs)
		webhookLogsGroup.GET("/stats", lr.GetWebhookLogStats)
	}

	// GitHook log route - simple githook
	githookLogsGroup := rg.Group("/logs/githook")
	{
		githookLogsGroup.GET("", lr.GetGitHookLogs)
		githookLogsGroup.GET("/stats", lr.GetGitHookLogStats)
	}

	// user activity log route
	userLogsGroup := rg.Group("/logs/users")
	{
		userLogsGroup.GET("", lr.GetUserActivities)
		userLogsGroup.GET("/stats", lr.GetUserActivityStats)
	}

	// system log route
	systemLogsGroup := rg.Group("/logs/system")
	{
		systemLogsGroup.GET("", lr.GetSystemLogs)
	}

	// project activity log route
	projectLogsGroup := rg.Group("/logs/projects")
	{
		projectLogsGroup.GET("", lr.GetProjectActivities)
	}

	// log management route
	logManagementGroup := rg.Group("/logs")
	{
		logManagementGroup.DELETE("/cleanup", lr.CleanupLogs)
	}
}

// GetWebhookLogs get webhook execution log
func (lr *LogRouter) GetWebhookLogs(c *gin.Context) {
	// parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// parse filter parameters
	hookID := c.Query("hook_id")
	hookName := c.Query("hook_name")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// parse time parameters
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

	// query data (Webhook type)
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

// GetWebhookLogStats get webhook log stats
func (lr *LogRouter) GetWebhookLogStats(c *gin.Context) {
	// parse time parameters
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

// GetSystemLogs get system log
func (lr *LogRouter) GetSystemLogs(c *gin.Context) {
	// parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// parse filter parameters
	level := c.Query("level")
	category := c.Query("category")
	userID := c.Query("user_id")

	// parse time parameters
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

	// query data
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

// GetUserActivities get user activity record
func (lr *LogRouter) GetUserActivities(c *gin.Context) {
	// parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// parse filter parameters
	username := c.Query("username")
	action := c.Query("action")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// parse time parameters
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

	// query data
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

// GetProjectActivities get project activity record
func (lr *LogRouter) GetProjectActivities(c *gin.Context) {
	// parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// parse filter parameters
	projectName := c.Query("project_name")
	action := c.Query("action")
	username := c.Query("username")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// parse time parameters
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

	// query data
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

// CleanupLogs clean old logs
func (lr *LogRouter) CleanupLogs(c *gin.Context) {
	// get retention days from query parameters, default 30 days
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

// GetGitHookLogs get GitHook execution log
func (lr *LogRouter) GetGitHookLogs(c *gin.Context) {
	// parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// parse filter parameters
	hookID := c.Query("hook_id")
	hookName := c.Query("hook_name")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// parse time parameters
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

	// query data (GitHook type)
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

// GetGitHookLogStats get GitHook log stats
func (lr *LogRouter) GetGitHookLogStats(c *gin.Context) {
	// parse time parameters
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

// GetUserActivityStats get user activity stats
func (lr *LogRouter) GetUserActivityStats(c *gin.Context) {
	// parse filter parameters
	username := c.Query("username")

	// parse time parameters
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

// HandleGetLogs unified log query interface
func HandleGetLogs(c *gin.Context) {
	// parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if pageSize > 100 {
		pageSize = 100
	}

	// parse filter parameters
	logType := c.Query("type")
	level := c.Query("level")
	category := c.Query("category")
	search := c.Query("search")
	user := c.Query("user")
	project := c.Query("project")

	var success *bool
	if successStr := c.Query("success"); successStr != "" {
		if successBool, err := strconv.ParseBool(successStr); err == nil {
			success = &successBool
		}
	}

	// parse time parameters
	var startTime, endTime *time.Time
	if startDate := c.Query("startDate"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			startTime = &t
		}
	}
	if endDate := c.Query("endDate"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			endTime = &t
		}
	}

	logService := database.NewLogService()

	// call different query methods based on type
	var logs interface{}
	var total int64
	var err error

	switch logType {
	case "hook":
		logs, total, err = logService.GetHookLogsForAPI(page, pageSize, "", "", "", success, startTime, endTime)
	case "system":
		logs, total, err = logService.GetSystemLogsForAPI(page, pageSize, level, category, user, startTime, endTime)
	case "user":
		logs, total, err = logService.GetUserActivitiesForAPI(page, pageSize, user, "", success, startTime, endTime)
	case "project":
		logs, total, err = logService.GetProjectActivitiesForAPI(page, pageSize, project, "", user, success, startTime, endTime)
	default:
		// query all types of logs (here need new method)
		logs, total, err = logService.GetAllLogs(page, pageSize, level, search, startTime, endTime)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	hasMore := int64(page*pageSize) < total

	c.JSON(http.StatusOK, gin.H{
		"logs":     logs,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
		"hasMore":  hasMore,
	})
}

// HandleExportLogs export log interface
func HandleExportLogs(c *gin.Context) {
	// parse filter parameters
	logType := c.Query("type")
	level := c.Query("level")
	search := c.Query("search")

	// parse time parameters
	var startTime, endTime *time.Time
	if startDate := c.Query("startDate"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			startTime = &t
		}
	}
	if endDate := c.Query("endDate"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			endTime = &t
		}
	}

	logService := database.NewLogService()

	// export CSV format logs
	csvData, err := logService.ExportLogsToCSV(logType, level, search, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// set response header
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=logs.csv")

	c.String(http.StatusOK, csvData)
}

// HandleCleanupLogs clean log interface
func HandleCleanupLogs(c *gin.Context) {
	// get retention days from query parameters, default 30 days
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter"})
		return
	}

	logService := database.NewLogService()
	err := logService.CleanOldLogs(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Old logs cleaned successfully"})
}
