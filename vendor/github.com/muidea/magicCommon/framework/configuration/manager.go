package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	fp "github.com/muidea/magicCommon/foundation/path"
	"log/slog"
)

// ConfigManagerImpl 配置管理器实现
type ConfigManagerImpl struct {
	options       *ConfigOptions
	loader        ConfigLoader
	envMerger     *EnvConfigMerger
	eventManager  *EventManager
	fileWatcher   FileWatcher
	globalConfig  map[string]any
	appConfig     map[string]any // 原始应用程序配置（不包含环境变量）
	moduleConfigs map[string]map[string]any
	mu            sync.RWMutex
	closed        bool
	debugMode     bool // 调试模式开关
}

// NewConfigManager 创建配置管理器
func NewConfigManager(options *ConfigOptions) (*ConfigManagerImpl, error) {
	if options == nil {
		options = DefaultConfigOptions()
	}

	// 创建配置加载器
	loader := NewTOMLConfigLoader(options.ConfigDir)

	// 创建环境变量合并器（使用空前缀，处理所有环境变量）
	envMerger := NewEnvConfigMerger("")

	// 创建事件管理器
	eventManager := NewEventManager()

	// 创建文件监听器
	var fileWatcher FileWatcher
	var err error
	if options.EnableHotReload {
		fileWatcher, err = NewFileWatcher()
		if err != nil {
			// 回退到简单文件监听器
			simpleWatcher := NewSimpleFileWatcher(options.WatchInterval)
			simpleWatcher.Start()
			fileWatcher = simpleWatcher
		}
	}

	// 检查是否启用调试模式
	debugMode := os.Getenv("DEBUG_MODE") == "true" || os.Getenv("DEBUG_MODE") == "1"

	manager := &ConfigManagerImpl{
		options:       options,
		loader:        loader,
		envMerger:     envMerger,
		eventManager:  eventManager,
		fileWatcher:   fileWatcher,
		globalConfig:  make(map[string]any),
		appConfig:     make(map[string]any),
		moduleConfigs: make(map[string]map[string]any),
		closed:        false,
		debugMode:     debugMode,
	}

	// 初始加载配置
	if err := manager.loadAllConfigs(); err != nil {
		return nil, fmt.Errorf("failed to load initial configs: %w", err)
	}

	// 设置文件监听
	if options.EnableHotReload && fileWatcher != nil {
		if err := manager.setupFileWatching(); err != nil {
			return nil, fmt.Errorf("failed to setup file watching: %w", err)
		}
	}

	return manager, nil
}

// getNestedValue 获取嵌套配置值
func (m *ConfigManagerImpl) getNestedValue(config map[string]any, key string) (any, error) {
	parts := strings.Split(key, ".")
	current := config

	for i, part := range parts {
		value, exists := current[part]
		if !exists {
			return nil, fmt.Errorf("config key not found: %s", key)
		}

		// 如果是最后一个部分，直接返回值
		if i == len(parts)-1 {
			return value, nil
		}

		// 如果不是最后一个部分，需要继续深入嵌套结构
		if nestedMap, ok := value.(map[string]any); ok {
			current = nestedMap
		} else {
			return nil, fmt.Errorf("config key %s is not a nested structure", strings.Join(parts[:i+1], "."))
		}
	}

	return nil, fmt.Errorf("config key not found: %s", key)
}

// IsDebugMode 检查是否启用调试模式
func (m *ConfigManagerImpl) IsDebugMode() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.debugMode
}

// SetDebugMode 设置调试模式
func (m *ConfigManagerImpl) SetDebugMode(debug bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugMode = debug
}

// Get 获取全局配置项（支持嵌套配置访问）
func (m *ConfigManagerImpl) Get(key string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, fmt.Errorf("config manager is closed")
	}

	// 如果key包含点号，尝试解析嵌套配置
	if strings.Contains(key, ".") {
		value, err := m.getNestedValue(m.globalConfig, key)
		if err != nil && m.debugMode {
			slog.Error("[CONFIG DEBUG] Config key not found", "key", key)
		}
		return value, err
	}

	// 否则使用原来的简单查找
	value, exists := m.globalConfig[key]
	if !exists {
		if m.debugMode {
			slog.Error("[CONFIG DEBUG] Config key not found", "key", key)
		}
		return nil, fmt.Errorf("config key not found: %s", key)
	}

	return value, nil
}

