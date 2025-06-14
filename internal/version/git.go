package version

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
