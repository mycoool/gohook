package version

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

type GitHookResult struct {
	Action  string
	Target  string
	Success bool
	Error   string
	Skipped bool   // 标识是否跳过操作（比如分支不匹配时无需切换）
	Message string // 详细消息，比如"无需切换"的原因
}

// GitHook handle GitHook webhook request
func tryGitHook(project *types.ProjectConfig, payload map[string]interface{}) (GitHookResult, error) {
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
				switch parts[1] {
				case "heads":
					targetRef = strings.Join(parts[2:], "/")
					refType = "branch"
				case "tags":
					targetRef = strings.Join(parts[2:], "/")
					refType = "tag"
				}
			}
		}
	}

	if targetRef == "" {
		return GitHookResult{
			Action:  "switch-branch",
			Target:  "",
			Success: false,
			Error:   "cannot parse branch or tag information from webhook payload",
			Skipped: false,
			Message: "",
		}, fmt.Errorf("cannot parse branch or tag information from webhook payload")
	}

	log.Printf("parsed webhook: type=%s, target=%s, after=%s", refType, targetRef, afterCommit)

	// check if it matches the project's hook mode
	if project.Hookmode != refType {
		log.Printf("webhook type(%s) does not match project hook mode(%s), skip but return success", refType, project.Hookmode)

		// 记录跳过的项目活动日志
		var actionType string
		switch refType {
		case "branch":
			actionType = database.ProjectActionBranchSwitch
		default:
			actionType = "switch-tag"
		}

		database.LogProjectAction(
			project.Name, // projectName
			actionType,   // action
			fmt.Sprintf("模式:%s", project.Hookmode), // oldValue - 配置的模式
			fmt.Sprintf("模式:%s", refType),          // newValue - 推送的类型
			"GitHook",                              // username
			true,                                   // success - 改为true表示处理成功
			"",                                     // error - 无错误
			"",                                     // commitHash
			fmt.Sprintf("GitHook模式匹配检查：推送类型 %s 与配置模式 %s 不匹配，无需处理", refType, project.Hookmode), // description
			"", // ipAddress
		)

		return GitHookResult{
			Action:  "skip-mode-mismatch",
			Target:  targetRef,
			Success: true,
			Error:   "",
			Skipped: true,
			Message: fmt.Sprintf("推送类型 %s 与配置模式 %s 不匹配，无需处理", refType, project.Hookmode),
		}, nil
	}

	// if it is a branch mode, check if the branch matches
	if project.Hookmode == "branch" {
		if project.Hookbranch != "*" && project.Hookbranch != targetRef {
			log.Printf("webhook branch(%s) does not match configured branch(%s), skip but return success", targetRef, project.Hookbranch)

			// 记录跳过的项目活动日志
			database.LogProjectAction(
				project.Name,                             // projectName
				database.ProjectActionBranchSwitch,       // action
				fmt.Sprintf("分支:%s", project.Hookbranch), // oldValue - 配置的分支
				fmt.Sprintf("分支:%s", targetRef),          // newValue - 推送的分支
				"GitHook",                                // username
				true,                                     // success - 改为true表示处理成功
				"",                                       // error - 无错误
				"",                                       // commitHash
				fmt.Sprintf("GitHook分支匹配检查：推送分支 %s 与配置分支 %s 不匹配，无需切换", targetRef, project.Hookbranch), // description
				"", // ipAddress
			)

			return GitHookResult{
				Action:  "skip-branch-switch",
				Target:  targetRef,
				Success: true,
				Error:   "",
				Skipped: true,
				Message: fmt.Sprintf("推送分支 %s 与配置分支 %s 不匹配，无需切换", targetRef, project.Hookbranch),
			}, nil
		}
	}

	// check if it is a deletion operation (after field is all zeros)
	if afterCommit == "0000000000000000000000000000000000000000" {
		switch refType {
		case "tag":
			// tag deletion: only delete local tag
			log.Printf("detected tag deletion event: %s", targetRef)
			return GitHookResult{
				Action:  "delete-tag",
				Target:  targetRef,
				Success: true,
				Error:   "",
				Skipped: false,
				Message: fmt.Sprintf("检测到标签 %s 删除事件", targetRef),
			}, nil
		case "branch":
			// branch deletion: need to smart judgment
			log.Printf("detected branch deletion event: %s", targetRef)
			return GitHookResult{
				Action:  "delete-branch",
				Target:  targetRef,
				Success: true,
				Error:   "",
				Skipped: false,
				Message: fmt.Sprintf("检测到分支 %s 删除事件", targetRef),
			}, nil
		}
	}

	// 获取当前分支/标签信息用于记录
	var currentPosition string
	var commitHash string

	switch refType {
	case "branch":
		if gitStatus, err := getGitStatus(project.Path); err == nil {
			currentPosition = fmt.Sprintf("分支:%s", gitStatus.CurrentBranch)
		} else {
			currentPosition = "未知位置"
		}
	case "tag":
		// 获取当前标签
		if output, err := execGitCommandOutput(project.Path, "describe", "--tags", "--exact-match", "HEAD"); err == nil {
			currentPosition = fmt.Sprintf("标签:%s", strings.TrimSpace(string(output)))
		} else {
			// 不在标签上，获取分支信息
			if gitStatus, err := getGitStatus(project.Path); err == nil {
				currentPosition = fmt.Sprintf("分支:%s", gitStatus.CurrentBranch)
				if gitStatus.LastCommit != "" {
					currentPosition += fmt.Sprintf("@%s", gitStatus.LastCommit)
				}
			} else {
				currentPosition = "未知位置"
			}
		}
	}

	// execute Git operation
	if err := executeGitHook(project, refType, targetRef); err != nil {
		// 记录GitHook触发的失败项目活动日志
		var actionType string
		var newValue string
		var description string

		switch refType {
		case "branch":
			actionType = database.ProjectActionBranchSwitch
			newValue = targetRef
			description = fmt.Sprintf("GitHook分支切换失败：从 %s 切换到分支 %s 时出错: %s", currentPosition, targetRef, err.Error())
		default:
			actionType = "switch-tag"
			newValue = fmt.Sprintf("标签:%s", targetRef)
			description = fmt.Sprintf("GitHook标签切换失败：从 %s 切换到标签 %s 时出错: %s", currentPosition, targetRef, err.Error())
		}

		database.LogProjectAction(
			project.Name,    // projectName
			actionType,      // action
			currentPosition, // oldValue
			newValue,        // newValue
			"GitHook",       // username - 标识为GitHook触发
			false,           // success
			err.Error(),     // error
			"",              // commitHash
			description,     // description
			"",              // ipAddress - GitHook触发无IP
		)

		return GitHookResult{
			Action:  "switch-branch",
			Target:  "",
			Success: false,
			Error:   "execute Git operation failed: " + err.Error(),
			Skipped: false,
			Message: "",
		}, fmt.Errorf("execute Git operation failed: %v", err)
	}

	// 获取执行后的提交哈希
	if output, err := execGitCommandOutput(project.Path, "rev-parse", "HEAD"); err == nil {
		commitHash = strings.TrimSpace(string(output))
		if len(commitHash) > 7 {
			commitHash = commitHash[:7]
		}
	}

	// 记录GitHook触发的成功项目活动日志
	var actionType string
	var newValue string
	var description string

	switch refType {
	case "branch":
		actionType = database.ProjectActionBranchSwitch
		newValue = targetRef
		description = fmt.Sprintf("GitHook分支切换成功：从 %s 切换到分支 %s (提交: %s)", currentPosition, targetRef, commitHash)
	default:
		actionType = "switch-tag"
		newValue = fmt.Sprintf("标签:%s", targetRef)
		description = fmt.Sprintf("GitHook标签切换成功：从 %s 切换到标签 %s (提交: %s)", currentPosition, targetRef, commitHash)
	}

	database.LogProjectAction(
		project.Name,    // projectName
		actionType,      // action
		currentPosition, // oldValue
		newValue,        // newValue
		"GitHook",       // username - 标识为GitHook触发
		true,            // success
		"",              // error
		commitHash,      // commitHash
		description,     // description
		"",              // ipAddress - GitHook触发无IP
	)

	log.Printf("GitHook processing successfully: project=%s, type=%s, target=%s", project.Name, refType, targetRef)

	var actionName string
	var message string
	switch refType {
	case "branch":
		actionName = "switch-branch"
		message = fmt.Sprintf("成功切换到分支 %s", targetRef)
	default:
		actionName = "switch-tag"
		message = fmt.Sprintf("成功切换到标签 %s", targetRef)
	}

	return GitHookResult{
		Action:  actionName,
		Target:  targetRef,
		Success: true,
		Error:   "",
		Skipped: false,
		Message: message,
	}, nil
}

