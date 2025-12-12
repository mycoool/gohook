package webhook

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gin-gonic/gin"
	"github.com/mycoool/gohook/internal/database"
	"github.com/mycoool/gohook/internal/middleware"
	"github.com/mycoool/gohook/internal/stream"
	"github.com/mycoool/gohook/internal/types"
)

// HookManager manage hook and config file loading
type hookManager struct {
	LoadedHooksFromFiles *map[string]Hooks
	HooksFiles           []string
	AsTemplate           bool
}

// global variable reference, used to access loaded hooks
var LoadedHooksFromFiles *map[string]Hooks
var HookManager *hookManager

// GetAllHooks get all hooks
func HandleGetAllHooks(c *gin.Context) {
	if LoadedHooksFromFiles == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hooks not loaded"})
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

// HandleGetHook 获取单个Hook的详细信息
func HandleGetHook(c *gin.Context) {
	hookID := c.Param("id")
	if hookID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hook ID is required"})
		return
	}

	// 查找Hook
	hook := HookManager.MatchLoadedHook(hookID)
	if hook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	// 转换Hook为前端需要的格式
	hookResponse := map[string]interface{}{
		"id":                                          hook.ID,
		"execute-command":                             hook.ExecuteCommand,
		"command-working-directory":                   hook.CommandWorkingDirectory,
		"response-message":                            hook.ResponseMessage,
		"http-methods":                                hook.HTTPMethods,
		"pass-arguments-to-command":                   hook.PassArgumentsToCommand,
		"pass-environment-to-command":                 hook.PassEnvironmentToCommand,
		"parse-parameters-as-json":                    hook.JSONStringParameters,
		"trigger-rule":                                hook.TriggerRule,
		"trigger-rule-mismatch-http-response-code":    hook.TriggerRuleMismatchHttpResponseCode,
		"include-command-output-in-response":          hook.CaptureCommandOutput,
		"include-command-output-in-response-on-error": hook.CaptureCommandOutputOnError,
	}

	// 转换ResponseHeaders为前端期望的map格式
	responseHeaders := make(map[string]string)
	for _, header := range hook.ResponseHeaders {
		responseHeaders[header.Name] = header.Value
	}
	hookResponse["response-headers"] = responseHeaders

	c.JSON(http.StatusOK, hookResponse)
}

