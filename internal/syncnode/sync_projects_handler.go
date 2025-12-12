package syncnode

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/types"
)

type syncProjectNodeSummary struct {
	NodeID        uint       `json:"nodeId"`
	NodeName      string     `json:"nodeName"`
	Health        string     `json:"health"`
	TargetPath    string     `json:"targetPath"`
	LastStatus    string     `json:"lastStatus,omitempty"`
	LastTaskAt    *time.Time `json:"lastTaskAt,omitempty"`
	LastSuccessAt *time.Time `json:"lastSuccessAt,omitempty"`
}

type syncProjectSummary struct {
	ProjectName string                   `json:"projectName"`
	Path        string                   `json:"path"`
	Sync        *types.ProjectSyncConfig `json:"sync"`
	Status      string                   `json:"status"`
	LastSyncAt  *time.Time               `json:"lastSyncAt,omitempty"`
	Nodes       []syncProjectNodeSummary `json:"nodes"`
}

// Admin: list all sync-enabled projects (Syncthing-like folders) with status summary.
func HandleListSyncProjects(c *gin.Context) {
	versionData := types.GoHookVersionData
	if versionData == nil {
		c.JSON(http.StatusOK, []syncProjectSummary{})
		return
	}

	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	var out []syncProjectSummary
	for i := range versionData.Projects {
		project := versionData.Projects[i]
		if !project.Enabled || project.Sync == nil || !project.Sync.Enabled {
			continue
		}

		nodeCfgs := project.Sync.Nodes
		nodeIDs := make([]uint, 0, len(nodeCfgs))
		cfgByID := map[uint]types.ProjectSyncNodeConfig{}
		for _, cfg := range nodeCfgs {
			id, err := strconv.ParseUint(strings.TrimSpace(cfg.NodeID), 10, 64)
			if err != nil || id == 0 {
				continue
			}
			uid := uint(id)
			nodeIDs = append(nodeIDs, uid)
			cfgByID[uid] = cfg
		}

		nodesByID := map[uint]database.SyncNode{}
		if len(nodeIDs) > 0 {
			var nodes []database.SyncNode
			_ = db.WithContext(c.Request.Context()).Where("id IN ?", nodeIDs).Find(&nodes).Error
			for i := range nodes {
				nodesByID[nodes[i].ID] = nodes[i]
			}
		}

		type lastTaskRow struct {
			NodeID    uint
			Status    string
			UpdatedAt time.Time
		}
		var lastTasks []lastTaskRow
		_ = db.WithContext(c.Request.Context()).Raw(
			`SELECT node_id, status, updated_at
			   FROM sync_tasks
			  WHERE project_name = ?
			    AND id IN (SELECT MAX(id) FROM sync_tasks WHERE project_name = ? GROUP BY node_id)`,
			project.Name, project.Name,
		).Scan(&lastTasks).Error
		lastTaskByNode := map[uint]lastTaskRow{}
		for i := range lastTasks {
			lastTaskByNode[lastTasks[i].NodeID] = lastTasks[i]
		}

		type lastSuccessRow struct {
			NodeID        uint
			LastSuccessAt sql.NullTime
		}
		var lastSuccess []lastSuccessRow
		_ = db.WithContext(c.Request.Context()).Raw(
			`SELECT node_id, MAX(updated_at) AS last_success_at
			   FROM sync_tasks
			  WHERE project_name = ?
			    AND status = 'success'
			  GROUP BY node_id`,
			project.Name,
		).Scan(&lastSuccess).Error
		lastSuccessByNode := map[uint]time.Time{}
		for i := range lastSuccess {
			if lastSuccess[i].LastSuccessAt.Valid {
				lastSuccessByNode[lastSuccess[i].NodeID] = lastSuccess[i].LastSuccessAt.Time
			}
		}

		summary := syncProjectSummary{
			ProjectName: project.Name,
			Path:        project.Path,
			Sync:        project.Sync,
			Status:      "PENDING",
			Nodes:       make([]syncProjectNodeSummary, 0, len(nodeIDs)),
		}

		var lastSyncAt *time.Time
		anyFailed := false
		anyRunning := false
		anyOffline := false

		for _, nodeID := range nodeIDs {
			node := nodesByID[nodeID]
			cfg := cfgByID[nodeID]

			n := syncProjectNodeSummary{
				NodeID:     nodeID,
				NodeName:   node.Name,
				Health:     node.Health,
				TargetPath: cfg.TargetPath,
			}

			if lt, ok := lastTaskByNode[nodeID]; ok {
				n.LastStatus = lt.Status
				n.LastTaskAt = &lt.UpdatedAt
				if lt.Status == "failed" {
					anyFailed = true
				}
				if lt.Status == "running" {
					anyRunning = true
				}
			}
			if ts, ok := lastSuccessByNode[nodeID]; ok {
				n.LastSuccessAt = &ts
				if lastSyncAt == nil || ts.After(*lastSyncAt) {
					copy := ts
					lastSyncAt = &copy
				}
			}

			if strings.EqualFold(node.Health, "OFFLINE") || strings.EqualFold(node.Status, "OFFLINE") {
				anyOffline = true
			}

			summary.Nodes = append(summary.Nodes, n)
		}

		switch {
		case len(nodeIDs) == 0:
			summary.Status = "MISCONFIGURED"
		case anyFailed || anyOffline:
			summary.Status = "DEGRADED"
		case anyRunning:
			summary.Status = "SYNCING"
		case lastSyncAt != nil:
			summary.Status = "HEALTHY"
		default:
			summary.Status = "PENDING"
		}
		summary.LastSyncAt = lastSyncAt

		out = append(out, summary)
	}

	c.JSON(http.StatusOK, out)
}

// Admin: update only the sync config for a project.
func HandleUpdateProjectSyncConfig(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Sync types.ProjectSyncConfig `json:"sync" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	versionData := types.GoHookVersionData
	if versionData == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "version data not initialized"})
		return
	}

	idx := -1
	for i := range versionData.Projects {
		if versionData.Projects[i].Name == projectName {
			idx = i
			break
		}
	}
	if idx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	versionData.Projects[idx].Sync = &req.Sync
	if err := config.SaveVersionConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Save config failed: " + err.Error()})
		return
	}

	// Ensure watchers reflect updated ignore rules / enabled state.
	RefreshProjectWatchers()

	c.JSON(http.StatusOK, gin.H{"message": "Sync config updated successfully"})
}
