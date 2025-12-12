package syncnode

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	Type                 string                 `json:"type"`
	Status               string                 `json:"status"`
	Health               string                 `json:"health"`
	AgentCertFingerprint string                 `json:"agentCertFingerprint"`
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

	c.JSON(http.StatusOK, mapNodes(nodes))
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

	c.JSON(http.StatusOK, mapNode(node))
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

	c.JSON(http.StatusCreated, mapNode(node))
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

	c.JSON(http.StatusOK, mapNode(node))
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

	c.JSON(http.StatusAccepted, mapNode(node))
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

	c.JSON(http.StatusOK, mapNode(node))
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
	c.JSON(http.StatusOK, mapNode(node))
}

func mapNodes(nodes []database.SyncNode) []nodeResponse {
	result := make([]nodeResponse, 0, len(nodes))
	for i := range nodes {
		result = append(result, mapNode(&nodes[i]))
	}
	return result
}

func mapNode(node *database.SyncNode) nodeResponse {
	return nodeResponse{
		ID:                   node.ID,
		Name:                 node.Name,
		Address:              node.Address,
		Type:                 node.Type,
		Status:               node.Status,
		Health:               node.Health,
		AgentCertFingerprint: node.AgentCertFingerprint,
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
		LastSeen:             node.LastSeen,
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
