package version

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/types"
	wsmanager "github.com/mycoool/gohook/websocket"
)

// GetVersionsHandler 获取所有项目版本信息
func GetVersionsHandler(c *gin.Context) {
	if configData == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "配置未加载"})
		return
	}

	var versions []types.VersionResponse
	for _, project := range configData.Projects {
		if !project.Enabled {
			continue
		}

		version := types.VersionResponse{
			Name:        project.Name,
			Path:        project.Path,
			Description: project.Description,
			Mode:        "none",
			Status:      getProjectStatus(project.Path),
		}

		if isGitRepository(project.Path) {
			// Git项目
			if currentBranch, err := getCurrentBranch(project.Path); err == nil {
				version.CurrentBranch = currentBranch
				version.Mode = "branch"
			}

			if currentTag, err := getCurrentTag(project.Path); err == nil && currentTag != "" {
				version.CurrentTag = currentTag
				version.Mode = "tag"
			}

			if lastCommit, lastCommitTime, err := getLastCommit(project.Path); err == nil {
				version.LastCommit = lastCommit
				version.LastCommitTime = lastCommitTime
			}
		}

		versions = append(versions, version)
	}

	c.JSON(http.StatusOK, versions)
}

// GetProjectBranchesHandler 获取项目分支列表
func GetProjectBranchesHandler(c *gin.Context) {
	projectName := c.Param("name")

	// 查找项目路径
	var projectPath string
	for _, proj := range configData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	if !isGitRepository(projectPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不是Git项目"})
		return
	}

	branches, err := getBranches(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分支列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, branches)
}

// GetProjectTagsHandler 获取项目标签列表
func GetProjectTagsHandler(c *gin.Context) {
	projectName := c.Param("name")

	// 查找项目路径
	var projectPath string
	for _, proj := range configData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	if !isGitRepository(projectPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不是Git项目"})
		return
	}

	tags, err := getTags(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取标签列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// SwitchBranchHandler 切换分支
func SwitchBranchHandler(c *gin.Context) {
	projectName := c.Param("name")
	branchName := c.Param("branchName")

	// 查找项目路径
	var projectPath string
	for _, proj := range configData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	if !isGitRepository(projectPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不是Git项目"})
		return
	}

	err := SwitchBranch(projectPath, branchName)

	// 推送WebSocket消息
	wsMessage := wsmanager.Message{
		Type:      "version_switched",
		Timestamp: time.Now(),
		Data: wsmanager.VersionSwitchMessage{
			ProjectName: projectName,
			Action:      "switch-branch",
			Target:      branchName,
			Success:     err == nil,
			Error: func() string {
				if err != nil {
					return err.Error()
				} else {
					return ""
				}
			}(),
		},
	}
	wsmanager.Global.Broadcast(wsMessage)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "分支切换成功"})
}

// SwitchTagHandler 切换标签
func SwitchTagHandler(c *gin.Context) {
	projectName := c.Param("name")
	tagName := c.Param("tagName")

	// 查找项目路径
	var projectPath string
	for _, proj := range configData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	if !isGitRepository(projectPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不是Git项目"})
		return
	}

	err := SwitchTag(projectPath, tagName)

	// 推送WebSocket消息
	wsMessage := wsmanager.Message{
		Type:      "version_switched",
		Timestamp: time.Now(),
		Data: wsmanager.VersionSwitchMessage{
			ProjectName: projectName,
			Action:      "switch-tag",
			Target:      tagName,
			Success:     err == nil,
			Error: func() string {
				if err != nil {
					return err.Error()
				} else {
					return ""
				}
			}(),
		},
	}
	wsmanager.Global.Broadcast(wsMessage)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "标签切换成功"})
}

// DeleteTagHandler 删除标签
func DeleteTagHandler(c *gin.Context) {
	projectName := c.Param("name")
	tagName := c.Param("tagName")

	// 查找项目路径
	var projectPath string
	for _, proj := range configData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	if !isGitRepository(projectPath) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不是Git项目"})
		return
	}

	err := DeleteTag(projectPath, tagName)

	// 推送WebSocket消息
	wsMessage := wsmanager.Message{
		Type:      "version_switched",
		Timestamp: time.Now(),
		Data: wsmanager.VersionSwitchMessage{
			ProjectName: projectName,
			Action:      "delete-tag",
			Target:      tagName,
			Success:     err == nil,
			Error: func() string {
				if err != nil {
					return err.Error()
				} else {
					return ""
				}
			}(),
		},
	}
	wsmanager.Global.Broadcast(wsMessage)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "标签删除成功"})
}

// GetEnvFileHandler 获取环境文件
func GetEnvFileHandler(c *gin.Context) {
	projectName := c.Param("name")

	// 查找项目路径
	var projectPath string
	for _, proj := range configData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	content, err := GetEnvFile(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content})
}

// SaveEnvFileHandler 保存环境文件
func SaveEnvFileHandler(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 查找项目路径
	var projectPath string
	for _, proj := range configData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	err := SaveEnvFile(projectPath, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "环境文件保存成功"})
}

// DeleteEnvFileHandler 删除环境文件
func DeleteEnvFileHandler(c *gin.Context) {
	projectName := c.Param("name")

	// 查找项目路径
	var projectPath string
	for _, proj := range configData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	err := DeleteEnvFile(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "环境文件删除成功"})
}

// AddProjectHandler 添加项目
func AddProjectHandler(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Path        string `json:"path" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// 检查项目是否已存在
	for _, proj := range configData.Projects {
		if proj.Name == req.Name {
			c.JSON(http.StatusConflict, gin.H{"error": "项目名称已存在"})
			return
		}
	}

	// 检查路径是否存在
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "项目路径不存在"})
		return
	}

	// 添加新项目
	newProject := types.ProjectConfig{
		Name:        req.Name,
		Path:        req.Path,
		Description: req.Description,
		Enabled:     true,
	}

	configData.Projects = append(configData.Projects, newProject)

	// 保存配置文件
	if err := SaveConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
		return
	}

	// 推送WebSocket消息
	wsMessage := wsmanager.Message{
		Type:      "project_managed",
		Timestamp: time.Now(),
		Data: wsmanager.ProjectManageMessage{
			Action:      "add",
			ProjectName: req.Name,
			ProjectPath: req.Path,
			Success:     true,
		},
	}
	wsmanager.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "项目添加成功",
		"project": newProject,
	})
}

// DeleteProjectHandler 删除项目
func DeleteProjectHandler(c *gin.Context) {
	projectName := c.Param("name")

	// 查找项目索引
	projectIndex := -1
	for i, proj := range configData.Projects {
		if proj.Name == projectName {
			projectIndex = i
			break
		}
	}

	if projectIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "项目未找到"})
		return
	}

	// 删除项目
	configData.Projects = append(configData.Projects[:projectIndex], configData.Projects[projectIndex+1:]...)

	// 保存配置文件
	if err := SaveConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败: " + err.Error()})
		return
	}

	// 推送WebSocket消息
	wsMessage := wsmanager.Message{
		Type:      "project_managed",
		Timestamp: time.Now(),
		Data: wsmanager.ProjectManageMessage{
			Action:      "delete",
			ProjectName: projectName,
			Success:     true,
		},
	}
	wsmanager.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "项目删除成功",
	})
}
