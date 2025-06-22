package main

import (
	"encoding/json"
	"log"

	"github.com/mycoool/gohook/internal/config"
	"github.com/mycoool/gohook/internal/database"
)

func main() {
	log.Println("=== GoHook 日志记录功能集成验证 ===")

	// 1. 初始化配置
	log.Println("1. 初始化配置...")
	if err := config.LoadAppConfig(); err != nil {
		log.Printf("Warning: failed to load config: %v", err)
	}

	// 2. 初始化数据库
	log.Println("2. 初始化数据库...")
	dbConfig := &database.DatabaseConfig{
		Type:     "sqlite",
		Database: "test_gohook.db",
	}
	if err := database.InitDatabase(dbConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 3. 自动迁移数据库表
	if err := database.AutoMigrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 4. 测试各模块日志记录功能
	testModuleLogs()

	// 5. 验证日志查询功能
	verifyLogQueries()

	// 6. 测试统计功能
	testStatistics()

	log.Println("=== 集成验证完成！===")
}

func testModuleLogs() {
	log.Println("3. 测试各模块日志记录...")

	// Webhook执行日志
	database.LogHookExecution(
		"test-webhook",
		"Test Webhook",
		"webhook",
		"POST",
		"192.168.1.100",
		map[string][]string{
			"Content-Type":   {"application/json"},
			"X-GitHub-Event": {"push"},
		},
		`{"ref":"refs/heads/main","after":"abc123"}`,
		true,
		"Webhook executed successfully",
		"",
		1250,
		"GitHub-Hookshot/1.0",
		map[string][]string{
			"trigger": {"auto"},
		},
	)

	// GitHook执行日志
	database.LogHookExecution(
		"project1",
		"GitHook-project1",
		"githook",
		"POST",
		"192.168.1.101",
		map[string][]string{
			"Content-Type":   {"application/json"},
			"X-GitHub-Event": {"push"},
		},
		`{"ref":"refs/heads/develop","after":"def456"}`,
		true,
		"Action: switch-branch, Target: develop",
		"",
		2300,
		"GitHub-Hookshot/1.0",
		map[string][]string{
			"project": {"project1"},
			"mode":    {"branch"},
		},
	)

	// 用户登录日志
	database.LogUserAction(
		"admin",
		database.UserActionLogin,
		"/client",
		"User logged in successfully",
		"192.168.1.200",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		true,
		map[string]interface{}{
			"client_name": "Chrome Browser",
			"role":        "admin",
		},
	)

	// 项目分支切换日志
	database.LogProjectAction(
		"project1",
		database.ProjectActionBranchSwitch,
		"main",
		"develop",
		"admin",
		true,
		"",
		"abc123",
		"Branch switched from main to develop successfully",
		"192.168.1.200",
	)

	// 系统事件日志
	database.LogSystemEvent(
		database.LogLevelInfo,
		database.LogCategorySystem,
		"Application started successfully",
		map[string]interface{}{
			"version": "1.0.0",
			"port":    "9000",
		},
		"system",
		"127.0.0.1",
		"GoHook/1.0",
	)

	log.Println("   ✓ 所有模块日志记录完成")
}

func verifyLogQueries() {
	log.Println("4. 验证日志查询功能...")
	logService := database.NewLogService()

	// 验证Hook日志
	hookLogs, hookTotal, err := logService.GetHookLogs(1, 10, "", "", "", nil, nil, nil)
	if err != nil {
		log.Printf("   ✗ Hook日志查询失败: %v", err)
	} else {
		log.Printf("   ✓ Hook日志查询成功，总数: %d", hookTotal)
		if len(hookLogs) > 0 {
			log.Printf("     示例: %s [%s] %s: %v",
				hookLogs[0].HookID, hookLogs[0].HookType, hookLogs[0].Method, hookLogs[0].Success)
		}
	}

	// 验证用户活动日志
	userLogs, userTotal, err := logService.GetUserActivities(1, 10, "", "", nil, nil, nil)
	if err != nil {
		log.Printf("   ✗ 用户活动日志查询失败: %v", err)
	} else {
		log.Printf("   ✓ 用户活动日志查询成功，总数: %d", userTotal)
		if len(userLogs) > 0 {
			log.Printf("     示例: %s [%s] %s: %v",
				userLogs[0].Username, userLogs[0].Action, userLogs[0].Resource, userLogs[0].Success)
		}
	}

	// 验证项目活动日志
	projectLogs, projectTotal, err := logService.GetProjectActivities(1, 10, "", "", "", nil, nil, nil)
	if err != nil {
		log.Printf("   ✗ 项目活动日志查询失败: %v", err)
	} else {
		log.Printf("   ✓ 项目活动日志查询成功，总数: %d", projectTotal)
		if len(projectLogs) > 0 {
			log.Printf("     示例: %s [%s] %s -> %s: %v",
				projectLogs[0].ProjectName, projectLogs[0].Action,
				projectLogs[0].OldValue, projectLogs[0].NewValue, projectLogs[0].Success)
		}
	}

	// 验证系统日志
	systemLogs, systemTotal, err := logService.GetSystemLogs(1, 10, "", "", "", nil, nil)
	if err != nil {
		log.Printf("   ✗ 系统日志查询失败: %v", err)
	} else {
		log.Printf("   ✓ 系统日志查询成功，总数: %d", systemTotal)
		if len(systemLogs) > 0 {
			log.Printf("     示例: [%s] %s: %s",
				systemLogs[0].Level, systemLogs[0].Category, systemLogs[0].Message)
		}
	}
}

func testStatistics() {
	log.Println("5. 测试统计功能...")
	logService := database.NewLogService()

	// Webhook统计
	webhookStats, err := logService.GetHookLogStats("webhook", nil, nil)
	if err != nil {
		log.Printf("   ✗ Webhook统计查询失败: %v", err)
	} else {
		statsJSON, _ := json.MarshalIndent(webhookStats, "     ", "  ")
		log.Printf("   ✓ Webhook统计:")
		log.Printf("     %s", statsJSON)
	}

	// GitHook统计
	githookStats, err := logService.GetHookLogStats("githook", nil, nil)
	if err != nil {
		log.Printf("   ✗ GitHook统计查询失败: %v", err)
	} else {
		statsJSON, _ := json.MarshalIndent(githookStats, "     ", "  ")
		log.Printf("   ✓ GitHook统计:")
		log.Printf("     %s", statsJSON)
	}

	// 用户活动统计
	userStats, err := logService.GetUserActivityStats("admin", nil, nil)
	if err != nil {
		log.Printf("   ✗ 用户活动统计查询失败: %v", err)
	} else {
		statsJSON, _ := json.MarshalIndent(userStats, "     ", "  ")
		log.Printf("   ✓ Admin用户活动统计:")
		log.Printf("     %s", statsJSON)
	}
}