// GetHookByID get Hook by ID
func GetHookByID(id string) *types.HookResponse {
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

// convertHookToResponse convert Hook to HookResponse
func convertHookToResponse(h *Hook) types.HookResponse {
	// parse trigger rule to readable description
	triggerDesc := "Any request"
	if h.TriggerRule != nil {
		triggerDesc = describeTriggerRule(h.TriggerRule)
	}

	// set HTTP methods
	httpMethods := h.HTTPMethods
	if len(httpMethods) == 0 {
		httpMethods = []string{"POST", "GET"} // default methods
	}

	// count arguments and environment variables
	argumentsCount := 0
	if h.PassArgumentsToCommand != nil {
		argumentsCount = len(h.PassArgumentsToCommand)
	}

	environmentCount := 0
	if h.PassEnvironmentToCommand != nil {
		environmentCount = len(h.PassEnvironmentToCommand)
	}

	return types.HookResponse{
		ID:                     h.ID,
		Name:                   h.ID, // use ID as name
		ExecuteCommand:         h.ExecuteCommand,
		WorkingDirectory:       h.CommandWorkingDirectory,
		ResponseMessage:        h.ResponseMessage,
		HTTPMethods:            httpMethods,
		ArgumentsCount:         argumentsCount,
		EnvironmentCount:       environmentCount,
		TriggerRuleDescription: triggerDesc,
		TriggerRule:            h.TriggerRule,
		LastUsed:               nil, // TODO: can add actual usage time tracking
		Status:                 "active",
	}
}

// describeTriggerRule generate readable description for trigger rule
func describeTriggerRule(rules *Rules) string {
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

// NewHookManager create new HookManager instance
func NewHookManager(loadedHooks *map[string]Hooks, hooksFiles []string, asTemplate bool) *hookManager {
	return &hookManager{
		LoadedHooksFromFiles: loadedHooks,
		HooksFiles:           hooksFiles,
		AsTemplate:           asTemplate,
	}
}

// MatchLoadedHook find matching hook in all loaded hooks
func (hm *hookManager) MatchLoadedHook(id string) *Hook {
	if hm.LoadedHooksFromFiles == nil {
		return nil
	}

	for _, hooks := range *hm.LoadedHooksFromFiles {
		if hook := hooks.Match(id); hook != nil {
			return hook
		}
	}
	return nil
}

// LenLoadedHooks return total number of loaded hooks
func (hm *hookManager) LenLoadedHooks() int {
	if hm.LoadedHooksFromFiles == nil {
		return 0
	}

	sum := 0
	for _, hooks := range *hm.LoadedHooksFromFiles {
		sum += len(hooks)
	}
	return sum
}

// ReloadHooks 加载指定文件的hooks
func (hm *hookManager) ReloadHooks(hooksFilePath string) error {
	log.Printf("reloading hooks from %s\n", hooksFilePath)

	newHooks := Hooks{}
	err := newHooks.LoadFromFile(hooksFilePath, hm.AsTemplate)

	if err != nil {
		log.Printf("couldn't load hooks from file! %+v\n", err)
		return err
	}

	seenHooksIds := make(map[string]bool)
	log.Printf("found %d hook(s) in file\n", len(newHooks))

	for _, hook := range newHooks {
		wasHookIDAlreadyLoaded := false

		// check if this hook ID has already been loaded in the current file
		if hm.LoadedHooksFromFiles != nil {
			if existingHooks, exists := (*hm.LoadedHooksFromFiles)[hooksFilePath]; exists {
				for _, loadedHook := range existingHooks {
					if loadedHook.ID == hook.ID {
						wasHookIDAlreadyLoaded = true
						break
					}
				}
			}
		}

		// check if hook ID is duplicated
		if (hm.MatchLoadedHook(hook.ID) != nil && !wasHookIDAlreadyLoaded) || seenHooksIds[hook.ID] {
			log.Printf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!", hook.ID)
			log.Println("reverting hooks back to the previous configuration")
			return nil // don't return error, just revert to previous configuration
		}

		seenHooksIds[hook.ID] = true
		log.Printf("\tloaded: %s\n", hook.ID)
	}

	// update loaded hooks
	if hm.LoadedHooksFromFiles != nil {
		(*hm.LoadedHooksFromFiles)[hooksFilePath] = newHooks
	}

	return nil
}

// ReloadAllHooks load all hooks files
func (hm *hookManager) ReloadAllHooks() error {
	var lastError error

	for _, hooksFilePath := range hm.HooksFiles {
		if err := hm.ReloadHooks(hooksFilePath); err != nil {
			lastError = err
			log.Printf("failed to reload hooks from %s: %v", hooksFilePath, err)
		}
	}

	return lastError
}

// RemoveHooks remove hooks from specified file
func (hm *hookManager) RemoveHooks(hooksFilePath string) {
	if hm.LoadedHooksFromFiles == nil {
		return
	}

	log.Printf("removing hooks from %s\n", hooksFilePath)

	if hooks, exists := (*hm.LoadedHooksFromFiles)[hooksFilePath]; exists {
		for _, hook := range hooks {
			log.Printf("\tremoving: %s\n", hook.ID)
		}

		removedHooksCount := len(hooks)
		delete(*hm.LoadedHooksFromFiles, hooksFilePath)

		log.Printf("removed %d hook(s) that were loaded from file %s\n", removedHooksCount, hooksFilePath)
	}
}

// GetHookCount get current loaded hooks count
func (hm *hookManager) GetHookCount() int {
	return hm.LenLoadedHooks()
}

// GetAllHooks get all loaded hooks
func (hm *hookManager) GetAllHooks() []Hook {
	if hm.LoadedHooksFromFiles == nil {
		return nil
	}

	var allHooks []Hook
	for _, hooks := range *hm.LoadedHooksFromFiles {
		allHooks = append(allHooks, hooks...)
	}

	return allHooks
}

// FindHookFile 查找指定Hook所在的配置文件路径
func (hm *hookManager) FindHookFile(hookID string) string {
	if hm.LoadedHooksFromFiles == nil {
		return ""
	}

	for filePath, hooks := range *hm.LoadedHooksFromFiles {
		for _, hook := range hooks {
			if hook.ID == hookID {
				return filePath
			}
		}
	}
	return ""
}

// SaveHooksToFile 保存指定文件的hooks配置
func (hm *hookManager) SaveHooksToFile(filePath string) error {
	if hm.LoadedHooksFromFiles == nil {
		return fmt.Errorf("no hooks loaded")
	}

	hooks, exists := (*hm.LoadedHooksFromFiles)[filePath]
	if !exists {
		return fmt.Errorf("hooks file %s not found in loaded hooks", filePath)
	}

	return hooks.SaveToFile(filePath)
}

// SaveHookChanges 保存Hook的更改到对应的配置文件
func (hm *hookManager) SaveHookChanges(hookID string) error {
	filePath := hm.FindHookFile(hookID)
	if filePath == "" {
		return fmt.Errorf("hook %s not found in any loaded files", hookID)
	}

	return hm.SaveHooksToFile(filePath)
}

func ReloadAllHooks(hooksFiles []string, asTemplate bool) {
	if HookManager != nil {
		if err := HookManager.ReloadAllHooks(); err != nil {
			log.Printf("failed to reload all hooks: %v", err)
		}
	} else {
		// revert to original logic
		for _, hooksFilePath := range hooksFiles {
			reloadHooks(hooksFilePath, asTemplate)
		}
	}
}

func reloadHooks(hooksFilePath string, asTemplate bool) {
	if HookManager != nil {
		if err := HookManager.ReloadHooks(hooksFilePath); err != nil {
			log.Printf("failed to reload hooks from %s: %v", hooksFilePath, err)
		}
		return
	}

	// revert to original logic
	log.Printf("reloading hooks from %s\n", hooksFilePath)

	newHooks := Hooks{}

	err := newHooks.LoadFromFile(hooksFilePath, asTemplate)

	if err != nil {
		log.Printf("couldn't load hooks from file! %+v\n", err)
	} else {
		seenHooksIds := make(map[string]bool)

		log.Printf("found %d hook(s) in file\n", len(newHooks))

		for _, hook := range newHooks {
			wasHookIDAlreadyLoaded := false

			for _, loadedHook := range (*HookManager.LoadedHooksFromFiles)[hooksFilePath] {
				if loadedHook.ID == hook.ID {
					wasHookIDAlreadyLoaded = true
					break
				}
			}

			if (HookManager.MatchLoadedHook(hook.ID) != nil && !wasHookIDAlreadyLoaded) || seenHooksIds[hook.ID] {
				log.Printf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!", hook.ID)
				log.Println("reverting hooks back to the previous configuration")
				return
			}

			seenHooksIds[hook.ID] = true
			log.Printf("\tloaded: %s\n", hook.ID)
		}

		(*HookManager.LoadedHooksFromFiles)[hooksFilePath] = newHooks
	}
}

func removeHooks(hooksFilePath string, hooksFiles []string) {
	if HookManager != nil {
		HookManager.RemoveHooks(hooksFilePath)

		// remove file path from hooksFiles list
		newHooksFiles := hooksFiles[:0]
		for _, filePath := range hooksFiles {
			if filePath != hooksFilePath {
				newHooksFiles = append(newHooksFiles, filePath)
			}
		}
		hooksFiles = newHooksFiles

		// update HookManager's file list
		HookManager.HooksFiles = hooksFiles

		return
	}

	// revert to original logic
	log.Printf("removing hooks from %s\n", hooksFilePath)

	for _, hook := range (*HookManager.LoadedHooksFromFiles)[hooksFilePath] {
		log.Printf("\tremoving: %s\n", hook.ID)
	}

	newHooksFiles := hooksFiles[:0]
	for _, filePath := range hooksFiles {
		if filePath != hooksFilePath {
			newHooksFiles = append(newHooksFiles, filePath)
		}
	}

	removedHooksCount := len((*HookManager.LoadedHooksFromFiles)[hooksFilePath])

	delete((*HookManager.LoadedHooksFromFiles), hooksFilePath)

	log.Printf("removed %d hook(s) that were loaded from file %s\n", removedHooksCount, hooksFilePath)

}

func WatchForFileChange(watcher *fsnotify.Watcher, loadedHooksFromFiles *map[string]Hooks, hooksFiles []string, asTemplate bool) {
	for {
		select {
		case event := <-(*watcher).Events:
			if !isWatchedHooksFile(event.Name, hooksFiles) {
				continue
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("hooks file %s modified\n", event.Name)
				reloadHooks(event.Name, asTemplate)
			} else if event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("hooks file %s created\n", event.Name)
				_ = (*watcher).Add(event.Name)
				reloadHooks(event.Name, asTemplate)
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				if _, err := os.Stat(event.Name); os.IsNotExist(err) {
					log.Printf("hooks file %s removed, no longer watching this file for changes, removing hooks that were loaded from it\n", event.Name)
					if err := (*watcher).Remove(event.Name); err != nil {
						log.Printf("Error removing watcher for %s: %v\n", event.Name, err)
					}
					removeHooks(event.Name, hooksFiles)
				}
			} else if event.Op&fsnotify.Rename == fsnotify.Rename {
				time.Sleep(100 * time.Millisecond)
				if _, err := os.Stat(event.Name); os.IsNotExist(err) {
					// file was removed
					log.Printf("hooks file %s removed, no longer watching this file for changes, and removing hooks that were loaded from it\n", event.Name)
					if err := (*watcher).Remove(event.Name); err != nil {
						log.Printf("Error removing watcher for %s: %v\n", event.Name, err)
					}
					removeHooks(event.Name, hooksFiles)
				} else {
					// file was overwritten
					log.Printf("hooks file %s overwritten\n", event.Name)
					reloadHooks(event.Name, asTemplate)
					if err := (*watcher).Remove(event.Name); err != nil {
						log.Printf("Error removing watcher for %s: %v\n", event.Name, err)
					}
					if err := (*watcher).Add(event.Name); err != nil {
						log.Printf("Error adding watcher for %s: %v\n", event.Name, err)
					}
				}
			}
		case err := <-(*watcher).Errors:
			log.Println("watcher error:", err)
		}
	}
}

func isWatchedHooksFile(path string, hooksFiles []string) bool {
	path = filepath.Clean(path)
	for _, f := range hooksFiles {
		if filepath.Clean(f) == path {
			return true
		}
	}
	return false
}

// makeHumanPattern builds a human-friendly URL for display.
func MakeHumanPattern(prefix *string) string {
	if prefix == nil || *prefix == "" {
		return "/{id}"
	}
	return "/" + *prefix + "/{id}"
}

func HandleHook(h *Hook, r *Request) (string, error) {
	var errors []error

	// check the command exists
	var lookpath string
	if filepath.IsAbs(h.ExecuteCommand) || h.CommandWorkingDirectory == "" {
		lookpath = h.ExecuteCommand
	} else {
		lookpath = filepath.Join(h.CommandWorkingDirectory, h.ExecuteCommand)
	}

	cmdPath, err := exec.LookPath(lookpath)
	if err != nil {
		log.Printf("[%s] error in %s", r.ID, err)

		// check if parameters specified in execute-command by mistake
		if strings.IndexByte(h.ExecuteCommand, ' ') != -1 {
			s := strings.Fields(h.ExecuteCommand)[0]
			log.Printf("[%s] use 'pass-arguments-to-command' to specify args for '%s'", r.ID, s)
		}

		return "", err
	}

	cmd := exec.Command(cmdPath)
	cmd.Dir = h.CommandWorkingDirectory

	cmd.Args, errors = h.ExtractCommandArguments(r)
	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments: %s\n", r.ID, err)
	}

	var envs []string
	envs, errors = h.ExtractCommandArgumentsForEnv(r)

	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments for environment: %s\n", r.ID, err)
	}

	files, errors := h.ExtractCommandArgumentsForFile(r)

	for _, err := range errors {
		log.Printf("[%s] error extracting command arguments for file: %s\n", r.ID, err)
	}

	for i := range files {
		tmpfile, err := os.CreateTemp(h.CommandWorkingDirectory, files[i].EnvName)
		if err != nil {
			log.Printf("[%s] error creating temp file [%s]", r.ID, err)
			continue
		}
		log.Printf("[%s] writing env %s file %s", r.ID, files[i].EnvName, tmpfile.Name())
		if _, err := tmpfile.Write(files[i].Data); err != nil {
			log.Printf("[%s] error writing file %s [%s]", r.ID, tmpfile.Name(), err)
			continue
		}
		if err := tmpfile.Close(); err != nil {
			log.Printf("[%s] error closing file %s [%s]", r.ID, tmpfile.Name(), err)
			continue
		}

		files[i].File = tmpfile
		envs = append(envs, files[i].EnvName+"="+tmpfile.Name())
	}

	cmd.Env = append(os.Environ(), envs...)

	log.Printf("[%s] executing %s (%s) with arguments %q and environment %s using %s as cwd\n", r.ID, h.ExecuteCommand, cmd.Path, cmd.Args, envs, cmd.Dir)

	out, err := cmd.CombinedOutput()

	log.Printf("[%s] command output: %s\n", r.ID, out)

	if err != nil {
		log.Printf("[%s] error occurred: %+v\n", r.ID, err)
	}

	for i := range files {
		if files[i].File != nil {
			log.Printf("[%s] removing file %s\n", r.ID, files[i].File.Name())
			err := os.Remove(files[i].File.Name())
			if err != nil {
				log.Printf("[%s] error removing file %s [%s]", r.ID, files[i].File.Name(), err)
			}
		}
	}

	log.Printf("[%s] finished handling %s\n", r.ID, h.ID)

	// 记录Webhook执行日志到数据库
	method := ""
	remoteAddr := ""
	headers := make(map[string][]string)
	queryParams := make(map[string][]string)
	userAgent := ""

	if r.RawRequest != nil {
		method = r.RawRequest.Method
		remoteAddr = r.RawRequest.RemoteAddr
		headers = r.RawRequest.Header
		userAgent = r.RawRequest.UserAgent()

		// 获取查询参数
		for k, v := range r.RawRequest.URL.Query() {
			queryParams[k] = v
		}
	}

	// 使用database包记录Hook执行日志
	database.LogHookExecution(
		h.ID,           // hookID
		h.ID,           // hookName
		"webhook",      // hookType
		method,         // method
		remoteAddr,     // remoteAddr
		headers,        // headers
		string(r.Body), // body
		err == nil,     // success
		string(out),    // output
		func() string { // error
			if err != nil {
				return err.Error()
			}
			return ""
		}(),
		cmd.ProcessState.UserTime().Nanoseconds()/1000000, // duration (毫秒)
		userAgent,   // userAgent
		queryParams, // queryParams
	)

	// push WebSocket message to notify hook execution completed
	wsMessage := stream.WsMessage{
		Type:      "hook_triggered",
		Timestamp: time.Now(),
		Data: stream.HookTriggeredMessage{
			HookID:     h.ID,
			HookName:   h.ID,
			Method:     method,
			RemoteAddr: remoteAddr,
			Success:    err == nil,
			Output:     string(out),
			Error: func() string {
				if err != nil {
					return err.Error()
				} else {
					return ""
				}
			}(),
		},
	}
	stream.Global.Broadcast(wsMessage)

	return string(out), err
}

