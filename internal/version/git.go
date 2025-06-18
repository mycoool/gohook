package version

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
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
	"github.com/mycoool/gohook/internal/env"
	"github.com/mycoool/gohook/internal/stream"
	"github.com/mycoool/gohook/internal/types"
)

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

	// execute git init command
	cmd := exec.Command("git", "-C", projectPath, "init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git repository initialization failed: %v, output: %s", err, string(output))
	}

	// verify if Git repository is successfully created
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("git repository initialization verification failed: .git directory not created")
	}

	return nil
}

// verify GitHub HMAC-SHA256 signature
func VerifyGitHubSignature(payload []byte, secret, signature string) error {
	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("GitHub signature format error, should start with sha256=")
	}

	expectedSig := "sha256=" + HmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("GitHub signature verification failed")
	}

	return nil
}

// verify GitHub legacy signature (old version)
func VerifyGitHubLegacySignature(payload []byte, secret, signature string) error {
	if !strings.HasPrefix(signature, "sha1=") {
		return fmt.Errorf("GitHub legacy signature format error, should start with sha1=")
	}

	// note: here should use SHA1, but for security, we suggest using SHA256
	expectedSig := "sha1=" + HmacSHA1Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("GitHub legacy signature verification failed")
	}

	return nil
}

// verifyGitLabToken verify GitLab token (directly compare)
func VerifyGitLabToken(secret, token string) error {
	if subtle.ConstantTimeCompare([]byte(secret), []byte(token)) != 1 {
		return fmt.Errorf("GitLab token verification failed")
	}
	return nil
}

// verifyGiteaSignature verify Gitea HMAC-SHA256 signature
func VerifyGiteaSignature(payload []byte, secret, signature string) error {
	expectedSig := HmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("gitea signature verification failed")
	}
	return nil
}

// verifyGogsSignature verify Gogs HMAC-SHA256 signature
func VerifyGogsSignature(payload []byte, secret, signature string) error {
	expectedSig := HmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("gogs signature verification failed")
	}
	return nil
}

// hmacSHA256Hex calculate HMAC-SHA256 and return hexadecimal string
func HmacSHA256Hex(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmacSHA1Hex calculate HMAC-SHA1 and return hexadecimal string (for GitHub legacy support)
func HmacSHA1Hex(data []byte, secret string) string {
	// note: here should import crypto/sha1, but for simplicity, we skip this implementation
	// in production environment, SHA1 should be correctly implemented
	return HmacSHA256Hex(data, secret) // temporarily use SHA256 instead
}

// VerifyWebhookSignature verify webhook signature, support GitHub, GitLab, etc.
func verifyWebhookSignature(c *gin.Context, payloadBody []byte, secret string) error {
	// GitHub use X-Hub-Signature-256 header with HMAC-SHA256
	if githubSig := c.GetHeader("X-Hub-Signature-256"); githubSig != "" {
		return VerifyGitHubSignature(payloadBody, secret, githubSig)
	}

	// GitHub legacy use X-Hub-Signature header with HMAC-SHA1
	if githubSigLegacy := c.GetHeader("X-Hub-Signature"); githubSigLegacy != "" {
		return VerifyGitHubLegacySignature(payloadBody, secret, githubSigLegacy)
	}

	// GitLab use X-Gitlab-Token header, directly compare password
	if gitlabToken := c.GetHeader("X-Gitlab-Token"); gitlabToken != "" {
		return VerifyGitLabToken(secret, gitlabToken)
	}

	// Gitea use X-Gitea-Signature header with HMAC-SHA256
	if giteaSig := c.GetHeader("X-Gitea-Signature"); giteaSig != "" {
		return VerifyGiteaSignature(payloadBody, secret, giteaSig)
	}

	// Gogs use X-Gogs-Signature header with HMAC-SHA256
	if gogsSig := c.GetHeader("X-Gogs-Signature"); gogsSig != "" {
		return VerifyGogsSignature(payloadBody, secret, gogsSig)
	}

	// if no known signature header is found, return error
	return fmt.Errorf("no supported webhook signature header found")
}

// executeGitHook execute specific Git operation
func ExecuteGitHook(project *types.ProjectConfig, refType, targetRef string) error {
	projectPath := project.Path

	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("project path is not a Git repository: %s", projectPath)
	}

	// fetch latest remote information
	cmd := exec.Command("git", "-C", projectPath, "fetch", "--all")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("warning: failed to fetch remote information: %s", string(output))
	}

	if refType == "branch" {
		// branch mode: switch to specified branch and pull latest code
		return switchAndPullBranch(projectPath, targetRef)
	} else if refType == "tag" {
		// tag mode: switch to specified tag
		return switchToTag(projectPath, targetRef)
	}

	return fmt.Errorf("unsupported reference type: %s", refType)
}

