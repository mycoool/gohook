package env

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// env file handle

// get env file content, always use .env filename but support TOML content inside
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

// save env file, always save to .env but content can be TOML format
func SaveEnvFile(projectPath, content string) error {
	// check project directory exist
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project directory not exist: %s", projectPath)
	}

	// always use .env filename
	envFilePath := filepath.Join(projectPath, ".env")

	// normalize content before saving
	normalizedContent := normalizeEnvContent(content)

	// write file, if file not exist, it will be created
	err := os.WriteFile(envFilePath, []byte(normalizedContent), 0644)
	if err != nil {
		return fmt.Errorf("save env file failed: %v", err)
	}

	return nil
}

// delete env file (only .env)
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

// detect if env file content uses TOML format (inside .env file)
func detectTomlContentFormat(content string) bool {
	lines := strings.Split(content, "\n")

	// check for TOML-specific syntax
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// skip empty lines and comments
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// check for TOML section headers [section]
		if strings.HasPrefix(trimmedLine, "[") && strings.HasSuffix(trimmedLine, "]") {
			return true
		}

		// check for TOML multiline strings
		if strings.Contains(trimmedLine, `"""`) {
			return true
		}

		// check for TOML arrays
		if regexp.MustCompile(`^\w+\s*=\s*\[.*\]`).MatchString(trimmedLine) {
			return true
		}

		// check for TOML dotted keys
		if regexp.MustCompile(`^\w+\.\w+\s*=`).MatchString(trimmedLine) {
			return true
		}
	}

	// default to .env format
	return false
}

// normalize env content (handle spaces and empty lines properly)
func normalizeEnvContent(content string) string {
	lines := strings.Split(content, "\n")
	var normalizedLines []string

	for _, line := range lines {
		// preserve empty lines
		if strings.TrimSpace(line) == "" {
			normalizedLines = append(normalizedLines, "")
			continue
		}

		// preserve comment lines as-is (including leading spaces)
		if strings.TrimSpace(line)[:1] == "#" {
			normalizedLines = append(normalizedLines, line)
			continue
		}

		// for key=value lines, preserve leading spaces but normalize around =
		if strings.Contains(line, "=") {
			leadingSpaces := ""
			trimmed := strings.TrimLeft(line, " \t")
			if len(line) > len(trimmed) {
				leadingSpaces = line[:len(line)-len(trimmed)]
			}

			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimLeft(parts[1], " \t") // preserve trailing spaces in value
				normalizedLines = append(normalizedLines, fmt.Sprintf("%s%s=%s", leadingSpaces, key, value))
				continue
			}
		}

		// preserve other lines as-is
		normalizedLines = append(normalizedLines, line)
	}

	return strings.Join(normalizedLines, "\n")
}

// validate env file content (support both .env and TOML content formats)
func ValidateEnvContent(content string) []string {
	var errors []string
	lines := strings.Split(content, "\n")

	// detect if this content uses TOML format (inside .env file)
	isTomlContent := detectTomlContentFormat(content)

	var currentSection string

	for i, line := range lines {
		lineNum := i + 1
		trimmedLine := strings.TrimSpace(line)

		// skip empty line and comment line
		if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
			continue
		}

		// handle TOML content format
		if isTomlContent {
			if validateTomlLine(trimmedLine, lineNum, &currentSection, &errors) {
				continue
			}
		}

		// handle standard .env format
		if validateEnvLine(trimmedLine, lineNum, &errors) {
			continue
		}
	}

	return errors
}

// validate TOML line
func validateTomlLine(line string, lineNum int, currentSection *string, errors *[]string) bool {
	// TOML section header [section]
	if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
		section := strings.Trim(line, "[]")
		if !IsValidTomlSection(section) {
			*errors = append(*errors, fmt.Sprintf("line %d: invalid TOML section '%s'", lineNum, section))
		}
		*currentSection = section
		return true
	}

	// TOML key-value pair
	if strings.Contains(line, "=") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			*errors = append(*errors, fmt.Sprintf("line %d: format error", lineNum))
			return true
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// validate key
		if key == "" {
			*errors = append(*errors, fmt.Sprintf("line %d: key can not be empty", lineNum))
			return true
		}

		// validate TOML key format (can contain dots)
		if !IsValidTomlKey(key) {
			*errors = append(*errors, fmt.Sprintf("line %d: key '%s' format invalid", lineNum, key))
			return true
		}

		// validate TOML value
		if !IsValidTomlValue(value) {
			*errors = append(*errors, fmt.Sprintf("line %d: value '%s' format invalid", lineNum, value))
		}
		return true
	}

	return false
}

// validate standard .env line
func validateEnvLine(line string, lineNum int, errors *[]string) bool {
	// check if contains equal sign
	if !strings.Contains(line, "=") {
		*errors = append(*errors, fmt.Sprintf("line %d: missing equal sign", lineNum))
		return true
	}

	// split key-value pair
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		*errors = append(*errors, fmt.Sprintf("line %d: format error", lineNum))
		return true
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// validate key
	if key == "" {
		*errors = append(*errors, fmt.Sprintf("line %d: key can not be empty", lineNum))
		return true
	}

	// validate key format (only allow letters, numbers, and underscores)
	if !IsValidEnvKey(key) {
		*errors = append(*errors, fmt.Sprintf("line %d: key '%s' format invalid, only allow letters, numbers and underscores", lineNum, key))
		return true
	}

	// validate value quote match
	if !IsValidEnvValue(value) {
		*errors = append(*errors, fmt.Sprintf("line %d: value '%s' quote not match", lineNum, value))
	}
	return true
}

// validate standard env key
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

// validate TOML key (more flexible, allows dots)
func IsValidTomlKey(key string) bool {
	if key == "" {
		return false
	}

	// TOML keys can contain letters, numbers, underscores, and dots
	for _, char := range key {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '.') {
			return false
		}
	}

	// must not start or end with dot
	return !strings.HasPrefix(key, ".") && !strings.HasSuffix(key, ".")
}

// validate TOML section name
func IsValidTomlSection(section string) bool {
	if section == "" {
		return false
	}

	// section can contain letters, numbers, underscores, dots, and hyphens
	for _, char := range section {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '.' || char == '-') {
			return false
		}
	}

	return true
}

// validate standard env value
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

// validate TOML value (more complex validation)
func IsValidTomlValue(value string) bool {
	if value == "" {
		return true
	}

	// string values (quoted)
	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return len(value) >= 2
	}
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return len(value) >= 2
	}

	// multiline strings
	if strings.HasPrefix(value, `"""`) && strings.HasSuffix(value, `"""`) {
		return len(value) >= 6
	}

	// boolean values
	if value == "true" || value == "false" {
		return true
	}

	// integer values
	if _, err := strconv.Atoi(value); err == nil {
		return true
	}

	// float values
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return true
	}

	// arrays (simple check)
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return true
	}

	// unquoted strings (should be quoted in TOML but we'll allow for flexibility)
	return !strings.ContainsAny(value, " \t#[]{}\"'")
}
