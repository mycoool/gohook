package config

import (
	"fmt"
	"log"
	"os"

	"github.com/mycoool/gohook/internal/types"
	"gopkg.in/yaml.v2"
)

// loadAppConfig 加载应用程序配置文件
func LoadAppConfig() error {
	filePath := "app.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// 如果配置文件不存在，创建默认配置并保存到文件
		types.GoHookAppConfig = &types.AppConfig{
			Port:              9000,
			JWTSecret:         "gohook-secret-key-change-in-production",
			JWTExpiryDuration: 24,
		}
		// 自动保存默认配置到文件
		if saveErr := SaveAppConfig(); saveErr != nil {
			log.Printf("Warning: failed to save default app config: %v", saveErr)
		} else {
			log.Printf("Created default app.yaml configuration file")
		}
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取应用配置文件失败: %v", err)
	}

	config := &types.AppConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析应用配置文件失败: %v", err)
	}

	types.GoHookAppConfig = config
	return nil
}

// saveAppConfig 保存应用程序配置文件
func SaveAppConfig() error {
	if types.GoHookAppConfig == nil {
		return fmt.Errorf("应用配置为空")
	}

	data, err := yaml.Marshal(types.GoHookAppConfig)
	if err != nil {
		return fmt.Errorf("序列化应用配置失败: %v", err)
	}

	if err := os.WriteFile("app.yaml", data, 0644); err != nil {
		return fmt.Errorf("保存应用配置文件失败: %v", err)
	}

	return nil
}

// loadConfig 加载版本配置文件
func LoadConfig() error {
	data, err := os.ReadFile("version.yaml")
	if err != nil {
		return fmt.Errorf("读取版本配置文件失败: %v", err)
	}

	config := &types.Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("解析版本配置文件失败: %v", err)
	}

	types.ConfigData = config
	return nil
}

// saveConfig 保存版本配置文件
func SaveConfig() error {
	if types.ConfigData == nil {
		return fmt.Errorf("版本配置数据为空")
	}

	data, err := yaml.Marshal(types.ConfigData)
	if err != nil {
		return fmt.Errorf("序列化版本配置失败: %v", err)
	}

	// 备份原配置文件
	if _, err := os.Stat("version.yaml"); err == nil {
		if err := os.Rename("version.yaml", "version.yaml.bak"); err != nil {
			log.Printf("Warning: failed to backup version config file: %v", err)
		}
	}

	err = os.WriteFile("version.yaml", data, 0644)
	if err != nil {
		// 如果保存失败，恢复备份
		if _, backupErr := os.Stat("version.yaml.bak"); backupErr == nil {
			if restoreErr := os.Rename("version.yaml.bak", "version.yaml"); restoreErr != nil {
				log.Printf("Error: failed to restore backup version config file: %v", restoreErr)
			}
		}
		return fmt.Errorf("保存版本配置文件失败: %v", err)
	}

	return nil
}