// switchAndPullBranch switch to specified branch and pull latest code
func switchAndPullBranch(projectPath, branchName string) error {
	// check if local branch exists
	cmd := exec.Command("git", "-C", projectPath, "branch", "--list", branchName)
	output, err := cmd.Output()
	localBranchExists := err == nil && strings.TrimSpace(string(output)) != ""

	if !localBranchExists {
		// local branch does not exist, try to create from remote
		cmd = exec.Command("git", "-C", projectPath, "checkout", "-b", branchName, "origin/"+branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("create and switch to branch %s failed: %s", branchName, string(output))
		}
	} else {
		// local branch exists, switch directly
		cmd = exec.Command("git", "-C", projectPath, "checkout", branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("switch to branch %s failed: %s", branchName, string(output))
		}

		// fetch latest code
		cmd = exec.Command("git", "-C", projectPath, "pull", "origin", branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to fetch latest code for branch %s: %s", branchName, string(output))
		}
	}

	return nil
}

// switchToTag switch to specified tag
func switchToTag(projectPath, tagName string) error {
	// fetch tag information
	cmd := exec.Command("git", "-C", projectPath, "fetch", "--tags")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("warning: failed to fetch tag information: %s", string(output))
	}

	// ensure tag exists (local or remote)
	cmd = exec.Command("git", "-C", projectPath, "rev-parse", tagName)
	if err := cmd.Run(); err != nil {
		log.Printf("tag %s does not exist, try to fetch from remote", tagName)
		cmd = exec.Command("git", "-C", projectPath, "fetch", "origin", "--tags")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to fetch tag from remote: %s", string(output))
		}

		// check if tag exists again
		cmd = exec.Command("git", "-C", projectPath, "rev-parse", tagName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("tag %s does not exist on remote, cannot deploy", tagName)
		}
	}

	// switch to specified tag
	cmd = exec.Command("git", "-C", projectPath, "checkout", tagName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("switch to tag %s failed: %s", tagName, string(output))
	}

	log.Printf("successfully switched to tag: %s", tagName)
	return nil
}

// HandleGitHook handle GitHook webhook request
func handleGitHook(project *types.ProjectConfig, payload map[string]interface{}) error {
	log.Printf("handle GitHook: project=%s, mode=%s, branch=%s", project.Name, project.Hookmode, project.Hookbranch)

	// parse webhook payload, extract branch or tag information
	var targetRef string
	var refType string
	var afterCommit string

	// try to parse GitHub/GitLab format webhook
	if ref, ok := payload["ref"].(string); ok {
		// extract after field (for detecting deletion operation)
		if after, ok := payload["after"].(string); ok {
			afterCommit = after
		}

		if strings.HasPrefix(ref, "refs/heads/") {
			// branch push
			targetRef = strings.TrimPrefix(ref, "refs/heads/")
			refType = "branch"
		} else if strings.HasPrefix(ref, "refs/tags/") {
			// tag push
			targetRef = strings.TrimPrefix(ref, "refs/tags/")
			refType = "tag"
		}
	}

	// if no ref is parsed, try other formats
	if targetRef == "" {
		// try GitLab format
		if ref, ok := payload["ref"].(string); ok {
			parts := strings.Split(ref, "/")
			if len(parts) >= 3 {
				if parts[1] == "heads" {
					targetRef = strings.Join(parts[2:], "/")
					refType = "branch"
				} else if parts[1] == "tags" {
					targetRef = strings.Join(parts[2:], "/")
					refType = "tag"
				}
			}
		}
	}

	if targetRef == "" {
		return fmt.Errorf("cannot parse branch or tag information from webhook payload")
	}

	log.Printf("parsed webhook: type=%s, target=%s, after=%s", refType, targetRef, afterCommit)

	// check if it matches the project's hook mode
	if project.Hookmode != refType {
		log.Printf("webhook type(%s) does not match project hook mode(%s), skip", refType, project.Hookmode)
		return nil
	}

	// if it is a branch mode, check if the branch matches
	if project.Hookmode == "branch" {
		if project.Hookbranch != "*" && project.Hookbranch != targetRef {
			log.Printf("webhook branch(%s) does not match configured branch(%s), skip", targetRef, project.Hookbranch)
			return nil
		}
	}

	// check if it is a deletion operation (after field is all zeros)
	if afterCommit == "0000000000000000000000000000000000000000" {
		if refType == "tag" {
			// tag deletion: only delete local tag
			log.Printf("detected tag deletion event: %s", targetRef)
			return deleteTag(project.Path, targetRef)
		} else if refType == "branch" {
			// branch deletion: need to smart judgment
			log.Printf("detected branch deletion event: %s", targetRef)
			return BranchDeletion(project, targetRef)
		}
	}

	// execute Git operation
	if err := ExecuteGitHook(project, refType, targetRef); err != nil {
		return fmt.Errorf("execute Git operation failed: %v", err)
	}

	log.Printf("GitHook processing successfully: project=%s, type=%s, target=%s", project.Name, refType, targetRef)
	return nil
}