func HandleTriggerHook(c *gin.Context) {
	hookID := c.Param("id")
	hookResponse := GetHookByID(hookID)
	if hookResponse == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	// execute hook command
	success := false
	output := ""
	errorMsg := ""

	if hookResponse.ExecuteCommand != "" {
		// execute command
		var cmd *exec.Cmd

		// 检查工作目录是否存在
		if hookResponse.WorkingDirectory != "" {
			if _, err := os.Stat(hookResponse.WorkingDirectory); os.IsNotExist(err) {
				errorMsg = fmt.Sprintf("工作目录不存在: %s", hookResponse.WorkingDirectory)
				output = fmt.Sprintf("错误：工作目录 '%s' 不存在，请检查Hook配置", hookResponse.WorkingDirectory)
			} else {
				cmd = exec.Command("bash", "-c", hookResponse.ExecuteCommand)
				cmd.Dir = hookResponse.WorkingDirectory
			}
		} else {
			cmd = exec.Command("bash", "-c", hookResponse.ExecuteCommand)
		}

		if cmd != nil {
			result, err := cmd.CombinedOutput()
			output = string(result)
			if err != nil {
				errorMsg = fmt.Sprintf("命令执行失败: %v", err)
				if output == "" {
					output = fmt.Sprintf("命令执行出错: %v", err)
				}
			} else {
				success = true
			}
		}
	} else {
		success = true
		output = "Hook triggered successfully (no execute command)"
	}

	// 记录手动触发的Webhook执行日志到数据库
	database.LogHookExecution(
		hookID,                    // hookID
		hookResponse.Name,         // hookName
		"webhook",                 // hookType
		c.Request.Method,          // method
		middleware.GetClientIP(c), // remoteAddr
		c.Request.Header,          // headers
		"",                        // body (手动触发无请求体)
		success,                   // success
		output,                    // output
		errorMsg,                  // error
		0,                         // duration (手动触发无精确执行时间)
		c.Request.UserAgent(),     // userAgent
		map[string][]string{ // queryParams
			"trigger": {"manual"},
		},
	)

	// push WebSocket message
	wsMessage := stream.WsMessage{
		Type:      "hook_triggered",
		Timestamp: time.Now(),
		Data: stream.HookTriggeredMessage{
			HookID:     hookID,
			HookName:   hookResponse.Name,
			Method:     c.Request.Method,
			RemoteAddr: middleware.GetClientIP(c),
			Success:    success,
			Output:     output,
			Error:      errorMsg,
		},
	}
	stream.Global.Broadcast(wsMessage)

	if success {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hook triggered successfully",
			"hook":    hookResponse.Name,
			"output":  output,
		})
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Hook triggered failed",
			"hook":    hookResponse.Name,
			"error":   errorMsg,
			"output":  output,
		})
	}
}

