package syncnode

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mycoool/gohook/internal/database"
	"gorm.io/gorm"
)

const (
	NodeTypeAgent = "agent"
	NodeTypeSSH   = "ssh"

	NodeStatusOnline  = "ONLINE"
	NodeStatusOffline = "OFFLINE"

	NodeHealthHealthy  = "HEALTHY"
	NodeHealthUnknown  = "UNKNOWN"
	NodeHealthDegraded = "DEGRADED"

	InstallStatusPending    = "pending"
	InstallStatusInstalling = "installing"
	InstallStatusSuccess    = "success"
	InstallStatusFailed     = "failed"
)

// ErrInvalidToken indicates the agent token was missing or incorrect
var ErrInvalidToken = errors.New("invalid node token")

// Service provides sync node management helpers
type Service struct {
	db          *gorm.DB
	changeQueue ChangeQueue
}

// NewService creates a sync node service
func NewService() *Service {
	return &Service{
		db:          database.GetDB(),
		changeQueue: NewDBChangeQueue(database.GetDB()),
	}
}

// NodeListFilter filters list queries
type NodeListFilter struct {
	Status string
	Type   string
	Search string
}

// CreateNodeRequest payload
type CreateNodeRequest struct {
	Name           string                 `json:"name" binding:"required"`
	Address        string                 `json:"address"`
	Type           string                 `json:"type" binding:"required"`
	SSHUser        string                 `json:"sshUser"`
	SSHPort        int                    `json:"sshPort"`
	AuthType       string                 `json:"authType"`
	CredentialRef  string                 `json:"credentialRef"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// UpdateNodeRequest payload (full replace)
type UpdateNodeRequest struct {
	Name           string                 `json:"name"`
	Address        string                 `json:"address"`
	Type           string                 `json:"type"`
	SSHUser        string                 `json:"sshUser"`
	SSHPort        int                    `json:"sshPort"`
	AuthType       string                 `json:"authType"`
	CredentialRef  string                 `json:"credentialRef"`
	Tags           []string               `json:"tags"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// InstallRequest controls agent installation
type InstallRequest struct {
	SSHUser       string `json:"sshUser"`
	SSHPort       int    `json:"sshPort"`
	CredentialRef string `json:"credentialRef"`
	AuthType      string `json:"authType"`
	Force         bool   `json:"force"`
}

// HeartbeatRequest payload sent by sync agents
type HeartbeatRequest struct {
	Token        string                 `json:"token" binding:"required"`
	Status       string                 `json:"status"`
	Health       string                 `json:"health"`
	AgentVersion string                 `json:"agentVersion"`
	Hostname     string                 `json:"hostname"`
	IPAddresses  []string               `json:"ipAddresses"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ListNodes returns all nodes with optional filters
func (s *Service) ListNodes(ctx context.Context, filter NodeListFilter) ([]database.SyncNode, error) {
	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}
	query := db.WithContext(ctx).Model(&database.SyncNode{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR address LIKE ?", like, like)
	}

	var nodes []database.SyncNode
	if err := query.Order("created_at DESC").Find(&nodes).Error; err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNode fetches a single node
func (s *Service) GetNode(ctx context.Context, id uint) (*database.SyncNode, error) {
	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}
	var node database.SyncNode
	if err := db.WithContext(ctx).First(&node, id).Error; err != nil {
		return nil, err
	}
	return &node, nil
}

// CreateNode stores a new node
func (s *Service) CreateNode(ctx context.Context, req CreateNodeRequest) (*database.SyncNode, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}
	token, err := GenerateNodeToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate node token: %w", err)
	}
	node := &database.SyncNode{
		Status:          NodeStatusOffline,
		Health:          NodeHealthUnknown,
		InstallStatus:   InstallStatusPending,
		CredentialValue: token,
	}
	s.applyCreateRequest(node, req)

	if err := db.WithContext(ctx).Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// UpdateNode replaces editable fields
func (s *Service) UpdateNode(ctx context.Context, id uint, req UpdateNodeRequest) (*database.SyncNode, error) {
	node, err := s.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}

	s.applyUpdateRequest(node, req)

	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}
	if err := db.WithContext(ctx).Save(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// DeleteNode removes a sync node
func (s *Service) DeleteNode(ctx context.Context, id uint) error {
	db, err := s.ensureDB()
	if err != nil {
		return err
	}
	res := db.WithContext(ctx).Delete(&database.SyncNode{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// RotateToken regenerates and persists a new agent token.
func (s *Service) RotateToken(ctx context.Context, id uint) (*database.SyncNode, error) {
	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}
	node, err := s.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}
	token, err := GenerateNodeToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate node token: %w", err)
	}
	node.CredentialValue = token
	if err := db.WithContext(ctx).Save(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// TriggerInstall sets install status and asynchronously installs agent
func (s *Service) TriggerInstall(ctx context.Context, id uint, req InstallRequest) (*database.SyncNode, error) {
	node, err := s.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}

	overrideSSHFields(node, req)
	node.InstallStatus = InstallStatusInstalling
	node.InstallLog = appendLogLine(node.InstallLog, "Starting Sync Agent installation job")

	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}
	if err := db.WithContext(ctx).Save(node).Error; err != nil {
		return nil, err
	}

	go s.runInstallRoutine(id, req)

	return node, nil
}

// RecordHeartbeat updates node status data from agent heartbeat
func (s *Service) RecordHeartbeat(ctx context.Context, id uint, req HeartbeatRequest) (*database.SyncNode, error) {
	node, err := s.ValidateAgentToken(ctx, id, req.Token)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	node.LastSeen = &now
	node.Status = normalizeNodeStatus(req.Status)
	node.Health = normalizeNodeHealth(req.Health)
	if req.AgentVersion != "" {
		node.AgentVersion = req.AgentVersion
	}
	if node.InstallStatus != InstallStatusSuccess {
		node.InstallStatus = InstallStatusSuccess
	}

	meta := decodeMap(node.Metadata)
	agentMeta := map[string]interface{}{
		"hostname":    req.Hostname,
		"ipAddresses": req.IPAddresses,
		"updatedAt":   now.Format(time.RFC3339),
	}
	for k, v := range req.Metadata {
		agentMeta[k] = v
	}
	meta["agent"] = agentMeta
	node.Metadata = encodeMap(meta)

	db, err := s.ensureDB()
	if err != nil {
		return nil, err
	}
	if err := db.WithContext(ctx).Save(node).Error; err != nil {
		return nil, err
	}

	return node, nil
}

// ValidateAgentToken loads the node and validates agent token.
func (s *Service) ValidateAgentToken(ctx context.Context, id uint, token string) (*database.SyncNode, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidToken
	}
	node, err := s.GetNode(ctx, id)
	if err != nil {
		return nil, err
	}
	if node.CredentialValue == "" || subtle.ConstantTimeCompare([]byte(node.CredentialValue), []byte(token)) != 1 {
		return nil, ErrInvalidToken
	}
	return node, nil
}

func (s *Service) runInstallRoutine(id uint, req InstallRequest) {
	ctx := context.Background()
	node, err := s.GetNode(ctx, id)
	if err != nil {
		return
	}

	lines := []string{}
	logStep := func(msg string) {
		lines = append(lines, fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), msg))
	}

	logStep("Validating SSH connectivity information")
	if node.SSHUser == "" {
		node.SSHUser = "root"
		logStep("SSH user not provided, defaulting to root")
	}
	if node.SSHPort == 0 {
		node.SSHPort = 22
	}

	logStep("Preparing Sync Agent package (stub)")
	logStep("Uploading agent binary via SSH/SCP (stub)")
	logStep("Configuring systemd service and ignore rules (stub)")
	logStep("Starting agent process and waiting for heartbeat (stub)")

	now := time.Now()
	node.InstallLog = combineLogs(node.InstallLog, strings.Join(lines, "\n"))
	node.InstallStatus = InstallStatusSuccess
	node.Status = NodeStatusOnline
	node.Health = NodeHealthHealthy
	node.LastSeen = &now
	node.AgentVersion = "v0.1.0-sync"

	db, err := s.ensureDB()
	if err != nil {
		return
	}
	if s.changeQueue == nil {
		s.changeQueue = NewDBChangeQueue(db)
	}
	if err := db.WithContext(ctx).Save(node).Error; err != nil {
		return
	}
}

func (s *Service) applyCreateRequest(node *database.SyncNode, req CreateNodeRequest) {
	node.Name = req.Name
	node.Address = req.Address
	node.Type = normalizeNodeType(req.Type)
	if node.Type == NodeTypeSSH {
		node.SSHUser = fallbackSSHUser(req.SSHUser)
		node.SSHPort = defaultPort(req.SSHPort)
	} else {
		node.SSHUser = ""
		node.SSHPort = 0
	}
	node.AuthType = normalizeAuthType(node.Type, req.AuthType)
	node.CredentialRef = req.CredentialRef
	node.Tags = encodeStringSlice(req.Tags)
	node.Metadata = encodeMap(req.Metadata)
}

func (s *Service) applyUpdateRequest(node *database.SyncNode, req UpdateNodeRequest) {
	if req.Name != "" {
		node.Name = req.Name
	}
	if req.Address != "" {
		node.Address = req.Address
	}
	if req.Type != "" {
		node.Type = normalizeNodeType(req.Type)
	}
	if node.Type == NodeTypeSSH {
		if req.SSHUser != "" {
			node.SSHUser = req.SSHUser
		} else if node.SSHUser == "" {
			node.SSHUser = fallbackSSHUser("")
		}
		if req.SSHPort != 0 {
			node.SSHPort = req.SSHPort
		} else if node.SSHPort == 0 {
			node.SSHPort = defaultPort(0)
		}
	} else {
		node.SSHUser = ""
		node.SSHPort = 0
	}
	if req.AuthType != "" || req.Type != "" {
		node.AuthType = normalizeAuthType(node.Type, req.AuthType)
	}
	if req.CredentialRef != "" {
		node.CredentialRef = req.CredentialRef
	}
	if req.Tags != nil {
		node.Tags = encodeStringSlice(req.Tags)
	}
	if req.Metadata != nil {
		node.Metadata = encodeMap(req.Metadata)
	}
}

func normalizeNodeType(value string) string {
	if value == "" {
		return NodeTypeAgent
	}
	switch value {
	case NodeTypeAgent, NodeTypeSSH:
		return value
	default:
		return NodeTypeAgent
	}
}

func normalizeNodeStatus(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case NodeStatusOffline:
		return NodeStatusOffline
	default:
		return NodeStatusOnline
	}
}

func normalizeNodeHealth(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case NodeHealthDegraded:
		return NodeHealthDegraded
	case NodeHealthHealthy:
		return NodeHealthHealthy
	default:
		return NodeHealthUnknown
	}
}

func defaultPort(port int) int {
	if port <= 0 {
		return 22
	}
	return port
}

func normalizeAuthType(nodeType, provided string) string {
	if provided != "" {
		return provided
	}
	if nodeType == NodeTypeAgent {
		return "key"
	}
	return provided
}

func fallbackSSHUser(value string) string {
	if value != "" {
		return value
	}
	return "root"
}

func encodeStringSlice(values []string) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func encodeMap(values map[string]interface{}) string {
	if len(values) == 0 {
		return ""
	}
	raw, _ := json.Marshal(values)
	return string(raw)
}

func appendLogLine(existing, line string) string {
	formatted := fmt.Sprintf("[%s] %s", time.Now().Format(time.RFC3339), line)
	if existing == "" {
		return formatted
	}
	return existing + "\n" + formatted
}

func combineLogs(existing, addition string) string {
	if strings.TrimSpace(existing) == "" {
		return addition
	}
	if strings.TrimSpace(addition) == "" {
		return existing
	}
	return existing + "\n" + addition
}

func overrideSSHFields(node *database.SyncNode, req InstallRequest) {
	if req.SSHUser != "" {
		node.SSHUser = req.SSHUser
	}
	if req.SSHPort != 0 {
		node.SSHPort = req.SSHPort
	}
	if req.CredentialRef != "" {
		node.CredentialRef = req.CredentialRef
	}
	if req.AuthType != "" {
		node.AuthType = req.AuthType
	}
}

// Decode helper functions for handlers
func decodeStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}

func decodeMap(raw string) map[string]interface{} {
	if strings.TrimSpace(raw) == "" {
		return map[string]interface{}{}
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

// Validate ensures create input contains essentials
func (req CreateNodeRequest) Validate() error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = NodeTypeAgent
	}
	return nil
}
func (s *Service) ensureDB() (*gorm.DB, error) {
	if s.db == nil {
		s.db = database.GetDB()
	}
	if s.db == nil {
		return nil, errors.New("database not initialized")
	}
	return s.db, nil
}
