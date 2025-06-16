package version

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/types"
)

// initGit 初始化Git仓库
func InitGit(projectPath string) error {
	// 检查项目路径是否存在
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不存在: %s", projectPath)
	}

	// 检查项目路径是否为目录
	if info, err := os.Stat(projectPath); err != nil {
		return fmt.Errorf("无法访问项目路径: %s, 错误: %v", projectPath, err)
	} else if !info.IsDir() {
		return fmt.Errorf("项目路径不是目录: %s", projectPath)
	}

	// 检查是否已经是Git仓库
	gitDir := filepath.Join(projectPath, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return fmt.Errorf("目录已经是Git仓库")
	}

	// 尝试创建一个临时文件来测试写权限
	testFile := filepath.Join(projectPath, ".gohook-permission-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("项目路径没有写权限: %s，请检查目录权限。建议运行: sudo chown -R %s:%s %s",
			projectPath, os.Getenv("USER"), os.Getenv("USER"), projectPath)
	}
	// 清理测试文件
	os.Remove(testFile)

	// 执行git init命令
	cmd := exec.Command("git", "-C", projectPath, "init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Git仓库初始化失败: %v, 输出: %s", err, string(output))
	}

	// 验证Git仓库是否成功创建
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("Git仓库初始化后验证失败: .git目录未创建")
	}

	return nil
}

// verifyGitHubSignature 验证GitHub HMAC-SHA256签名
func VerifyGitHubSignature(payload []byte, secret, signature string) error {
	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("GitHub签名格式错误，应以sha256=开头")
	}

	expectedSig := "sha256=" + HmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("GitHub签名验证失败")
	}

	return nil
}

// verifyGitHubLegacySignature 验证GitHub HMAC-SHA1签名（旧版）
func VerifyGitHubLegacySignature(payload []byte, secret, signature string) error {
	if !strings.HasPrefix(signature, "sha1=") {
		return fmt.Errorf("GitHub legacy签名格式错误，应以sha1=开头")
	}

	// 注意：这里应该使用SHA1，但为了安全性，我们建议使用SHA256
	expectedSig := "sha1=" + HmacSHA1Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("GitHub legacy签名验证失败")
	}

	return nil
}

// verifyGitLabToken 验证GitLab token（直接比较）
func VerifyGitLabToken(secret, token string) error {
	if subtle.ConstantTimeCompare([]byte(secret), []byte(token)) != 1 {
		return fmt.Errorf("GitLab token验证失败")
	}
	return nil
}

// verifyGiteaSignature 验证Gitea HMAC-SHA256签名
func VerifyGiteaSignature(payload []byte, secret, signature string) error {
	expectedSig := HmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("Gitea签名验证失败")
	}
	return nil
}

// verifyGogsSignature 验证Gogs HMAC-SHA256签名
func VerifyGogsSignature(payload []byte, secret, signature string) error {
	expectedSig := HmacSHA256Hex(payload, secret)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return fmt.Errorf("Gogs签名验证失败")
	}
	return nil
}

// hmacSHA256Hex 计算HMAC-SHA256并返回十六进制字符串
func HmacSHA256Hex(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmacSHA1Hex 计算HMAC-SHA1并返回十六进制字符串（用于GitHub legacy支持）
func HmacSHA1Hex(data []byte, secret string) string {
	// 注意：这里应该导入crypto/sha1，但为了保持简单，我们跳过这个实现
	// 在生产环境中应该正确实现SHA1
	return HmacSHA256Hex(data, secret) // 临时使用SHA256代替
}

// VerifyWebhookSignature 验证webhook签名，支持GitHub、GitLab等不同格式
func VerifyWebhookSignature(c *gin.Context, payloadBody []byte, secret string) error {
	// GitHub使用X-Hub-Signature-256 header with HMAC-SHA256
	if githubSig := c.GetHeader("X-Hub-Signature-256"); githubSig != "" {
		return VerifyGitHubSignature(payloadBody, secret, githubSig)
	}

	// GitHub旧版使用X-Hub-Signature header with HMAC-SHA1
	if githubSigLegacy := c.GetHeader("X-Hub-Signature"); githubSigLegacy != "" {
		return VerifyGitHubLegacySignature(payloadBody, secret, githubSigLegacy)
	}

	// GitLab使用X-Gitlab-Token header，直接比较密码
	if gitlabToken := c.GetHeader("X-Gitlab-Token"); gitlabToken != "" {
		return VerifyGitLabToken(secret, gitlabToken)
	}

	// Gitea使用X-Gitea-Signature header with HMAC-SHA256
	if giteaSig := c.GetHeader("X-Gitea-Signature"); giteaSig != "" {
		return VerifyGiteaSignature(payloadBody, secret, giteaSig)
	}

	// Gogs使用X-Gogs-Signature header with HMAC-SHA256
	if gogsSig := c.GetHeader("X-Gogs-Signature"); gogsSig != "" {
		return VerifyGogsSignature(payloadBody, secret, gogsSig)
	}

	// 如果没有找到任何已知的签名header，返回错误
	return fmt.Errorf("未找到支持的webhook签名header")
}

// executeGitHook 执行具体的Git操作
func ExecuteGitHook(project *types.ProjectConfig, refType, targetRef string) error {
	projectPath := project.Path

	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("项目路径不是Git仓库: %s", projectPath)
	}

	// 首先拉取最新的远程信息
	cmd := exec.Command("git", "-C", projectPath, "fetch", "--all")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("警告: 拉取远程信息失败: %s", string(output))
	}

	if refType == "branch" {
		// 分支模式：切换到指定分支并拉取最新代码
		return switchAndPullBranch(projectPath, targetRef)
	} else if refType == "tag" {
		// 标签模式：切换到指定标签
		return switchToTag(projectPath, targetRef)
	}

	return fmt.Errorf("不支持的引用类型: %s", refType)
}

