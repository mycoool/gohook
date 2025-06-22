package database

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
)

// LogService 日志服务
type LogService struct {
	db *gorm.DB
}

// NewLogService 创建日志服务实例
func NewLogService() *LogService {
	return &LogService{db: GetDB()}
}

// CreateHookLog 创建Hook执行日志
func (s *LogService) CreateHookLog(hookID, hookName, hookType, method, remoteAddr string,
	headers map[string][]string, body string, success bool, output, error string,
	duration int64, userAgent string, queryParams map[string][]string) error {

	headersJSON, _ := json.Marshal(headers)
	queryParamsJSON, _ := json.Marshal(queryParams)

	log := &HookLog{
		HookID:      hookID,
		HookName:    hookName,
		HookType:    hookType,
		Method:      method,
		RemoteAddr:  remoteAddr,
		Headers:     string(headersJSON),
		Body:        body,
		Success:     success,
		Output:      output,
		Error:       error,
		Duration:    duration,
		UserAgent:   userAgent,
		QueryParams: string(queryParamsJSON),
	}

	return s.db.Create(log).Error
}

// CreateSystemLog 创建系统日志
func (s *LogService) CreateSystemLog(level, category, message string, details interface{},
	userID, ipAddress, userAgent string) error {

	var detailsJSON string
	if details != nil {
		detailsBytes, _ := json.Marshal(details)
		detailsJSON = string(detailsBytes)
	}

	log := &SystemLog{
		Level:     level,
		Category:  category,
		Message:   message,
		Details:   detailsJSON,
		UserID:    userID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	return s.db.Create(log).Error
}

// CreateUserActivity 创建用户活动记录
func (s *LogService) CreateUserActivity(username, action, resource, description,
	ipAddress, userAgent string, success bool, details interface{}) error {

	var detailsJSON string
	if details != nil {
		detailsBytes, _ := json.Marshal(details)
		detailsJSON = string(detailsBytes)
	}

	activity := &UserActivity{
		Username:    username,
		Action:      action,
		Resource:    resource,
		Description: description,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		Success:     success,
		Details:     detailsJSON,
	}

	return s.db.Create(activity).Error
}

// CreateProjectActivity 创建项目活动记录
func (s *LogService) CreateProjectActivity(projectName, action, oldValue, newValue,
	username string, success bool, error, commitHash, description, ipAddress string) error {

	activity := &ProjectActivity{
		ProjectName: projectName,
		Action:      action,
		OldValue:    oldValue,
		NewValue:    newValue,
		Username:    username,
		Success:     success,
		Error:       error,
		CommitHash:  commitHash,
		Description: description,
		IPAddress:   ipAddress,
	}

	return s.db.Create(activity).Error
}

// GetHookLogs 获取Hook日志列表（支持分页和过滤）
func (s *LogService) GetHookLogs(page, pageSize int, hookID, hookName, hookType string, success *bool,
	startTime, endTime *time.Time) ([]HookLog, int64, error) {

	query := s.db.Model(&HookLog{})

	// 添加过滤条件
	if hookID != "" {
		query = query.Where("hook_id = ?", hookID)
	}
	if hookName != "" {
		query = query.Where("hook_name LIKE ?", "%"+hookName+"%")
	}
	if hookType != "" {
		query = query.Where("hook_type = ?", hookType)
	}
	if success != nil {
		query = query.Where("success = ?", *success)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var logs []HookLog
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error

	return logs, total, err
}

// GetSystemLogs 获取系统日志列表
func (s *LogService) GetSystemLogs(page, pageSize int, level, category, userID string,
	startTime, endTime *time.Time) ([]SystemLog, int64, error) {

	query := s.db.Model(&SystemLog{})

	// 添加过滤条件
	if level != "" {
		query = query.Where("level = ?", level)
	}
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var logs []SystemLog
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error

	return logs, total, err
}

// GetUserActivities 获取用户活动记录
func (s *LogService) GetUserActivities(page, pageSize int, username, action string,
	success *bool, startTime, endTime *time.Time) ([]UserActivity, int64, error) {

	query := s.db.Model(&UserActivity{})

	// 添加过滤条件
	if username != "" {
		query = query.Where("username = ?", username)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if success != nil {
		query = query.Where("success = ?", *success)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var activities []UserActivity
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&activities).Error

	return activities, total, err
}

// GetProjectActivities 获取项目活动记录
func (s *LogService) GetProjectActivities(page, pageSize int, projectName, action, username string,
	success *bool, startTime, endTime *time.Time) ([]ProjectActivity, int64, error) {

	query := s.db.Model(&ProjectActivity{})

	// 添加过滤条件
	if projectName != "" {
		query = query.Where("project_name = ?", projectName)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if username != "" {
		query = query.Where("username = ?", username)
	}
	if success != nil {
		query = query.Where("success = ?", *success)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	var activities []ProjectActivity
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&activities).Error

	return activities, total, err
}

// GetHookLogStats 获取Hook日志统计信息
func (s *LogService) GetHookLogStats(hookType string, startTime, endTime *time.Time) (map[string]interface{}, error) {
	query := s.db.Model(&HookLog{})

	if hookType != "" {
		query = query.Where("hook_type = ?", hookType)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	stats := make(map[string]interface{})

	// 总执行次数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	if total == 0 {
		stats["success"] = 0
		stats["success_rate"] = 0
		stats["avg_duration"] = 0
		return stats, nil
	}

	// 成功次数
	var success int64
	successQuery := s.db.Model(&HookLog{}).Where("success = ?", true)
	if hookType != "" {
		successQuery = successQuery.Where("hook_type = ?", hookType)
	}
	if startTime != nil {
		successQuery = successQuery.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		successQuery = successQuery.Where("created_at <= ?", *endTime)
	}
	if err := successQuery.Count(&success).Error; err != nil {
		return nil, err
	}
	stats["success"] = success
	stats["success_rate"] = float64(success) / float64(total) * 100

	// 平均执行时长
	var avgDuration float64
	avgQuery := s.db.Model(&HookLog{}).Select("AVG(duration)")
	if hookType != "" {
		avgQuery = avgQuery.Where("hook_type = ?", hookType)
	}
	if startTime != nil {
		avgQuery = avgQuery.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		avgQuery = avgQuery.Where("created_at <= ?", *endTime)
	}
	if err := avgQuery.Row().Scan(&avgDuration); err != nil {
		return nil, err
	}
	stats["avg_duration"] = avgDuration

	return stats, nil
}

// GetUserActivityStats 获取用户活动统计信息
func (s *LogService) GetUserActivityStats(username string, startTime, endTime *time.Time) (map[string]interface{}, error) {
	query := s.db.Model(&UserActivity{})

	if username != "" {
		query = query.Where("username = ?", username)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	stats := make(map[string]interface{})

	// 总活动次数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total

	if total == 0 {
		stats["success"] = 0
		stats["success_rate"] = 0
		return stats, nil
	}

	// 成功次数
	var success int64
	successQuery := s.db.Model(&UserActivity{}).Where("success = ?", true)
	if username != "" {
		successQuery = successQuery.Where("username = ?", username)
	}
	if startTime != nil {
		successQuery = successQuery.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		successQuery = successQuery.Where("created_at <= ?", *endTime)
	}
	if err := successQuery.Count(&success).Error; err != nil {
		return nil, err
	}
	stats["success"] = success
	stats["success_rate"] = float64(success) / float64(total) * 100

	return stats, nil
}

// CleanOldLogs 清理旧日志（保留指定天数）
func (s *LogService) CleanOldLogs(days int) error {
	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 清理Hook日志
	if err := s.db.Where("created_at < ?", cutoffTime).Delete(&HookLog{}).Error; err != nil {
		return fmt.Errorf("failed to clean hook logs: %v", err)
	}

	// 清理系统日志
	if err := s.db.Where("created_at < ?", cutoffTime).Delete(&SystemLog{}).Error; err != nil {
		return fmt.Errorf("failed to clean system logs: %v", err)
	}

	// 清理用户活动记录
	if err := s.db.Where("created_at < ?", cutoffTime).Delete(&UserActivity{}).Error; err != nil {
		return fmt.Errorf("failed to clean user activities: %v", err)
	}

	// 清理项目活动记录
	if err := s.db.Where("created_at < ?", cutoffTime).Delete(&ProjectActivity{}).Error; err != nil {
		return fmt.Errorf("failed to clean project activities: %v", err)
	}

	return nil
}

// GetAllLogs 获取所有类型的日志（统一接口）
func (s *LogService) GetAllLogs(page, pageSize int, level, search string, startTime, endTime *time.Time) ([]map[string]interface{}, int64, error) {
	var allLogs []map[string]interface{}

	// 先获取总数
	hookTotal := s.getHookLogsCount(search, startTime, endTime)
	systemTotal := s.getSystemLogsCount(level, search, startTime, endTime)
	userTotal := s.getUserActivitiesCount(search, startTime, endTime)
	projectTotal := s.getProjectActivitiesCount(search, startTime, endTime)

	total := hookTotal + systemTotal + userTotal + projectTotal

	// 计算偏移量和限制
	offset := (page - 1) * pageSize
	limit := pageSize

	// 从所有表中获取数据并合并排序
	hookLogs, err := s.getHookLogsAsInterfaceAll(search, startTime, endTime)
	if err != nil {
		return nil, 0, err
	}
	allLogs = append(allLogs, hookLogs...)

	systemLogs, err := s.getSystemLogsAsInterfaceAll(level, search, startTime, endTime)
	if err != nil {
		return nil, 0, err
	}
	allLogs = append(allLogs, systemLogs...)

	userLogs, err := s.getUserActivitiesAsInterfaceAll(search, startTime, endTime)
	if err != nil {
		return nil, 0, err
	}
	allLogs = append(allLogs, userLogs...)

	projectLogs, err := s.getProjectActivitiesAsInterfaceAll(search, startTime, endTime)
	if err != nil {
		return nil, 0, err
	}
	allLogs = append(allLogs, projectLogs...)

	// 按时间排序（最新的在前）
	sort.Slice(allLogs, func(i, j int) bool {
		timeI, _ := time.Parse(time.RFC3339, allLogs[i]["timestamp"].(time.Time).Format(time.RFC3339))
		timeJ, _ := time.Parse(time.RFC3339, allLogs[j]["timestamp"].(time.Time).Format(time.RFC3339))
		return timeI.After(timeJ)
	})

	// 应用分页
	if offset >= len(allLogs) {
		return []map[string]interface{}{}, total, nil
	}

	end := offset + limit
	if end > len(allLogs) {
		end = len(allLogs)
	}

	pagedLogs := allLogs[offset:end]

	return pagedLogs, total, nil
}

// getHookLogsCount 获取Hook日志总数
func (s *LogService) getHookLogsCount(search string, startTime, endTime *time.Time) int64 {
	query := s.db.Model(&HookLog{})
	if search != "" {
		query = query.Where("hook_name LIKE ? OR output LIKE ? OR error LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}
	var count int64
	query.Count(&count)
	return count
}

// getSystemLogsCount 获取系统日志总数
func (s *LogService) getSystemLogsCount(level, search string, startTime, endTime *time.Time) int64 {
	query := s.db.Model(&SystemLog{})
	if level != "" {
		query = query.Where("level = ?", level)
	}
	if search != "" {
		query = query.Where("message LIKE ? OR details LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}
	var count int64
	query.Count(&count)
	return count
}

// getUserActivitiesCount 获取用户活动总数
func (s *LogService) getUserActivitiesCount(search string, startTime, endTime *time.Time) int64 {
	query := s.db.Model(&UserActivity{})
	if search != "" {
		query = query.Where("username LIKE ? OR action LIKE ? OR description LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}
	var count int64
	query.Count(&count)
	return count
}

// getProjectActivitiesCount 获取项目活动总数
func (s *LogService) getProjectActivitiesCount(search string, startTime, endTime *time.Time) int64 {
	query := s.db.Model(&ProjectActivity{})
	if search != "" {
		query = query.Where("project_name LIKE ? OR action LIKE ? OR description LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}
	var count int64
	query.Count(&count)
	return count
}

// getHookLogsAsInterfaceAll 获取所有Hook日志转为通用接口
func (s *LogService) getHookLogsAsInterfaceAll(search string, startTime, endTime *time.Time) ([]map[string]interface{}, error) {
	query := s.db.Model(&HookLog{})
	if search != "" {
		query = query.Where("hook_name LIKE ? OR output LIKE ? OR error LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	var logs []HookLog
	err := query.Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, log := range logs {
		result = append(result, map[string]interface{}{
			"id":         log.ID,
			"type":       "hook",
			"timestamp":  log.CreatedAt,
			"message":    fmt.Sprintf("Hook %s executed", log.HookName),
			"hookName":   log.HookName,
			"hookType":   log.HookType,
			"method":     log.Method,
			"remoteAddr": log.RemoteAddr,
			"success":    log.Success,
			"output":     log.Output,
			"error":      log.Error,
			"duration":   log.Duration,
			"userAgent":  log.UserAgent,
		})
	}
	return result, nil
}

// getSystemLogsAsInterfaceAll 获取所有系统日志转为通用接口
func (s *LogService) getSystemLogsAsInterfaceAll(level, search string, startTime, endTime *time.Time) ([]map[string]interface{}, error) {
	query := s.db.Model(&SystemLog{})
	if level != "" {
		query = query.Where("level = ?", level)
	}
	if search != "" {
		query = query.Where("message LIKE ? OR details LIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	var logs []SystemLog
	err := query.Order("created_at DESC").Find(&logs).Error
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, log := range logs {
		result = append(result, map[string]interface{}{
			"id":        log.ID,
			"type":      "system",
			"timestamp": log.CreatedAt,
			"level":     log.Level,
			"category":  log.Category,
			"message":   log.Message,
			"userId":    log.UserID,
			"ipAddress": log.IPAddress,
			"userAgent": log.UserAgent,
			"details":   log.Details,
		})
	}
	return result, nil
}

// getUserActivitiesAsInterfaceAll 获取所有用户活动转为通用接口
func (s *LogService) getUserActivitiesAsInterfaceAll(search string, startTime, endTime *time.Time) ([]map[string]interface{}, error) {
	query := s.db.Model(&UserActivity{})
	if search != "" {
		query = query.Where("username LIKE ? OR action LIKE ? OR description LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	var activities []UserActivity
	err := query.Order("created_at DESC").Find(&activities).Error
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, activity := range activities {
		// 优先使用详细的description，如果为空则使用简单格式
		message := activity.Description
		if message == "" {
			message = fmt.Sprintf("User %s performed %s", activity.Username, activity.Action)
		}

		result = append(result, map[string]interface{}{
			"id":          activity.ID,
			"type":        "user",
			"timestamp":   activity.CreatedAt,
			"message":     message,
			"username":    activity.Username,
			"action":      activity.Action,
			"resource":    activity.Resource,
			"description": activity.Description,
			"success":     activity.Success,
			"ipAddress":   activity.IPAddress,
			"userAgent":   activity.UserAgent,
			"details":     activity.Details,
		})
	}
	return result, nil
}

// getProjectActivitiesAsInterfaceAll 项目活动转为通用接口（全部）
func (s *LogService) getProjectActivitiesAsInterfaceAll(search string, startTime, endTime *time.Time) ([]map[string]interface{}, error) {
	query := s.db.Model(&ProjectActivity{})

	if search != "" {
		query = query.Where("project_name LIKE ? OR action LIKE ? OR description LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	var activities []ProjectActivity
	err := query.Order("created_at DESC").Find(&activities).Error
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, activity := range activities {
		// 优先使用详细的description，如果为空则使用简单格式
		message := activity.Description
		if message == "" {
			message = fmt.Sprintf("Project %s: %s", activity.ProjectName, activity.Action)
		}

		result = append(result, map[string]interface{}{
			"id":          activity.ID,
			"type":        "project",
			"timestamp":   activity.CreatedAt,
			"message":     message,
			"projectName": activity.ProjectName,
			"action":      activity.Action,
			"username":    activity.Username,
			"success":     activity.Success,
			"oldValue":    activity.OldValue,
			"newValue":    activity.NewValue,
			"commitHash":  activity.CommitHash,
			"description": activity.Description,
			"ipAddress":   activity.IPAddress,
		})
	}

	return result, nil
}

// ExportLogsToCSV 导出日志为CSV格式
func (s *LogService) ExportLogsToCSV(logType, level, search string, startTime, endTime *time.Time) (string, error) {
	var csvData string

	// CSV头部
	csvData = "ID,Type,Timestamp,Message,Level,User,Success,Details\n"

	switch logType {
	case "hook":
		logs, _, err := s.GetHookLogs(1, 1000, "", "", "", nil, startTime, endTime)
		if err != nil {
			return "", err
		}
		for _, log := range logs {
			csvData += fmt.Sprintf("%d,hook,%s,Hook %s executed,,,%t,%s\n",
				log.ID, log.CreatedAt.Format("2006-01-02 15:04:05"), log.HookName, log.Success, log.Output)
		}
	case "system":
		logs, _, err := s.GetSystemLogs(1, 1000, level, "", "", startTime, endTime)
		if err != nil {
			return "", err
		}
		for _, log := range logs {
			csvData += fmt.Sprintf("%d,system,%s,%s,%s,%s,,%s\n",
				log.ID, log.CreatedAt.Format("2006-01-02 15:04:05"), log.Message, log.Level, log.UserID, log.Details)
		}
	case "user":
		logs, _, err := s.GetUserActivities(1, 1000, "", "", nil, startTime, endTime)
		if err != nil {
			return "", err
		}
		for _, log := range logs {
			csvData += fmt.Sprintf("%d,user,%s,User %s: %s,,%s,%t,%s\n",
				log.ID, log.CreatedAt.Format("2006-01-02 15:04:05"), log.Username, log.Action, log.Username, log.Success, log.Details)
		}
	case "project":
		logs, _, err := s.GetProjectActivities(1, 1000, "", "", "", nil, startTime, endTime)
		if err != nil {
			return "", err
		}
		for _, log := range logs {
			csvData += fmt.Sprintf("%d,project,%s,Project %s: %s,,%s,%t,%s\n",
				log.ID, log.CreatedAt.Format("2006-01-02 15:04:05"), log.ProjectName, log.Action, log.Username, log.Success, log.Description)
		}
	default:
		// 导出所有日志
		allLogs, _, err := s.GetAllLogs(1, 1000, level, search, startTime, endTime)
		if err != nil {
			return "", err
		}
		for _, log := range allLogs {
			csvData += fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v\n",
				log["id"], log["type"], log["timestamp"], log["message"],
				log["level"], log["username"], log["success"], log["details"])
		}
	}

	return csvData, nil
}
