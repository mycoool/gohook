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
	PanelAlias        string         `yaml:"panel_alias"` // 面板别名，用于浏览器标题
}

// DatabaseConfig database config
type DatabaseConfig struct {
	Type             string `yaml:"type"`     // sqlite, mysql, postgres
	Database         string `yaml:"database"` // database name or file path
	Host             string `yaml:"host,omitempty"`
	Port             int    `yaml:"port,omitempty"`
	Username         string `yaml:"username,omitempty"`
	Password         string `yaml:"password,omitempty"`
	LogRetentionDays int    `yaml:"log_retention_days"` // log retention days
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
	Name        string             `yaml:"name"`
	Path        string             `yaml:"path"`
	Description string             `yaml:"description"`
	Enabled     bool               `yaml:"enabled"`
	Enhook      bool               `yaml:"enhook,omitempty"`
	Hookmode    string             `yaml:"hookmode,omitempty"`
	Hookbranch  string             `yaml:"hookbranch,omitempty"`
	Hooksecret  string             `yaml:"hooksecret,omitempty"`
	ForceSync   bool               `yaml:"forcesync,omitempty"` // GitHook 是否使用强制同步模式
	Sync        *ProjectSyncConfig `yaml:"sync,omitempty"`      // Sync node settings
}

// ProjectSyncConfig describes sync strategy for a project
type ProjectSyncConfig struct {
	Enabled           bool     `yaml:"enabled" json:"enabled"`
	Driver            string   `yaml:"driver,omitempty" json:"driver,omitempty"`                        // agent | rsync | inherit
	MaxParallelNodes  int      `yaml:"max_parallel_nodes,omitempty" json:"maxParallelNodes,omitempty"`  // concurrency guard
	IgnoreDefaults    bool     `yaml:"ignore_defaults,omitempty" json:"ignoreDefaults,omitempty"`       // include built-in ignore set (.git/, runtime/)
	IgnorePatterns    []string `yaml:"ignore_patterns,omitempty" json:"ignorePatterns,omitempty"`       // extra ignore globs
	IgnoreFile        string   `yaml:"ignore_file,omitempty" json:"ignoreFile,omitempty"`               // optional ignore file path
	IgnorePermissions bool     `yaml:"ignore_permissions,omitempty" json:"ignorePermissions,omitempty"` // do not sync chmod/chown
	// Advanced performance/correctness knobs (UI-managed, env can still override if set).
	DeltaIndexOverlay       *bool                   `yaml:"delta_index_overlay,omitempty" json:"deltaIndexOverlay,omitempty"`             // enable overlay delta index (needs baseline scan for drift healing)
	DeltaMaxFiles           int                     `yaml:"delta_max_files,omitempty" json:"deltaMaxFiles,omitempty"`                     // max files per delta batch (fallback to full walk when exceeded)
	OverlayFullScanEvery    int                     `yaml:"overlay_fullscan_every,omitempty" json:"overlayFullScanEvery,omitempty"`       // force full index every N tasks (overlay)
	OverlayFullScanInterval string                  `yaml:"overlay_fullscan_interval,omitempty" json:"overlayFullScanInterval,omitempty"` // force full index at least every duration (e.g. 30m)
	Nodes                   []ProjectSyncNodeConfig `yaml:"nodes,omitempty" json:"nodes,omitempty"`
}

// ProjectSyncNodeConfig describes one target node for the project
type ProjectSyncNodeConfig struct {
	NodeID         string   `yaml:"node_id" json:"nodeId"`
	TargetPath     string   `yaml:"target_path" json:"targetPath"`
	Strategy       string   `yaml:"strategy,omitempty" json:"strategy,omitempty"`              // mirror | overlay
	Driver         string   `yaml:"driver,omitempty" json:"driver,omitempty"`                  // override driver per node
	Include        []string `yaml:"include,omitempty" json:"include,omitempty"`                // whitelist globs
	Exclude        []string `yaml:"exclude,omitempty" json:"exclude,omitempty"`                // blacklist globs
	IgnoreFile     string   `yaml:"ignore_file,omitempty" json:"ignoreFile,omitempty"`         // per-node ignore file path
	IgnorePatterns []string `yaml:"ignore_patterns,omitempty" json:"ignorePatterns,omitempty"` // per-node ignore
	// Mirror optimization knobs (agent-side behavior, delivered via task payload).
	MirrorFastDelete        bool `yaml:"mirror_fast_delete,omitempty" json:"mirrorFastDelete,omitempty"`
	MirrorFastFullscanEvery int  `yaml:"mirror_fast_fullscan_every,omitempty" json:"mirrorFastFullscanEvery,omitempty"`
	MirrorCleanEmptyDirs    bool `yaml:"mirror_clean_empty_dirs,omitempty" json:"mirrorCleanEmptyDirs,omitempty"`
}

// VersionResponse version response structure
type VersionResponse struct {
	Name           string             `json:"name"`
	Path           string             `json:"path"`
	Description    string             `json:"description"`
	CurrentBranch  string             `json:"currentBranch"`
	CurrentTag     string             `json:"currentTag"`
	Mode           string             `json:"mode"` // "branch" or "tag"
	Status         string             `json:"status"`
	LastCommit     string             `json:"lastCommit"`
	LastCommitTime string             `json:"lastCommitTime"`
	Enhook         bool               `json:"enhook,omitempty"`
	Hookmode       string             `json:"hookmode,omitempty"`
	Hookbranch     string             `json:"hookbranch,omitempty"`
	Hooksecret     string             `json:"hooksecret,omitempty"`
	ForceSync      bool               `json:"forcesync,omitempty"` // GitHook 是否使用强制同步模式
	Sync           *ProjectSyncConfig `json:"sync,omitempty"`
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

// UpdateAppConfig update app config in memory
func UpdateAppConfig(systemConfig interface{}) {
	if GoHookAppConfig == nil {
		GoHookAppConfig = &AppConfig{}
	}

	// use reflection to get field values, avoid type assertion problem
	configValue := reflect.ValueOf(systemConfig)
	if configValue.Kind() == reflect.Struct {
		// get JWTSecret field
		if jwtSecretField := configValue.FieldByName("JWTSecret"); jwtSecretField.IsValid() {
			GoHookAppConfig.JWTSecret = jwtSecretField.String()
		}

		// get JWTExpiryDuration field
		if jwtExpiryField := configValue.FieldByName("JWTExpiryDuration"); jwtExpiryField.IsValid() {
			GoHookAppConfig.JWTExpiryDuration = int(jwtExpiryField.Int())
		}

		// get Mode field
		if modeField := configValue.FieldByName("Mode"); modeField.IsValid() {
			GoHookAppConfig.Mode = modeField.String()
		}

		// get PanelAlias field
		if panelAliasField := configValue.FieldByName("PanelAlias"); panelAliasField.IsValid() {
			GoHookAppConfig.PanelAlias = panelAliasField.String()
		}
	}
}
