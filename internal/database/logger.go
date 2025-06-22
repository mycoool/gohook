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
func LogHookExecution(hookID, hookName, method, remoteAddr string,
	headers map[string][]string, body string, success bool, output, error string,
	duration int64, userAgent string, queryParams map[string][]string) {

	if globalLogService == nil {
		InitLogService()
	}

	if globalLogService != nil {
		err := globalLogService.CreateHookLog(hookID, hookName, method, remoteAddr,
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

// ScheduleLogCleanup 启动定期日志清理任务
func ScheduleLogCleanup(retentionDays int) {
	if retentionDays <= 0 {
		retentionDays = 30 // 默认保留30天
	}

	go func() {
		ticker := time.NewTicker(24 * time.Hour) // 每天检查一次
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if globalLogService != nil {
					err := globalLogService.CleanOldLogs(retentionDays)
					if err != nil {
						log.Printf("Failed to clean old logs: %v", err)
					} else {
						log.Printf("Successfully cleaned logs older than %d days", retentionDays)
					}
				}
			}
		}
	}()

	log.Printf("Started automatic log cleanup task (retention: %d days)", retentionDays)
}
