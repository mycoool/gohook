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

// LogHookManagement record hook management operation log (global function)
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

// getHookManagementDescription get hook management operation description
func getHookManagementDescription(action, hookName string) string {
	switch action {
	case "CREATE_HOOK":
		return "create hook: " + hookName
	case "UPDATE_HOOK_BASIC":
		return "update hook basic info: " + hookName
	case "UPDATE_HOOK_PARAMETERS":
		return "update hook parameters: " + hookName
	case "UPDATE_HOOK_TRIGGERS":
		return "update hook triggers: " + hookName
	case "UPDATE_HOOK_RESPONSE":
		return "update hook response: " + hookName
	case "UPDATE_HOOK_SCRIPT":
		return "update hook script: " + hookName
	case "DELETE_HOOK":
		return "delete hook: " + hookName
	default:
		return "hook management operation: " + hookName
	}
}

// ScheduleLogCleanup start periodic log cleanup task
func ScheduleLogCleanup(retentionDays int) {
	if retentionDays <= 0 {
		retentionDays = 30 // default retention 30 days
	}

	go func() {
		ticker := time.NewTicker(24 * time.Hour) // check once per day
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
