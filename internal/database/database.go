package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string `yaml:"type"` // sqlite, mysql, postgres
	DSN      string `yaml:"dsn"`  // 数据源名称
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// DefaultDatabaseConfig 返回默认数据库配置
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Type:     "sqlite",
		Database: "gohook.db",
	}
}

// InitDatabase 初始化数据库连接
func InitDatabase(config *DatabaseConfig) error {
	var dsn string
	var dialector gorm.Dialector

	switch config.Type {
	case "sqlite":
		if config.Database == "" {
			config.Database = "gohook.db"
		}

		// 确保数据库目录存在
		dbDir := filepath.Dir(config.Database)
		if dbDir != "." && dbDir != "" {
			if err := os.MkdirAll(dbDir, 0755); err != nil {
				return fmt.Errorf("failed to create database directory: %v", err)
			}
		}

		dsn = config.Database
		dialector = sqlite.Open(dsn)
	default:
		return fmt.Errorf("unsupported database type: %s", config.Type)
	}

	// 配置日志级别
	logLevel := logger.Error
	if os.Getenv("DB_DEBUG") == "true" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	DB = db
	log.Printf("Database connected successfully (type: %s)", config.Type)

	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// 迁移所有模型
	err := DB.AutoMigrate(
		&HookLog{},
		&SystemLog{},
		&UserActivity{},
		&ProjectActivity{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}
