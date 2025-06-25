package types

import (
	"reflect"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserConfig user config structure
type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Role     string `yaml:"role"`
}

// UsersConfig user config file structure (original AppConfig)
type UsersConfig struct {
	Users []UserConfig `yaml:"users"`
}

// AppConfig application config structure
type AppConfig struct {
	Port              int            `yaml:"port"`
	JWTSecret         string         `yaml:"jwt_secret"`
	JWTExpiryDuration int            `yaml:"jwt_expiry_duration"`
	Mode              string         `yaml:"mode"` // "dev" | "prod" | "test"
	Database          DatabaseConfig `yaml:"database"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type             string `yaml:"type"`     // sqlite, mysql, postgres
	Database         string `yaml:"database"` // 数据库名称或文件路径
	Host             string `yaml:"host,omitempty"`
	Port             int    `yaml:"port,omitempty"`
	Username         string `yaml:"username,omitempty"`
	Password         string `yaml:"password,omitempty"`
	LogRetentionDays int    `yaml:"log_retention_days"` // 日志保留天数
}

// Claims JWT claim structure
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// UserResponse user response structure
type UserResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Config config file structure
type VersionConfig struct {
	Projects []ProjectConfig `yaml:"projects"`
}

// ProjectConfig project config structure
type ProjectConfig struct {
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
	Enhook      bool   `yaml:"enhook,omitempty"`
	Hookmode    string `yaml:"hookmode,omitempty"`
	Hookbranch  string `yaml:"hookbranch,omitempty"`
	Hooksecret  string `yaml:"hooksecret,omitempty"`
}

// VersionResponse version response structure
type VersionResponse struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Description    string `json:"description"`
	CurrentBranch  string `json:"currentBranch"`
	CurrentTag     string `json:"currentTag"`
	Mode           string `json:"mode"` // "branch" or "tag"
	Status         string `json:"status"`
	LastCommit     string `json:"lastCommit"`
	LastCommitTime string `json:"lastCommitTime"`
	Enhook         bool   `json:"enhook,omitempty"`
	Hookmode       string `json:"hookmode,omitempty"`
	Hookbranch     string `json:"hookbranch,omitempty"`
	Hooksecret     string `json:"hooksecret,omitempty"`
}

// BranchResponse branch response structure
type BranchResponse struct {
	Name           string `json:"name"`
	IsCurrent      bool   `json:"isCurrent"`
	LastCommit     string `json:"lastCommit"`
	LastCommitTime string `json:"lastCommitTime"`
	Type           string `json:"type"` // "local", "remote", or "detached"
}

// TagResponse tag response structure
type TagResponse struct {
	Name       string `json:"name"`
	IsCurrent  bool   `json:"isCurrent"`
	CommitHash string `json:"commitHash"`
	Date       string `json:"date"`
	Message    string `json:"message"`
}

// ClientSession client session structure
type ClientSession struct {
	ID        int       `json:"id"`
	Token     string    `json:"token"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	LastUsed  time.Time `json:"lastUsed"`
	CreatedAt time.Time `json:"createdAt"`
}

// WebSocket message type
type WSMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	HookID    string      `json:"hookId"`
	HookName  string      `json:"hookName"`
	Method    string      `json:"method"`
}

// Hook triggered message
type HookTriggeredMessage struct {
	HookID     string `json:"hookId"`
	HookName   string `json:"hookName"`
	Method     string `json:"method"`
	RemoteAddr string `json:"remoteAddr"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

// version switch message
type VersionSwitchMessage struct {
	ProjectName string `json:"projectName"`
	Action      string `json:"action"` // "switch-branch" | "switch-tag"
	Target      string `json:"target"` // branch name or tag name
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// project manage message
type ProjectManageMessage struct {
	Action      string `json:"action"` // "add" | "delete"
	ProjectName string `json:"projectName"`
	ProjectPath string `json:"projectPath,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

var GoHookAppConfig *AppConfig // application config

var GoHookUsersConfig *UsersConfig // user config

var GoHookVersionData *VersionConfig

// ClientResponse client response structure
type ClientResponse struct {
	Token string `json:"token"`
	ID    int    `json:"id"`
	Name  string `json:"name"`
}

// HookResponse Hook response structure
type HookResponse struct {
	ID                     string      `json:"id"`
	Name                   string      `json:"name"`
	ExecuteCommand         string      `json:"executeCommand"`
	WorkingDirectory       string      `json:"workingDirectory"`
	ResponseMessage        string      `json:"responseMessage"`
	HTTPMethods            []string    `json:"httpMethods"`
	ArgumentsCount         int         `json:"argumentsCount"`
	EnvironmentCount       int         `json:"environmentCount"`
	TriggerRuleDescription string      `json:"triggerRuleDescription"`
	TriggerRule            interface{} `json:"trigger-rule,omitempty"`
	LastUsed               *string     `json:"lastUsed"`
	Status                 string      `json:"status"` // active, inactive
}

func (c *AppConfig) SetMode(mode string) {
	// only "test" or "dev" or "prod" is allowed
	if mode != "test" && mode != "dev" && mode != "prod" {
		mode = "test"
	}
	c.Mode = mode
}

// UpdateAppConfig 更新内存中的应用配置
func UpdateAppConfig(systemConfig interface{}) {
	if GoHookAppConfig == nil {
		GoHookAppConfig = &AppConfig{}
	}

	// 使用反射来获取字段值，避免类型断言问题
	configValue := reflect.ValueOf(systemConfig)
	if configValue.Kind() == reflect.Struct {
		// 获取 JWTSecret 字段
		if jwtSecretField := configValue.FieldByName("JWTSecret"); jwtSecretField.IsValid() {
			GoHookAppConfig.JWTSecret = jwtSecretField.String()
		}

		// 获取 JWTExpiryDuration 字段
		if jwtExpiryField := configValue.FieldByName("JWTExpiryDuration"); jwtExpiryField.IsValid() {
			GoHookAppConfig.JWTExpiryDuration = int(jwtExpiryField.Int())
		}

		// 获取 Mode 字段
		if modeField := configValue.FieldByName("Mode"); modeField.IsValid() {
			GoHookAppConfig.Mode = modeField.String()
		}
	}
}