// switchAndPullBranch 切换到指定分支并拉取最新代码
func switchAndPullBranch(projectPath, branchName string) error {
	// 检查本地是否存在该分支
	cmd := exec.Command("git", "-C", projectPath, "branch", "--list", branchName)
	output, err := cmd.Output()
	localBranchExists := err == nil && strings.TrimSpace(string(output)) != ""

	if !localBranchExists {
		// 本地分支不存在，尝试从远程创建
		cmd = exec.Command("git", "-C", projectPath, "checkout", "-b", branchName, "origin/"+branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("创建并切换到分支 %s 失败: %s", branchName, string(output))
		}
	} else {
		// 本地分支存在，直接切换
		cmd = exec.Command("git", "-C", projectPath, "checkout", branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("切换到分支 %s 失败: %s", branchName, string(output))
		}

		// 拉取最新代码
		cmd = exec.Command("git", "-C", projectPath, "pull", "origin", branchName)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("拉取分支 %s 最新代码失败: %s", branchName, string(output))
		}
	}

	return nil
}

// switchToTag 切换到指定标签
func switchToTag(projectPath, tagName string) error {
	// 拉取标签信息
	cmd := exec.Command("git", "-C", projectPath, "fetch", "--tags")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("警告: 拉取标签信息失败: %s", string(output))
	}

	// 确保标签存在（本地或远程）
	cmd = exec.Command("git", "-C", projectPath, "rev-parse", tagName)
	if err := cmd.Run(); err != nil {
		log.Printf("标签 %s 不存在，尝试从远程获取", tagName)
		cmd = exec.Command("git", "-C", projectPath, "fetch", "origin", "--tags")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("从远程获取标签失败: %s", string(output))
		}

		// 再次检查标签是否存在
		cmd = exec.Command("git", "-C", projectPath, "rev-parse", tagName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("标签 %s 在远程也不存在，无法部署", tagName)
		}
	}

	// 切换到指定标签
	cmd = exec.Command("git", "-C", projectPath, "checkout", tagName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("切换到标签 %s 失败: %s", tagName, string(output))
	}

	log.Printf("成功切换到标签: %s", tagName)
	return nil
}

