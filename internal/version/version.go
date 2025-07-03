package version

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/middleware"
	"github.com/mycoool/gohook/internal/stream"
	"github.com/mycoool/gohook/internal/types"
)

// execGitCommand 执行Git命令，自动处理safe.directory权限问题
func execGitCommand(projectPath string, args ...string) ([]byte, error) {
	// 首先尝试正常执行Git命令
	cmd := exec.Command("git", append([]string{"-C", projectPath}, args...)...)
	output, err := cmd.CombinedOutput()

	// 如果成功或者不是safe.directory相关错误，直接返回
	if err == nil {
		return output, nil
	}

	outputStr := string(output)
	// 检查是否是safe.directory相关错误
	if !strings.Contains(outputStr, "safe.directory") && !strings.Contains(outputStr, "detected dubious ownership") {
		return output, err
	}

	log.Printf("检测到Git安全目录问题，尝试自动修复: %s", projectPath)

	// 尝试添加到safe.directory (全局系统级配置)
	safeCmd := exec.Command("git", "config", "--system", "--add", "safe.directory", projectPath)
	if safeOutput, safeErr := safeCmd.CombinedOutput(); safeErr != nil {
		log.Printf("尝试系统级safe.directory配置失败: %s", string(safeOutput))

		// 如果系统级配置失败，尝试全局用户级配置
		safeCmd = exec.Command("git", "config", "--global", "--add", "safe.directory", projectPath)
		if safeOutput, safeErr := safeCmd.CombinedOutput(); safeErr != nil {
			log.Printf("尝试全局safe.directory配置也失败: %s", string(safeOutput))
			return output, fmt.Errorf("git safe.directory configuration failed: %v. Original error: %v", safeErr, err)
		} else {
			log.Printf("成功配置全局safe.directory: %s", projectPath)
		}
	} else {
		log.Printf("成功配置系统级safe.directory: %s", projectPath)
	}

	// 重新尝试执行原始Git命令
	cmd = exec.Command("git", append([]string{"-C", projectPath}, args...)...)
	retryOutput, retryErr := cmd.CombinedOutput()
	if retryErr != nil {
		log.Printf("配置safe.directory后重试仍失败: %s", string(retryOutput))
		return retryOutput, fmt.Errorf("git command failed even after safe.directory configuration: %v", retryErr)
	}

	log.Printf("配置safe.directory后Git命令执行成功: %s", projectPath)
	return retryOutput, nil
}

// execGitCommandOutput 执行Git命令并返回输出，使用safe.directory自动修复
func execGitCommandOutput(projectPath string, args ...string) ([]byte, error) {
	return execGitCommand(projectPath, args...)
}

// execGitCommandRun 执行Git命令，只返回错误，使用safe.directory自动修复
func execGitCommandRun(projectPath string, args ...string) error {
	_, err := execGitCommand(projectPath, args...)
	return err
}

// init Git repository
func initGit(projectPath string) error {
	// check if project path exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", projectPath)
	}

	// check if it is a directory
	if info, err := os.Stat(projectPath); err != nil {
		return fmt.Errorf("cannot access project path: %s, error: %v", projectPath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("project path is not a directory: %s", projectPath)
	}

	// check if it is already a Git repository
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return fmt.Errorf("directory is already a Git repository")
	}

	// try to create a temporary file to test write permission
	testFile := filepath.Join(projectPath, ".gohook-permission-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("project path does not have write permission: %s, please check directory permission. recommended: sudo chown -R %s:%s %s",
			projectPath, os.Getenv("USER"), os.Getenv("USER"), projectPath)
	}
	// clean up test file
	os.Remove(testFile)

	// execute git init command with safe.directory support
	output, err := execGitCommand(projectPath, "init")
	if err != nil {
		return fmt.Errorf("git repository initialization failed: %v, output: %s", err, string(output))
	}

	// verify if Git repository is successfully created
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("git repository initialization verification failed: .git directory not created")
	}

	return nil
}

// switchAndPullBranch switch to specified branch and pull latest code
func switchAndPullBranch(projectPath, branchName string) error {
	// check if local branch exists
	output, err := execGitCommandOutput(projectPath, "branch", "--list", branchName)
	localBranchExists := err == nil && strings.TrimSpace(string(output)) != ""

	if !localBranchExists {
		// local branch does not exist, try to create from remote
		if output, err := execGitCommand(projectPath, "checkout", "-b", branchName, "origin/"+branchName); err != nil {
			return fmt.Errorf("create and switch to branch %s failed: %s", branchName, string(output))
		}
	} else {
		// local branch exists, switch directly
		if output, err := execGitCommand(projectPath, "checkout", branchName); err != nil {
			return fmt.Errorf("switch to branch %s failed: %s", branchName, string(output))
		}

		// fetch latest code
		if output, err := execGitCommand(projectPath, "pull", "origin", branchName); err != nil {
			return fmt.Errorf("failed to fetch latest code for branch %s: %s", branchName, string(output))
		}
	}

	return nil
}

