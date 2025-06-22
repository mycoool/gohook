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

// createSQLiteDialector 创建SQLite方言器，自动选择可用的驱动
func createSQLiteDialector(dsn string) gorm.Dialector {
	// 首先尝试标准SQLite驱动，如果失败则使用纯Go驱动
	dialector := sqlite.Open(dsn)

	// 尝试用GORM打开连接测试
	testDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil && (err.Error() == "Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub" ||
		err.Error() == "CGO_ENABLED=0" ||
		err.Error() == "cgo not available") {
		log.Printf("Standard SQLite driver (go-sqlite3) not available, using pure Go driver")

		// 使用纯Go SQLite驱动
		dialector = sqlite.Dialector{
			DriverName: "sqlite",
			DSN:        dsn,
		}
	} else if err != nil {
		log.Printf("SQLite driver test failed: %v, trying pure Go driver", err)
		dialector = sqlite.Dialector{
			DriverName: "sqlite",
			DSN:        dsn,
		}
	} else {
		// 标准驱动可用
		log.Printf("Using standard SQLite driver (go-sqlite3)")
		if testDB != nil {
			sqlDB, _ := testDB.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}
	}

	return dialector
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
		dialector = createSQLiteDialector(dsn)

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