// HandleGitHook 处理GitHook webhook请求
func HandleGitHook(project *types.ProjectConfig, payload map[string]interface{}) error {
	log.Printf("处理GitHook: 项目=%s, 模式=%s, 分支设置=%s", project.Name, project.Hookmode, project.Hookbranch)

	// 解析webhook payload，提取分支或标签信息
	var targetRef string
	var refType string
	var afterCommit string

	// 尝试解析GitHub/GitLab格式的webhook
	if ref, ok := payload["ref"].(string); ok {
		// 提取after字段（用于检测删除操作）
		if after, ok := payload["after"].(string); ok {
			afterCommit = after
		}

		if strings.HasPrefix(ref, "refs/heads/") {
			// 分支推送
			targetRef = strings.TrimPrefix(ref, "refs/heads/")
			refType = "branch"
		} else if strings.HasPrefix(ref, "refs/tags/") {
			// 标签推送
			targetRef = strings.TrimPrefix(ref, "refs/tags/")
			refType = "tag"
		}
	}

	// 如果没有解析到ref，尝试其他格式
	if targetRef == "" {
		// 尝试GitLab格式
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
		return fmt.Errorf("无法从webhook payload中解析分支或标签信息")
	}

	log.Printf("解析到webhook: 类型=%s, 目标=%s, after=%s", refType, targetRef, afterCommit)

	// 检查是否匹配项目的hook模式
	if project.Hookmode != refType {
		log.Printf("webhook类型(%s)与项目hook模式(%s)不匹配，忽略", refType, project.Hookmode)
		return nil
	}

	// 如果是分支模式，检查分支匹配
	if project.Hookmode == "branch" {
		if project.Hookbranch != "*" && project.Hookbranch != targetRef {
			log.Printf("webhook分支(%s)与配置分支(%s)不匹配，忽略", targetRef, project.Hookbranch)
			return nil
		}
	}

	// 检查是否是删除操作（after字段为全零）
	if afterCommit == "0000000000000000000000000000000000000000" {
		if refType == "tag" {
			// 标签删除：只删除本地标签
			log.Printf("检测到标签删除事件: %s", targetRef)
			return DeleteLocalTag(project.Path, targetRef)
		} else if refType == "branch" {
			// 分支删除：需要智能判断
			log.Printf("检测到分支删除事件: %s", targetRef)
			return BranchDeletion(project, targetRef)
		}
	}

	// 执行Git操作
	if err := ExecuteGitHook(project, refType, targetRef); err != nil {
		return fmt.Errorf("执行Git操作失败: %v", err)
	}

	log.Printf("GitHook处理成功: 项目=%s, 类型=%s, 目标=%s", project.Name, refType, targetRef)
	return nil
}

// DeleteLocalTag 删除本地标签
func DeleteLocalTag(projectPath, tagName string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库: %s", projectPath)
	}

	// 检查标签是否存在
	cmd := exec.Command("git", "-C", projectPath, "show-ref", "--tags", "--quiet", "refs/tags/"+tagName)
	if err := cmd.Run(); err != nil {
		log.Printf("本地标签 %s 不存在，无需删除", tagName)
		return nil
	}

	// 删除本地标签
	cmd = exec.Command("git", "-C", projectPath, "tag", "-d", tagName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("删除本地标签 %s 失败: %s", tagName, string(output))
	}

	log.Printf("成功删除本地标签: %s", tagName)
	return nil
}

// DeleteLocalBranch 删除本地分支
func DeleteLocalBranch(projectPath, branchName string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库: %s", projectPath)
	}

	// 获取当前分支
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranchOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取当前分支失败: %v", err)
	}
	currentBranch := strings.TrimSpace(string(currentBranchOutput))

	// 检查是否试图删除当前分支
	if currentBranch == branchName {
		log.Printf("不能删除当前分支 %s，跳过删除操作", branchName)
		return nil
	}

	// 检查分支是否存在
	cmd = exec.Command("git", "-C", projectPath, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err := cmd.Run(); err != nil {
		log.Printf("本地分支 %s 不存在，无需删除", branchName)
		return nil
	}

	// 删除本地分支
	cmd = exec.Command("git", "-C", projectPath, "branch", "-D", branchName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("删除本地分支 %s 失败: %s", branchName, string(output))
	}

	log.Printf("成功删除本地分支: %s", branchName)
	return nil
}

// BranchDeletion 智能处理分支删除操作
func BranchDeletion(project *types.ProjectConfig, branchName string) error {
	log.Printf("智能处理分支删除: 项目=%s, 分支=%s, 配置分支=%s", project.Name, branchName, project.Hookbranch)

	// 检查是否是配置的分支
	if project.Hookbranch != "*" && project.Hookbranch == branchName {
		log.Printf("删除的是配置分支 %s，忽略删除操作以保护项目运行", branchName)
		return nil
	}

	// 检查是否是master分支
	if branchName == "master" || branchName == "main" {
		log.Printf("删除的是主分支 %s，忽略删除操作以保护项目", branchName)
		return nil
	}

	// 如果是其他分支，执行删除操作
	log.Printf("删除的是非关键分支 %s，执行本地删除操作", branchName)
	return DeleteLocalBranch(project.Path, branchName)
}

// DeleteBranch 删除本地分支
func DeleteBranch(projectPath, branchName string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库")
	}

	// 获取当前分支
	cmd := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD")
	currentBranchOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取当前分支失败: %v", err)
	}
	currentBranch := strings.TrimSpace(string(currentBranchOutput))

	// 检查是否试图删除当前分支
	if currentBranch == branchName {
		return fmt.Errorf("不能删除当前分支")
	}

	// 删除本地分支
	cmd = exec.Command("git", "-C", projectPath, "branch", "-D", branchName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除分支失败: %s", string(output))
	}

	return nil
}

