package hook

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
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
func GetAllHooks(c *gin.Context) {
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
