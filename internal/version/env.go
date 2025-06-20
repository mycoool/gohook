package version

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/env"
	"github.com/mycoool/gohook/internal/types"
)

// get project environment variable file (.env)
func GetEnv(c *gin.Context) {
	projectName := c.Param("name")

	// find project path
	var projectPath string
	for _, proj := range types.GoHookVersionData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	envContent, exists, err := env.GetEnvFile(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content": envContent,
		"exists":  exists,
		"path":    filepath.Join(projectPath, ".env"),
	})
}

// SaveEnv save project environment variable file (.env)
func SaveEnv(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// find project path
	var projectPath string
	for _, proj := range types.GoHookVersionData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// validate environment variable file format
	if errors := env.ValidateEnvContent(req.Content); len(errors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Environment variable file format validation failed",
			"details": errors,
		})
		return
	}

	// save environment variable file
	if err := env.SaveEnvFile(projectPath, req.Content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Environment variable file saved successfully",
		"path":    filepath.Join(projectPath, ".env"),
	})
}

// DeleteEnv delete project environment variable file (.env)
func DeleteEnv(c *gin.Context) {
	projectName := c.Param("name")

	// find project path
	var projectPath string
	for _, proj := range types.GoHookVersionData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if err := env.DeleteEnvFile(projectPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Environment variable file deleted successfully",
	})
}