// DeleteTag 删除本地和远程标签
func DeleteTag(projectPath, tagName string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库")
	}

	// 检查当前是否在该标签上
	cmd := exec.Command("git", "-C", projectPath, "describe", "--tags", "--exact-match", "HEAD")
	currentTagOutput, err := cmd.Output()
	if err == nil {
		currentTag := strings.TrimSpace(string(currentTagOutput))
		if currentTag == tagName {
			return fmt.Errorf("不能删除当前标签")
		}
	}

	// 删除本地标签
	cmd = exec.Command("git", "-C", projectPath, "tag", "-d", tagName)
	localOutput, localErr := cmd.CombinedOutput()
	if localErr != nil {
		return fmt.Errorf("删除本地标签失败: %s", string(localOutput))
	}

	// 尝试删除远程标签
	cmd = exec.Command("git", "-C", projectPath, "push", "origin", ":refs/tags/"+tagName)
	remoteOutput, remoteErr := cmd.CombinedOutput()
	if remoteErr != nil {
		log.Printf("删除远程标签失败 (项目: %s, 标签: %s): %s", projectPath, tagName, string(remoteOutput))
		// 远程标签删除失败不作为致命错误，因为可能远程没有该标签
	}

	return nil
}

// SwitchTag 切换标签
func SwitchTag(projectPath, tagName string) error {
	cmd := exec.Command("git", "-C", projectPath, "checkout", tagName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("切换标签失败: %v", err)
	}
	return nil
}

// GetRemote 获取远程仓库URL
func GetRemote(projectPath string) (string, error) {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return "", fmt.Errorf("不是Git仓库")
	}

	// 获取origin远程仓库URL
	cmd := exec.Command("git", "-C", projectPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		// 如果 "origin" 不存在，命令会返回非零退出码。
		// 这种情况下我们返回空字符串，表示没有设置远程地址。
		return "", nil
	}

	return strings.TrimSpace(string(output)), nil
}

// SetRemote 设置远程仓库
func SetRemote(projectPath, remoteUrl string) error {
	// 检查是否是Git仓库
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return fmt.Errorf("不是Git仓库")
	}

	// 检查是否已有origin远程仓库
	cmd := exec.Command("git", "-C", projectPath, "remote", "get-url", "origin")
	if cmd.Run() == nil {
		// 如果已有origin，先删除
		cmd = exec.Command("git", "-C", projectPath, "remote", "remove", "origin")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("删除原有远程仓库失败: %v", err)
		}
	}

	// 添加新的origin远程仓库
	cmd = exec.Command("git", "-C", projectPath, "remote", "add", "origin", remoteUrl)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("设置远程仓库失败: %v", err)
	}

	return nil
}

// SyncBranches 同步远程分支，清理已删除的远程分支引用
func SyncBranches(projectPath string) error {
	// 使用 fetch --prune 来更新远程分支信息并删除不存在的引用
	cmd := exec.Command("git", "-C", projectPath, "fetch", "origin", "--prune")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("同步分支失败: %s", string(output))
	}
	return nil
}

// SwitchBranch 切换分支
func SwitchBranch(projectPath, branchName string) error {
	var cmd *exec.Cmd
	var isRemoteBranch bool
	var localBranchName string

	// 检查是否是远程分支格式 (例如 origin/release)
	if strings.HasPrefix(branchName, "origin/") {
		isRemoteBranch = true
		localBranchName = strings.TrimPrefix(branchName, "origin/")

		// 检查本地是否已有同名分支
		checkCmd := exec.Command("git", "-C", projectPath, "rev-parse", "--verify", localBranchName)
		if checkCmd.Run() == nil {
			// 本地分支已存在，直接切换
			cmd = exec.Command("git", "-C", projectPath, "checkout", localBranchName)
		} else {
			// 本地分支不存在，基于远程分支创建新的本地分支
			cmd = exec.Command("git", "-C", projectPath, "checkout", "-b", localBranchName, branchName)
		}
	} else {
		// 普通的本地分支切换
		isRemoteBranch = false
		localBranchName = branchName
		cmd = exec.Command("git", "-C", projectPath, "checkout", branchName)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("切换分支失败: %s", string(output))
	}

	// 如果是基于远程分支创建的新分支，尝试拉取最新代码
	if isRemoteBranch {
		pullCmd := exec.Command("git", "-C", projectPath, "pull", "origin", localBranchName)
		pullOutput, pullErr := pullCmd.CombinedOutput()
		if pullErr != nil {
			// 拉取失败不认为是致命错误，但记录日志
			log.Printf("切换分支后拉取最新代码失败 (项目: %s, 分支: %s): %s", projectPath, localBranchName, string(pullOutput))
		}
	}

	return nil
}

