package hook

import (
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/hook"
	"github.com/mycoool/gohook/internal/types"
	wsmanager "github.com/mycoool/gohook/websocket"
)

// 声明外部变量引用
var LoadedHooksFromFiles *map[string]hook.Hooks
var HookManager *hook.HookManager

// SetHookReferences 设置Hook相关引用
func SetHookReferences(loadedHooks *map[string]hook.Hooks, hookManager *hook.HookManager) {
	LoadedHooksFromFiles = loadedHooks
	HookManager = hookManager
}

// getHookByID 根据ID获取Hook
func getHookByID(id string) *types.HookResponse {
	if LoadedHooksFromFiles == nil {
		return nil
	}

	for _, hooksInFile := range *LoadedHooksFromFiles {
		if hook := hooksInFile.Match(id); hook != nil {
			hookResponse := convertHookToResponse(hook)
			return &hookResponse
		}
	}

	return nil
}

// convertHookToResponse 将Hook转换为HookResponse
func convertHookToResponse(h *hook.Hook) types.HookResponse {
	description := fmt.Sprintf("Execute: %s", h.ExecuteCommand)
	if h.ResponseMessage != "" {
		description = h.ResponseMessage
	}

	// 解析触发规则为可读描述
	triggerDesc := "Any request"
	if h.TriggerRule != nil {
		triggerDesc = describeTriggerRule(h.TriggerRule)
	}

	// 设置HTTP方法
	httpMethods := h.HTTPMethods
	if len(httpMethods) == 0 {
		httpMethods = []string{"POST", "GET"} // 默认方法
	}

	return types.HookResponse{
		ID:                     h.ID,
		Name:                   h.ID, // 使用ID作为名称
		Description:            description,
		ExecuteCommand:         h.ExecuteCommand,
		WorkingDirectory:       h.CommandWorkingDirectory,
		ResponseMessage:        h.ResponseMessage,
		HTTPMethods:            httpMethods,
		TriggerRuleDescription: triggerDesc,
		LastUsed:               nil, // TODO: 可以添加实际的使用时间跟踪
		Status:                 "active",
	}
}

// describeTriggerRule 生成触发规则的可读描述
func describeTriggerRule(rules *hook.Rules) string {
	if rules == nil {
		return "No rules"
	}

	if rules.Match != nil {
		return fmt.Sprintf("Match %s: %s", rules.Match.Type, rules.Match.Value)
	}

	if rules.And != nil {
		return fmt.Sprintf("Multiple conditions required (%d rules)", len(*rules.And))
	}

	if rules.Or != nil {
		return fmt.Sprintf("Any condition satisfied (%d rules)", len(*rules.Or))
	}

	if rules.Not != nil {
		return "Negated condition"
	}

	return "Complex rules"
}

// GetHooksHandler 获取所有hooks
func GetHooksHandler(c *gin.Context) {
	if LoadedHooksFromFiles == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hooks未加载"})
		return
	}

	var hooks []types.HookResponse
	for _, hooksInFile := range *LoadedHooksFromFiles {
		for _, h := range hooksInFile {
			hookResponse := convertHookToResponse(&h)
			hooks = append(hooks, hookResponse)
		}
	}

	c.JSON(http.StatusOK, hooks)
}

// GetHookHandler 获取单个Hook详情
func GetHookHandler(c *gin.Context) {
	hookID := c.Param("id")
	hookResponse := getHookByID(hookID)
	if hookResponse == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}
	c.JSON(http.StatusOK, hookResponse)
}

// TriggerHookHandler 触发Hook（测试接口）
func TriggerHookHandler(c *gin.Context) {
	hookID := c.Param("id")
	hookResponse := getHookByID(hookID)
	if hookResponse == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	// 执行Hook命令
	success := false
	output := ""
	errorMsg := ""

	if hookResponse.ExecuteCommand != "" {
		// 执行命令
		var cmd *exec.Cmd
		if hookResponse.WorkingDirectory != "" {
			cmd = exec.Command("bash", "-c", hookResponse.ExecuteCommand)
			cmd.Dir = hookResponse.WorkingDirectory
		} else {
			cmd = exec.Command("bash", "-c", hookResponse.ExecuteCommand)
		}

		result, err := cmd.CombinedOutput()
		output = string(result)
		if err != nil {
			errorMsg = err.Error()
		} else {
			success = true
		}
	} else {
		success = true
		output = "Hook触发成功（无执行命令）"
	}

	// 推送WebSocket消息
	wsMessage := wsmanager.Message{
		Type:      "hook_triggered",
		Timestamp: time.Now(),
		Data: wsmanager.HookTriggeredMessage{
			HookID:     hookID,
			HookName:   hookResponse.Name,
			Method:     c.Request.Method,
			RemoteAddr: c.ClientIP(),
			Success:    success,
			Output:     output,
			Error:      errorMsg,
		},
	}
	wsmanager.Global.Broadcast(wsMessage)

	if success {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hook触发成功",
			"hook":    hookResponse.Name,
			"output":  output,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Hook触发失败",
			"hook":    hookResponse.Name,
			"error":   errorMsg,
			"output":  output,
		})
	}
}

// ReloadConfigHandler 重新加载Hooks配置的专用接口
func ReloadConfigHandler(c *gin.Context) {
	if HookManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Hook管理器未初始化",
		})
		return
	}

	// 执行实际的重新加载
	err := HookManager.ReloadAllHooks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "重新加载Hooks配置失败",
			"details":   err.Error(),
			"hookCount": HookManager.GetHookCount(),
		})
		return
	}

	// 获取重新加载后的hooks数量
	hookCount := HookManager.GetHookCount()

	c.JSON(http.StatusOK, gin.H{
		"message":   "Hooks配置重新加载成功",
		"hookCount": hookCount,
	})
}