// GetWithDefault 获取全局配置项，如果不存在则返回默认值
func (m *ConfigManagerImpl) GetWithDefault(key string, defaultValue any) any {
	value, err := m.Get(key)
	if err != nil {
		return defaultValue
	}
	return value
}

// GetModuleConfig 获取模块隔离配置项（支持嵌套配置访问）
func (m *ConfigManagerImpl) GetModuleConfig(moduleName, key string) (any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, fmt.Errorf("config manager is closed")
	}

	moduleConfig, exists := m.moduleConfigs[moduleName]
	if !exists {
		if m.debugMode {
			slog.Error("[CONFIG DEBUG] Module not found", "module", moduleName)
		}
		return nil, fmt.Errorf("module not found: %s", moduleName)
	}

	// 如果key包含点号，尝试解析嵌套配置
	if strings.Contains(key, ".") {
		value, err := m.getNestedValue(moduleConfig, key)
		if err != nil && m.debugMode {
			slog.Error("[CONFIG DEBUG] Config key not found in module %s: %s", moduleName, key)
		}
		return value, err
	}

	// 否则使用原来的简单查找
	value, exists := moduleConfig[key]
	if !exists {
		if m.debugMode {
			slog.Error("[CONFIG DEBUG] Config key not found in module %s: %s", moduleName, key)
		}
		return nil, fmt.Errorf("config key not found in module %s: %s", moduleName, key)
	}

	return value, nil
}

// GetModuleConfigWithDefault 获取模块隔离配置项，如果不存在则返回默认值
func (m *ConfigManagerImpl) GetModuleConfigWithDefault(moduleName, key string, defaultValue any) any {
	value, err := m.GetModuleConfig(moduleName, key)
	if err != nil {
		return defaultValue
	}
	return value
}

// ExportAllConfigs 导出所有配置项为JSON对象，保留层级结构
// 只包含应用程序配置，不包含系统环境变量
func (m *ConfigManagerImpl) ExportAllConfigs() (map[string]any, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return nil, fmt.Errorf("config manager is closed")
	}

	// 创建结果map
	result := make(map[string]any)

	// 复制应用程序配置（不包含环境变量）
	appConfig := make(map[string]any)
	for k, v := range m.appConfig {
		appConfig[k] = m.deepCopyValue(v)
	}
	result["application"] = appConfig

	// 复制模块配置
	moduleConfigs := make(map[string]any)
	for moduleName, moduleConfig := range m.moduleConfigs {
		moduleCopy := make(map[string]any)
		for k, v := range moduleConfig {
			moduleCopy[k] = m.deepCopyValue(v)
		}
		moduleConfigs[moduleName] = moduleCopy
	}
	result["modules"] = moduleConfigs

	return result, nil
}

// deepCopyValue 深度复制配置值，确保返回的是可安全序列化的值
func (m *ConfigManagerImpl) deepCopyValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		// 递归复制map
		copyMap := make(map[string]any)
		for key, val := range v {
			copyMap[key] = m.deepCopyValue(val)
		}
		return copyMap
	case []any:
		// 递归复制slice
		copySlice := make([]any, len(v))
		for i, val := range v {
			copySlice[i] = m.deepCopyValue(val)
		}
		return copySlice
	default:
		// 基本类型直接返回
		return v
	}
}

// GetSection 获取指定section的配置并反序列化为对象
func (m *ConfigManagerImpl) GetSection(sectionPath string, target any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	// 获取section的配置值
	sectionValue, err := m.getNestedValue(m.globalConfig, sectionPath)
	if err != nil {
		return fmt.Errorf("failed to get section %s: %w", sectionPath, err)
	}

	// 将配置值转换为JSON格式
	sectionMap, ok := sectionValue.(map[string]any)
	if !ok {
		return fmt.Errorf("section %s is not a map structure", sectionPath)
	}

	// 将map转换为JSON字节
	jsonBytes, err := json.Marshal(sectionMap)
	if err != nil {
		return fmt.Errorf("failed to marshal section %s to JSON: %w", sectionPath, err)
	}

	// 将JSON反序列化到目标对象
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal section %s to target object: %w", sectionPath, err)
	}

	return nil
}