// switchToTag switch to specified tag
func switchToTag(projectPath, tagName string) error {
	// fetch tag information
	if output, err := execGitCommand(projectPath, "fetch", "--tags"); err != nil {
		log.Printf("warning: failed to fetch tag information: %s", string(output))
	}

	// ensure tag exists (local or remote)
	if err := execGitCommandRun(projectPath, "rev-parse", tagName); err != nil {
		log.Printf("tag %s does not exist, try to fetch from remote", tagName)
		if output, err := execGitCommand(projectPath, "fetch", "origin", "--tags"); err != nil {
			return fmt.Errorf("failed to fetch tag from remote: %s", string(output))
		}

		// check if tag exists again
		if err := execGitCommandRun(projectPath, "rev-parse", tagName); err != nil {
			return fmt.Errorf("tag %s does not exist on remote, cannot deploy", tagName)
		}
	}

	// switch to specified tag
	if output, err := execGitCommand(projectPath, "checkout", tagName); err != nil {
		return fmt.Errorf("switch to tag %s failed: %s", tagName, string(output))
	}

	log.Printf("successfully switched to tag: %s", tagName)
	return nil
}

// deleteLocalTag delete local tag
func deleteLocalTag(projectPath, tagName string) error {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository: %s", projectPath)
	}

	// check if tag exists
	if err := execGitCommandRun(projectPath, "show-ref", "--tags", "--quiet", "refs/tags/"+tagName); err != nil {
		log.Printf("local tag %s does not exist, skip deletion", tagName)
		return nil
	}

	// delete local tag
	if output, err := execGitCommand(projectPath, "tag", "-d", tagName); err != nil {
		return fmt.Errorf("delete local tag %s failed: %s", tagName, string(output))
	}

	log.Printf("successfully deleted local tag: %s", tagName)
	return nil
}

func HandleDeleteLocalBranch(c *gin.Context) {
	projectName := c.Param("name")
	branchName := c.Param("branchName")
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

	if err := deleteLocalBranch(projectPath, branchName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Branch deleted successfully"})
}

// deleteLocalBranch delete local branch
func deleteLocalBranch(projectPath, branchName string) error {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository: %s", projectPath)
	}

	// get current branch
	currentBranchOutput, err := execGitCommandOutput(projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("get current branch failed: %v", err)
	}
	currentBranch := strings.TrimSpace(string(currentBranchOutput))

	// check if trying to delete current branch
	if currentBranch == branchName {
		log.Printf("cannot delete current branch %s, skip deletion", branchName)
		return nil
	}

	// check if branch exists
	if err := execGitCommandRun(projectPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName); err != nil {
		log.Printf("local branch %s does not exist, skip deletion", branchName)
		return nil
	}

	// delete local branch
	if output, err := execGitCommand(projectPath, "branch", "-D", branchName); err != nil {
		return fmt.Errorf("delete local branch %s failed: %s", branchName, string(output))
	}

	log.Printf("successfully deleted local branch: %s", branchName)
	return nil
}

// DeleteBranch delete local branch
func deleteBranch(projectPath, branchName string) error {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository")
	}

	// get current branch
	currentBranchOutput, err := execGitCommandOutput(projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return fmt.Errorf("get current branch failed: %v", err)
	}
	currentBranch := strings.TrimSpace(string(currentBranchOutput))

	// check if trying to delete current branch
	if currentBranch == branchName {
		return fmt.Errorf("cannot delete current branch")
	}

	// delete local branch
	output, err := execGitCommand(projectPath, "branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("delete branch failed: %s", string(output))
	}

	return nil
}

// DeleteTag delete local and remote tag
func deleteTag(projectPath, tagName string) error {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository")
	}

	// check if current is on the tag
	currentTagOutput, err := execGitCommandOutput(projectPath, "describe", "--tags", "--exact-match", "HEAD")
	if err == nil {
		currentTag := strings.TrimSpace(string(currentTagOutput))
		if currentTag == tagName {
			return fmt.Errorf("cannot delete current tag")
		}
	}

	// delete local tag
	localOutput, localErr := execGitCommand(projectPath, "tag", "-d", tagName)
	if localErr != nil {
		return fmt.Errorf("delete local tag failed: %s", string(localOutput))
	}

	// try to delete remote tag
	remoteOutput, remoteErr := execGitCommand(projectPath, "push", "origin", ":refs/tags/"+tagName)
	if remoteErr != nil {
		log.Printf("delete remote tag failed (project: %s, tag: %s): %s", projectPath, tagName, string(remoteOutput))
		// remote tag deletion failed is not a fatal error, because it may not exist on remote
	}

	return nil
}

// SwitchTag switch tag
func switchTag(projectPath, tagName string) error {
	if err := execGitCommandRun(projectPath, "checkout", tagName); err != nil {
		return fmt.Errorf("switch tag failed: %v", err)
	}
	return nil
}

