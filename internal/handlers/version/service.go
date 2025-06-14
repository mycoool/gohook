package version

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mycoool/gohook/internal/types"
	"gopkg.in/yaml.v2"
)

var configData *types.Config

// LoadConfig 加载配置文件
func LoadConfig() error {
	data, err := os.ReadFile("version.yaml")
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	config := &types.Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	configData = config
	return nil
}

// GetConfig 获取配置
func GetConfig() *types.Config {
	return configData
}

// SaveConfig 保存配置文件
func SaveConfig() error {
	if configData == nil {
		return fmt.Errorf("配置数据为空")
	}

	data, err := yaml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	// 备份原配置文件
	if _, err := os.Stat("version.yaml"); err == nil {
		if err := os.Rename("version.yaml", "version.yaml.bak"); err != nil {
			log.Printf("Warning: failed to backup config file: %v", err)
		}
	}

	err = os.WriteFile("version.yaml", data, 0644)
	if err != nil {
		// 如果保存失败，恢复备份
		if _, backupErr := os.Stat("version.yaml.bak"); backupErr == nil {
			if restoreErr := os.Rename("version.yaml.bak", "version.yaml"); restoreErr != nil {
				log.Printf("Error: failed to restore backup config file: %v", restoreErr)
			}
		}
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	return nil
}

// isGitRepository 检查目录是否是Git仓库
func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if stat, err := os.Stat(gitDir); err == nil {
		return stat.IsDir()
	}
	return false
}

// getCurrentBranch 获取当前分支
func getCurrentBranch(projectPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getCurrentTag 获取当前标签
func getCurrentTag(projectPath string) (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--exact-match", "HEAD")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		// 如果没有标签，返回空字符串而不是错误
		return "", nil
	}
	return strings.TrimSpace(string(output)), nil
}

// getLastCommit 获取最后一次提交信息
func getLastCommit(projectPath string) (string, string, error) {
	cmd := exec.Command("git", "log", "-1", "--format=%H|%ci")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "", "", err
	}

	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected git log output format")
	}

	return parts[0], parts[1], nil
}

// getBranches 获取所有分支
func getBranches(projectPath string) ([]types.BranchResponse, error) {
	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)|%(objectname:short)|%(committerdate:iso8601)")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var branches []types.BranchResponse
	currentBranch, _ := getCurrentBranch(projectPath)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}

		branchName := parts[0]
		commitHash := parts[1]
		commitTime := parts[2]

		// 跳过HEAD指针
		if strings.Contains(branchName, "HEAD") {
			continue
		}

		// 确定分支类型
		branchType := "local"
		if strings.HasPrefix(branchName, "origin/") {
			branchType = "remote"
			// 检查是否有对应的本地分支
			localName := strings.TrimPrefix(branchName, "origin/")
			for _, b := range branches {
				if b.Name == localName && b.Type == "local" {
					continue // 跳过已有本地分支的远程分支
				}
			}
		}

		branches = append(branches, types.BranchResponse{
			Name:           branchName,
			IsCurrent:      branchName == currentBranch,
			LastCommit:     commitHash,
			LastCommitTime: commitTime,
			Type:           branchType,
		})
	}

	return branches, nil
}

// getTags 获取所有标签
func getTags(projectPath string) ([]types.TagResponse, error) {
	cmd := exec.Command("git", "tag", "-l", "--format=%(refname:short)|%(objectname:short)|%(creatordate:iso8601)|%(subject)")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var tags []types.TagResponse
	currentTag, _ := getCurrentTag(projectPath)

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		tagName := parts[0]
		commitHash := parts[1]
		date := parts[2]
		message := ""
		if len(parts) > 3 {
			message = parts[3]
		}

		tags = append(tags, types.TagResponse{
			Name:       tagName,
			IsCurrent:  tagName == currentTag,
			CommitHash: commitHash,
			Date:       date,
			Message:    message,
		})
	}

	// 按版本号排序（尝试语义化版本排序）
	sort.Slice(tags, func(i, j int) bool {
		return compareVersions(tags[i].Name, tags[j].Name) > 0
	})

	return tags, nil
}

