package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// evn file handle

// get env file content
func GetEnvFile(projectPath string) (string, bool, error) {
	envFilePath := filepath.Join(projectPath, ".env")

	// check file exist
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return "", false, nil
	}

	// read file content
	content, err := os.ReadFile(envFilePath)
	if err != nil {
		return "", true, fmt.Errorf("read env file failed: %v", err)
	}

	return string(content), true, nil
}

// save env file
func SaveEnvFile(projectPath, content string) error {
	envFilePath := filepath.Join(projectPath, ".env")

	// check project directory exist
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project directory not exist: %s", projectPath)
	}

	// write file, if file not exist, it will be created
	err := os.WriteFile(envFilePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("save env file failed: %v", err)
	}

	return nil
}

// delete env file
func DeleteEnvFile(projectPath string) error {
	envFilePath := filepath.Join(projectPath, ".env")

	// check file exist
	if _, err := os.Stat(envFilePath); os.IsNotExist(err) {
		return fmt.Errorf("env file not exist")
	}

	// delete file
	err := os.Remove(envFilePath)
	if err != nil {
		return fmt.Errorf("delete env file failed: %v", err)
	}

	return nil
}

// validate env file content
func ValidateEnvContent(content string) []string {
	var errors []string
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)

		// skip empty line and comment line
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// check if contains equal sign
		if !strings.Contains(trimmedLine, "=") {
			errors = append(errors, fmt.Sprintf("line %d: missing equal sign", lineNum))
			continue
		}

		// split key-value pair
		parts := strings.SplitN(trimmedLine, "=", 2)
		if len(parts) != 2 {
			errors = append(errors, fmt.Sprintf("line %d: format error", lineNum))
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// validate key
		if key == "" {
			errors = append(errors, fmt.Sprintf("line %d: key can not be empty", lineNum))
			continue
		}

		// validate key format (only allow letters, numbers, and underscores)
		if !IsValidEnvKey(key) {
			errors = append(errors, fmt.Sprintf("line %d: key '%s' format invalid, only allow letters, numbers and underscores", lineNum, key))
			continue
		}

		// validate value quote match
		if !IsValidEnvValue(value) {
			errors = append(errors, fmt.Sprintf("line %d: value '%s' quote not match", lineNum, value))
		}
	}

	return errors
}

// validate env key
func IsValidEnvKey(key string) bool {
	if key == "" {
		return false
	}

	// first char must be letter or underscore
	firstChar := key[0]
	if !((firstChar >= 'A' && firstChar <= 'Z') ||
		(firstChar >= 'a' && firstChar <= 'z') ||
		firstChar == '_') {
		return false
	}

	// other chars must be letters, numbers or underscores
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

// validate env value
func IsValidEnvValue(value string) bool {
	if value == "" {
		return true
	}

	// check single quote
	if strings.HasPrefix(value, "'") {
		return strings.HasSuffix(value, "'") && len(value) >= 2
	}

	// check double quote
	if strings.HasPrefix(value, "\"") {
		return strings.HasSuffix(value, "\"") && len(value) >= 2
	}

	// value without quote is also valid
	return true
}