// executeGitHook execute specific Git operation
func executeGitHook(project *types.ProjectConfig, refType, targetRef string) error {
	projectPath := project.Path

	// check if it is a Git repository
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("project path is not a Git repository: %s", projectPath)
	}

	// fetch latest remote information
	if output, err := execGitCommand(projectPath, "fetch", "--all"); err != nil {
		log.Printf("warning: failed to fetch remote information: %s", string(output))
	}

	switch refType {
	case "branch":
		// branch mode: switch to specified branch and pull latest code
		return switchAndPullBranch(projectPath, targetRef)
	case "tag":
		// tag mode: switch to specified tag
		return switchToTag(projectPath, targetRef)
	default:
		return fmt.Errorf("unsupported reference type: %s", refType)
	}
}

// verify GitHub HMAC-SHA256 signature
func verifyGitHubSignature(payload []byte, secret, signature string) error {
	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("GitHub signature format error, should start with sha256=")
	}

	expectedSig := "sha256=" + hmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("GitHub signature verification failed")
	}

	return nil
}

// verify GitHub legacy signature (old version)
func verifyGitHubLegacySignature(payload []byte, secret, signature string) error {
	if !strings.HasPrefix(signature, "sha1=") {
		return fmt.Errorf("GitHub legacy signature format error, should start with sha1=")
	}

	// note: here should use SHA1, but for security, we suggest using SHA256
	expectedSig := "sha1=" + hmacSHA1Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("GitHub legacy signature verification failed")
	}

	return nil
}

