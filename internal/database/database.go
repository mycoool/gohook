package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// DatabaseConfig database config
type DatabaseConfig struct {
	Type     string `yaml:"type"` // sqlite, mysql, postgres
	DSN      string `yaml:"dsn"`  // data source name
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// DefaultDatabaseConfig return default database config
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Type:     "sqlite",
		Database: "gohook.db",
	}
}

// createSQLiteDialector create SQLite dialect, automatically select available driver
func createSQLiteDialector(dsn string) gorm.Dialector {
	// first try standard SQLite driver, if failed, use pure Go driver
	dialector := sqlite.Open(dsn)

	// try to open connection with GORM
	testDB, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil && (err.Error() == "Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub" ||
		err.Error() == "CGO_ENABLED=0" ||
		err.Error() == "cgo not available") {
		//log.Printf("Standard SQLite driver (go-sqlite3) not available, using pure Go driver")

		// use pure Go SQLite driver
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
		// standard driver available
		//log.Printf("Using standard SQLite driver (go-sqlite3)")
		if testDB != nil {
			sqlDB, _ := testDB.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}
	}

	return dialector
}

// InitDatabase initialize database connection
func InitDatabase(config *DatabaseConfig) error {
	var dsn string
	var dialector gorm.Dialector

	switch config.Type {
	case "sqlite":
		if config.Database == "" {
			config.Database = "gohook.db"
		}

		// ensure database directory exists
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

	// set log level
	logLevel := logger.Error
	if os.Getenv("DB_DEBUG") == "true" {
		logLevel = logger.Info
	}

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// SQLite tuning:
	// - WAL reduces reader/writer blocking.
	// - busy_timeout makes transient lock contention wait instead of failing fast.
	// - single open connection avoids multi-conn write contention in a single-process server.
	if config.Type == "sqlite" {
		if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
			sqlDB.SetMaxOpenConns(1)
			sqlDB.SetMaxIdleConns(1)
			sqlDB.SetConnMaxLifetime(1 * time.Hour)
		}
		// Best-effort pragmas; ignore errors to stay compatible across sqlite drivers.
		_ = db.Exec("PRAGMA journal_mode=WAL").Error
		_ = db.Exec("PRAGMA synchronous=NORMAL").Error
		_ = db.Exec("PRAGMA foreign_keys=ON").Error
		_ = db.Exec("PRAGMA busy_timeout=5000").Error
	}

	DB = db
	log.Printf("Database connected successfully (type: %s)", config.Type)

	return nil
}

// GetDB get database instance
func GetDB() *gorm.DB {
	return DB
}

// AutoMigrate auto migrate database table structure
func AutoMigrate() error {
	if DB == nil {
		return fmt.Errorf("database not initialized")
	}

	// migrate all models
	err := DB.AutoMigrate(
		&HookLog{},
		&SystemLog{},
		&UserActivity{},
		&ProjectActivity{},
		&SyncNode{},
		&SyncTask{},
		&SyncFileChange{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}

// CloseDB close database connection
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
