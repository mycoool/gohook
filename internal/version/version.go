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

// execGitCommand execute git command, automatically handle safe.directory permission issues
func execGitCommand(projectPath string, args ...string) ([]byte, error) {
	// first try to execute git command normally
	cmd := exec.Command("git", append([]string{"-C", projectPath}, args...)...)
	output, err := cmd.CombinedOutput()

	// if successful or not safe.directory related error, return directly
	if err == nil {
		return output, nil
	}

	outputStr := string(output)
	// check if it is safe.directory related error
	if !strings.Contains(outputStr, "safe.directory") && !strings.Contains(outputStr, "detected dubious ownership") {
		return output, err
	}

	log.Printf("detected Git safe.directory issue, trying to fix: %s", projectPath)

	// try to add to safe.directory (global system-level configuration)
	safeCmd := exec.Command("git", "config", "--system", "--add", "safe.directory", projectPath)
	if safeOutput, safeErr := safeCmd.CombinedOutput(); safeErr != nil {
		log.Printf("system-level safe.directory configuration failed: %s", string(safeOutput))

		// if system-level configuration failed, try global user-level configuration
		safeCmd = exec.Command("git", "config", "--global", "--add", "safe.directory", projectPath)
		if safeOutput, safeErr := safeCmd.CombinedOutput(); safeErr != nil {
			log.Printf("global safe.directory configuration also failed: %s", string(safeOutput))
			return output, fmt.Errorf("git safe.directory configuration failed: %v. Original error: %v", safeErr, err)
		} else {
			log.Printf("successfully configured global safe.directory: %s", projectPath)
		}
	} else {
		log.Printf("successfully configured system-level safe.directory: %s", projectPath)
	}

	// retry to execute original git command
	cmd = exec.Command("git", append([]string{"-C", projectPath}, args...)...)
	retryOutput, retryErr := cmd.CombinedOutput()
	if retryErr != nil {
		log.Printf("retry after safe.directory configuration failed: %s", string(retryOutput))
		return retryOutput, fmt.Errorf("git command failed even after safe.directory configuration: %v", retryErr)
	}

	log.Printf("successfully executed git command after safe.directory configuration: %s", projectPath)
	return retryOutput, nil
}

// execGitCommandOutput execute git command and return output, using safe.directory to automatically fix
func execGitCommandOutput(projectPath string, args ...string) ([]byte, error) {
	return execGitCommand(projectPath, args...)
}

// execGitCommandRun execute
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

// forceCleanWorkingDirectory force clean working directory, discard all local changes
// Note: Only resets tracked files, does NOT clean untracked files (to preserve .env, runtime/, etc.)
func forceCleanWorkingDirectory(projectPath string) error {
	log.Printf("Force cleaning working directory: %s", projectPath)

	// Reset all changes to tracked files (staged and unstaged)
	// This will discard all local modifications but preserve untracked files like .env, runtime/, etc.
	if output, err := execGitCommand(projectPath, "reset", "--hard", "HEAD"); err != nil {
		return fmt.Errorf("git reset --hard failed: %s", string(output))
	}

	log.Printf("Working directory cleaned successfully (tracked files only): %s", projectPath)
	return nil
}

// switchAndPullBranch switch to specified branch and pull latest code
// force: if true, will discard all local changes before switching
func switchAndPullBranch(projectPath, branchName string, force bool) error {
	// if force mode, clean working directory first
	if force {
		if err := forceCleanWorkingDirectory(projectPath); err != nil {
			return fmt.Errorf("force clean failed: %v", err)
		}
	}

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

		// if force mode, use reset to sync with remote instead of pull
		if force {
			if output, err := execGitCommand(projectPath, "reset", "--hard", "origin/"+branchName); err != nil {
				return fmt.Errorf("failed to force sync with remote branch %s: %s", branchName, string(output))
			}
		} else {
			// normal mode: pull latest code
			if output, err := execGitCommand(projectPath, "pull", "origin", branchName); err != nil {
				return fmt.Errorf("failed to fetch latest code for branch %s: %s", branchName, string(output))
			}
		}
	}

	return nil
}