// SyncTags sync remote tags
func syncTags(projectPath string) error {
	output, err := execGitCommand(projectPath, "fetch", "origin", "--prune", "--tags")
	if err != nil {
		return fmt.Errorf("sync tags failed: %s", string(output))
	}
	return nil
}

// GetRemote get remote repository URL
func getRemote(projectPath string) (string, error) {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("not a Git repository")
	}

	// get origin remote repository URL
	output, err := execGitCommandOutput(projectPath, "remote", "get-url", "origin")
	if err != nil {
		// if "origin" does not exist, the command will return a non-zero exit code
		// in this case, we return an empty string, indicating that no remote address is set
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// SetRemote set remote repository
func setRemote(projectPath, remoteUrl string) error {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository")
	}

	// check if origin remote repository already exists
	if execGitCommandRun(projectPath, "remote", "get-url", "origin") == nil {
		// if origin already exists, delete it first
		if err := execGitCommandRun(projectPath, "remote", "remove", "origin"); err != nil {
			return fmt.Errorf("delete existing remote repository failed: %v", err)
		}
	}

	// add new origin remote repository
	if err := execGitCommandRun(projectPath, "remote", "add", "origin", remoteUrl); err != nil {
		return fmt.Errorf("set remote repository failed: %v", err)
	}

	return nil
}

// SyncBranches sync remote branches, clean up deleted remote branch references
func syncBranches(projectPath string) error {
	// use fetch --prune to update remote branch information and delete non-existent references
	output, err := execGitCommand(projectPath, "fetch", "origin", "--prune")
	if err != nil {
		return fmt.Errorf("sync branches failed: %s", string(output))
	}
	return nil
}

// SwitchBranch switch branch
func switchBranch(projectPath, branchName string) error {
	var isRemoteBranch bool
	var localBranchName string

	// check if it is a remote branch format (for example origin/release)
	if strings.HasPrefix(branchName, "origin/") {
		isRemoteBranch = true
		localBranchName = strings.TrimPrefix(branchName, "origin/")

		// check if local branch already exists
		if execGitCommandRun(projectPath, "rev-parse", "--verify", localBranchName) == nil {
			// local branch already exists, switch directly
			if output, err := execGitCommand(projectPath, "checkout", localBranchName); err != nil {
				return fmt.Errorf("switch branch failed: %s", string(output))
			}
		} else {
			// local branch does not exist, create a new local branch based on the remote branch
			if output, err := execGitCommand(projectPath, "checkout", "-b", localBranchName, branchName); err != nil {
				return fmt.Errorf("switch branch failed: %s", string(output))
			}
		}
	} else {
		// normal local branch switch
		isRemoteBranch = false
		localBranchName = branchName
		if output, err := execGitCommand(projectPath, "checkout", branchName); err != nil {
			return fmt.Errorf("switch branch failed: %s", string(output))
		}
	}

	// if a new branch is created based on a remote branch, try to pull the latest code
	if isRemoteBranch {
		pullOutput, pullErr := execGitCommand(projectPath, "pull", "origin", localBranchName)
		if pullErr != nil {
			// pull failed is not a fatal error, but log it
			log.Printf("pull latest code after switching branch failed (project: %s, branch: %s): %s", projectPath, localBranchName, string(pullOutput))
		}
	}

	return nil
}

// GetTags get tag list
func getTags(projectPath string) ([]types.TagResponse, error) {
	// get current tag
	currentOutput, _ := execGitCommandOutput(projectPath, "describe", "--exact-match", "--tags", "HEAD")
	currentTag := strings.TrimSpace(string(currentOutput))

	// get all tags
	output, err := execGitCommandOutput(projectPath, "tag", "-l", "--sort=-version:refname", "--format=%(refname:short)|%(creatordate)|%(objectname:short)|%(subject)")
	if err != nil {
		return nil, fmt.Errorf("get tag list failed: %v", err)
	}

	var tags []types.TagResponse
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			tagName := parts[0]
			tags = append(tags, types.TagResponse{
				Name:       tagName,
				IsCurrent:  tagName == currentTag,
				Date:       parts[1],
				CommitHash: parts[2],
				Message:    parts[3],
			})
		}
	}

	return tags, nil
}

