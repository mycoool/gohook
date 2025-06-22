package database

import (
	"encoding/json"
	"fmt"
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
func (s *LogService) CreateHookLog(hookID, hookName, method, remoteAddr string,
	headers map[string][]string, body string, success bool, output, error string,
	duration int64, userAgent string, queryParams map[string][]string) error {

	headersJSON, _ := json.Marshal(headers)
	queryParamsJSON, _ := json.Marshal(queryParams)

	log := &HookLog{
		HookID:      hookID,
		HookName:    hookName,
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
func (s *LogService) GetHookLogs(page, pageSize int, hookID, hookName string, success *bool,
	startTime, endTime *time.Time) ([]HookLog, int64, error) {

	query := s.db.Model(&HookLog{})

	// 添加过滤条件
	if hookID != "" {
		query = query.Where("hook_id = ?", hookID)
	}
	if hookName != "" {
		query = query.Where("hook_name LIKE ?", "%"+hookName+"%")
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
func (s *LogService) GetHookLogStats(startTime, endTime *time.Time) (map[string]interface{}, error) {
	query := s.db.Model(&HookLog{})

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

	// 成功次数
	var success int64
	if err := query.Where("success = ?", true).Count(&success).Error; err != nil {
		return nil, err
	}
	stats["success"] = success
	stats["success_rate"] = float64(success) / float64(total) * 100

	// 平均执行时长
	var avgDuration float64
	if err := s.db.Model(&HookLog{}).Select("AVG(duration)").Row().Scan(&avgDuration); err != nil {
		return nil, err
	}
	stats["avg_duration"] = avgDuration

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