// Watch 监听配置变更
func (m *ConfigManagerImpl) Watch(key string, handler ConfigChangeHandler) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	m.eventManager.RegisterGlobalWatcher(key, handler)
	return nil
}

// WatchModule 监听模块配置变更
func (m *ConfigManagerImpl) WatchModule(moduleName, key string, handler ConfigChangeHandler) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	m.eventManager.RegisterModuleWatcher(moduleName, key, handler)
	return nil
}

// Unwatch 取消监听
func (m *ConfigManagerImpl) Unwatch(key string, handler ConfigChangeHandler) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	m.eventManager.UnregisterGlobalWatcher(key, handler)
	return nil
}

// UnwatchModule 取消模块配置监听
func (m *ConfigManagerImpl) UnwatchModule(moduleName, key string, handler ConfigChangeHandler) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	m.eventManager.UnregisterModuleWatcher(moduleName, key, handler)
	return nil
}

// WatchSection 监听section配置变更
func (m *ConfigManagerImpl) WatchSection(sectionPath string, handler ConfigChangeHandler) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	// 使用全局配置监听器来监听section变更
	m.eventManager.RegisterGlobalWatcher(sectionPath, handler)
	return nil
}

// UnwatchSection 取消section配置监听
func (m *ConfigManagerImpl) UnwatchSection(sectionPath string, handler ConfigChangeHandler) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	m.eventManager.UnregisterGlobalWatcher(sectionPath, handler)
	return nil
}

// Reload 重新加载所有配置
func (m *ConfigManagerImpl) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return fmt.Errorf("config manager is closed")
	}

	return m.loadAllConfigs()
}

// Close 关闭配置管理器
func (m *ConfigManagerImpl) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true

	if m.fileWatcher != nil {
		return m.fileWatcher.Close()
	}

	return nil
}

// loadAllConfigs 加载所有配置
func (m *ConfigManagerImpl) loadAllConfigs() error {
	// 保存旧配置用于比较
	oldGlobalConfig := m.globalConfig
	oldModuleConfigs := m.moduleConfigs

	// 加载全局配置
	appConfig, err := m.loader.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("failed to load global config: %w", err)
	}

	// 保存原始应用程序配置
	m.appConfig = appConfig

	// 合并环境变量配置
	globalConfig, err := m.envMerger.Merge(appConfig)
	if err != nil {
		return fmt.Errorf("failed to merge env config: %w", err)
	}

	// 验证全局配置
	if m.options.Validator != nil {
		if err := m.options.Validator.ValidateGlobalConfig(globalConfig); err != nil {
			return fmt.Errorf("global config validation failed: %w", err)
		}
	}

	// 加载模块配置
	moduleConfigs, err := m.loader.LoadAllModuleConfigs()
	if err != nil {
		// 模块配置加载失败不影响全局配置
		slog.Warn("Failed to load some module configs", "error", err)
	}

	// 验证模块配置
	if m.options.Validator != nil {
		for moduleName, config := range moduleConfigs {
			if err := m.options.Validator.ValidateModuleConfig(moduleName, config); err != nil {
				slog.Warn("Module config validation failed", "module", moduleName, "error", err)
				// 验证失败的模块配置不加载
				delete(moduleConfigs, moduleName)
			}
		}
	}

	// 更新配置
	m.globalConfig = globalConfig
	m.moduleConfigs = moduleConfigs

	// 触发配置变更事件
	m.triggerConfigChangeEvents(oldGlobalConfig, oldModuleConfigs)

	return nil
}

