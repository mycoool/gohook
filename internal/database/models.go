package database

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel base model, contains common fields
type BaseModel struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// HookLog hook execution log
type HookLog struct {
	BaseModel
	HookID      string `json:"hook_id" gorm:"size:100;index"`  // hook id
	HookName    string `json:"hook_name" gorm:"size:200"`      // hook name
	HookType    string `json:"hook_type" gorm:"size:50;index"` // hook type: webhook, githook
	Method      string `json:"method" gorm:"size:10"`          // http method
	RemoteAddr  string `json:"remote_addr" gorm:"size:45"`     // client ip address
	Headers     string `json:"headers" gorm:"type:text"`       // request headers
	Body        string `json:"body" gorm:"type:text"`          // request body
	Success     bool   `json:"success" gorm:"index"`           // success
	Output      string `json:"output" gorm:"type:text"`        // output
	Error       string `json:"error" gorm:"type:text"`         // error
	Duration    int64  `json:"duration"`                       // duration (milliseconds)
	UserAgent   string `json:"user_agent" gorm:"size:500"`     // user agent
	QueryParams string `json:"query_params" gorm:"type:text"`  // query params
}

// SystemLog system log
type SystemLog struct {
	BaseModel
	Level     string `json:"level" gorm:"size:10;index"`    // log level: DEBUG, INFO, WARN, ERROR
	Category  string `json:"category" gorm:"size:50;index"` // log category: AUTH, CONFIG, DATABASE, etc.
	Message   string `json:"message" gorm:"type:text"`      // log message
	Details   string `json:"details" gorm:"type:text"`      // details
	UserID    string `json:"user_id" gorm:"size:50;index"`  // user id
	IPAddress string `json:"ip_address" gorm:"size:45"`     // client ip address
	UserAgent string `json:"user_agent" gorm:"size:500"`    // User Agent
}

// UserActivity user activity record
type UserActivity struct {
	BaseModel
	Username    string `json:"username" gorm:"size:100;index"` // username
	Action      string `json:"action" gorm:"size:100;index"`   // action type
	Resource    string `json:"resource" gorm:"size:200"`       // resource
	Description string `json:"description" gorm:"type:text"`   // description
	IPAddress   string `json:"ip_address" gorm:"size:45"`      // IP address
	UserAgent   string `json:"user_agent" gorm:"size:500"`     // User Agent
	Success     bool   `json:"success" gorm:"index"`           // success
	Details     string `json:"details" gorm:"type:text"`       // details
}

// ProjectActivity project activity record
type ProjectActivity struct {
	BaseModel
	ProjectName string `json:"project_name" gorm:"size:200;index"` // project name
	Action      string `json:"action" gorm:"size:100;index"`       // action type: branch_switch, tag_switch, pull, etc.
	OldValue    string `json:"old_value" gorm:"size:200"`          // old value (e.g. old branch name)
	NewValue    string `json:"new_value" gorm:"size:200"`          // new value (e.g. new branch name)
	Username    string `json:"username" gorm:"size:100;index"`     // username
	Success     bool   `json:"success" gorm:"index"`               // success
	Error       string `json:"error" gorm:"type:text"`             // error
	CommitHash  string `json:"commit_hash" gorm:"size:40"`         // commit hash
	Description string `json:"description" gorm:"type:text"`       // description
	IPAddress   string `json:"ip_address" gorm:"size:45"`          // IP address
}

// LogLevel log level constant
const (
	LogLevelDebug = "DEBUG"
	LogLevelInfo  = "INFO"
	LogLevelWarn  = "WARN"
	LogLevelError = "ERROR"
)

// LogCategory log category constant
const (
	LogCategoryAuth     = "AUTH"
	LogCategoryConfig   = "CONFIG"
	LogCategoryDatabase = "DATABASE"
	LogCategoryHook     = "HOOK"
	LogCategoryProject  = "PROJECT"
	LogCategorySystem   = "SYSTEM"
	LogCategoryAPI      = "API"
)

// UserAction user action constant
const (
	UserActionLogin        = "LOGIN"
	UserActionLogout       = "LOGOUT"
	UserActionCreateUser   = "CREATE_USER"
	UserActionUpdateUser   = "UPDATE_USER"
	UserActionDeleteUser   = "DELETE_USER"
	UserActionChangePasswd = "CHANGE_PASSWORD"
	// Hook management operation
	UserActionCreateHook         = "CREATE_HOOK"
	UserActionUpdateHookBasic    = "UPDATE_HOOK_BASIC"
	UserActionUpdateHookParam    = "UPDATE_HOOK_PARAMETERS"
	UserActionUpdateHookTrigger  = "UPDATE_HOOK_TRIGGERS"
	UserActionUpdateHookResponse = "UPDATE_HOOK_RESPONSE"
	UserActionUpdateHookScript   = "UPDATE_HOOK_SCRIPT"
	UserActionDeleteHook         = "DELETE_HOOK"
	// Add missing constants
	UserActionUpdateHookParameters = "UPDATE_HOOK_PARAMETERS"
	UserActionUpdateHookTriggers   = "UPDATE_HOOK_TRIGGERS"
	UserActionSaveHookScript       = "SAVE_HOOK_SCRIPT"

	// System configuration management operation
	UserActionViewSystemConfig   = "VIEW_SYSTEM_CONFIG"
	UserActionUpdateSystemConfig = "UPDATE_SYSTEM_CONFIG"
)

// ProjectAction project action constant
const (
	ProjectActionBranchSwitch = "BRANCH_SWITCH"
	ProjectActionTagSwitch    = "TAG_SWITCH"
	ProjectActionPull         = "PULL"
	ProjectActionAdd          = "ADD"
	ProjectActionDelete       = "DELETE"
	ProjectActionUpdate       = "UPDATE"
)

// HookType hook type constant
const (
	HookTypeWebhook = "webhook" // user-defined webhook
	HookTypeGitHook = "githook" // simple githook
)
