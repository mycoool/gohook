package database

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 基础模型，包含通用字段
type BaseModel struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// HookLog Hook执行日志
type HookLog struct {
	BaseModel
	HookID      string `json:"hook_id" gorm:"size:100;index"`  // Hook ID
	HookName    string `json:"hook_name" gorm:"size:200"`      // Hook名称
	HookType    string `json:"hook_type" gorm:"size:50;index"` // Hook类型：webhook, githook
	Method      string `json:"method" gorm:"size:10"`          // HTTP方法
	RemoteAddr  string `json:"remote_addr" gorm:"size:45"`     // 客户端IP
	Headers     string `json:"headers" gorm:"type:text"`       // 请求头JSON
	Body        string `json:"body" gorm:"type:text"`          // 请求体
	Success     bool   `json:"success" gorm:"index"`           // 是否成功
	Output      string `json:"output" gorm:"type:text"`        // 执行输出
	Error       string `json:"error" gorm:"type:text"`         // 错误信息
	Duration    int64  `json:"duration"`                       // 执行时长（毫秒）
	UserAgent   string `json:"user_agent" gorm:"size:500"`     // User Agent
	QueryParams string `json:"query_params" gorm:"type:text"`  // 查询参数JSON
}

// SystemLog 系统日志
type SystemLog struct {
	BaseModel
	Level     string `json:"level" gorm:"size:10;index"`    // 日志级别: DEBUG, INFO, WARN, ERROR
	Category  string `json:"category" gorm:"size:50;index"` // 日志分类: AUTH, CONFIG, DATABASE, etc.
	Message   string `json:"message" gorm:"type:text"`      // 日志消息
	Details   string `json:"details" gorm:"type:text"`      // 详细信息JSON
	UserID    string `json:"user_id" gorm:"size:50;index"`  // 相关用户ID
	IPAddress string `json:"ip_address" gorm:"size:45"`     // IP地址
	UserAgent string `json:"user_agent" gorm:"size:500"`    // User Agent
}

// UserActivity 用户活动记录
type UserActivity struct {
	BaseModel
	Username    string `json:"username" gorm:"size:100;index"` // 用户名
	Action      string `json:"action" gorm:"size:100;index"`   // 操作类型
	Resource    string `json:"resource" gorm:"size:200"`       // 操作资源
	Description string `json:"description" gorm:"type:text"`   // 操作描述
	IPAddress   string `json:"ip_address" gorm:"size:45"`      // IP地址
	UserAgent   string `json:"user_agent" gorm:"size:500"`     // User Agent
	Success     bool   `json:"success" gorm:"index"`           // 是否成功
	Details     string `json:"details" gorm:"type:text"`       // 详细信息JSON
}

// ProjectActivity 项目活动记录
type ProjectActivity struct {
	BaseModel
	ProjectName string `json:"project_name" gorm:"size:200;index"` // 项目名称
	Action      string `json:"action" gorm:"size:100;index"`       // 操作类型: branch_switch, tag_switch, pull, etc.
	OldValue    string `json:"old_value" gorm:"size:200"`          // 旧值（如旧分支名）
	NewValue    string `json:"new_value" gorm:"size:200"`          // 新值（如新分支名）
	Username    string `json:"username" gorm:"size:100;index"`     // 操作用户
	Success     bool   `json:"success" gorm:"index"`               // 是否成功
	Error       string `json:"error" gorm:"type:text"`             // 错误信息
	CommitHash  string `json:"commit_hash" gorm:"size:40"`         // 相关提交哈希
	Description string `json:"description" gorm:"type:text"`       // 操作描述
	IPAddress   string `json:"ip_address" gorm:"size:45"`          // IP地址
}

// LogLevel 日志级别常量
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

// LogCategory 日志分类常量
const (
	LogCategoryAuth     = "AUTH"
	LogCategoryConfig   = "CONFIG"
	LogCategoryDatabase = "DATABASE"
	LogCategoryHook     = "HOOK"
	LogCategoryProject  = "PROJECT"
	LogCategorySystem   = "SYSTEM"
	LogCategoryAPI      = "API"
)

// UserAction 用户操作常量
const (
	UserActionLogin        = "LOGIN"
	UserActionLogout       = "LOGOUT"
	UserActionCreateUser   = "CREATE_USER"
	UserActionUpdateUser   = "UPDATE_USER"
	UserActionDeleteUser   = "DELETE_USER"
	UserActionChangePasswd = "CHANGE_PASSWORD"
	// Hook管理操作
	UserActionCreateHook         = "CREATE_HOOK"
	UserActionUpdateHookBasic    = "UPDATE_HOOK_BASIC"
	UserActionUpdateHookParam    = "UPDATE_HOOK_PARAMETERS"
	UserActionUpdateHookTrigger  = "UPDATE_HOOK_TRIGGERS"
	UserActionUpdateHookResponse = "UPDATE_HOOK_RESPONSE"
	UserActionUpdateHookScript   = "UPDATE_HOOK_SCRIPT"
	UserActionDeleteHook         = "DELETE_HOOK"
	// 添加缺失的常量
	UserActionUpdateHookParameters = "UPDATE_HOOK_PARAMETERS"
	UserActionUpdateHookTriggers   = "UPDATE_HOOK_TRIGGERS"
	UserActionSaveHookScript       = "SAVE_HOOK_SCRIPT"
)

// ProjectAction 项目操作常量
const (
	ProjectActionBranchSwitch = "BRANCH_SWITCH"
	ProjectActionTagSwitch    = "TAG_SWITCH"
	ProjectActionPull         = "PULL"
	ProjectActionAdd          = "ADD"
	ProjectActionDelete       = "DELETE"
	ProjectActionUpdate       = "UPDATE"
)

// HookType Hook类型常量
const (
	HookTypeWebhook = "webhook" // 用户手动建立规则和脚本的webhook
	HookTypeGitHook = "githook" // 简易版本的githook
)
