package syncnode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/database"
	"gorm.io/gorm"
)

var defaultService = NewService()

type nodeResponse struct {
	ID                   uint                   `json:"id"`
	Name                 string                 `json:"name"`
	Address              string                 `json:"address"`
	Remark               string                 `json:"remark"`
	Type                 string                 `json:"type"`
	Status               string                 `json:"status"`
	Health               string                 `json:"health"`
	AgentCertFingerprint string                 `json:"agentCertFingerprint"`
	ConnectionStatus     string                 `json:"connectionStatus"`
	SyncStatus           string                 `json:"syncStatus"`
	LastSyncAt           *time.Time             `json:"lastSyncAt"`
	LastTaskProject      string                 `json:"lastTaskProject,omitempty"`
	LastTaskTargetPath   string                 `json:"lastTaskTargetPath,omitempty"`
	LastError            string                 `json:"lastError,omitempty"`
	LastErrorCode        string                 `json:"lastErrorCode,omitempty"`
	Tags                 []string               `json:"tags"`
	Metadata             map[string]interface{} `json:"metadata"`
	SSHUser              string                 `json:"sshUser"`
	SSHPort              int                    `json:"sshPort"`
	AuthType             string                 `json:"authType"`
	CredentialRef        string                 `json:"credentialRef"`
	AgentToken           string                 `json:"agentToken,omitempty"`
	InstallStatus        string                 `json:"installStatus"`
	InstallLog           string                 `json:"installLog"`
	AgentVersion         string                 `json:"agentVersion"`
	LastSeen             *time.Time             `json:"lastSeen"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
}

func HandleListNodes(c *gin.Context) {
	filter := NodeListFilter{
		Status: c.Query("status"),
		Type:   c.Query("type"),
		Search: c.Query("search"),
	}

	nodes, err := defaultService.ListNodes(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	taskSummary := loadLastTaskSummary(c.Request.Context(), nodes)
	c.JSON(http.StatusOK, mapNodes(nodes, taskSummary))
}

func HandleGetNode(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := defaultService.GetNode(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	taskSummary := loadLastTaskSummary(c.Request.Context(), []database.SyncNode{*node})
	c.JSON(http.StatusOK, mapNode(node, taskSummary[node.ID]))
}

func HandleCreateNode(c *gin.Context) {
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := defaultService.CreateNode(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, mapNode(node, nodeTaskSummary{}))
}

func HandleUpdateNode(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := defaultService.UpdateNode(c.Request.Context(), id, req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapNode(node, nodeTaskSummary{}))
}

func HandleDeleteNode(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := defaultService.DeleteNode(c.Request.Context(), id); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func HandleInstallNode(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req InstallRequest
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := defaultService.TriggerInstall(c.Request.Context(), id, req)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, mapNode(node, nodeTaskSummary{}))
}

func HandleRotateToken(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := defaultService.RotateToken(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, mapNode(node, nodeTaskSummary{}))
}

func HandleResetPairing(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	node, err := defaultService.ResetPairing(c.Request.Context(), id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mapNode(node, nodeTaskSummary{}))
}

type nodeTaskSummary struct {
	ProjectName string
	Status      string
	UpdatedAt   *time.Time
	LastError   string
	ErrorCode   string
	TargetPath  string
}

func loadLastTaskSummary(ctx context.Context, nodes []database.SyncNode) map[uint]nodeTaskSummary {
	out := map[uint]nodeTaskSummary{}
	if len(nodes) == 0 {
		return out
	}
	db := database.GetDB()
	if db == nil {
		return out
	}

	ids := make([]uint, 0, len(nodes))
	for i := range nodes {
		ids = append(ids, nodes[i].ID)
	}

	type row struct {
		NodeID      uint      `gorm:"column:node_id"`
		ProjectName string    `gorm:"column:project_name"`
		Status      string    `gorm:"column:status"`
		UpdatedAt   time.Time `gorm:"column:updated_at"`
		LastError   string    `gorm:"column:last_error"`
		ErrorCode   string    `gorm:"column:error_code"`
		Payload     string    `gorm:"column:payload"`
	}
	var rows []row
	_ = db.WithContext(ctx).Raw(
		`SELECT t.node_id, t.project_name, t.status, t.updated_at, t.last_error, t.error_code, t.payload
		   FROM sync_tasks t
		   JOIN (SELECT node_id, MAX(id) AS max_id
		           FROM sync_tasks
		          WHERE node_id IN (?)
		          GROUP BY node_id) x
		     ON x.max_id = t.id`,
		ids,
	).Scan(&rows).Error

	for i := range rows {
		t := rows[i].UpdatedAt
		targetPath := ""
		if strings.TrimSpace(rows[i].Payload) != "" {
			var payload struct {
				TargetPath string `json:"targetPath"`
			}
			_ = json.Unmarshal([]byte(rows[i].Payload), &payload)
			targetPath = strings.TrimSpace(payload.TargetPath)
		}
		out[rows[i].NodeID] = nodeTaskSummary{
			ProjectName: rows[i].ProjectName,
			Status:      rows[i].Status,
			UpdatedAt:   &t,
			LastError:   rows[i].LastError,
			ErrorCode:   rows[i].ErrorCode,
			TargetPath:  targetPath,
		}
	}
	return out
}

func mapNodes(nodes []database.SyncNode, summary map[uint]nodeTaskSummary) []nodeResponse {
	result := make([]nodeResponse, 0, len(nodes))
	for i := range nodes {
		result = append(result, mapNode(&nodes[i], summary[nodes[i].ID]))
	}
	return result
}

func mapNode(node *database.SyncNode, summary nodeTaskSummary) nodeResponse {
	// ConnectionStatus prefers in-memory TCP connection registry for real-time status.
	// Fallback to persisted pairing/lastSeen when registry has no entry (e.g. after restart).
	lastSeen := node.LastSeen
	hasHistory := lastSeen != nil || strings.TrimSpace(node.AgentCertFingerprint) != ""
	connected := false
	if st, ok := getConnState(node.ID); ok {
		connected = st.Connected
		if !st.LastSeen.IsZero() {
			ts := st.LastSeen
			lastSeen = &ts
		}
	} else if node.LastSeen != nil {
		// Backward-compat: if we have DB lastSeen but no in-memory entry, assume disconnected.
		connected = false
	}

	connectionStatus := "UNPAIRED"
	if connected {
		connectionStatus = "CONNECTED"
	} else if hasHistory {
		connectionStatus = "DISCONNECTED"
	}

	// Normalize status/health for UI consistency.
	status := node.Status
	health := node.Health
	if connected {
		status = NodeStatusOnline
		health = NodeHealthHealthy
	} else {
		// When not connected, don't show stale HEALTHY from old install routines.
		status = NodeStatusOffline
		health = NodeHealthUnknown
	}

	syncStatus := "IDLE"
	lastSyncAt := (*time.Time)(nil)
	lastTaskProject := ""
	lastTaskTargetPath := ""
	lastError := ""
	lastErrorCode := ""
	if strings.TrimSpace(summary.Status) != "" {
		syncStatus = strings.ToUpper(summary.Status)
		lastSyncAt = summary.UpdatedAt
		lastTaskProject = strings.TrimSpace(summary.ProjectName)
		lastTaskTargetPath = strings.TrimSpace(summary.TargetPath)
		switch strings.ToLower(summary.Status) {
		case "failed", "retrying":
			lastError = strings.TrimSpace(summary.LastError)
			lastErrorCode = strings.TrimSpace(summary.ErrorCode)
		}
	}

	return nodeResponse{
		ID:                   node.ID,
		Name:                 node.Name,
		Address:              node.Address,
		Remark:               node.Remark,
		Type:                 node.Type,
		Status:               status,
		Health:               health,
		AgentCertFingerprint: node.AgentCertFingerprint,
		ConnectionStatus:     connectionStatus,
		SyncStatus:           syncStatus,
		LastSyncAt:           lastSyncAt,
		LastTaskProject:      lastTaskProject,
		LastTaskTargetPath:   lastTaskTargetPath,
		LastError:            lastError,
		LastErrorCode:        lastErrorCode,
		Tags:                 decodeStringSlice(node.Tags),
		Metadata:             decodeMap(node.Metadata),
		SSHUser:              node.SSHUser,
		SSHPort:              node.SSHPort,
		AuthType:             node.AuthType,
		CredentialRef:        node.CredentialRef,
		AgentToken:           node.CredentialValue,
		InstallStatus:        node.InstallStatus,
		InstallLog:           node.InstallLog,
		AgentVersion:         node.AgentVersion,
		LastSeen:             lastSeen,
		CreatedAt:            node.CreatedAt,
		UpdatedAt:            node.UpdatedAt,
	}
}

func parseIDParam(param string) (uint, error) {
	id, err := strconv.Atoi(param)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid id: %s", param)
	}
	return uint(id), nil
}
