package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// SystemConfig 系统配置结构
type SystemConfig struct {
	JWTSecret         string `yaml:"jwt_secret" json:"jwt_secret"`
	JWTExpiryDuration int    `yaml:"jwt_expiry_duration" json:"jwt_expiry_duration"`
	Mode              string `yaml:"mode" json:"mode"`
}

const configFilePath = "app.yaml"

// LoadSystemConfig 加载系统配置
func LoadSystemConfig() (*SystemConfig, error) {
	// 如果文件不存在，返回默认配置
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return &SystemConfig{
			JWTSecret:         "gohook-secret-key-change-in-production",
			JWTExpiryDuration: 24,
			Mode:              "dev",
		}, nil
	}

	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config SystemConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// 设置默认值
	if config.JWTSecret == "" {
		config.JWTSecret = "gohook-secret-key-change-in-production"
	}
	if config.JWTExpiryDuration <= 0 {
		config.JWTExpiryDuration = 24
	}
	if config.Mode == "" {
		config.Mode = "dev"
	}

	return &config, nil
}

// SaveSystemConfig 保存系统配置
func SaveSystemConfig(config *SystemConfig) error {
	// 验证配置
	if config.JWTSecret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}
	if config.JWTExpiryDuration <= 0 || config.JWTExpiryDuration > 8760 {
		return fmt.Errorf("JWT expiry duration must be between 1 and 8760 hours")
	}
	if config.Mode != "dev" && config.Mode != "test" && config.Mode != "prod" {
		return fmt.Errorf("mode must be one of: dev, test, prod")
	}

	// 读取现有的完整配置文件
	var existingConfig map[string]interface{}
	if _, err := os.Stat(configFilePath); err == nil {
		data, err := ioutil.ReadFile(configFilePath)
		if err != nil {
			return fmt.Errorf("failed to read existing config file: %v", err)
		}

		err = yaml.Unmarshal(data, &existingConfig)
		if err != nil {
			return fmt.Errorf("failed to unmarshal existing config: %v", err)
		}
	} else {
		existingConfig = make(map[string]interface{})
	}

	// 只更新系统配置相关的字段
	existingConfig["jwt_secret"] = config.JWTSecret
	existingConfig["jwt_expiry_duration"] = config.JWTExpiryDuration
	existingConfig["mode"] = config.Mode

	// 确保port字段存在且有效
	if _, exists := existingConfig["port"]; !exists {
		existingConfig["port"] = 9000
	}

	data, err := yaml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// 备份原文件
	if _, err := os.Stat(configFilePath); err == nil {
		backupPath := configFilePath + ".backup"
		if err := copyFile(configFilePath, backupPath); err != nil {
			return fmt.Errorf("failed to backup config file: %v", err)
		}
	}

	err = ioutil.WriteFile(configFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, 0644)
}