func HandleGetHookByID(c *gin.Context) {
	hookID := c.Param("id")
	hookResponse := GetHookByID(hookID)
	if hookResponse == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}
	c.JSON(http.StatusOK, hookResponse)
}

func HandleReloadHooksConfig(c *gin.Context) {
	if HookManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Hook manager not initialized",
		})
		return
	}

	// execute actual reload
	err := HookManager.ReloadAllHooks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":     "Load Hook failed",
			"details":   err.Error(),
			"hookCount": HookManager.GetHookCount(),
		})
		return
	}

	// get loaded hooks count
	hookCount := HookManager.GetHookCount()

	c.JSON(http.StatusOK, gin.H{
		"message":   "Hooks config loaded successfully",
		"hookCount": hookCount,
	})
}

// isExecutableFile 检查路径是否指向系统可执行文件
func isExecutableFile(path string) bool {
	// 如果路径包含空格或参数，提取第一个单词作为命令名
	commandParts := strings.Fields(path)
	if len(commandParts) == 0 {
		return false
	}

	commandName := commandParts[0]

	// 检查是否为绝对路径且在标准系统目录中
	if strings.HasPrefix(commandName, "/bin/") ||
		strings.HasPrefix(commandName, "/usr/bin/") ||
		strings.HasPrefix(commandName, "/usr/local/bin/") ||
		strings.HasPrefix(commandName, "/sbin/") ||
		strings.HasPrefix(commandName, "/usr/sbin/") {
		return true
	}

	// 如果是脚本文件扩展名，很可能是脚本文件
	if strings.Contains(commandName, ".sh") ||
		strings.Contains(commandName, ".py") ||
		strings.Contains(commandName, ".js") ||
		strings.Contains(commandName, ".pl") ||
		strings.Contains(commandName, ".rb") {
		return false
	}

	// 如果命令名包含路径分隔符，提取文件名
	if strings.Contains(commandName, "/") {
		commandName = filepath.Base(commandName)
	}

	// 检查是否为常见的系统命令（只检查纯命令名，不包含参数）
	commonCommands := []string{
		"echo", "cat", "ls", "cp", "mv", "rm", "mkdir", "rmdir",
		"chmod", "chown", "grep", "sed", "awk", "cut", "sort",
		"tar", "gzip", "curl", "wget", "git", "npm", "yarn",
		"node", "python", "python3", "php", "ruby", "java",
		"docker", "kubectl", "systemctl", "service",
	}

	for _, cmd := range commonCommands {
		if commandName == cmd {
			return true
		}
	}

	return false
}

// isBinaryFile 检查文件是否为二进制文件
func isBinaryFile(content []byte) bool {
	// 检查文件前512字节中是否包含空字节
	checkBytes := content
	if len(content) > 512 {
		checkBytes = content[:512]
	}

	for _, b := range checkBytes {
		if b == 0 {
			return true
		}
	}

	return false
}

// 脚本文件管理 - 获取脚本内容
func HandleGetHookScript(c *gin.Context) {
	hookID := c.Param("id")

	// 检查 hook 是否存在，并获取配置
	hook := HookManager.MatchLoadedHook(hookID)
	if hook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	// 使用hook配置中的execute-command作为脚本路径
	scriptPath := hook.ExecuteCommand
	if scriptPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hook has no execute-command configured"})
		return
	}

	// 检查是否为系统可执行文件
	if isExecutableFile(scriptPath) {
		c.JSON(http.StatusOK, gin.H{
			"content":      "",
			"exists":       true,
			"path":         scriptPath,
			"isExecutable": true,
			"editable":     false,
			"message":      "这是一个系统可执行文件，不可编辑",
			"suggestion":   "如需使用脚本，请在基本信息中将执行命令修改为脚本文件路径，例如：/path/to/your-script.sh",
		})
		return
	}

	// 检查文件是否存在
	fileInfo, err := os.Stat(scriptPath)
	if os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{
			"content":      "",
			"exists":       false,
			"path":         scriptPath,
			"isExecutable": false,
			"editable":     true,
		})
		return
	}

	// 检查是否为目录
	if fileInfo.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "指定的路径是一个目录，不是文件",
			"path":       scriptPath,
			"editable":   false,
			"suggestion": "请在基本信息中修改执行命令为具体的脚本文件路径",
		})
		return
	}

	// 读取文件内容
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取脚本文件失败: " + err.Error()})
		return
	}

	// 检查是否为二进制文件
	if isBinaryFile(content) {
		c.JSON(http.StatusOK, gin.H{
			"content":      "",
			"exists":       true,
			"path":         scriptPath,
			"isExecutable": true,
			"editable":     false,
			"message":      "这是一个二进制可执行文件，不可编辑",
			"suggestion":   "如需使用脚本，请在基本信息中将执行命令修改为脚本文件路径，例如：/path/to/your-script.sh",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"content":      string(content),
		"exists":       true,
		"path":         scriptPath,
		"isExecutable": false,
		"editable":     true,
	})
}

