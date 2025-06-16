package types

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// UserConfig 用户配置结构
type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Role     string `yaml:"role"`
}

// UsersConfig 用户配置文件结构（原AppConfig）
type UsersConfig struct {
	Users []UserConfig `yaml:"users"`
}

// AppConfig 应用程序配置结构
type AppConfig struct {
	Port              int    `yaml:"port"`
	JWTSecret         string `yaml:"jwt_secret"`
	JWTExpiryDuration int    `yaml:"jwt_expiry_duration"`
}

// Claims JWT声明结构
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// UserResponse 用户响应结构
type UserResponse struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

// Config 配置文件结构
type Config struct {
	Projects []ProjectConfig `yaml:"projects"`
}

// ProjectConfig 项目配置
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

// VersionResponse 版本响应结构
type VersionResponse struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	Description    string `json:"description"`
	CurrentBranch  string `json:"currentBranch"`
	CurrentTag     string `json:"currentTag"`
	Mode           string `json:"mode"` // "branch" 或 "tag"
	Status         string `json:"status"`
	LastCommit     string `json:"lastCommit"`
	LastCommitTime string `json:"lastCommitTime"`
	Enhook         bool   `json:"enhook,omitempty"`
	Hookmode       string `json:"hookmode,omitempty"`
	Hookbranch     string `json:"hookbranch,omitempty"`
	Hooksecret     string `json:"hooksecret,omitempty"`
}

// BranchResponse 分支响应结构
type BranchResponse struct {
	Name           string `json:"name"`
	IsCurrent      bool   `json:"isCurrent"`
	LastCommit     string `json:"lastCommit"`
	LastCommitTime string `json:"lastCommitTime"`
	Type           string `json:"type"` // "local", "remote", or "detached"
}

// TagResponse 标签响应结构
type TagResponse struct {
	Name       string `json:"name"`
	IsCurrent  bool   `json:"isCurrent"`
	CommitHash string `json:"commitHash"`
	Date       string `json:"date"`
	Message    string `json:"message"`
}

// ClientSession 客户端会话结构
type ClientSession struct {
	ID        int       `json:"id"`
	Token     string    `json:"token"`
	Name      string    `json:"name"`
	Username  string    `json:"username"`
	LastUsed  time.Time `json:"lastUsed"`
	CreatedAt time.Time `json:"createdAt"`
}

// WebSocket消息类型
type WSMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	HookID    string      `json:"hookId"`
	HookName  string      `json:"hookName"`
	Method    string      `json:"method"`
}

// Hook触发消息
type HookTriggeredMessage struct {
	HookID     string `json:"hookId"`
	HookName   string `json:"hookName"`
	Method     string `json:"method"`
	RemoteAddr string `json:"remoteAddr"`
	Success    bool   `json:"success"`
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
}

// 版本切换消息
type VersionSwitchMessage struct {
	ProjectName string `json:"projectName"`
	Action      string `json:"action"` // "switch-branch" | "switch-tag"
	Target      string `json:"target"` // 分支名或标签名
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

// 项目管理消息
type ProjectManageMessage struct {
	Action      string `json:"action"` // "add" | "delete"
	ProjectName string `json:"projectName"`
	ProjectPath string `json:"projectPath,omitempty"`
	Success     bool   `json:"success"`
	Error       string `json:"error,omitempty"`
}

var GoHookAppConfig *AppConfig // 应用程序配置

var GoHookUsersConfig *UsersConfig // 用户配置