// switchToTag switch to specified tag
// force: if true, will discard all local changes before switching
func switchToTag(projectPath, tagName string, force bool) error {
	// if force mode, clean working directory first
	if force {
		if err := forceCleanWorkingDirectory(projectPath); err != nil {
			return fmt.Errorf("force clean failed: %v", err)
		}
	}

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

// SwitchTag switch tag (wrapper for backward compatibility)
func switchTag(projectPath, tagName string, force bool) error {
	return switchToTag(projectPath, tagName, force)
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
// force: if true, will discard all local changes before switching
func switchBranch(projectPath, branchName string, force bool) error {
	// if force mode, clean working directory first
	if force {
		if err := forceCleanWorkingDirectory(projectPath); err != nil {
			return fmt.Errorf("force clean failed: %v", err)
		}
	}

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
		if force {
			// force mode: use reset instead of pull
			resetOutput, resetErr := execGitCommand(projectPath, "reset", "--hard", "origin/"+localBranchName)
			if resetErr != nil {
				log.Printf("force reset after switching branch failed (project: %s, branch: %s): %s", projectPath, localBranchName, string(resetOutput))
			}
		} else {
			// normal mode: try to pull
			pullOutput, pullErr := execGitCommand(projectPath, "pull", "origin", localBranchName)
			if pullErr != nil {
				// pull failed is not a fatal error, but log it
				log.Printf("pull latest code after switching branch failed (project: %s, branch: %s): %s", projectPath, localBranchName, string(pullOutput))
			}
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
		Name        string                 `json:"name" binding:"required"`
		Path        string                 `json:"path" binding:"required"`
		Description string                 `json:"description"`
		Sync        *types.ProjectSyncConfig `json:"sync,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// 智能清理路径末尾的斜杠
	if len(req.Path) > 1 {
		req.Path = strings.TrimRight(req.Path, string(os.PathSeparator))
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
		Enabled:     currentProject.Enabled,
		Enhook:      currentProject.Enhook,
		Hookmode:    currentProject.Hookmode,
		Hookbranch:  currentProject.Hookbranch,
		Hooksecret:  currentProject.Hooksecret,
		ForceSync:   currentProject.ForceSync,
		Sync:        currentProject.Sync,
	}
	if req.Sync != nil {
		types.GoHookVersionData.Projects[projectIndex].Sync = req.Sync
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
		Name        string                  `json:"name" binding:"required"`
		Path        string                  `json:"path" binding:"required"`
		Description string                  `json:"description"`
		Sync        *types.ProjectSyncConfig `json:"sync,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// 智能清理路径末尾的斜杠
	if len(req.Path) > 1 {
		req.Path = strings.TrimRight(req.Path, string(os.PathSeparator))
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
		Sync:        req.Sync,
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

	// execute git status check in background to trigger safe.directory etc.
	go func(p types.ProjectConfig) {
		log.Printf("project '%s' added successfully, starting background git status check...", p.Name)
		_, err := getGitStatus(p.Path)
		if err != nil {
			log.Printf("background git status check failed for project '%s': %v", p.Name, err)
		} else {
			log.Printf("background git status check completed for project '%s'", p.Name)
		}
	}(newProject)

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
			// check tag name filter
			nameMatch := filter == "" || strings.HasPrefix(tag.Name, filter)
			// check message filter (case-insensitive contains match)
			messageMatch := messageFilter == "" || strings.Contains(strings.ToLower(tag.Message), strings.ToLower(messageFilter))

			// only add to results if both conditions are met
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
		Force  bool   `json:"force"` // force switch, discard local changes
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
		// log failed branch switch attempt
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

	// get current branch for logging
	currentBranch := ""
	if gitStatus, err := getGitStatus(projectPath); err == nil {
		currentBranch = gitStatus.CurrentBranch
	}

	if err := switchBranch(projectPath, req.Branch, req.Force); err != nil {
		// log failed branch switch attempt
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

	// log successful branch switch
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
		Tag   string `json:"tag"`
		Force bool   `json:"force"` // force switch, discard local changes
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// get current user information
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
		// log failed tag switch attempt
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

	// get current tag/branch information for logging
	currentTag := ""
	currentBranch := ""
	currentCommit := ""

	// try to get current tag
	if output, err := execGitCommandOutput(projectPath, "describe", "--tags", "--exact-match", "HEAD"); err == nil {
		currentTag = strings.TrimSpace(string(output))
	}

	// if not on a tag, get current branch
	if currentTag == "" {
		if gitStatus, err := getGitStatus(projectPath); err == nil {
			currentBranch = gitStatus.CurrentBranch
			currentCommit = gitStatus.LastCommit
		}
	}

	// build current position description
	currentPosition := ""
	if currentTag != "" {
		currentPosition = fmt.Sprintf("Tag:%s", currentTag)
	} else if currentBranch != "" {
		currentPosition = fmt.Sprintf("Branch:%s", currentBranch)
		if currentCommit != "" && len(currentCommit) > 7 {
			currentPosition += fmt.Sprintf("@%s", currentCommit[:7])
		}
	} else {
		currentPosition = "Unknown position"
	}

	if err := switchTag(projectPath, req.Tag, req.Force); err != nil {
		// log failed project action
		database.LogProjectAction(
			projectName,
			"switch-tag",
			currentPosition,                // oldValue - current position
			fmt.Sprintf("Tag:%s", req.Tag), // newValue - target tag
			currentUserStr,                 // username - use the user information we got
			false,                          // success
			err.Error(),                    // error
			"",                             // commitHash
			fmt.Sprintf("Switch tag failed: from %s to tag %s: %s", currentPosition, req.Tag, err.Error()), // description
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

	// get new commit hash after switch
	newCommit := ""
	if output, err := execGitCommandOutput(projectPath, "rev-parse", "HEAD"); err == nil {
		newCommit = strings.TrimSpace(string(output))
		if len(newCommit) > 7 {
			newCommit = newCommit[:7]
		}
	}

	// log successful project action
	database.LogProjectAction(
		projectName,
		"switch-tag",
		currentPosition,                // oldValue - previous position
		fmt.Sprintf("Tag:%s", req.Tag), // newValue - target tag
		currentUserStr,                 // username - use the user information we got
		true,                           // success
		"",                             // error
		newCommit,                      // commitHash - new commit hash after switch
		fmt.Sprintf("Switch tag successfully: from %s to tag %s (commit: %s)", currentPosition, req.Tag, newCommit), // description
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

	// get current user information
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
		// log failed tag delete attempt
		database.LogProjectAction(
			projectName,                    // projectName
			"delete-tag",                   // action
			fmt.Sprintf("Tag:%s", tagName), // oldValue
			"",                             // newValue
			currentUserStr,                 // username
			false,                          // success
			"Project not found",            // error
			"",                             // commitHash
			fmt.Sprintf("Delete tag failed: project %s not found", projectName), // description
			middleware.GetClientIP(c), // ipAddress
		)
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// get tag information for detailed logging
	tagCommit := ""
	tagDate := ""

	// get tag commit hash
	if output, err := execGitCommandOutput(projectPath, "rev-list", "-n", "1", tagName); err == nil {
		tagCommit = strings.TrimSpace(string(output))
		if len(tagCommit) > 7 {
			tagCommit = tagCommit[:7]
		}
	}

	// get tag creation date
	if output, err := execGitCommandOutput(projectPath, "log", "-1", "--format=%ci", tagName); err == nil {
		if t, err := time.Parse("2006-01-02 15:04:05 -0700", strings.TrimSpace(string(output))); err == nil {
			tagDate = t.Format("2006-01-02 15:04")
		}
	}

	if err := deleteTag(projectPath, tagName); err != nil {
		// log failed project action
		database.LogProjectAction(
			projectName,
			"delete-tag",
			fmt.Sprintf("Tag:%s", tagName), // oldValue
			"",                             // newValue
			currentUserStr,                 // username - use the user information we got
			false,                          // success
			err.Error(),                    // error
			tagCommit,                      // commitHash
			fmt.Sprintf("Delete tag failed: delete tag %s: %s (commit: %s, created at: %s)", tagName, tagCommit, tagDate, err.Error()), // description
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

	// log successful project action
	database.LogProjectAction(
		projectName,
		"delete-tag",
		fmt.Sprintf("Tag:%s", tagName), // oldValue
		"",                             // newValue
		currentUserStr,                 // username - use the user information we got
		true,                           // success
		"",                             // error
		tagCommit,                      // commitHash
		fmt.Sprintf("Delete tag successfully: deleted tag %s (commit: %s, created at: %s)", tagName, tagCommit, tagDate), // description
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
				Sync:        proj.Sync,
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
		gitStatus.ForceSync = proj.ForceSync
		gitStatus.Sync = proj.Sync
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