// GetBranches get branch list
func getBranches(projectPath string) ([]types.BranchResponse, error) {
	var branches []types.BranchResponse
	branchSet := make(map[string]bool) // used to prevent duplicate addition

	// 1. get whether current is in detached head state
	_, err := execGitCommandOutput(projectPath, "symbolic-ref", "-q", "HEAD")
	isDetached := err != nil

	// 2. get current branch or commit reference
	var currentRef string
	if isDetached {
		// detached head state, get HEAD short hash
		headSha, err := execGitCommandOutput(projectPath, "rev-parse", "--short", "HEAD")
		if err != nil {
			return nil, fmt.Errorf("get HEAD commit failed: %v", err)
		}
		currentRef = strings.TrimSpace(string(headSha))
	} else {
		// on a branch, get branch name
		branchName, err := execGitCommandOutput(projectPath, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return nil, fmt.Errorf("get current branch name failed: %v", err)
		}
		currentRef = strings.TrimSpace(string(branchName))
	}

	// 3. handle detached head state
	if isDetached {
		// try to get tag name
		tagName, err := execGitCommandOutput(projectPath, "describe", "--tags", "--exact-match", "HEAD")
		var displayName string
		if err == nil {
			displayName = strings.TrimSpace(string(tagName))
		} else {
			displayName = currentRef
		}

		// get last commit information
		commitOutput, _ := execGitCommandOutput(projectPath, "log", "-1", "HEAD", "--format=%H|%ci")
		parts := strings.Split(strings.TrimSpace(string(commitOutput)), "|")
		lastCommit, lastCommitTime := "", ""
		if len(parts) > 0 {
			lastCommit = parts[0][:8]
		}
		if len(parts) > 1 {
			lastCommitTime = parts[1]
		}

		branches = append(branches, types.BranchResponse{
			Name:           fmt.Sprintf("(currently pointing to %s)", displayName),
			IsCurrent:      true,
			LastCommit:     lastCommit,
			LastCommitTime: lastCommitTime,
			Type:           "detached",
		})
	}

	// 4. get all local branches
	localOutput, err := execGitCommandOutput(projectPath, "for-each-ref", "refs/heads", "--format=%(refname:short)|%(committerdate:iso)|%(objectname:short)")
	if err != nil {
		return nil, fmt.Errorf("get local branch list failed: %v", err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(localOutput)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) >= 3 {
			branchName := parts[0]
			if branchSet[branchName] {
				continue
			}
			branchSet[branchName] = true
			branches = append(branches, types.BranchResponse{
				Name:           branchName,
				IsCurrent:      !isDetached && branchName == currentRef,
				LastCommitTime: parts[1],
				LastCommit:     parts[2],
				Type:           "local",
			})
		}
	}

	// 5. get all remote branches
	remoteOutput, err := execGitCommandOutput(projectPath, "for-each-ref", "refs/remotes", "--format=%(refname:short)|%(committerdate:iso)|%(objectname:short)")
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(remoteOutput)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "|", 3)
			if len(parts) >= 3 {
				remoteRef := parts[0]
				if strings.HasSuffix(remoteRef, "/HEAD") {
					continue // ignore HEAD pointer
				}
				branchName := remoteRef // for example "origin/master"
				if branchSet[branchName] {
					continue
				}
				branchSet[branchName] = true
				branches = append(branches, types.BranchResponse{
					Name:           branchName,
					IsCurrent:      false,
					LastCommitTime: parts[1],
					LastCommit:     parts[2],
					Type:           "remote",
				})
			}
		}
	} else {
		log.Printf("Get remote branch list failed (project: %s): %v", projectPath, err)
	}

	return branches, nil
}

// getGitStatus get Git status
func getGitStatus(projectPath string) (*types.VersionResponse, error) {
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a Git repository")
	}

	// get current branch
	branchOutput, _ := execGitCommandOutput(projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranch := strings.TrimSpace(string(branchOutput))

	// get current tag (if on a tag) - only if HEAD exactly matches a tag
	tagOutput, tagErr := execGitCommandOutput(projectPath, "describe", "--exact-match", "--tags", "HEAD")
	currentTag := ""
	if tagErr == nil {
		currentTag = strings.TrimSpace(string(tagOutput))
	}

	// determine mode - only tag mode if HEAD exactly matches a tag
	mode := "branch"
	if tagErr == nil && currentTag != "" {
		mode = "tag"
	}

	// get last commit information
	commitOutput, _ := execGitCommandOutput(projectPath, "log", "-1", "--format=%H|%ci|%s")
	commitInfo := strings.TrimSpace(string(commitOutput))

	parts := strings.Split(commitInfo, "|")
	lastCommit := ""
	lastCommitTime := ""
	if len(parts) >= 2 {
		lastCommit = parts[0][:8] // short hash
		lastCommitTime = parts[1]
	}

	return &types.VersionResponse{
		CurrentBranch:  currentBranch,
		CurrentTag:     currentTag,
		Mode:           mode,
		Status:         "active",
		LastCommit:     lastCommit,
		LastCommitTime: lastCommitTime,
	}, nil
}