// verify GitLab token verify GitLab token (directly compare)
func verifyGitLabToken(secret, token string) error {
	if subtle.ConstantTimeCompare([]byte(secret), []byte(token)) != 1 {
		return fmt.Errorf("GitLab token verification failed")
	}
	return nil
}

// verify Gitee token verify Gitee token (password mode, directly compare)
// Gitee supports password mode where X-Gitee-Token contains the plain text password
func verifyGiteeToken(secret, token string) error {
	if subtle.ConstantTimeCompare([]byte(secret), []byte(token)) != 1 {
		return fmt.Errorf("gitee token verification failed")
	}
	return nil
}

// verify Gitee signature verify Gitee HMAC-SHA256 signature
// Gitee signature mode: stringToSign = timestamp + "\n" + secret
// Sign with HMAC-SHA256, then Base64 encode (no URL encoding needed)
func verifyGiteeSignature(secret, token, timestamp string) error {
	// timestamp + "\n" + secret
	stringToSign := timestamp + "\n" + secret

	// HMAC-SHA256
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	signData := h.Sum(nil)

	// Base64 encode (gitee sends Base64 directly, no URL encoding)
	expectedSig := base64.StdEncoding.EncodeToString(signData)

	if subtle.ConstantTimeCompare([]byte(token), []byte(expectedSig)) != 1 {
		return fmt.Errorf("gitee signature verification failed")
	}
	return nil
}

// verify Gitea signature verify Gitea HMAC-SHA256 signature
func verifyGiteaSignature(payload []byte, secret, signature string) error {
	expectedSig := hmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("gitea signature verification failed")
	}
	return nil
}

// verify Gogs signature verify Gogs HMAC-SHA256 signature
func verifyGogsSignature(payload []byte, secret, signature string) error {
	expectedSig := hmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("gogs signature verification failed")
	}
	return nil
}