// HandleUpdateHookResponse 更新Hook响应配置
func HandleUpdateHookResponse(c *gin.Context) {
	hookID := c.Param("id")
	existingHook := HookManager.MatchLoadedHook(hookID)
	if existingHook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	var request struct {
		HTTPMethods                           []string          `json:"http-methods,omitempty"`
		ResponseHeaders                       map[string]string `json:"response-headers,omitempty"`
		IncludeCommandOutputInResponse        bool              `json:"include-command-output-in-response,omitempty"`
		IncludeCommandOutputInResponseOnError bool              `json:"include-command-output-in-response-on-error,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// 验证HTTP方法
	validMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true}
	for _, method := range request.HTTPMethods {
		if !validMethods[strings.ToUpper(method)] {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("无效的HTTP方法: %s", method)})
			return
		}
	}

	// 备份原值，以便保存失败时恢复和记录日志
	originalHTTPMethods := existingHook.HTTPMethods
	originalResponseHeaders := existingHook.ResponseHeaders
	originalCaptureCommandOutput := existingHook.CaptureCommandOutput
	originalCaptureCommandOutputOnError := existingHook.CaptureCommandOutputOnError

	// 更新响应配置
	if len(request.HTTPMethods) > 0 {
		existingHook.HTTPMethods = request.HTTPMethods
	}

	// 转换ResponseHeaders格式
	if request.ResponseHeaders != nil {
		existingHook.ResponseHeaders = make(ResponseHeaders, 0, len(request.ResponseHeaders))
		for name, value := range request.ResponseHeaders {
			existingHook.ResponseHeaders = append(existingHook.ResponseHeaders, Header{
				Name:  name,
				Value: value,
			})
		}
	}

	existingHook.CaptureCommandOutput = request.IncludeCommandOutputInResponse
	existingHook.CaptureCommandOutputOnError = request.IncludeCommandOutputInResponseOnError

	// 保存到配置文件
	if err := HookManager.SaveHookChanges(hookID); err != nil {
		// 保存失败，恢复原值
		existingHook.HTTPMethods = originalHTTPMethods
		existingHook.ResponseHeaders = originalResponseHeaders
		existingHook.CaptureCommandOutput = originalCaptureCommandOutput
		existingHook.CaptureCommandOutputOnError = originalCaptureCommandOutputOnError

		// 记录失败的日志
		username, _ := c.Get("username")
		usernameStr := "unknown"
		if username != nil {
			usernameStr = username.(string)
		}
		database.LogHookManagement(
			database.UserActionUpdateHookResponse,
			hookID,
			hookID,
			usernameStr,
			middleware.GetClientIP(c),
			c.Request.UserAgent(),
			false,
			map[string]interface{}{
				"hookId": hookID,
				"error":  err.Error(),
				"action": "update_hook_response",
				"changes": map[string]interface{}{
					"httpMethods":     request.HTTPMethods,
					"responseHeaders": request.ResponseHeaders,
					"captureOutput":   request.IncludeCommandOutputInResponse,
					"captureError":    request.IncludeCommandOutputInResponseOnError,
				},
			},
		)

		// 推送失败消息
		wsMessage := stream.WsMessage{
			Type:      "hook_managed",
			Timestamp: time.Now(),
			Data: stream.HookManageMessage{
				Action:   "update_response",
				HookID:   hookID,
				HookName: hookID,
				Success:  false,
				Error:    "保存Hook响应配置失败: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save hook changes: " + err.Error()})
		return
	}

	// 记录成功的日志
	username, _ := c.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}
	database.LogHookManagement(
		database.UserActionUpdateHookResponse,
		hookID,
		hookID,
		usernameStr,
		middleware.GetClientIP(c),
		c.Request.UserAgent(),
		true,
		map[string]interface{}{
			"hookId": hookID,
			"action": "update_hook_response",
			"changes": map[string]interface{}{
				"httpMethods":     request.HTTPMethods,
				"responseHeaders": request.ResponseHeaders,
				"captureOutput":   request.IncludeCommandOutputInResponse,
				"captureError":    request.IncludeCommandOutputInResponseOnError,
			},
		},
	)

	// 推送成功消息
	wsMessage := stream.WsMessage{
		Type:      "hook_managed",
		Timestamp: time.Now(),
		Data: stream.HookManageMessage{
			Action:   "update_response",
			HookID:   hookID,
			HookName: hookID,
			Success:  true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Hook响应配置更新成功",
		"hookId":  hookID,
	})
}

// HandleSaveHookScript 脚本文件管理 - 保存脚本内容
func HandleSaveHookScript(c *gin.Context) {
	hookID := c.Param("id")

	// 检查 hook 是否存在，并获取配置
	hook := HookManager.MatchLoadedHook(hookID)
	if hook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	var req struct {
		Content string `json:"content"`
		Path    string `json:"path,omitempty"` // 允许前端指定脚本路径
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters"})
		return
	}

	// 确定脚本路径：优先使用前端指定的路径，否则使用execute-command
	var scriptPath string
	if req.Path != "" {
		scriptPath = req.Path
	} else {
		scriptPath = hook.ExecuteCommand
		if scriptPath == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hook has no execute-command configured and no path provided"})
			return
		}
	}

	// 确保脚本文件所在目录存在
	scriptDir := filepath.Dir(scriptPath)
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		// 记录失败的日志
		username, _ := c.Get("username")
		usernameStr := "unknown"
		if username != nil {
			usernameStr = username.(string)
		}
		database.LogHookManagement(
			database.UserActionSaveHookScript,
			hookID,
			hookID,
			usernameStr,
			middleware.GetClientIP(c),
			c.Request.UserAgent(),
			false,
			map[string]interface{}{
				"hookId":     hookID,
				"error":      "创建脚本目录失败: " + err.Error(),
				"action":     "save_hook_script",
				"scriptPath": scriptPath,
			},
		)

		// 推送失败消息
		wsMessage := stream.WsMessage{
			Type:      "hook_managed",
			Timestamp: time.Now(),
			Data: stream.HookManageMessage{
				Action:   "update_script",
				HookID:   hookID,
				HookName: hookID,
				Success:  false,
				Error:    "创建脚本目录失败: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建脚本目录失败: " + err.Error()})
		return
	}

	// 写入文件
	err := os.WriteFile(scriptPath, []byte(req.Content), 0755)
	if err != nil {
		// 记录失败的日志
		username, _ := c.Get("username")
		usernameStr := "unknown"
		if username != nil {
			usernameStr = username.(string)
		}
		database.LogHookManagement(
			database.UserActionSaveHookScript,
			hookID,
			hookID,
			usernameStr,
			middleware.GetClientIP(c),
			c.Request.UserAgent(),
			false,
			map[string]interface{}{
				"hookId":     hookID,
				"error":      err.Error(),
				"action":     "save_hook_script",
				"scriptPath": scriptPath,
			},
		)

		// 推送失败消息
		wsMessage := stream.WsMessage{
			Type:      "hook_managed",
			Timestamp: time.Now(),
			Data: stream.HookManageMessage{
				Action:   "update_script",
				HookID:   hookID,
				HookName: hookID,
				Success:  false,
				Error:    "保存脚本文件失败: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存脚本文件失败: " + err.Error()})
		return
	}

	// 记录成功的日志
	username, _ := c.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}
	database.LogHookManagement(
		database.UserActionSaveHookScript,
		hookID,
		hookID,
		usernameStr,
		middleware.GetClientIP(c),
		c.Request.UserAgent(),
		true,
		map[string]interface{}{
			"hookId":      hookID,
			"action":      "save_hook_script",
			"scriptPath":  scriptPath,
			"contentSize": len(req.Content),
		},
	)

	// 推送成功消息
	wsMessage := stream.WsMessage{
		Type:      "hook_managed",
		Timestamp: time.Now(),
		Data: stream.HookManageMessage{
			Action:   "update_script",
			HookID:   hookID,
			HookName: hookID,
			Success:  true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "脚本文件保存成功",
		"path":    scriptPath,
	})
}

// HandleCreateHook 创建新的Hook
func HandleCreateHook(c *gin.Context) {
	var request struct {
		ID                      string `json:"id" binding:"required"`
		ExecuteCommand          string `json:"execute-command" binding:"required"`
		CommandWorkingDirectory string `json:"command-working-directory,omitempty"`
		ResponseMessage         string `json:"response-message,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// 检查Hook ID是否已存在
	if HookManager.MatchLoadedHook(request.ID) != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Hook with this ID already exists"})
		return
	}

	// 创建新的Hook，使用默认值
	newHook := Hook{
		ID:                                  request.ID,
		ExecuteCommand:                      request.ExecuteCommand,
		CommandWorkingDirectory:             request.CommandWorkingDirectory,
		ResponseMessage:                     request.ResponseMessage,
		HTTPMethods:                         []string{"POST"},  // 默认方法
		CaptureCommandOutput:                false,             // 默认不包含输出
		CaptureCommandOutputOnError:         false,             // 默认不包含错误输出
		PassArgumentsToCommand:              []Argument{},      // 默认无参数
		PassEnvironmentToCommand:            []Argument{},      // 默认无环境变量
		JSONStringParameters:                []Argument{},      // 默认无JSON参数
		TriggerRule:                         nil,               // 默认无触发规则
		TriggerRuleMismatchHttpResponseCode: 400,               // 默认错误码
		ResponseHeaders:                     ResponseHeaders{}, // 默认无响应头
	}

	// 添加到内存中的第一个配置文件
	var targetFilePath string
	if LoadedHooksFromFiles != nil {
		for filePath, hooks := range *LoadedHooksFromFiles {
			updatedHooks := append(hooks, newHook)
			(*LoadedHooksFromFiles)[filePath] = updatedHooks
			targetFilePath = filePath
			break
		}
	}

	// 保存到配置文件
	if targetFilePath != "" {
		if err := HookManager.SaveHooksToFile(targetFilePath); err != nil {
			// 如果保存失败，从内存中移除刚添加的Hook
			if LoadedHooksFromFiles != nil {
				if hooks, exists := (*LoadedHooksFromFiles)[targetFilePath]; exists {
					// 移除最后添加的Hook
					if len(hooks) > 0 {
						(*LoadedHooksFromFiles)[targetFilePath] = hooks[:len(hooks)-1]
					}
				}
			}

			// 记录失败的日志
			username, _ := c.Get("username")
			usernameStr := "unknown"
			if username != nil {
				usernameStr = username.(string)
			}
			database.LogHookManagement(
				database.UserActionCreateHook,
				request.ID,
				request.ID,
				usernameStr,
				middleware.GetClientIP(c),
				c.Request.UserAgent(),
				false,
				map[string]interface{}{
					"hookId":   request.ID,
					"error":    err.Error(),
					"action":   "create_hook",
					"filePath": targetFilePath,
				},
			)

			// 推送失败消息
			wsMessage := stream.WsMessage{
				Type:      "hook_managed",
				Timestamp: time.Now(),
				Data: stream.HookManageMessage{
					Action:   "create",
					HookID:   request.ID,
					HookName: request.ID,
					Success:  false,
					Error:    "保存Hook配置失败: " + err.Error(),
				},
			}
			stream.Global.Broadcast(wsMessage)

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save hook to file: " + err.Error()})
			return
		}
	}

	// 记录成功的日志
	username, _ := c.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}
	database.LogHookManagement(
		database.UserActionCreateHook,
		request.ID,
		request.ID,
		usernameStr,
		middleware.GetClientIP(c),
		c.Request.UserAgent(),
		true,
		map[string]interface{}{
			"hookId":          request.ID,
			"executeCommand":  request.ExecuteCommand,
			"workingDir":      request.CommandWorkingDirectory,
			"responseMessage": request.ResponseMessage,
			"action":          "create_hook",
			"filePath":        targetFilePath,
		},
	)

	// 推送成功消息
	wsMessage := stream.WsMessage{
		Type:      "hook_managed",
		Timestamp: time.Now(),
		Data: stream.HookManageMessage{
			Action:   "create",
			HookID:   request.ID,
			HookName: request.ID,
			Success:  true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Hook创建成功",
		"hookId":  request.ID,
	})
}

