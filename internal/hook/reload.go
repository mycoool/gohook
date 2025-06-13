package hook

import (
	"log"
)

// HookManager 管理hook的加载和重新加载
type HookManager struct {
	LoadedHooksFromFiles *map[string]Hooks
	HooksFiles           []string
	AsTemplate           bool
}

// NewHookManager 创建新的HookManager实例
func NewHookManager(loadedHooks *map[string]Hooks, hooksFiles []string, asTemplate bool) *HookManager {
	return &HookManager{
		LoadedHooksFromFiles: loadedHooks,
		HooksFiles:           hooksFiles,
		AsTemplate:           asTemplate,
	}
}

// MatchLoadedHook 在所有已加载的hooks中查找匹配的hook
func (hm *HookManager) MatchLoadedHook(id string) *Hook {
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

// LenLoadedHooks 返回已加载的hooks总数
func (hm *HookManager) LenLoadedHooks() int {
	if hm.LoadedHooksFromFiles == nil {
		return 0
	}

	sum := 0
	for _, hooks := range *hm.LoadedHooksFromFiles {
		sum += len(hooks)
	}
	return sum
}

// ReloadHooks 重新加载指定文件的hooks
func (hm *HookManager) ReloadHooks(hooksFilePath string) error {
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

		// 检查这个hook ID是否已经在当前文件中加载过
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

		// 检查hook ID是否重复
		if (hm.MatchLoadedHook(hook.ID) != nil && !wasHookIDAlreadyLoaded) || seenHooksIds[hook.ID] {
			log.Printf("error: hook with the id %s has already been loaded!\nplease check your hooks file for duplicate hooks ids!", hook.ID)
			log.Println("reverting hooks back to the previous configuration")
			return nil // 不返回错误，而是恢复到之前的配置
		}

		seenHooksIds[hook.ID] = true
		log.Printf("\tloaded: %s\n", hook.ID)
	}

	// 更新已加载的hooks
	if hm.LoadedHooksFromFiles != nil {
		(*hm.LoadedHooksFromFiles)[hooksFilePath] = newHooks
	}

	return nil
}

// ReloadAllHooks 重新加载所有hooks文件
func (hm *HookManager) ReloadAllHooks() error {
	var lastError error

	for _, hooksFilePath := range hm.HooksFiles {
		if err := hm.ReloadHooks(hooksFilePath); err != nil {
			lastError = err
			log.Printf("failed to reload hooks from %s: %v", hooksFilePath, err)
		}
	}

	return lastError
}

// RemoveHooks 移除指定文件的hooks
func (hm *HookManager) RemoveHooks(hooksFilePath string) {
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

// GetHookCount 获取当前加载的hooks数量
func (hm *HookManager) GetHookCount() int {
	return hm.LenLoadedHooks()
}

// GetAllHooks 获取所有已加载的hooks
func (hm *HookManager) GetAllHooks() []Hook {
	if hm.LoadedHooksFromFiles == nil {
		return nil
	}

	var allHooks []Hook
	for _, hooks := range *hm.LoadedHooksFromFiles {
		allHooks = append(allHooks, hooks...)
	}

	return allHooks
}
