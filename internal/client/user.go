package client

import (
	"fmt"
	"os"
	"strings"

	"github.com/mycoool/gohook/internal/types"
	"gopkg.in/yaml.v2"
)

// loadUsersConfig 加载用户配置文件
func LoadUsersConfig() error {
	filePath := "user.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("用户配置文件 %s 不存在", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取用户配置文件失败: %v", err)
	}

	config := &types.UsersConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析用户配置文件失败: %v", err)
	}

	types.GoHookUsersConfig = config
	return nil
}

// saveUsersConfig 保存用户配置文件
func SaveUsersConfig() error {
	if types.GoHookUsersConfig == nil {
		return fmt.Errorf("用户配置为空")
	}

	// 创建带注释的YAML内容
	var yamlContent strings.Builder
	yamlContent.WriteString("# GoHook 用户配置文件\n")
	yamlContent.WriteString("# 用户账户信息\n")
	yamlContent.WriteString("users:\n")

	for _, user := range types.GoHookUsersConfig.Users {
		yamlContent.WriteString(fmt.Sprintf("  - username: %s\n", user.Username))
		yamlContent.WriteString(fmt.Sprintf("    password: %s\n", user.Password))
		yamlContent.WriteString(fmt.Sprintf("    role: %s\n", user.Role))

		// 如果是默认admin用户且密码是哈希值，添加原始密码注释
		if user.Username == "admin" && strings.HasPrefix(user.Password, "$2a$") {
			// 检查是否是新创建的默认用户（通过检查是否只有一个用户来判断）
			if len(types.GoHookUsersConfig.Users) == 1 {
				yamlContent.WriteString("    # 默认密码: admin123 (请及时修改)\n")
			}
		}
	}

	if err := os.WriteFile("user.yaml", []byte(yamlContent.String()), 0644); err != nil {
		return fmt.Errorf("保存用户配置文件失败: %v", err)
	}

	return nil
}
