package config

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/types"
	"gopkg.in/yaml.v2"
)

// load app config file
func LoadAppConfig() error {
	filePath := "app.yaml"
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// if config file not exist, create default config and save to file
		types.GoHookAppConfig = &types.AppConfig{
			Port:              9000,
			JWTSecret:         "gohook-secret-key-change-in-production",
			JWTExpiryDuration: 1440, // default 24 hours, unit: minutes
			Mode:              "test",
			PanelAlias:        "GoHook", // 默认面板别名
			Database: types.DatabaseConfig{
				Type:             "sqlite",
				Database:         "gohook.db",
				LogRetentionDays: 30,
			},
		}
		// auto save default config to file
		if saveErr := SaveAppConfig(); saveErr != nil {
			log.Printf("Warning: failed to save default app config: %v", saveErr)
		} else {
			log.Printf("Created default app.yaml configuration file")
		}
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read app config file failed: %v", err)
	}

	config := &types.AppConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("parse app config file failed: %v", err)
	}

	types.GoHookAppConfig = config
	return nil
}

// save app config file
func SaveAppConfig() error {
	if types.GoHookAppConfig == nil {
		return fmt.Errorf("app config is nil")
	}

	data, err := yaml.Marshal(types.GoHookAppConfig)
	if err != nil {
		return fmt.Errorf("serialize app config failed: %v", err)
	}

	if err := os.WriteFile("app.yaml", data, 0644); err != nil {
		return fmt.Errorf("save app config file failed: %v", err)
	}

	return nil
}

// load version config file
func LoadVersionConfig() error {
	data, err := os.ReadFile("version.yaml")
	if err != nil {
		return fmt.Errorf("read version config file failed: %v", err)
	}

	config := &types.VersionConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("parse version config file failed: %v", err)
	}

	types.GoHookVersionData = config
	return nil
}

// save version config file
func SaveVersionConfig() error {
	if types.GoHookVersionData == nil {
		return fmt.Errorf("version config data is nil")
	}

	data, err := yaml.Marshal(types.GoHookVersionData)
	if err != nil {
		return fmt.Errorf("serialize version config failed: %v", err)
	}

	// backup original config file
	if _, err := os.Stat("version.yaml"); err == nil {
		if err := os.Rename("version.yaml", "version.yaml.bak"); err != nil {
			log.Printf("Warning: failed to backup version config file: %v", err)
		}
	}

	err = os.WriteFile("version.yaml", data, 0644)
	if err != nil {
		// if save failed, restore backup
		if _, backupErr := os.Stat("version.yaml.bak"); backupErr == nil {
			if restoreErr := os.Rename("version.yaml.bak", "version.yaml"); restoreErr != nil {
				log.Printf("Error: failed to restore backup version config file: %v", restoreErr)
			}
		}
		return fmt.Errorf("save version config file failed: %v", err)
	}

	return nil
}

// GetAppConfig get application configuration
func GetAppConfig() *types.AppConfig {
	return types.GoHookAppConfig
}

// GetUsersConfig get users configuration
func GetUsersConfig() *types.UsersConfig {
	return types.GoHookUsersConfig
}

// GetConfiguredPort get configured port
func GetConfiguredPort() int {
	if types.GoHookAppConfig != nil {
		return types.GoHookAppConfig.Port
	}
	return 9000 // default port
}

func HandleGetAppConfig(c *gin.Context) {
	if types.GoHookAppConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "App config not loaded"})
		return
	}

	// only return safe config fields, not including secrets
	c.JSON(http.StatusOK, gin.H{
		"mode":        types.GoHookAppConfig.Mode,
		"panel_alias": types.GoHookAppConfig.PanelAlias,
	})
}