// compareVersions 比较版本号
func compareVersions(v1, v2 string) int {
	// 移除v前缀
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	// 分割版本号
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// 补齐长度
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for len(parts1) < maxLen {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < maxLen {
		parts2 = append(parts2, "0")
	}

	// 逐个比较
	for i := 0; i < maxLen; i++ {
		// 尝试作为数字比较
		num1, err1 := strconv.Atoi(parts1[i])
		num2, err2 := strconv.Atoi(parts2[i])

		if err1 == nil && err2 == nil {
			if num1 != num2 {
				return num1 - num2
			}
		} else {
			// 作为字符串比较
			if parts1[i] != parts2[i] {
				if parts1[i] > parts2[i] {
					return 1
				}
				return -1
			}
		}
	}

	return 0
}

// switchBranch 切换分支
func SwitchBranch(projectPath, branchName string) error {
	// 检查是否是远程分支
	if strings.HasPrefix(branchName, "origin/") {
		localBranch := strings.TrimPrefix(branchName, "origin/")
		// 创建并切换到本地分支
		cmd := exec.Command("git", "checkout", "-b", localBranch, branchName)
		cmd.Dir = projectPath
		if err := cmd.Run(); err != nil {
			// 如果创建失败，尝试直接切换
			cmd = exec.Command("git", "checkout", localBranch)
			cmd.Dir = projectPath
			return cmd.Run()
		}
		return nil
	}

	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = projectPath
	return cmd.Run()
}

// switchTag 切换到标签
func SwitchTag(projectPath, tagName string) error {
	cmd := exec.Command("git", "checkout", tagName)
	cmd.Dir = projectPath
	return cmd.Run()
}

// deleteTag 删除标签
func DeleteTag(projectPath, tagName string) error {
	cmd := exec.Command("git", "tag", "-d", tagName)
	cmd.Dir = projectPath
	return cmd.Run()
}

// getEnvFile 获取环境文件内容
func GetEnvFile(projectPath string) (string, error) {
	envPath := filepath.Join(projectPath, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return "", nil // 文件不存在，返回空内容
	}

	content, err := os.ReadFile(envPath)
	if err != nil {
		return "", fmt.Errorf("读取环境文件失败: %v", err)
	}

	return string(content), nil
}

// saveEnvFile 保存环境文件
func SaveEnvFile(projectPath, content string) error {
	envPath := filepath.Join(projectPath, ".env")

	// 备份原文件
	if _, err := os.Stat(envPath); err == nil {
		backupPath := envPath + ".bak"
		if err := os.Rename(envPath, backupPath); err != nil {
			log.Printf("Warning: failed to backup env file: %v", err)
		}
	}

	err := os.WriteFile(envPath, []byte(content), 0644)
	if err != nil {
		// 如果保存失败，恢复备份
		backupPath := envPath + ".bak"
		if _, backupErr := os.Stat(backupPath); backupErr == nil {
			if restoreErr := os.Rename(backupPath, envPath); restoreErr != nil {
				log.Printf("Error: failed to restore backup env file: %v", restoreErr)
			}
		}
		return fmt.Errorf("保存环境文件失败: %v", err)
	}

	return nil
}

// deleteEnvFile 删除环境文件
func DeleteEnvFile(projectPath string) error {
	envPath := filepath.Join(projectPath, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil // 文件不存在，认为删除成功
	}

	return os.Remove(envPath)
}

// getProjectStatus 获取项目状态
func getProjectStatus(projectPath string) string {
	if !isGitRepository(projectPath) {
		return "非Git项目"
	}

	// 检查是否有未提交的更改
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = projectPath
	output, err := cmd.Output()
	if err != nil {
		return "状态未知"
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		return "有未提交更改"
	}

	return "干净"
}
