package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 环境变量文件处理函数

// getEnvFile 读取项目的.env文件内容
func GetEnvFile(projectPath string) (string, bool, error) {
	envFilePath := filepath.Join(projectPath, ".env")

	// 检查文件是否存在
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return "", false, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(envFilePath)
	if err != nil {
		return "", true, fmt.Errorf("读取环境变量文件失败: %v", err)
	}

	return string(content), true, nil
}

// saveEnvFile 保存项目的.env文件
func SaveEnvFile(projectPath, content string) error {
	envFilePath := filepath.Join(projectPath, ".env")

	// 确保项目目录存在
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("项目目录不存在: %s", projectPath)
	}

	// 写入文件，如果文件不存在会自动创建
	err := os.WriteFile(envFilePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("保存环境变量文件失败: %v", err)
	}

	return nil
}

// deleteEnvFile 删除项目的.env文件
func DeleteEnvFile(projectPath string) error {
	envFilePath := filepath.Join(projectPath, ".env")

	// 检查文件是否存在
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return fmt.Errorf("环境变量文件不存在")
	}

	// 删除文件
	err := os.Remove(envFilePath)
	if err != nil {
		return fmt.Errorf("删除环境变量文件失败: %v", err)
	}

	return nil
}

// validateEnvContent 验证环境变量文件格式
func ValidateEnvContent(content string) []string {
	var errors []string
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)

		// 跳过空行和注释行
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// 检查是否包含等号
		if !strings.Contains(trimmedLine, "=") {
			errors = append(errors, fmt.Sprintf("第%d行: 缺少等号分隔符", lineNum))
			continue
		}

		// 分割键值对
		parts := strings.SplitN(trimmedLine, "=", 2)
		if len(parts) != 2 {
			errors = append(errors, fmt.Sprintf("第%d行: 格式错误", lineNum))
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// 验证键名
		if key == "" {
			errors = append(errors, fmt.Sprintf("第%d行: 变量名不能为空", lineNum))
			continue
		}

		// 验证键名格式（只允许字母、数字、下划线）
		if !IsValidEnvKey(key) {
			errors = append(errors, fmt.Sprintf("第%d行: 变量名'%s'格式无效，只允许字母、数字和下划线", lineNum, key))
			continue
		}

		// 验证值的引号匹配
		if !IsValidEnvValue(value) {
			errors = append(errors, fmt.Sprintf("第%d行: 变量值'%s'的引号不匹配", lineNum, value))
		}
	}

	return errors
}

// isValidEnvKey 检查环境变量键名是否有效
func IsValidEnvKey(key string) bool {
	if key == "" {
		return false
	}

	// 第一个字符必须是字母或下划线
	firstChar := key[0]
	if !((firstChar >= 'A' && firstChar <= 'Z') ||
		(firstChar >= 'a' && firstChar <= 'z') ||
		firstChar == '_') {
		return false
	}

	// 其余字符必须是字母、数字或下划线
	for _, char := range key[1:] {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}

	return true
}

// isValidEnvValue 检查环境变量值的引号是否匹配
func IsValidEnvValue(value string) bool {
	if value == "" {
		return true
	}

	// 检查单引号
	if strings.HasPrefix(value, "'") {
		return strings.HasSuffix(value, "'") && len(value) >= 2
	}

	// 检查双引号
	if strings.HasPrefix(value, "\"") {
		return strings.HasSuffix(value, "\"") && len(value) >= 2
	}

	// 没有引号的值也是有效的
	return true
}
