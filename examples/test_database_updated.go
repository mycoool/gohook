package main

import (
	"log"
	"time"

	"github.com/mycoool/gohook/internal/database"
)

func main() {
	log.Println("Testing updated GoHook database integration...")

	// Initialize database with default SQLite config
	config := database.DefaultDatabaseConfig()
	config.Database = "test_gohook_updated.db"

	err := database.InitDatabase(config)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Perform migration
	err = database.AutoMigrate()
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("Database initialized successfully!")

	// Initialize log service
	database.InitLogService()
	log.Println("Log service initialized!")

	// Test creating some sample logs
	log.Println("Creating sample logs...")

	// Test Webhook log
	database.LogHookExecution(
		"test-webhook-1",
		"Test Webhook",
		database.HookTypeWebhook,
		"POST",
		"127.0.0.1:12345",
		map[string][]string{"Content-Type": {"application/json"}},
		`{"test": "data"}`,
		true,
		"Webhook executed successfully",
		"",
		150,
		"curl/7.68.0",
		map[string][]string{"param": {"value"}},
	)

	// Test GitHook log
	database.LogHookExecution(
		"test-githook-1",
		"Test GitHook",
		database.HookTypeGitHook,
		"POST",
		"127.0.0.1:12346",
		map[string][]string{"Content-Type": {"application/json"}},
		`{"ref": "refs/heads/main"}`,
		true,
		"GitHook executed successfully",
		"",
		80,
		"git/2.39.0",
		map[string][]string{"repository": {"test-repo"}},
	)

	// Test System log
	database.LogSystemEvent(
		database.LogLevelInfo,
		database.LogCategorySystem,
		"Application started",
		map[string]interface{}{"version": "1.0.0"},
		"",
		"127.0.0.1",
		"GoHook/1.0.0",
	)

	// Test User activity
	database.LogUserAction(
		"admin",
		database.UserActionLogin,
		"/login",
		"User logged in successfully",
		"127.0.0.1",
		"Mozilla/5.0",
		true,
		map[string]interface{}{"login_time": time.Now()},
	)

	// Test Project activity
	database.LogProjectAction(
		"test-project",
		database.ProjectActionBranchSwitch,
		"main",
		"develop",
		"admin",
		true,
		"",
		"abc123def456",
		"Switched from main to develop branch",
		"127.0.0.1",
	)

	log.Println("Sample logs created successfully!")

	// Test querying logs
	logService := database.NewLogService()

	// Query Webhook logs
	webhookLogs, total, err := logService.GetHookLogs(1, 10, "", "", database.HookTypeWebhook, nil, nil, nil)
	if err != nil {
		log.Printf("Error querying webhook logs: %v", err)
	} else {
		log.Printf("Found %d webhook logs (total: %d)", len(webhookLogs), total)
		for i, hookLog := range webhookLogs {
			log.Printf("  %d. Webhook: %s, Method: %s, Success: %v, Duration: %dms",
				i+1, hookLog.HookName, hookLog.Method, hookLog.Success, hookLog.Duration)
		}
	}

	// Query GitHook logs
	githookLogs, total, err := logService.GetHookLogs(1, 10, "", "", database.HookTypeGitHook, nil, nil, nil)
	if err != nil {
		log.Printf("Error querying githook logs: %v", err)
	} else {
		log.Printf("Found %d githook logs (total: %d)", len(githookLogs), total)
		for i, hookLog := range githookLogs {
			log.Printf("  %d. GitHook: %s, Method: %s, Success: %v, Duration: %dms",
				i+1, hookLog.HookName, hookLog.Method, hookLog.Success, hookLog.Duration)
		}
	}

	// Query System logs
	systemLogs, total, err := logService.GetSystemLogs(1, 10, "", "", "", nil, nil)
	if err != nil {
		log.Printf("Error querying system logs: %v", err)
	} else {
		log.Printf("Found %d system logs (total: %d)", len(systemLogs), total)
		for i, sysLog := range systemLogs {
			log.Printf("  %d. Level: %s, Category: %s, Message: %s",
				i+1, sysLog.Level, sysLog.Category, sysLog.Message)
		}
	}

	// Query User activities
	userActivities, total, err := logService.GetUserActivities(1, 10, "", "", nil, nil, nil)
	if err != nil {
		log.Printf("Error querying user activities: %v", err)
	} else {
		log.Printf("Found %d user activities (total: %d)", len(userActivities), total)
		for i, activity := range userActivities {
			log.Printf("  %d. User: %s, Action: %s, Success: %v",
				i+1, activity.Username, activity.Action, activity.Success)
		}
	}

	// Query Project activities
	projectActivities, total, err := logService.GetProjectActivities(1, 10, "", "", "", nil, nil, nil)
	if err != nil {
		log.Printf("Error querying project activities: %v", err)
	} else {
		log.Printf("Found %d project activities (total: %d)", len(projectActivities), total)
		for i, activity := range projectActivities {
			log.Printf("  %d. Project: %s, Action: %s, %s -> %s, User: %s",
				i+1, activity.ProjectName, activity.Action, activity.OldValue, activity.NewValue, activity.Username)
		}
	}

	// Get Webhook statistics
	webhookStats, err := logService.GetHookLogStats(database.HookTypeWebhook, nil, nil)
	if err != nil {
		log.Printf("Error getting webhook stats: %v", err)
	} else {
		log.Printf("Webhook statistics: %+v", webhookStats)
	}

	// Get GitHook statistics
	githookStats, err := logService.GetHookLogStats(database.HookTypeGitHook, nil, nil)
	if err != nil {
		log.Printf("Error getting githook stats: %v", err)
	} else {
		log.Printf("GitHook statistics: %+v", githookStats)
	}

	// Get User activity statistics
	userStats, err := logService.GetUserActivityStats("", nil, nil)
	if err != nil {
		log.Printf("Error getting user activity stats: %v", err)
	} else {
		log.Printf("User activity statistics: %+v", userStats)
	}

	log.Println("Database test completed successfully!")
	log.Println("You can now run the main application and use the updated log APIs:")
	log.Println("  GET /logs/webhooks")
	log.Println("  GET /logs/webhooks/stats")
	log.Println("  GET /logs/githook")
	log.Println("  GET /logs/githook/stats")
	log.Println("  GET /logs/users")
	log.Println("  GET /logs/users/stats")
	log.Println("  GET /logs/system")
	log.Println("  GET /logs/projects")
	log.Println("  DELETE /logs/cleanup")
}