// HandleUpdateHookBasic 更新Hook基本信息
func HandleUpdateHookBasic(c *gin.Context) {
	hookID := c.Param("id")
	existingHook := HookManager.MatchLoadedHook(hookID)
	if existingHook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	var request struct {
		ExecuteCommand          string `json:"execute-command" binding:"required"`
		CommandWorkingDirectory string `json:"command-working-directory,omitempty"`
		ResponseMessage         string `json:"response-message,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// 备份原值，以便保存失败时恢复和记录日志
	originalExecuteCommand := existingHook.ExecuteCommand
	originalCommandWorkingDirectory := existingHook.CommandWorkingDirectory
	originalResponseMessage := existingHook.ResponseMessage

	// 更新基本信息
	existingHook.ExecuteCommand = request.ExecuteCommand
	existingHook.CommandWorkingDirectory = request.CommandWorkingDirectory
	existingHook.ResponseMessage = request.ResponseMessage

	// 保存到配置文件
	if err := HookManager.SaveHookChanges(hookID); err != nil {
		// 保存失败，恢复原值
		existingHook.ExecuteCommand = originalExecuteCommand
		existingHook.CommandWorkingDirectory = originalCommandWorkingDirectory
		existingHook.ResponseMessage = originalResponseMessage

		// 记录失败的日志
		username, _ := c.Get("username")
		usernameStr := "unknown"
		if username != nil {
			usernameStr = username.(string)
		}
		database.LogHookManagement(
			database.UserActionUpdateHookBasic,
			hookID,
			hookID,
			usernameStr,
			middleware.GetClientIP(c),
			c.Request.UserAgent(),
			false,
			map[string]interface{}{
				"hookId": hookID,
				"error":  err.Error(),
				"action": "update_hook_basic",
				"changes": map[string]interface{}{
					"executeCommand":  request.ExecuteCommand,
					"workingDir":      request.CommandWorkingDirectory,
					"responseMessage": request.ResponseMessage,
				},
			},
		)

		// 推送失败消息
		wsMessage := stream.WsMessage{
			Type:      "hook_managed",
			Timestamp: time.Now(),
			Data: stream.HookManageMessage{
				Action:   "update_basic",
				HookID:   hookID,
				HookName: hookID,
				Success:  false,
				Error:    "保存Hook基本信息失败: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save hook changes: " + err.Error()})
		return
	}

	// 记录成功的日志
	username, _ := c.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}
	database.LogHookManagement(
		database.UserActionUpdateHookBasic,
		hookID,
		hookID,
		usernameStr,
		middleware.GetClientIP(c),
		c.Request.UserAgent(),
		true,
		map[string]interface{}{
			"hookId": hookID,
			"action": "update_hook_basic",
			"changes": map[string]interface{}{
				"executeCommand": map[string]string{
					"old": originalExecuteCommand,
					"new": request.ExecuteCommand,
				},
				"workingDir": map[string]string{
					"old": originalCommandWorkingDirectory,
					"new": request.CommandWorkingDirectory,
				},
				"responseMessage": map[string]string{
					"old": originalResponseMessage,
					"new": request.ResponseMessage,
				},
			},
		},
	)

	// 推送成功消息
	wsMessage := stream.WsMessage{
		Type:      "hook_managed",
		Timestamp: time.Now(),
		Data: stream.HookManageMessage{
			Action:   "update_basic",
			HookID:   hookID,
			HookName: hookID,
			Success:  true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Hook基本信息更新成功",
		"hookId":  hookID,
	})
}

// HandleUpdateHookParameters 更新Hook参数配置
func HandleUpdateHookParameters(c *gin.Context) {
	hookID := c.Param("id")
	existingHook := HookManager.MatchLoadedHook(hookID)
	if existingHook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	var request struct {
		PassArgumentsToCommand   []Argument `json:"pass-arguments-to-command,omitempty"`
		PassEnvironmentToCommand []Argument `json:"pass-environment-to-command,omitempty"`
		ParseParametersAsJSON    []Argument `json:"parse-parameters-as-json,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// 验证参数配置
	for i, arg := range request.PassArgumentsToCommand {
		if arg.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("参数%d的名称不能为空", i+1)})
			return
		}
		if arg.Source == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("参数%d的来源不能为空", i+1)})
			return
		}
	}

	for i, env := range request.PassEnvironmentToCommand {
		if env.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("环境变量%d的名称不能为空", i+1)})
			return
		}
		if env.Source == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("环境变量%d的来源不能为空", i+1)})
			return
		}
	}

	// 备份原值，以便保存失败时恢复和记录日志
	originalPassArgumentsToCommand := existingHook.PassArgumentsToCommand
	originalPassEnvironmentToCommand := existingHook.PassEnvironmentToCommand
	originalJSONStringParameters := existingHook.JSONStringParameters

	// 更新参数配置
	existingHook.PassArgumentsToCommand = request.PassArgumentsToCommand
	existingHook.PassEnvironmentToCommand = request.PassEnvironmentToCommand
	existingHook.JSONStringParameters = request.ParseParametersAsJSON

	// 保存到配置文件
	if err := HookManager.SaveHookChanges(hookID); err != nil {
		// 保存失败，恢复原值
		existingHook.PassArgumentsToCommand = originalPassArgumentsToCommand
		existingHook.PassEnvironmentToCommand = originalPassEnvironmentToCommand
		existingHook.JSONStringParameters = originalJSONStringParameters

		// 记录失败的日志
		username, _ := c.Get("username")
		usernameStr := "unknown"
		if username != nil {
			usernameStr = username.(string)
		}
		database.LogHookManagement(
			database.UserActionUpdateHookParameters,
			hookID,
			hookID,
			usernameStr,
			middleware.GetClientIP(c),
			c.Request.UserAgent(),
			false,
			map[string]interface{}{
				"hookId": hookID,
				"error":  err.Error(),
				"action": "update_hook_parameters",
				"changes": map[string]interface{}{
					"argumentsCount":      len(request.PassArgumentsToCommand),
					"environmentCount":    len(request.PassEnvironmentToCommand),
					"jsonParametersCount": len(request.ParseParametersAsJSON),
				},
			},
		)

		// 推送失败消息
		wsMessage := stream.WsMessage{
			Type:      "hook_managed",
			Timestamp: time.Now(),
			Data: stream.HookManageMessage{
				Action:   "update_parameters",
				HookID:   hookID,
				HookName: hookID,
				Success:  false,
				Error:    "保存Hook参数配置失败: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save hook changes: " + err.Error()})
		return
	}

	// 记录成功的日志
	username, _ := c.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}
	database.LogHookManagement(
		database.UserActionUpdateHookParameters,
		hookID,
		hookID,
		usernameStr,
		middleware.GetClientIP(c),
		c.Request.UserAgent(),
		true,
		map[string]interface{}{
			"hookId": hookID,
			"action": "update_hook_parameters",
			"changes": map[string]interface{}{
				"arguments": map[string]interface{}{
					"oldCount": len(originalPassArgumentsToCommand),
					"newCount": len(request.PassArgumentsToCommand),
				},
				"environment": map[string]interface{}{
					"oldCount": len(originalPassEnvironmentToCommand),
					"newCount": len(request.PassEnvironmentToCommand),
				},
				"jsonParameters": map[string]interface{}{
					"oldCount": len(originalJSONStringParameters),
					"newCount": len(request.ParseParametersAsJSON),
				},
			},
		},
	)

	// 推送成功消息
	wsMessage := stream.WsMessage{
		Type:      "hook_managed",
		Timestamp: time.Now(),
		Data: stream.HookManageMessage{
			Action:   "update_parameters",
			HookID:   hookID,
			HookName: hookID,
			Success:  true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Hook参数配置更新成功",
		"hookId":  hookID,
	})
}