// HandleEditProject edit project
func HandleEditProject(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Path        string `json:"path" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// find project index
	projectIndex := -1
	for i, proj := range types.GoHookVersionData.Projects {
		if proj.Name == projectName {
			projectIndex = i
			break
		}
	}

	if projectIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// check if new name conflicts with existing projects (except current one)
	if req.Name != projectName {
		for _, proj := range types.GoHookVersionData.Projects {
			if proj.Name == req.Name {
				c.JSON(http.StatusConflict, gin.H{"error": "Project name already exists"})
				return
			}
		}
	}

	// check if path exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Specified path does not exist"})
		return
	}

	// preserve existing configuration that is not being updated
	currentProject := types.GoHookVersionData.Projects[projectIndex]

	// update project while preserving existing fields
	types.GoHookVersionData.Projects[projectIndex] = types.ProjectConfig{
		Name:        req.Name,
		Path:        req.Path,
		Description: req.Description,
		Enabled:     currentProject.Enabled,    // preserve enabled status
		Enhook:      currentProject.Enhook,     // preserve hook configuration
		Hookmode:    currentProject.Hookmode,   // preserve hook mode
		Hookbranch:  currentProject.Hookbranch, // preserve hook branch
		Hooksecret:  currentProject.Hooksecret, // preserve hook secret
	}

	// save config file
	if err := config.SaveVersionConfig(); err != nil {
		// push failed message
		wsMessage := stream.WsMessage{
			Type:      "project_managed",
			Timestamp: time.Now(),
			Data: stream.ProjectManageMessage{
				Action:      "edit",
				ProjectName: req.Name,
				ProjectPath: req.Path,
				Success:     false,
				Error:       "Save config failed: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Save config failed: " + err.Error()})
		return
	}

	// push success message
	wsMessage := stream.WsMessage{
		Type:      "project_managed",
		Timestamp: time.Now(),
		Data: stream.ProjectManageMessage{
			Action:      "edit",
			ProjectName: req.Name,
			ProjectPath: req.Path,
			Success:     true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{"message": "Project updated successfully"})
}

// AddProject add project
func HandleAddProject(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Path        string `json:"path" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// check if project name already exists
	for _, proj := range types.GoHookVersionData.Projects {
		if proj.Name == req.Name {
			c.JSON(http.StatusConflict, gin.H{"error": "Project name already exists"})
			return
		}
	}

	// check if path exists
	if _, err := os.Stat(req.Path); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Specified path does not exist"})
		return
	}

	// add new project
	newProject := types.ProjectConfig{
		Name:        req.Name,
		Path:        req.Path,
		Description: req.Description,
		Enabled:     true,
	}

	types.GoHookVersionData.Projects = append(types.GoHookVersionData.Projects, newProject)

	// save config file
	if err := config.SaveVersionConfig(); err != nil {
		// push failed message
		wsMessage := stream.WsMessage{
			Type:      "project_managed",
			Timestamp: time.Now(),
			Data: stream.ProjectManageMessage{
				Action:      "add",
				ProjectName: req.Name,
				ProjectPath: req.Path,
				Success:     false,
				Error:       "Save config failed: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Save config failed: " + err.Error()})
		return
	}

	// push success message
	wsMessage := stream.WsMessage{
		Type:      "project_managed",
		Timestamp: time.Now(),
		Data: stream.ProjectManageMessage{
			Action:      "add",
			ProjectName: req.Name,
			ProjectPath: req.Path,
			Success:     true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Project added successfully",
		"project": newProject,
	})
}

// DeleteProject delete project
func HandleDeleteProject(c *gin.Context) {
	projectName := c.Param("name")

	// find project index
	projectIndex := -1
	for i, proj := range types.GoHookVersionData.Projects {
		if proj.Name == projectName {
			projectIndex = i
			break
		}
	}

	if projectIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// delete project
	types.GoHookVersionData.Projects = append(types.GoHookVersionData.Projects[:projectIndex], types.GoHookVersionData.Projects[projectIndex+1:]...)

	// save config file
	if err := config.SaveVersionConfig(); err != nil {
		// push failed message
		wsMessage := stream.WsMessage{
			Type:      "project_managed",
			Timestamp: time.Now(),
			Data: stream.ProjectManageMessage{
				Action:      "delete",
				ProjectName: projectName,
				Success:     false,
				Error:       "Save config failed: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Save config failed: " + err.Error()})
		return
	}

	// push success message
	wsMessage := stream.WsMessage{
		Type:      "project_managed",
		Timestamp: time.Now(),
		Data: stream.ProjectManageMessage{
			Action:      "delete",
			ProjectName: projectName,
			Success:     true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Project deleted successfully",
		"name":    projectName,
	})
}

// GetBranches get branch list
func HandleGetBranches(c *gin.Context) {
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

	branches, err := getBranches(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, branches)
}

