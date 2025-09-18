package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// SystemConfig system config struct
type SystemConfig struct {
	JWTSecret         string `yaml:"jwt_secret" json:"jwt_secret"`
	JWTExpiryDuration int    `yaml:"jwt_expiry_duration" json:"jwt_expiry_duration"`
	Mode              string `yaml:"mode" json:"mode"`
}

const configFilePath = "app.yaml"

// LoadSystemConfig load system config
func LoadSystemConfig() (*SystemConfig, error) {
	// if file not exist, return default config
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		return &SystemConfig{
			JWTSecret:         "gohook-secret-key-change-in-production",
			JWTExpiryDuration: 1440, // 1440 minutes = 24 hours
			Mode:              "dev",
		}, nil
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config SystemConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// set default value
	if config.JWTSecret == "" {
		config.JWTSecret = "gohook-secret-key-change-in-production"
	}
	if config.JWTExpiryDuration <= 0 {
		config.JWTExpiryDuration = 1440 // 1440 minutes = 24 hours
	}
	if config.Mode == "" {
		config.Mode = "dev"
	}

	return &config, nil
}

// SaveSystemConfig save system config
func SaveSystemConfig(config *SystemConfig) error {
	// validate config
	if config.JWTSecret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}
	if config.JWTExpiryDuration <= 0 || config.JWTExpiryDuration > 525600 {
		return fmt.Errorf("JWT expiry duration must be between 1 and 525600 minutes")
	}
	if config.Mode != "dev" && config.Mode != "test" && config.Mode != "prod" {
		return fmt.Errorf("mode must be one of: dev, test, prod")
	}

	// read existing complete config file
	var existingConfig map[string]interface{}
	if _, err := os.Stat(configFilePath); err == nil {
		data, err := os.ReadFile(configFilePath)
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

	// only update system config related fields
	existingConfig["jwt_secret"] = config.JWTSecret
	existingConfig["jwt_expiry_duration"] = config.JWTExpiryDuration
	existingConfig["mode"] = config.Mode

	// ensure port field exists and is valid
	if _, exists := existingConfig["port"]; !exists {
		existingConfig["port"] = 9000
	}

	data, err := yaml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	// backup original file
	if _, err := os.Stat(configFilePath); err == nil {
		backupPath := configFilePath + ".backup"
		if err := copyFile(configFilePath, backupPath); err != nil {
			return fmt.Errorf("failed to backup config file: %v", err)
		}
	}

	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// copyFile copy file
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
