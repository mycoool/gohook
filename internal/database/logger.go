package database

import (
	"log"
	"time"
)

var globalLogService *LogService

// InitLogService 初始化全局日志服务
func InitLogService() {
	globalLogService = NewLogService()
}

// LogHookExecution 记录Hook执行日志（全局函数）
func LogHookExecution(hookID, hookName, hookType, method, remoteAddr string,
	headers map[string][]string, body string, success bool, output, error string,
	duration int64, userAgent string, queryParams map[string][]string) {

	if globalLogService == nil {
		InitLogService()
	}

	if globalLogService != nil {
		err := globalLogService.CreateHookLog(hookID, hookName, hookType, method, remoteAddr,
			headers, body, success, output, error, duration, userAgent, queryParams)
		if err != nil {
			log.Printf("Failed to log hook execution: %v", err)
		}
	}
}

// LogSystemEvent 记录系统事件日志（全局函数）
func LogSystemEvent(level, category, message string, details interface{},
	userID, ipAddress, userAgent string) {

	if globalLogService == nil {
		InitLogService()
	}

	if globalLogService != nil {
		err := globalLogService.CreateSystemLog(level, category, message, details,
			userID, ipAddress, userAgent)
		if err != nil {
			log.Printf("Failed to log system event: %v", err)
		}
	}
}

// LogUserAction 记录用户活动（全局函数）
func LogUserAction(username, action, resource, description,
	ipAddress, userAgent string, success bool, details interface{}) {

	if globalLogService == nil {
		InitLogService()
	}

	if globalLogService != nil {
		err := globalLogService.CreateUserActivity(username, action, resource, description,
			ipAddress, userAgent, success, details)
		if err != nil {
			log.Printf("Failed to log user activity: %v", err)
		}
	}
}

// LogProjectAction 记录项目活动（全局函数）
func LogProjectAction(projectName, action, oldValue, newValue,
	username string, success bool, error, commitHash, description, ipAddress string) {

	if globalLogService == nil {
		InitLogService()
	}

	if globalLogService != nil {
		err := globalLogService.CreateProjectActivity(projectName, action, oldValue, newValue,
			username, success, error, commitHash, description, ipAddress)
		if err != nil {
			log.Printf("Failed to log project activity: %v", err)
		}
	}
}

// LogHookManagement 记录Hook管理操作日志（全局函数）
func LogHookManagement(action, hookID, hookName, username, ipAddress, userAgent string, success bool, details interface{}) {
	if globalLogService == nil {
		InitLogService()
	}

	if globalLogService != nil {
		resource := "hook:" + hookID
		description := getHookManagementDescription(action, hookName)

		err := globalLogService.CreateUserActivity(username, action, resource, description,
			ipAddress, userAgent, success, details)
		if err != nil {
			log.Printf("Failed to log hook management activity: %v", err)
		}
	}
}

// getHookManagementDescription 获取Hook管理操作的描述
func getHookManagementDescription(action, hookName string) string {
	switch action {
	case "CREATE_HOOK":
		return "创建Hook: " + hookName
	case "UPDATE_HOOK_BASIC":
		return "更新Hook基本信息: " + hookName
	case "UPDATE_HOOK_PARAMETERS":
		return "更新Hook参数配置: " + hookName
	case "UPDATE_HOOK_TRIGGERS":
		return "更新Hook触发规则: " + hookName
	case "UPDATE_HOOK_RESPONSE":
		return "更新Hook响应配置: " + hookName
	case "UPDATE_HOOK_SCRIPT":
		return "更新Hook脚本: " + hookName
	case "DELETE_HOOK":
		return "删除Hook: " + hookName
	default:
		return "Hook管理操作: " + hookName
	}
}

// ScheduleLogCleanup 启动定期日志清理任务
func ScheduleLogCleanup(retentionDays int) {
	if retentionDays <= 0 {
		retentionDays = 30 // 默认保留30天
	}

	go func() {
		ticker := time.NewTicker(24 * time.Hour) // 每天检查一次
		defer ticker.Stop()

		for range ticker.C {
			if globalLogService != nil {
				err := globalLogService.CleanOldLogs(retentionDays)
				if err != nil {
					log.Printf("Failed to clean old logs: %v", err)
				} else {
					log.Printf("Successfully cleaned logs older than %d days", retentionDays)
				}
			}
		}
	}()

	log.Printf("Started automatic log cleanup task (retention: %d days)", retentionDays)
}
