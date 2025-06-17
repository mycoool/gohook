package types

import (
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
	Port              int    `yaml:"port"`
	JWTSecret         string `yaml:"jwt_secret"`
	JWTExpiryDuration int    `yaml:"jwt_expiry_duration"`
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
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	ExecuteCommand         string   `json:"executeCommand"`
	WorkingDirectory       string   `json:"workingDirectory"`
	ResponseMessage        string   `json:"responseMessage"`
	HTTPMethods            []string `json:"httpMethods"`
	TriggerRuleDescription string   `json:"triggerRuleDescription"`
	LastUsed               *string  `json:"lastUsed"`
	Status                 string   `json:"status"` // active, inactive
}
