package configuration

import (
	"reflect"
	"sync"
	"time"
)

// EventManager 事件管理器
type EventManager struct {
	mu             sync.RWMutex
	globalWatchers map[string][]ConfigChangeHandler
	moduleWatchers map[string]map[string][]ConfigChangeHandler
}

// NewEventManager 创建事件管理器
func NewEventManager() *EventManager {
	return &EventManager{
		globalWatchers: make(map[string][]ConfigChangeHandler),
		moduleWatchers: make(map[string]map[string][]ConfigChangeHandler),
	}
}

// RegisterGlobalWatcher 注册全局配置监听器
func (em *EventManager) RegisterGlobalWatcher(key string, handler ConfigChangeHandler) {
	if handler == nil {
		return
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.globalWatchers[key]; !exists {
		em.globalWatchers[key] = make([]ConfigChangeHandler, 0)
	}

	em.globalWatchers[key] = append(em.globalWatchers[key], handler)
}

// RegisterModuleWatcher 注册模块配置监听器
func (em *EventManager) RegisterModuleWatcher(moduleName, key string, handler ConfigChangeHandler) {
	if handler == nil {
		return
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.moduleWatchers[moduleName]; !exists {
		em.moduleWatchers[moduleName] = make(map[string][]ConfigChangeHandler)
	}

	if _, exists := em.moduleWatchers[moduleName][key]; !exists {
		em.moduleWatchers[moduleName][key] = make([]ConfigChangeHandler, 0)
	}

	em.moduleWatchers[moduleName][key] = append(em.moduleWatchers[moduleName][key], handler)
}

// UnregisterGlobalWatcher 取消注册全局配置监听器
func (em *EventManager) UnregisterGlobalWatcher(key string, handler ConfigChangeHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()

	handlers, exists := em.globalWatchers[key]
	if !exists {
		return
	}

	// 使用函数值比较而不是指针比较
	for i, h := range handlers {
		// 由于函数值不能直接比较，我们使用反射或其他方法
		// 这里我们简单地移除第一个匹配的处理器
		// 在实际应用中，可能需要更复杂的比较逻辑
		if isSameHandler(h, handler) {
			em.globalWatchers[key] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	// 如果没有监听器了，删除该键
	if len(em.globalWatchers[key]) == 0 {
		delete(em.globalWatchers, key)
	}
}

// isSameHandler 检查两个处理器是否相同
// 注意：这是一个简化的实现，在实际应用中可能需要更复杂的逻辑
func isSameHandler(a, b ConfigChangeHandler) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

// UnregisterModuleWatcher 取消注册模块配置监听器
func (em *EventManager) UnregisterModuleWatcher(moduleName, key string, handler ConfigChangeHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()

	moduleWatchers, exists := em.moduleWatchers[moduleName]
	if !exists {
		return
	}

	handlers, exists := moduleWatchers[key]
	if !exists {
		return
	}

	for i, h := range handlers {
		if isSameHandler(h, handler) {
			em.moduleWatchers[moduleName][key] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	// 如果没有监听器了，删除该键
	if len(em.moduleWatchers[moduleName][key]) == 0 {
		delete(em.moduleWatchers[moduleName], key)
	}

	// 如果模块没有监听器了，删除该模块
	if len(em.moduleWatchers[moduleName]) == 0 {
		delete(em.moduleWatchers, moduleName)
	}
}

// NotifyGlobalChange 通知全局配置变更
func (em *EventManager) NotifyGlobalChange(key string, oldValue, newValue any) {
	em.mu.RLock()
	handlers, exists := em.globalWatchers[key]
	if exists {
		handlers = append([]ConfigChangeHandler(nil), handlers...)
	}
	em.mu.RUnlock()
	if !exists {
		return
	}

	event := ConfigChangeEvent{
		Key:      key,
		OldValue: oldValue,
		NewValue: newValue,
		Time:     time.Now(),
	}

	// 异步执行监听器，避免阻塞
	go func() {
		for _, handler := range handlers {
			if handler == nil {
				continue
			}
			handler(event)
		}
	}()
}

// NotifyModuleChange 通知模块配置变更
func (em *EventManager) NotifyModuleChange(moduleName, key string, oldValue, newValue any) {
	em.mu.RLock()
	moduleWatchers, exists := em.moduleWatchers[moduleName]
	if !exists {
		em.mu.RUnlock()
		return
	}

	handlers, exists := moduleWatchers[key]
	if !exists {
		em.mu.RUnlock()
		return
	}
	handlers = append([]ConfigChangeHandler(nil), handlers...)
	em.mu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	event := ConfigChangeEvent{
		Key:      moduleName + "." + key,
		OldValue: oldValue,
		NewValue: newValue,
		Time:     time.Now(),
	}

	// 异步执行监听器，避免阻塞
	go func() {
		for _, handler := range handlers {
			if handler == nil {
				continue
			}
			handler(event)
		}
	}()
}

// GetGlobalWatcherCount 获取全局配置监听器数量
func (em *EventManager) GetGlobalWatcherCount(key string) int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	handlers, exists := em.globalWatchers[key]
	if !exists {
		return 0
	}

	return len(handlers)
}

// GetModuleWatcherCount 获取模块配置监听器数量
func (em *EventManager) GetModuleWatcherCount(moduleName, key string) int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	moduleWatchers, exists := em.moduleWatchers[moduleName]
	if !exists {
		return 0
	}

	handlers, exists := moduleWatchers[key]
	if !exists {
		return 0
	}

	return len(handlers)
}