// GetTags get tag list
func HandleGetTags(c *gin.Context) {
	projectName := c.Param("name")

	// get filter parameters
	filter := c.Query("filter")
	messageFilter := c.Query("messageFilter")

	// get pagination parameter
	page := 1
	limit := 20
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
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

	allTags, err := getTags(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// if there is filter condition, filter tags
	var filteredTags []types.TagResponse
	if filter != "" || messageFilter != "" {
		for _, tag := range allTags {
			// 检查标签名称筛选
			nameMatch := filter == "" || strings.HasPrefix(tag.Name, filter)
			// 检查说明内容筛选（不区分大小写的包含匹配）
			messageMatch := messageFilter == "" || strings.Contains(strings.ToLower(tag.Message), strings.ToLower(messageFilter))

			// 只有当两个条件都满足时才添加到结果中
			if nameMatch && messageMatch {
				filteredTags = append(filteredTags, tag)
			}
		}
	} else {
		filteredTags = allTags
	}

	// calculate pagination
	total := len(filteredTags)
	totalPages := (total + limit - 1) / limit
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		// out of range, return empty array
		c.JSON(http.StatusOK, gin.H{
			"tags":       []types.TagResponse{},
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
			"hasMore":    false,
		})
		return
	}

	if end > total {
		end = total
	}

	paginatedTags := filteredTags[start:end]
	hasMore := page < totalPages

	c.JSON(http.StatusOK, gin.H{
		"tags":       paginatedTags,
		"total":      total,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
		"hasMore":    hasMore,
	})
}