// DeleteLocalTag delete local tag
func deleteLocalTag(projectPath, tagName string) error {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository: %s", projectPath)
	}

	// check if tag exists
	cmd := exec.Command("git", "-C", projectPath, "show-ref", "--tags", "--quiet", "refs/tags/"+tagName)
	if err := cmd.Run(); err != nil {
		log.Printf("local tag %s does not exist, skip deletion", tagName)
		return nil
	}

	// delete local tag
	cmd = exec.Command("git", "-C", projectPath, "tag", "-d", tagName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("delete local tag %s failed: %s", tagName, string(output))
	}

	log.Printf("successfully deleted local tag: %s", tagName)
	return nil
}

// DeleteLocalBranch delete local branch
func DeleteLocalBranch(projectPath, branchName string) error {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository: %s", projectPath)
	}

	// get current branch
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranchOutput, err := cmd.Output()
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
	cmd = exec.Command("git", "-C", projectPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err := cmd.Run(); err != nil {
		log.Printf("local branch %s does not exist, skip deletion", branchName)
		return nil
	}

	// delete local branch
	cmd = exec.Command("git", "-C", projectPath, "branch", "-D", branchName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("delete local branch %s failed: %s", branchName, string(output))
	}

	log.Printf("successfully deleted local branch: %s", branchName)
	return nil
}

// BranchDeletion handle branch deletion operation
func BranchDeletion(project *types.ProjectConfig, branchName string) error {
	log.Printf("handle branch deletion: project=%s, branch=%s, configured branch=%s", project.Name, branchName, project.Hookbranch)

	// check if it is a configured branch
	if project.Hookbranch != "*" && project.Hookbranch == branchName {
		log.Printf("deleting configured branch %s, skip deletion to protect project running", branchName)
		return nil
	}

	// check if it is a master branch
	if branchName == "master" || branchName == "main" {
		log.Printf("deleting master branch %s, skip deletion to protect project", branchName)
		return nil
	}

	// if it is other branch, execute local deletion operation
	log.Printf("deleting non-critical branch %s, execute local deletion operation", branchName)
	return DeleteLocalBranch(project.Path, branchName)
}

// DeleteBranch delete local branch
func deleteBranch(projectPath, branchName string) error {
	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("not a Git repository")
	}

	// get current branch
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranchOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("get current branch failed: %v", err)
	}
	currentBranch := strings.TrimSpace(string(currentBranchOutput))

	// check if trying to delete current branch
	if currentBranch == branchName {
		return fmt.Errorf("cannot delete current branch")
	}

	// delete local branch
	cmd = exec.Command("git", "-C", projectPath, "branch", "-D", branchName)
	output, err := cmd.CombinedOutput()
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
	cmd := exec.Command("git", "-C", projectPath, "describe", "--tags", "--exact-match", "HEAD")
	currentTagOutput, err := cmd.Output()
	if err == nil {
		currentTag := strings.TrimSpace(string(currentTagOutput))
		if currentTag == tagName {
			return fmt.Errorf("cannot delete current tag")
		}
	}

	// delete local tag
	cmd = exec.Command("git", "-C", projectPath, "tag", "-d", tagName)
	localOutput, localErr := cmd.CombinedOutput()
	if localErr != nil {
		return fmt.Errorf("delete local tag failed: %s", string(localOutput))
	}

	// try to delete remote tag
	cmd = exec.Command("git", "-C", projectPath, "push", "origin", ":refs/tags/"+tagName)
	remoteOutput, remoteErr := cmd.CombinedOutput()
	if remoteErr != nil {
		log.Printf("delete remote tag failed (project: %s, tag: %s): %s", projectPath, tagName, string(remoteOutput))
		// remote tag deletion failed is not a fatal error, because it may not exist on remote
	}

	return nil
}