// HandleUpdateHookTriggers 更新Hook触发规则
func HandleUpdateHookTriggers(c *gin.Context) {
	hookID := c.Param("id")
	existingHook := HookManager.MatchLoadedHook(hookID)
	if existingHook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	var request struct {
		TriggerRule                         *Rules `json:"trigger-rule,omitempty"`
		TriggerRuleMismatchHTTPResponseCode int    `json:"trigger-rule-mismatch-http-response-code,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// 验证触发规则
	if request.TriggerRuleMismatchHTTPResponseCode != 0 &&
		(request.TriggerRuleMismatchHTTPResponseCode < 200 || request.TriggerRuleMismatchHTTPResponseCode > 599) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "HTTP响应码必须在200-599范围内"})
		return
	}

	// 备份原值，以便保存失败时恢复和记录日志
	originalTriggerRule := existingHook.TriggerRule
	originalTriggerRuleMismatchHttpResponseCode := existingHook.TriggerRuleMismatchHttpResponseCode

	// 更新触发规则
	existingHook.TriggerRule = request.TriggerRule
	if request.TriggerRuleMismatchHTTPResponseCode > 0 {
		existingHook.TriggerRuleMismatchHttpResponseCode = request.TriggerRuleMismatchHTTPResponseCode
	}

	// 保存到配置文件
	if err := HookManager.SaveHookChanges(hookID); err != nil {
		// 保存失败，恢复原值
		existingHook.TriggerRule = originalTriggerRule
		existingHook.TriggerRuleMismatchHttpResponseCode = originalTriggerRuleMismatchHttpResponseCode

		// 记录失败的日志
		username, _ := c.Get("username")
		usernameStr := "unknown"
		if username != nil {
			usernameStr = username.(string)
		}
		database.LogHookManagement(
			database.UserActionUpdateHookTriggers,
			hookID,
			hookID,
			usernameStr,
			middleware.GetClientIP(c),
			c.Request.UserAgent(),
			false,
			map[string]interface{}{
				"hookId": hookID,
				"error":  err.Error(),
				"action": "update_hook_triggers",
				"changes": map[string]interface{}{
					"hasRules":     request.TriggerRule != nil,
					"responseCode": request.TriggerRuleMismatchHTTPResponseCode,
				},
			},
		)

		// 推送失败消息
		wsMessage := stream.WsMessage{
			Type:      "hook_managed",
			Timestamp: time.Now(),
			Data: stream.HookManageMessage{
				Action:   "update_triggers",
				HookID:   hookID,
				HookName: hookID,
				Success:  false,
				Error:    "保存Hook触发规则失败: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save hook changes: " + err.Error()})
		return
	}

	// 记录成功的日志
	username, _ := c.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}
	database.LogHookManagement(
		database.UserActionUpdateHookTriggers,
		hookID,
		hookID,
		usernameStr,
		middleware.GetClientIP(c),
		c.Request.UserAgent(),
		true,
		map[string]interface{}{
			"hookId": hookID,
			"action": "update_hook_triggers",
			"changes": map[string]interface{}{
				"triggerRule": map[string]interface{}{
					"hadRules": originalTriggerRule != nil,
					"hasRules": request.TriggerRule != nil,
				},
				"responseCode": map[string]interface{}{
					"old": originalTriggerRuleMismatchHttpResponseCode,
					"new": request.TriggerRuleMismatchHTTPResponseCode,
				},
			},
		},
	)

	// 推送成功消息
	wsMessage := stream.WsMessage{
		Type:      "hook_managed",
		Timestamp: time.Now(),
		Data: stream.HookManageMessage{
			Action:   "update_triggers",
			HookID:   hookID,
			HookName: hookID,
			Success:  true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Hook触发规则更新成功",
		"hookId":  hookID,
	})
}

// HandleUpdateHookExecuteCommand 更新Hook的执行命令
func HandleUpdateHookExecuteCommand(c *gin.Context) {
	hookID := c.Param("id")
	existingHook := HookManager.MatchLoadedHook(hookID)
	if existingHook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	var request struct {
		ExecuteCommand string `json:"execute-command" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters: " + err.Error()})
		return
	}

	// 备份原值，以便保存失败时恢复
	originalExecuteCommand := existingHook.ExecuteCommand

	// 更新执行命令
	existingHook.ExecuteCommand = request.ExecuteCommand

	// 保存到配置文件
	if err := HookManager.SaveHookChanges(hookID); err != nil {
		// 保存失败，恢复原值
		existingHook.ExecuteCommand = originalExecuteCommand

		// 记录失败的日志
		username, _ := c.Get("username")
		usernameStr := "unknown"
		if username != nil {
			usernameStr = username.(string)
		}
		database.LogHookManagement(
			database.UserActionUpdateHookScript,
			hookID,
			hookID,
			usernameStr,
			middleware.GetClientIP(c),
			c.Request.UserAgent(),
			false,
			map[string]interface{}{
				"hookId":     hookID,
				"error":      err.Error(),
				"action":     "update_execute_command",
				"oldCommand": originalExecuteCommand,
				"newCommand": request.ExecuteCommand,
			},
		)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save hook changes: " + err.Error()})
		return
	}

	// 记录成功的日志
	username, _ := c.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}
	database.LogHookManagement(
		database.UserActionUpdateHookScript,
		hookID,
		hookID,
		usernameStr,
		middleware.GetClientIP(c),
		c.Request.UserAgent(),
		true,
		map[string]interface{}{
			"hookId": hookID,
			"action": "update_execute_command",
			"changes": map[string]interface{}{
				"executeCommand": map[string]interface{}{
					"old": originalExecuteCommand,
					"new": request.ExecuteCommand,
				},
			},
		},
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Hook执行命令更新成功",
		"hookId":  hookID,
	})
}

