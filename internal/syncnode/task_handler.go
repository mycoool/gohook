package syncnode

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/database"
	"gorm.io/gorm"
)

var defaultTaskService = NewTaskService()

type taskResponse struct {
	ID          uint   `json:"id"`
	ProjectName string `json:"projectName"`
	NodeID      uint   `json:"nodeId"`
	NodeName    string `json:"nodeName"`
	Driver      string `json:"driver"`
	Status      string `json:"status"`
	Attempt     int    `json:"attempt"`
	Payload     string `json:"payload"`
	Logs        string `json:"logs"`
	LastError   string `json:"lastError"`
	ErrorCode   string `json:"errorCode"`
}

func mapTask(t *database.SyncTask) taskResponse {
	return taskResponse{
		ID:          t.ID,
		ProjectName: t.ProjectName,
		NodeID:      t.NodeID,
		NodeName:    t.NodeName,
		Driver:      t.Driver,
		Status:      t.Status,
		Attempt:     t.Attempt,
		Payload:     t.Payload,
		Logs:        t.Logs,
		LastError:   t.LastError,
		ErrorCode:   t.ErrorCode,
	}
}

// Admin: manually enqueue tasks for a project (useful until controller is implemented).
func HandleRunProjectSync(c *gin.Context) {
	projectName := c.Param("name")
	tasks, err := defaultTaskService.CreateProjectTasks(c.Request.Context(), projectName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out := make([]taskResponse, 0, len(tasks))
	for i := range tasks {
		out = append(out, mapTask(&tasks[i]))
	}
	c.JSON(http.StatusCreated, out)
}

// Agent: pull next pending task for node, marking it running.
func HandlePullTask(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := defaultTaskService.PullNextTask(c.Request.Context(), id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mapTask(task))
}

// Agent: report task completion.
func HandleReportTask(c *gin.Context) {
	nodeID, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	taskID, err := parseIDParam(c.Param("taskId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req TaskReport
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := defaultTaskService.ReportTask(c.Request.Context(), nodeID, taskID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mapTask(task))
}

// Agent: download a tar.gz bundle for the task.
func HandleDownloadBundle(c *gin.Context) {
	nodeID, err := parseIDParam(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	taskID, err := parseIDParam(c.Param("taskId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	if db == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialized"})
		return
	}
	var task database.SyncTask
	if err := db.WithContext(c.Request.Context()).First(&task, taskID).Error; err != nil {
		status := http.StatusInternalServerError
		if err == gorm.ErrRecordNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	if task.NodeID != nodeID {
		c.JSON(http.StatusForbidden, gin.H{"error": "task does not belong to node"})
		return
	}

	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", "attachment; filename=\"gohook-sync-bundle.tar.gz\"")
	if err := defaultTaskService.StreamBundle(c.Request.Context(), c.Writer, task); err != nil {
		// headers already sent; best-effort abort by closing connection
		_ = c.Error(err)
		return
	}
}