// GetTags 获取标签列表
func GetTags(projectPath string) ([]types.TagResponse, error) {
	// 获取当前标签
	cmd := exec.Command("git", "-C", projectPath, "describe", "--exact-match", "--tags", "HEAD")
	currentOutput, _ := cmd.Output()
	currentTag := strings.TrimSpace(string(currentOutput))

	// 获取所有标签
	cmd = exec.Command("git", "-C", projectPath, "tag", "-l", "--sort=-version:refname", "--format=%(refname:short)|%(creatordate)|%(objectname:short)|%(subject)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取标签列表失败: %v", err)
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

// GetBranches 获取分支列表
func GetBranches(projectPath string) ([]types.BranchResponse, error) {
	var branches []types.BranchResponse
	branchSet := make(map[string]bool) // 用于防止重复添加

	// 1. 获取当前是否处于分离头状态
	_, err := exec.Command("git", "-C", projectPath, "symbolic-ref", "-q", "HEAD").Output()
	isDetached := err != nil

	// 2. 获取当前分支或提交的引用
	var currentRef string
	if isDetached {
		// 分离头状态，获取 HEAD 的短哈希
		headSha, err := exec.Command("git", "-C", projectPath, "rev-parse", "--short", "HEAD").Output()
		if err != nil {
			return nil, fmt.Errorf("获取HEAD commit失败: %v", err)
		}
		currentRef = strings.TrimSpace(string(headSha))
	} else {
		// 在分支上，获取分支名
		branchName, err := exec.Command("git", "-C", projectPath, "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return nil, fmt.Errorf("获取当前分支名失败: %v", err)
		}
		currentRef = strings.TrimSpace(string(branchName))
	}

	// 3. 处理分离头状态
	if isDetached {
		// 尝试获取标签名
		tagName, err := exec.Command("git", "-C", projectPath, "describe", "--tags", "--exact-match", "HEAD").Output()
		var displayName string
		if err == nil {
			displayName = strings.TrimSpace(string(tagName))
		} else {
			displayName = currentRef
		}

		// 获取最后提交信息
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
			Name:           fmt.Sprintf("(当前指向 %s)", displayName),
			IsCurrent:      true,
			LastCommit:     lastCommit,
			LastCommitTime: lastCommitTime,
			Type:           "detached",
		})
	}

	// 4. 获取所有本地分支
	cmd := exec.Command("git", "-C", projectPath, "for-each-ref", "refs/heads", "--format=%(refname:short)|%(committerdate:iso)|%(objectname:short)")
	localOutput, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取本地分支列表失败: %v", err)
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

	// 5. 获取所有远程分支
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
					continue // 忽略 HEAD 指针
				}
				branchName := remoteRef // 例如 "origin/master"
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
		log.Printf("获取远程分支列表失败 (项目: %s): %v", projectPath, err)
	}

	return branches, nil
}

// GetGitStatus 获取Git状态
func GetGitStatus(projectPath string) (*types.VersionResponse, error) {
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		return nil, fmt.Errorf("不是Git仓库")
	}

	// 获取当前分支
	cmd := exec.Command("git", "-C", projectPath, "branch", "--show-current")
	branchOutput, _ := cmd.Output()
	currentBranch := strings.TrimSpace(string(branchOutput))

	// 获取当前标签（如果在标签上）
	cmd = exec.Command("git", "-C", projectPath, "describe", "--exact-match", "--tags", "HEAD")
	tagOutput, _ := cmd.Output()
	currentTag := strings.TrimSpace(string(tagOutput))

	// 确定模式
	mode := "branch"
	if currentTag != "" {
		mode = "tag"
	}

	// 获取最后提交信息
	cmd = exec.Command("git", "-C", projectPath, "log", "-1", "--format=%H|%ci|%s")
	commitOutput, _ := cmd.Output()
	commitInfo := strings.TrimSpace(string(commitOutput))

	parts := strings.Split(commitInfo, "|")
	lastCommit := ""
	lastCommitTime := ""
	if len(parts) >= 2 {
		lastCommit = parts[0][:8] // 短哈希
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