// HandleDeleteHook 删除Hook
func HandleDeleteHook(c *gin.Context) {
	hookID := c.Param("id")
	if hookID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hook ID is required"})
		return
	}

	// 查找Hook是否存在
	existingHook := HookManager.MatchLoadedHook(hookID)
	if existingHook == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook not found"})
		return
	}

	// 查找Hook所在的配置文件
	var targetFilePath string
	var hookIndex = -1
	if LoadedHooksFromFiles != nil {
		for filePath, hooks := range *LoadedHooksFromFiles {
			for i, hook := range hooks {
				if hook.ID == hookID {
					targetFilePath = filePath
					hookIndex = i
					break
				}
			}
			if hookIndex != -1 {
				break
			}
		}
	}

	if targetFilePath == "" || hookIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Hook configuration not found"})
		return
	}

	// 从内存中删除Hook
	hooks := (*LoadedHooksFromFiles)[targetFilePath]
	updatedHooks := append(hooks[:hookIndex], hooks[hookIndex+1:]...)
	(*LoadedHooksFromFiles)[targetFilePath] = updatedHooks

	// 保存配置文件
	if err := HookManager.SaveHooksToFile(targetFilePath); err != nil {
		// 保存失败，恢复Hook到内存中
		hooks = append(hooks[:hookIndex], append([]Hook{*existingHook}, hooks[hookIndex:]...)...)
		(*LoadedHooksFromFiles)[targetFilePath] = hooks

		// 记录失败的日志
		username, _ := c.Get("username")
		usernameStr := "unknown"
		if username != nil {
			usernameStr = username.(string)
		}
		database.LogHookManagement(
			database.UserActionDeleteHook,
			hookID,
			hookID,
			usernameStr,
			middleware.GetClientIP(c),
			c.Request.UserAgent(),
			false,
			map[string]interface{}{
				"hookId":   hookID,
				"error":    err.Error(),
				"action":   "delete_hook",
				"filePath": targetFilePath,
			},
		)

		// 推送失败消息
		wsMessage := stream.WsMessage{
			Type:      "hook_managed",
			Timestamp: time.Now(),
			Data: stream.HookManageMessage{
				Action:   "delete",
				HookID:   hookID,
				HookName: hookID,
				Success:  false,
				Error:    "保存Hook配置失败: " + err.Error(),
			},
		}
		stream.Global.Broadcast(wsMessage)

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save hook configuration: " + err.Error()})
		return
	}

	// 记录成功的日志
	username, _ := c.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}
	database.LogHookManagement(
		database.UserActionDeleteHook,
		hookID,
		hookID,
		usernameStr,
		middleware.GetClientIP(c),
		c.Request.UserAgent(),
		true,
		map[string]interface{}{
			"hookId":         hookID,
			"executeCommand": existingHook.ExecuteCommand,
			"action":         "delete_hook",
			"filePath":       targetFilePath,
			"remainingHooks": len(updatedHooks),
		},
	)

	// 推送成功消息
	wsMessage := stream.WsMessage{
		Type:      "hook_managed",
		Timestamp: time.Now(),
		Data: stream.HookManageMessage{
			Action:   "delete",
			HookID:   hookID,
			HookName: hookID,
			Success:  true,
		},
	}
	stream.Global.Broadcast(wsMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Hook删除成功",
		"hookId":  hookID,
	})
}