// triggerConfigChangeEvents 触发配置变更事件
func (m *ConfigManagerImpl) triggerConfigChangeEvents(oldGlobalConfig map[string]any, oldModuleConfigs map[string]map[string]any) {
	// 检查全局配置变更
	for key, newValue := range m.globalConfig {
		oldValue, exists := oldGlobalConfig[key]
		if !exists || !m.valuesEqual(oldValue, newValue) {
			m.eventManager.NotifyGlobalChange(key, oldValue, newValue)
		}
	}

	// 检查被删除的全局配置项
	for key, oldValue := range oldGlobalConfig {
		if _, exists := m.globalConfig[key]; !exists {
			m.eventManager.NotifyGlobalChange(key, oldValue, nil)
		}
	}

	// 检查模块配置变更
	for moduleName, newModuleConfig := range m.moduleConfigs {
		oldModuleConfig, exists := oldModuleConfigs[moduleName]
		if !exists {
			// 新模块
			for key, newValue := range newModuleConfig {
				m.eventManager.NotifyModuleChange(moduleName, key, nil, newValue)
			}
			continue
		}

		// 检查变更的配置项
		for key, newValue := range newModuleConfig {
			oldValue, exists := oldModuleConfig[key]
			if !exists || !m.valuesEqual(oldValue, newValue) {
				m.eventManager.NotifyModuleChange(moduleName, key, oldValue, newValue)
			}
		}

		// 检查被删除的模块配置项
		for key, oldValue := range oldModuleConfig {
			if _, exists := newModuleConfig[key]; !exists {
				m.eventManager.NotifyModuleChange(moduleName, key, oldValue, nil)
			}
		}
	}

	// 检查被删除的模块
	for moduleName, oldModuleConfig := range oldModuleConfigs {
		if _, exists := m.moduleConfigs[moduleName]; !exists {
			for key, oldValue := range oldModuleConfig {
				m.eventManager.NotifyModuleChange(moduleName, key, oldValue, nil)
			}
		}
	}
}

// valuesEqual 比较两个值是否相等
func (m *ConfigManagerImpl) valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// 对于基本类型，使用直接比较
	switch aVal := a.(type) {
	case string, int, int64, float64, bool:
		return a == b
	case map[string]any:
		// 对于map类型，进行深度比较
		if bVal, ok := b.(map[string]any); ok {
			return m.compareMaps(aVal, bVal)
		}
		return false
	case []any:
		// 对于slice类型，进行深度比较
		if bVal, ok := b.([]any); ok {
			return m.compareSlices(aVal, bVal)
		}
		return false
	default:
		// 对于其他类型，使用字符串表示进行比较
		return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
	}
}

// compareMaps 比较两个map是否相等
func (m *ConfigManagerImpl) compareMaps(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}

	for key, aVal := range a {
		bVal, exists := b[key]
		if !exists {
			return false
		}
		if !m.valuesEqual(aVal, bVal) {
			return false
		}
	}

	return true
}

// compareSlices 比较两个slice是否相等
func (m *ConfigManagerImpl) compareSlices(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !m.valuesEqual(a[i], b[i]) {
			return false
		}
	}

	return true
}

// setupFileWatching 设置文件监听
func (m *ConfigManagerImpl) setupFileWatching() error {
	if m.fileWatcher == nil {
		return nil
	}

	// 监听全局配置文件
	globalConfigPath := filepath.Join(m.options.ConfigDir, "application.toml")
	if fp.Exist(globalConfigPath) {
		if err := m.fileWatcher.Watch(globalConfigPath, func() {
			slog.Info("Global config file changed, reloading...")
			_ = m.Reload()
		}); err != nil {
			slog.Error("Failed to watch global config file:%s, err:%v", globalConfigPath, err)
		}
	}

	// 监听模块配置目录
	moduleConfigDir := filepath.Join(m.options.ConfigDir, "config.d")
	if fp.Exist(moduleConfigDir) {
		if err := m.fileWatcher.WatchDirectory(moduleConfigDir, func(filePath string) {
			if strings.HasSuffix(filePath, ".toml") {
				slog.Info("Module config file changed", "file", filePath)
				_ = m.Reload()
			}
		}); err != nil {
			slog.Error("Failed to watch module config directory", "directory", moduleConfigDir, "error", err)
		}
	}

	return nil
}

// GetGlobalConfig 获取全局配置（用于调试）
func (m *ConfigManagerImpl) GetGlobalConfig() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := make(map[string]any)
	for k, v := range m.globalConfig {
		config[k] = v
	}
	return config
}

// GetModuleNames 获取所有模块名称（用于调试）
func (m *ConfigManagerImpl) GetModuleNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	modules := make([]string, 0, len(m.moduleConfigs))
	for moduleName := range m.moduleConfigs {
		modules = append(modules, moduleName)
	}
	return modules
}
