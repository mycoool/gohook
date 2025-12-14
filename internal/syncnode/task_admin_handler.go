package syncnode

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/database"
	"gorm.io/gorm"
)

type adminTaskResponse struct {
	ID          uint      `json:"id"`
	ProjectName string    `json:"projectName"`
	NodeID      uint      `json:"nodeId"`
	NodeName    string    `json:"nodeName"`
	Driver      string    `json:"driver"`
	Status      string    `json:"status"`
	Attempt     int       `json:"attempt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	LastError   string    `json:"lastError,omitempty"`
	ErrorCode   string    `json:"errorCode,omitempty"`
	Files       int       `json:"files,omitempty"`
	Blocks      int       `json:"blocks,omitempty"`
	Bytes       int64     `json:"bytes,omitempty"`
	DurationMs  int64     `json:"durationMs,omitempty"`
	Logs        string    `json:"logs,omitempty"`
}

// Admin: list recent tasks for debugging.
// Query:
// - projectName (optional)
// - nodeId (optional)
// - status (optional)
// - limit (default 50, max 200)
// - includeLogs (optional, true/false)
func HandleListTasks(c *gin.Context) {
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	limit := 50
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	if limit > 200 {
		limit = 200
	}
	includeLogs := strings.EqualFold(strings.TrimSpace(c.Query("includeLogs")), "true")
	beforeID := uint(0)
	if raw := strings.TrimSpace(c.Query("beforeId")); raw != "" {
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil && v > 0 {
			beforeID = uint(v)
		}
	}

	query := db.WithContext(c.Request.Context()).Model(&database.SyncTask{})
	if pn := strings.TrimSpace(c.Query("projectName")); pn != "" {
		query = query.Where("project_name = ?", pn)
	}
	if raw := strings.TrimSpace(c.Query("nodeId")); raw != "" {
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil && v > 0 {
			query = query.Where("node_id = ?", uint(v))
		}
	}
	if st := strings.TrimSpace(c.Query("status")); st != "" {
		query = query.Where("status = ?", strings.ToLower(st))
	}
	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}

	var tasks []database.SyncTask
	if err := query.Order("id DESC").Limit(limit).Find(&tasks).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]adminTaskResponse, 0, len(tasks))
	for i := range tasks {
		t := tasks[i]
		resp := adminTaskResponse{
			ID:          t.ID,
			ProjectName: t.ProjectName,
			NodeID:      t.NodeID,
			NodeName:    t.NodeName,
			Driver:      t.Driver,
			Status:      t.Status,
			Attempt:     t.Attempt,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
			LastError:   t.LastError,
			ErrorCode:   t.ErrorCode,
			Files:       t.FilesTotal,
			Blocks:      t.BlocksTotal,
			Bytes:       t.BytesTotal,
			DurationMs:  t.DurationMs,
		}
		if includeLogs {
			resp.Logs = t.Logs
		}
		out = append(out, resp)
	}

	c.JSON(http.StatusOK, out)
}

// Admin: clear task records (optionally filtered).
// Query:
// - projectName (optional)
// - nodeId (optional)
// - status (optional)
// - includeActive (optional, true/false; default false)
func HandleClearTasks(c *gin.Context) {
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	includeActive := strings.EqualFold(strings.TrimSpace(c.Query("includeActive")), "true")

	query := db.WithContext(c.Request.Context()).Model(&database.SyncTask{})
	if pn := strings.TrimSpace(c.Query("projectName")); pn != "" {
		query = query.Where("project_name = ?", pn)
	}
	if raw := strings.TrimSpace(c.Query("nodeId")); raw != "" {
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil && v > 0 {
			query = query.Where("node_id = ?", uint(v))
		}
	}
	if st := strings.TrimSpace(c.Query("status")); st != "" {
		query = query.Where("status = ?", strings.ToLower(st))
	} else if !includeActive {
		// Default: only clear completed records.
		query = query.Where("status IN ?", []string{"success", "failed"})
	}

	res := query.Delete(&database.SyncTask{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

// Admin: get one task by id (optionally include logs).
// Query:
// - includeLogs (optional, true/false; default true)
func HandleGetTask(c *gin.Context) {
	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}

	id, err := parseIDParam(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	includeLogs := true
	if raw := strings.TrimSpace(c.Query("includeLogs")); raw != "" {
		includeLogs = strings.EqualFold(raw, "true")
	}

	var t database.SyncTask
	if err := db.WithContext(c.Request.Context()).First(&t, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := adminTaskResponse{
		ID:          t.ID,
		ProjectName: t.ProjectName,
		NodeID:      t.NodeID,
		NodeName:    t.NodeName,
		Driver:      t.Driver,
		Status:      t.Status,
		Attempt:     t.Attempt,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		LastError:   t.LastError,
		ErrorCode:   t.ErrorCode,
		Files:       t.FilesTotal,
		Blocks:      t.BlocksTotal,
		Bytes:       t.BytesTotal,
		DurationMs:  t.DurationMs,
	}
	if includeLogs {
		resp.Logs = t.Logs
	}
	c.JSON(http.StatusOK, resp)
}