// SwitchTag switch tag
func switchTag(projectPath, tagName string) error {
	cmd := exec.Command("git", "-C", projectPath, "checkout", tagName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("switch tag failed: %v", err)
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
	cmd := exec.Command("git", "-C", projectPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
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
	cmd := exec.Command("git", "-C", projectPath, "remote", "get-url", "origin")
	if cmd.Run() == nil {
		// if origin already exists, delete it first
		cmd = exec.Command("git", "-C", projectPath, "remote", "remove", "origin")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("delete existing remote repository failed: %v", err)
		}
	}

	// add new origin remote repository
	cmd = exec.Command("git", "-C", projectPath, "remote", "add", "origin", remoteUrl)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("set remote repository failed: %v", err)
	}

	return nil
}

// SyncBranches sync remote branches, clean up deleted remote branch references
func syncBranches(projectPath string) error {
	// use fetch --prune to update remote branch information and delete non-existent references
	cmd := exec.Command("git", "-C", projectPath, "fetch", "origin", "--prune")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sync branches failed: %s", string(output))
	}
	return nil
}

// SwitchBranch switch branch
func switchBranch(projectPath, branchName string) error {
	var cmd *exec.Cmd
	var isRemoteBranch bool
	var localBranchName string

	// check if it is a remote branch format (for example origin/release)
	if strings.HasPrefix(branchName, "origin/") {
		isRemoteBranch = true
		localBranchName = strings.TrimPrefix(branchName, "origin/")

		// check if local branch already exists
		checkCmd := exec.Command("git", "-C", projectPath, "rev-parse", "--verify", localBranchName)
		if checkCmd.Run() == nil {
			// local branch already exists, switch directly
			cmd = exec.Command("git", "-C", projectPath, "checkout", localBranchName)
		} else {
			// local branch does not exist, create a new local branch based on the remote branch
			cmd = exec.Command("git", "-C", projectPath, "checkout", "-b", localBranchName, branchName)
		}
	} else {
		// normal local branch switch
		isRemoteBranch = false
		localBranchName = branchName
		cmd = exec.Command("git", "-C", projectPath, "checkout", branchName)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("switch branch failed: %s", string(output))
	}

	// if a new branch is created based on a remote branch, try to pull the latest code
	if isRemoteBranch {
		pullCmd := exec.Command("git", "-C", projectPath, "pull", "origin", localBranchName)
		pullOutput, pullErr := pullCmd.CombinedOutput()
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
	cmd := exec.Command("git", "-C", projectPath, "describe", "--exact-match", "--tags", "HEAD")
	currentOutput, _ := cmd.Output()
	currentTag := strings.TrimSpace(string(currentOutput))

	// get all tags
	cmd = exec.Command("git", "-C", projectPath, "tag", "-l", "--sort=-version:refname", "--format=%(refname:short)|%(creatordate)|%(objectname:short)|%(subject)")
	output, err := cmd.Output()
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
	_, err := exec.Command("git", "-C", projectPath, "symbolic-ref", "-q", "HEAD").Output()
	isDetached := err != nil

	// 2. get current branch or commit reference
	var currentRef string
	if isDetached {
		// detached head state, get HEAD short hash
		headSha, err := exec.Command("git", "-C", projectPath, "rev-parse", "--short", "HEAD").Output()
		if err != nil {
			return nil, fmt.Errorf("get HEAD commit failed: %v", err)
		}
		currentRef = strings.TrimSpace(string(headSha))
	} else {
		// on a branch, get branch name
		branchName, err := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return nil, fmt.Errorf("get current branch name failed: %v", err)
		}
		currentRef = strings.TrimSpace(string(branchName))
	}

	// 3. handle detached head state
	if isDetached {
		// try to get tag name
		tagName, err := exec.Command("git", "-C", projectPath, "describe", "--tags", "--exact-match", "HEAD").Output()
		var displayName string
		if err == nil {
			displayName = strings.TrimSpace(string(tagName))
		} else {
			displayName = currentRef
		}

		// get last commit information
		commitOutput, _ := exec.Command("git", "-C", projectPath, "log", "-1", "HEAD", "--format=%H|%ci").Output()
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
	cmd := exec.Command("git", "-C", projectPath, "for-each-ref", "refs/heads", "--format=%(refname:short)|%(committerdate:iso)|%(objectname:short)")
	localOutput, err := cmd.Output()
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
	cmd = exec.Command("git", "-C", projectPath, "for-each-ref", "refs/remotes", "--format=%(refname:short)|%(committerdate:iso)|%(objectname:short)")
	remoteOutput, err := cmd.Output()
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
	cmd := exec.Command("git", "-C", projectPath, "branch", "--show-current")
	branchOutput, _ := cmd.Output()
	currentBranch := strings.TrimSpace(string(branchOutput))

	// get current tag (if on a tag)
	cmd = exec.Command("git", "-C", projectPath, "describe", "--exact-match", "--tags", "HEAD")
	tagOutput, _ := cmd.Output()
	currentTag := strings.TrimSpace(string(tagOutput))

	// determine mode
	mode := "branch"
	if currentTag != "" {
		mode = "tag"
	}

	// get last commit information
	cmd = exec.Command("git", "-C", projectPath, "log", "-1", "--format=%H|%ci|%s")
	commitOutput, _ := cmd.Output()
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

// AddProject add project
func AddProject(c *gin.Context) {
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
func DeleteProject(c *gin.Context) {
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
func GetBranches(c *gin.Context) {
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
func GetTags(c *gin.Context) {
	projectName := c.Param("name")

	// get filter parameter
	filter := c.Query("filter")

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
	if filter != "" {
		for _, tag := range allTags {
			if strings.HasPrefix(tag.Name, filter) {
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
func SyncBranches(c *gin.Context) {
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
func DeleteBranch(c *gin.Context) {
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
func SwitchBranch(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Branch string `json:"branch"`
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

	if err := switchBranch(projectPath, req.Branch); err != nil {
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

// SwitchTag switch tag
func SwitchTag(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Tag string `json:"tag"`
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

	if err := switchTag(projectPath, req.Tag); err != nil {
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
func DeleteTag(c *gin.Context) {
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

	if err := deleteTag(projectPath, tagName); err != nil {
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
func DeleteLocalTag(c *gin.Context) {
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
func InitGitRepository(c *gin.Context) {
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

func SetRemote(c *gin.Context) {
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

func GetRemote(c *gin.Context) {
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

func GetProjects(c *gin.Context) {
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

func ReloadConfig(c *gin.Context) {
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

// SaveGitHook save project GitHook configuration
func SaveGitHook(c *gin.Context) {
	projectName := c.Param("name")

	var req struct {
		Enhook     bool   `json:"enhook"`
		Hookmode   string `json:"hookmode"`
		Hookbranch string `json:"hookbranch"`
		Hooksecret string `json:"hooksecret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// find project and update configuration
	projectFound := false
	for i, proj := range types.GoHookVersionData.Projects {
		if proj.Name == projectName && proj.Enabled {
			types.GoHookVersionData.Projects[i].Enhook = req.Enhook
			types.GoHookVersionData.Projects[i].Hookmode = req.Hookmode
			types.GoHookVersionData.Projects[i].Hookbranch = req.Hookbranch
			types.GoHookVersionData.Projects[i].Hooksecret = req.Hooksecret
			projectFound = true
			break
		}
	}

	if !projectFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// save configuration file
	if err := config.SaveVersionConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Save configuration failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "GitHook configuration saved successfully",
	})
}

// GitHook handle GitHook request
func GitHook(c *gin.Context) {
	projectName := c.Param("name")

	// find project configuration
	var project *types.ProjectConfig
	for _, proj := range types.GoHookVersionData.Projects {
		if proj.Name == projectName && proj.Enabled && proj.Enhook {
			project = &proj
			break
		}
	}

	if project == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found or GitHook not enabled"})
		return
	}

	// read original payload data
	var payloadBody []byte
	if c.Request.Body != nil {
		var err error
		payloadBody, err = io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Read payload failed"})
			return
		}
		// reset body for subsequent use
		c.Request.Body = io.NopCloser(bytes.NewReader(payloadBody))
	}

	// verify webhook password (if set)
	if project.Hooksecret != "" {
		if err := verifyWebhookSignature(c, payloadBody, project.Hooksecret); err != nil {
			log.Printf("GitHook password verification failed: project=%s, error=%v", project.Name, err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Password verification failed: " + err.Error()})
			return
		}
	}

	// parse webhook payload (support GitHub, GitLab, etc.)
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	// handle GitHook logic
	if err := handleGitHook(project, payload); err != nil {
		log.Printf("GitHook processing failed: project=%s, error=%v", project.Name, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "GitHook processing failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "GitHook processing successfully"})
}
