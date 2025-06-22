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
	description := fmt.Sprintf("Execute: %s", h.ExecuteCommand)
	if h.ResponseMessage != "" {
		description = h.ResponseMessage
	}

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

	return types.HookResponse{
		ID:                     h.ID,
		Name:                   h.ID, // use ID as name
		Description:            description,
		ExecuteCommand:         h.ExecuteCommand,
		WorkingDirectory:       h.CommandWorkingDirectory,
		ResponseMessage:        h.ResponseMessage,
		HTTPMethods:            httpMethods,
		TriggerRuleDescription: triggerDesc,
		LastUsed:               nil, // TODO: can add actual usage time trackingl usage time tracking
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
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("hooks file %s modified\n", event.Name)
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
		output = "Hook triggered successfully (no execute command)"
	}

	// 记录手动触发的Webhook执行日志到数据库
	database.LogHookExecution(
		hookID,                // hookID
		hookResponse.Name,     // hookName
		"webhook",             // hookType
		c.Request.Method,      // method
		c.ClientIP(),          // remoteAddr
		c.Request.Header,      // headers
		"",                    // body (手动触发无请求体)
		success,               // success
		output,                // output
		errorMsg,              // error
		0,                     // duration (手动触发无精确执行时间)
		c.Request.UserAgent(), // userAgent
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
			RemoteAddr: c.ClientIP(),
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