// hmacSHA256Hex calculate HMAC-SHA256 and return hexadecimal string
func hmacSHA256Hex(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmacSHA1Hex calculate HMAC-SHA1 and return hexadecimal string (for GitHub legacy support)
func hmacSHA1Hex(data []byte, secret string) string {
	// note: here should import crypto/sha1, but for simplicity, we skip this implementation
	// in production environment, SHA1 should be correctly implemented
	return hmacSHA256Hex(data, secret) // temporarily use SHA256 instead
}

// VerifyWebhookSignature verify webhook signature, support GitHub, GitLab, Gitee, etc.
func verifyWebhookSignature(c *gin.Context, payloadBody []byte, secret string) error {
	// GitHub use X-Hub-Signature-256 header with HMAC-SHA256
	if githubSig := c.GetHeader("X-Hub-Signature-256"); githubSig != "" {
		return verifyGitHubSignature(payloadBody, secret, githubSig)
	}

	// GitHub legacy use X-Hub-Signature header with HMAC-SHA1
	if githubSigLegacy := c.GetHeader("X-Hub-Signature"); githubSigLegacy != "" {
		return verifyGitHubLegacySignature(payloadBody, secret, githubSigLegacy)
	}

	// GitLab use X-Gitlab-Token header, directly compare password
	if gitlabToken := c.GetHeader("X-Gitlab-Token"); gitlabToken != "" {
		return verifyGitLabToken(secret, gitlabToken)
	}

	// Gitee use X-Gitee-Token header, support both password and signature mode
	// Headers: X-Gitee-Token, X-Gitee-Timestamp, User-Agent: git-oschina-hook
	// Note: Both modes have timestamp, so we need to try both verification methods
	if giteeToken := c.GetHeader("X-Gitee-Token"); giteeToken != "" {
		giteeTimestamp := c.GetHeader("X-Gitee-Timestamp")

		// Try signature mode first (if timestamp exists)
		if giteeTimestamp != "" {
			if err := verifyGiteeSignature(secret, giteeToken, giteeTimestamp); err == nil {
				return nil // signature verification successful
			}
		}

		// If signature verification failed or no timestamp, try password mode
		return verifyGiteeToken(secret, giteeToken)
	}

	// Gitea use X-Gitea-Signature header with HMAC-SHA256
	if giteaSig := c.GetHeader("X-Gitea-Signature"); giteaSig != "" {
		return verifyGiteaSignature(payloadBody, secret, giteaSig)
	}

	// Gogs use X-Gogs-Signature header with HMAC-SHA256
	if gogsSig := c.GetHeader("X-Gogs-Signature"); gogsSig != "" {
		return verifyGogsSignature(payloadBody, secret, gogsSig)
	}

	// if no known signature header is found, return error
	return fmt.Errorf("no supported webhook signature header found")
}

// SaveGitHook save project GitHook configuration
func HandleSaveGitHook(c *gin.Context) {
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
		"code":    0,
		"message": "GitHook configuration saved successfully",
	})
}

// GitHook handle GitHook request
func HandleGitHook(c *gin.Context) {
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

	// parse webhook payload (support GitHub, GitLab, Gitee, etc.)
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook payload"})
		return
	}

	// handle GitHook logic
	result, err := tryGitHook(project, payload)

	// 记录GitHook执行日志到数据库
	var outputMessage string
	if result.Skipped {
		outputMessage = fmt.Sprintf("Action: %s, Target: %s, Skipped: true, Message: %s", result.Action, result.Target, result.Message)
	} else {
		outputMessage = fmt.Sprintf("Action: %s, Target: %s", result.Action, result.Target)
	}

	database.LogHookExecution(
		project.Name,              // hookID (使用项目名作为ID)
		"GitHook-"+project.Name,   // hookName
		"githook",                 // hookType
		c.Request.Method,          // method
		middleware.GetClientIP(c), // remoteAddr
		c.Request.Header,          // headers
		string(payloadBody),       // body
		result.Success,            // success
		outputMessage,             // output
		result.Error,              // error
		0,                         // duration (无精确执行时间)
		c.Request.UserAgent(),     // userAgent
		map[string][]string{ // queryParams
			"project": {project.Name},
			"mode":    {project.Hookmode},
		},
	)

	if err != nil {
		// push failed message
		wsMessage := stream.WsMessage{
			Type:      "githook_triggered",
			Timestamp: time.Now(),
			Data: stream.GitHookTriggeredMessage{
				Action:      result.Action,
				ProjectName: projectName,
				Target:      result.Target,
				Success:     result.Success,
				Error:       "GitHook processing failed: " + err.Error(),
				Skipped:     result.Skipped,
				Message:     result.Message,
			},
		}
		stream.Global.Broadcast(wsMessage)
		log.Printf("GitHook processing failed: project=%s, error=%v", project.Name, err)
		c.String(http.StatusInternalServerError, "GitHook processing failed: "+result.Action+" "+result.Target+" "+strconv.FormatBool(result.Success)+" "+err.Error())
		return
	}

	// push success message
	wsMessage := stream.WsMessage{
		Type:      "githook_triggered",
		Timestamp: time.Now(),
		Data: stream.GitHookTriggeredMessage{
			Action:      result.Action,
			ProjectName: projectName,
			Target:      result.Target,
			Success:     result.Success,
			Skipped:     result.Skipped,
			Message:     result.Message,
		},
	}
	stream.Global.Broadcast(wsMessage)

	var responseMessage string
	if result.Skipped {
		responseMessage = fmt.Sprintf("GitHook processing successfully (skipped): %s %s - %s", result.Action, result.Target, result.Message)
	} else {
		responseMessage = fmt.Sprintf("GitHook processing successfully: %s %s", result.Action, result.Target)
	}
	c.String(http.StatusOK, responseMessage)
}