// SyncBranches sync remote branches, clean up deleted remote branch references
func HandleSyncBranches(c *gin.Context) {
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

	if err := syncBranches(projectPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Branches synced successfully"})
}

// DeleteBranch delete local branch
func HandleDeleteBranch(c *gin.Context) {
	projectName := c.Param("name")
	branchName := c.Param("branchName")

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

	if err := deleteBranch(projectPath, branchName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Branch deleted successfully"})
}

// SwitchBranch switch branch
func HandleSwitchBranch(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Branch string `json:"branch"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	currentUser, _ := c.Get("username")
	currentUserStr := "unknown"
	if currentUser != nil {
		currentUserStr = currentUser.(string)
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
		// 记录失败的分支切换尝试
		database.LogProjectAction(
			projectName,                        // projectName
			database.ProjectActionBranchSwitch, // action
			"",                                 // oldValue
			req.Branch,                         // newValue
			currentUserStr,                     // username
			false,                              // success
			"Project not found",                // error
			"",                                 // commitHash
			fmt.Sprintf("Switch branch failed: Project %s not found", projectName), // description
			middleware.GetClientIP(c), // ipAddress
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// 获取当前分支用于记录
	currentBranch := ""
	if gitStatus, err := getGitStatus(projectPath); err == nil {
		currentBranch = gitStatus.CurrentBranch
	}

	if err := switchBranch(projectPath, req.Branch); err != nil {
		// 记录失败的分支切换
		database.LogProjectAction(
			projectName,                        // projectName
			database.ProjectActionBranchSwitch, // action
			currentBranch,                      // oldValue
			req.Branch,                         // newValue
			currentUserStr,                     // username
			false,                              // success
			err.Error(),                        // error
			"",                                 // commitHash
			fmt.Sprintf("Switch branch failed: %s", err.Error()), // description
			middleware.GetClientIP(c),                            // ipAddress
		)

		// push failed message
		wsMessage := stream.WsMessage{
			Type:      "version_switched",
			Timestamp: time.Now(),
			Data: stream.VersionSwitchMessage{
				ProjectName: projectName,
				Action:      "switch-branch",
				Target:      req.Branch,
				Success:     false,
				Error:       err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 记录成功的分支切换
	database.LogProjectAction(
		projectName,                        // projectName
		database.ProjectActionBranchSwitch, // action
		currentBranch,                      // oldValue
		req.Branch,                         // newValue
		currentUserStr,                     // username
		true,                               // success
		"",                                 // error
		"",                                 // commitHash
		fmt.Sprintf("Branch switched from %s to %s successfully", currentBranch, req.Branch), // description
		middleware.GetClientIP(c), // ipAddress
	)

	// push success message
	wsMessage := stream.WsMessage{
		Type:      "version_switched",
		Timestamp: time.Now(),
		Data: stream.VersionSwitchMessage{
			ProjectName: projectName,
			Action:      "switch-branch",
			Target:      req.Branch,
			Success:     true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{"message": "Branch switched successfully", "branch": req.Branch})
}

// SyncTags sync remote tags
func HandleSyncTags(c *gin.Context) {
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

	if err := syncTags(projectPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tags synced successfully"})
}

// SwitchTag switch tag
func HandleSwitchTag(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Tag string `json:"tag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// 获取当前用户信息
	currentUser, _ := c.Get("username")
	currentUserStr := "unknown"
	if currentUser != nil {
		currentUserStr = currentUser.(string)
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
		// 记录失败的标签切换尝试
		database.LogProjectAction(
			projectName,                   // projectName
			"switch-tag",                  // action
			"",                            // oldValue
			fmt.Sprintf("标签:%s", req.Tag), // newValue
			currentUserStr,                // username
			false,                         // success
			"Project not found",           // error
			"",                            // commitHash
			fmt.Sprintf("标签切换失败：项目 %s 未找到", projectName), // description
			middleware.GetClientIP(c), // ipAddress
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// 获取当前标签/分支信息用于记录
	currentTag := ""
	currentBranch := ""
	currentCommit := ""

	// 尝试获取当前标签
	if output, err := execGitCommandOutput(projectPath, "describe", "--tags", "--exact-match", "HEAD"); err == nil {
		currentTag = strings.TrimSpace(string(output))
	}

	// 如果不在标签上，获取当前分支
	if currentTag == "" {
		if gitStatus, err := getGitStatus(projectPath); err == nil {
			currentBranch = gitStatus.CurrentBranch
			currentCommit = gitStatus.LastCommit
		}
	}

	// 构建当前位置描述
	currentPosition := ""
	if currentTag != "" {
		currentPosition = fmt.Sprintf("标签:%s", currentTag)
	} else if currentBranch != "" {
		currentPosition = fmt.Sprintf("分支:%s", currentBranch)
		if currentCommit != "" && len(currentCommit) > 7 {
			currentPosition += fmt.Sprintf("@%s", currentCommit[:7])
		}
	} else {
		currentPosition = "未知位置"
	}

	if err := switchTag(projectPath, req.Tag); err != nil {
		// 记录失败的项目活动日志
		database.LogProjectAction(
			projectName,
			"switch-tag",
			currentPosition,               // oldValue - 当前位置
			fmt.Sprintf("标签:%s", req.Tag), // newValue - 目标标签
			currentUserStr,                // username - 使用获取到的用户信息
			false,                         // success
			err.Error(),                   // error
			"",                            // commitHash
			fmt.Sprintf("标签切换失败：从 %s 切换到标签 %s 时出错: %s", currentPosition, req.Tag, err.Error()), // description
			middleware.GetClientIP(c), // ipAddress
		)

		// push failed message
		wsMessage := stream.WsMessage{
			Type:      "version_switched",
			Timestamp: time.Now(),
			Data: stream.VersionSwitchMessage{
				ProjectName: projectName,
				Action:      "switch-tag",
				Target:      req.Tag,
				Success:     false,
				Error:       err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取切换后的提交哈希
	newCommit := ""
	if output, err := execGitCommandOutput(projectPath, "rev-parse", "HEAD"); err == nil {
		newCommit = strings.TrimSpace(string(output))
		if len(newCommit) > 7 {
			newCommit = newCommit[:7]
		}
	}

	// 记录成功的项目活动日志
	database.LogProjectAction(
		projectName,
		"switch-tag",
		currentPosition,               // oldValue - 之前的位置
		fmt.Sprintf("标签:%s", req.Tag), // newValue - 目标标签
		currentUserStr,                // username - 使用获取到的用户信息
		true,                          // success
		"",                            // error
		newCommit,                     // commitHash - 切换后的提交哈希
		fmt.Sprintf("标签切换成功：从 %s 切换到标签 %s (提交: %s)", currentPosition, req.Tag, newCommit), // description
		middleware.GetClientIP(c), // ipAddress
	)

	// push success message
	wsMessage := stream.WsMessage{
		Type:      "version_switched",
		Timestamp: time.Now(),
		Data: stream.VersionSwitchMessage{
			ProjectName: projectName,
			Action:      "switch-tag",
			Target:      req.Tag,
			Success:     true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{"message": "Tag switched successfully", "tag": req.Tag})
}

// DeleteTag delete local and remote tag
func HandleDeleteTag(c *gin.Context) {
	projectName := c.Param("name")
	tagName := c.Param("tagName")

	// 获取当前用户信息
	currentUser, _ := c.Get("username")
	currentUserStr := "unknown"
	if currentUser != nil {
		currentUserStr = currentUser.(string)
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
		// 记录失败的标签删除尝试
		database.LogProjectAction(
			projectName,                   // projectName
			"delete-tag",                  // action
			fmt.Sprintf("标签:%s", tagName), // oldValue
			"",                            // newValue
			currentUserStr,                // username
			false,                         // success
			"Project not found",           // error
			"",                            // commitHash
			fmt.Sprintf("标签删除失败：项目 %s 未找到", projectName), // description
			middleware.GetClientIP(c), // ipAddress
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// 获取标签信息用于详细记录
	tagCommit := ""
	tagDate := ""

	// 获取标签对应的提交哈希
	if output, err := execGitCommandOutput(projectPath, "rev-list", "-n", "1", tagName); err == nil {
		tagCommit = strings.TrimSpace(string(output))
		if len(tagCommit) > 7 {
			tagCommit = tagCommit[:7]
		}
	}

	// 获取标签创建日期
	if output, err := execGitCommandOutput(projectPath, "log", "-1", "--format=%ci", tagName); err == nil {
		if t, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(string(output))); err == nil {
			tagDate = t.Format("2006-01-02 15:04")
		}
	}

	if err := deleteTag(projectPath, tagName); err != nil {
		// 记录失败的项目活动日志
		database.LogProjectAction(
			projectName,
			"delete-tag",
			fmt.Sprintf("标签:%s", tagName), // oldValue
			"",                            // newValue
			currentUserStr,                // username - 使用获取到的用户信息
			false,                         // success
			err.Error(),                   // error
			tagCommit,                     // commitHash
			fmt.Sprintf("标签删除失败：删除标签 %s 时出错 (提交: %s, 创建时间: %s): %s", tagName, tagCommit, tagDate, err.Error()), // description
			middleware.GetClientIP(c), // ipAddress
		)

		// push failed message
		wsMessage := stream.WsMessage{
			Type:      "version_switched",
			Timestamp: time.Now(),
			Data: stream.VersionSwitchMessage{
				ProjectName: projectName,
				Action:      "delete-tag",
				Target:      tagName,
				Success:     false,
				Error:       err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 记录成功的项目活动日志
	database.LogProjectAction(
		projectName,
		"delete-tag",
		fmt.Sprintf("标签:%s", tagName), // oldValue
		"",                            // newValue
		currentUserStr,                // username - 使用获取到的用户信息
		true,                          // success
		"",                            // error
		tagCommit,                     // commitHash
		fmt.Sprintf("标签删除成功：已删除标签 %s (提交: %s, 创建时间: %s)", tagName, tagCommit, tagDate), // description
		middleware.GetClientIP(c), // ipAddress
	)

	// push success message
	wsMessage := stream.WsMessage{
		Type:      "version_switched",
		Timestamp: time.Now(),
		Data: stream.VersionSwitchMessage{
			ProjectName: projectName,
			Action:      "delete-tag",
			Target:      tagName,
			Success:     true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted successfully"})
}

// DeleteLocalTag delete local tag
func HandleDeleteLocalTag(c *gin.Context) {
	projectName := c.Param("name")
	tagName := c.Param("tagName")

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

	if err := deleteLocalTag(projectPath, tagName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted successfully"})
}

// InitGitRepository initialize git repository
func HandleInitGitRepository(c *gin.Context) {
	projectName := c.Param("name")
	fmt.Printf("Received Git initialization request: project name=%s\n", projectName)

	// find project path
	var projectPath string
	for _, proj := range types.GoHookVersionData.Projects {
		if proj.Name == projectName && proj.Enabled {
			projectPath = proj.Path
			break
		}
	}

	if projectPath == "" {
		fmt.Printf("Git initialization failed: project not found, project name=%s\n", projectName)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	fmt.Printf("Git initialization: project name=%s, path=%s\n", projectName, projectPath)

	if err := initGit(projectPath); err != nil {
		fmt.Printf("Git initialization failed: project name=%s, path=%s, error=%v\n", projectName, projectPath, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fmt.Printf("Git initialization successful: project name=%s, path=%s\n", projectName, projectPath)
	c.JSON(http.StatusOK, gin.H{"message": "Git repository initialized successfully"})
}

func HandleSetRemote(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		RemoteUrl string `json:"remoteUrl"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	if req.RemoteUrl == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Remote repository URL cannot be empty"})
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

	if err := setRemote(projectPath, req.RemoteUrl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Remote repository set successfully"})
}

func HandleGetRemote(c *gin.Context) {
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

	remoteURL, err := getRemote(projectPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": remoteURL})
}

func HandleGetProjects(c *gin.Context) {
	// load config file every time get projects list
	if err := config.LoadVersionConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Load version config failed: " + err.Error()})
		return
	}

	if types.GoHookVersionData == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Version config not loaded"})
		return
	}

	var projects []types.VersionResponse
	for _, proj := range types.GoHookVersionData.Projects {
		if !proj.Enabled {
			continue
		}

		gitStatus, err := getGitStatus(proj.Path)
		if err != nil {
			// if not Git repository, still display but mark as non-Git project
			projects = append(projects, types.VersionResponse{
				Name:        proj.Name,
				Path:        proj.Path,
				Description: proj.Description,
				Mode:        "none",
				Status:      "not-git",
			})
			continue
		}

		gitStatus.Name = proj.Name
		gitStatus.Path = proj.Path
		gitStatus.Description = proj.Description
		gitStatus.Enhook = proj.Enhook
		gitStatus.Hookmode = proj.Hookmode
		gitStatus.Hookbranch = proj.Hookbranch
		gitStatus.Hooksecret = proj.Hooksecret
		projects = append(projects, *gitStatus)
	}

	c.JSON(http.StatusOK, projects)
}

func HandleReloadConfig(c *gin.Context) {
	if err := config.LoadVersionConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Load version config failed: " + err.Error(),
		})
		return
	}

	projectCount := 0
	if types.GoHookVersionData != nil {
		for _, proj := range types.GoHookVersionData.Projects {
			if proj.Enabled {
				projectCount++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Version config loaded successfully",
		"projectCount": projectCount,
	})
}
