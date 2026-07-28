package configuration

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultConfigManager 默认配置管理器实例
var DefaultConfigManager ConfigManager

// InitDefaultConfigManager 初始化默认配置管理器
func InitDefaultConfigManager(configDir string) error {
	// 如果未指定配置目录，使用默认值
	if configDir == "" {
		// 尝试从环境变量获取配置目录
		if envConfigDir := os.Getenv("CONFIG_PATH"); envConfigDir != "" {
			configDir = envConfigDir
		} else {
			// 使用当前工作目录下的config目录
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			configDir = filepath.Join(wd, "config")
		}
	}

	// 创建配置选项
	options := &ConfigOptions{
		ConfigDir:       configDir,
		WatchInterval:   time.Second * 5,
		EnableHotReload: true,
		Validator:       &defaultValidator{},
	}

	// 创建配置管理器
	manager, err := NewConfigManager(options)
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}

	DefaultConfigManager = manager
	return nil
}

// GetDefaultConfigManager 获取默认配置管理器
func GetDefaultConfigManager() ConfigManager {
	return DefaultConfigManager
}

// MustInitDefaultConfigManager 初始化默认配置管理器，如果失败则panic
// 注意：此函数在初始化失败时会panic，仅用于测试或程序启动阶段
// 生产环境建议使用 InitDefaultConfigManager 并处理错误
func MustInitDefaultConfigManager(configDir string) {
	if err := InitDefaultConfigManager(configDir); err != nil {
		panic(fmt.Sprintf("Failed to initialize default config manager: %v", err))
	}
}

// SafeInitDefaultConfigManager 安全初始化默认配置管理器，返回错误而不是panic
// 这是 MustInitDefaultConfigManager 的安全替代版本
func SafeInitDefaultConfigManager(configDir string) error {
	return InitDefaultConfigManager(configDir)
}

// CreateConfigManager 创建配置管理器（工厂函数）
func CreateConfigManager(options *ConfigOptions) (ConfigManager, error) {
	return NewConfigManager(options)
}

// CreateConfigManagerWithDir 使用配置目录创建配置管理器
func CreateConfigManagerWithDir(configDir string, enableHotReload bool) (ConfigManager, error) {
	options := &ConfigOptions{
		ConfigDir:       configDir,
		WatchInterval:   time.Second * 5,
		EnableHotReload: enableHotReload,
		Validator:       &defaultValidator{},
	}
	return NewConfigManager(options)
}

// CreateSimpleConfigManager 创建简单的配置管理器（无热加载）
func CreateSimpleConfigManager(configDir string) (ConfigManager, error) {
	return CreateConfigManagerWithDir(configDir, false)
}

// CreateConfigManagerWithValidator 创建带验证器的配置管理器
func CreateConfigManagerWithValidator(configDir string, validator ConfigValidator) (ConfigManager, error) {
	options := &ConfigOptions{
		ConfigDir:       configDir,
		WatchInterval:   time.Second * 5,
		EnableHotReload: true,
		Validator:       validator,
	}
	return NewConfigManager(options)
}

// Helper functions for common configuration operations

// GetString 获取字符串配置值
func GetString(key string) (string, error) {
	if DefaultConfigManager == nil {
		return "", fmt.Errorf("default config manager not initialized")
	}

	value, err := DefaultConfigManager.Get(key)
	if err != nil {
		return "", err
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("config value is not a string: %s", key)
	}

	return str, nil
}

// GetStringWithDefault 获取字符串配置值，如果不存在则返回默认值
func GetStringWithDefault(key, defaultValue string) string {
	if DefaultConfigManager == nil {
		return defaultValue
	}

	value := DefaultConfigManager.GetWithDefault(key, defaultValue)
	str, ok := value.(string)
	if !ok {
		return defaultValue
	}

	return str
}

// GetInt 获取整数配置值
func GetInt(key string) (int, error) {
	if DefaultConfigManager == nil {
		return 0, fmt.Errorf("default config manager not initialized")
	}

	value, err := DefaultConfigManager.Get(key)
	if err != nil {
		return 0, err
	}

	// 尝试转换为整数
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("config value is not an integer: %s", key)
	}
}

// GetIntWithDefault 获取整数配置值，如果不存在则返回默认值
func GetIntWithDefault(key string, defaultValue int) int {
	if DefaultConfigManager == nil {
		return defaultValue
	}

	value := DefaultConfigManager.GetWithDefault(key, defaultValue)

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultValue
	}
}

// GetBool 获取布尔配置值
func GetBool(key string) (bool, error) {
	if DefaultConfigManager == nil {
		return false, fmt.Errorf("default config manager not initialized")
	}

	value, err := DefaultConfigManager.Get(key)
	if err != nil {
		return false, err
	}

	boolVal, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("config value is not a boolean: %s", key)
	}

	return boolVal, nil
}

// GetBoolWithDefault 获取布尔配置值，如果不存在则返回默认值
func GetBoolWithDefault(key string, defaultValue bool) bool {
	if DefaultConfigManager == nil {
		return defaultValue
	}

	value := DefaultConfigManager.GetWithDefault(key, defaultValue)
	boolVal, ok := value.(bool)
	if !ok {
		return defaultValue
	}

	return boolVal
}

// GetModuleString 获取模块字符串配置值
func GetModuleString(moduleName, key string) (string, error) {
	if DefaultConfigManager == nil {
		return "", fmt.Errorf("default config manager not initialized")
	}

	value, err := DefaultConfigManager.GetModuleConfig(moduleName, key)
	if err != nil {
		return "", err
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("module config value is not a string: %s.%s", moduleName, key)
	}

	return str, nil
}

// GetModuleStringWithDefault 获取模块字符串配置值，如果不存在则返回默认值
func GetModuleStringWithDefault(moduleName, key, defaultValue string) string {
	if DefaultConfigManager == nil {
		return defaultValue
	}

	value := DefaultConfigManager.GetModuleConfigWithDefault(moduleName, key, defaultValue)
	str, ok := value.(string)
	if !ok {
		return defaultValue
	}

	return str
}

// WatchConfig 监听配置变更（使用默认配置管理器）
func WatchConfig(key string, handler ConfigChangeHandler) error {
	if DefaultConfigManager == nil {
		return fmt.Errorf("default config manager not initialized")
	}

	return DefaultConfigManager.Watch(key, handler)
}

// WatchModuleConfig 监听模块配置变更（使用默认配置管理器）
func WatchModuleConfig(moduleName, key string, handler ConfigChangeHandler) error {
	if DefaultConfigManager == nil {
		return fmt.Errorf("default config manager not initialized")
	}

	return DefaultConfigManager.WatchModule(moduleName, key, handler)
}

// GetSection 获取指定section的配置并反序列化为对象（使用默认配置管理器）
func GetSection(sectionPath string, target any) error {
	if DefaultConfigManager == nil {
		return fmt.Errorf("default config manager not initialized")
	}

	return DefaultConfigManager.GetSection(sectionPath, target)
}

// WatchSection 监听section配置变更（使用默认配置管理器）
func WatchSection(sectionPath string, handler ConfigChangeHandler) error {
	if DefaultConfigManager == nil {
		return fmt.Errorf("default config manager not initialized")
	}

	return DefaultConfigManager.WatchSection(sectionPath, handler)
}

// UnwatchSection 取消section配置监听（使用默认配置管理器）
func UnwatchSection(sectionPath string, handler ConfigChangeHandler) error {
	if DefaultConfigManager == nil {
		return fmt.Errorf("default config manager not initialized")
	}

	return DefaultConfigManager.UnwatchSection(sectionPath, handler)
}

// ReloadConfig 重新加载所有配置（使用默认配置管理器）
func ReloadConfig() error {
	if DefaultConfigManager == nil {
		return fmt.Errorf("default config manager not initialized")
	}

	return DefaultConfigManager.Reload()
}

// CloseConfigManager 关闭配置管理器（使用默认配置管理器）
func CloseConfigManager() error {
	if DefaultConfigManager == nil {
		return nil // 如果未初始化，直接返回成功
	}

	return DefaultConfigManager.Close()
}

// GetFloat64 获取浮点数配置值
func GetFloat64(key string) (float64, error) {
	if DefaultConfigManager == nil {
		return 0, fmt.Errorf("default config manager not initialized")
	}

	value, err := DefaultConfigManager.Get(key)
	if err != nil {
		return 0, err
	}

	// 尝试转换为浮点数
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("config value is not a float: %s", key)
	}
}

// GetFloat64WithDefault 获取浮点数配置值，如果不存在则返回默认值
func GetFloat64WithDefault(key string, defaultValue float64) float64 {
	if DefaultConfigManager == nil {
		return defaultValue
	}

	value := DefaultConfigManager.GetWithDefault(key, defaultValue)

	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return defaultValue
	}
}

// GetModuleInt 获取模块整数配置值
func GetModuleInt(moduleName, key string) (int, error) {
	if DefaultConfigManager == nil {
		return 0, fmt.Errorf("default config manager not initialized")
	}

	value, err := DefaultConfigManager.GetModuleConfig(moduleName, key)
	if err != nil {
		return 0, err
	}

	// 尝试转换为整数
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("module config value is not an integer: %s.%s", moduleName, key)
	}
}

// GetModuleIntWithDefault 获取模块整数配置值，如果不存在则返回默认值
func GetModuleIntWithDefault(moduleName, key string, defaultValue int) int {
	if DefaultConfigManager == nil {
		return defaultValue
	}

	value := DefaultConfigManager.GetModuleConfigWithDefault(moduleName, key, defaultValue)

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return defaultValue
	}
}

// GetModuleBool 获取模块布尔配置值
func GetModuleBool(moduleName, key string) (bool, error) {
	if DefaultConfigManager == nil {
		return false, fmt.Errorf("default config manager not initialized")
	}

	value, err := DefaultConfigManager.GetModuleConfig(moduleName, key)
	if err != nil {
		return false, err
	}

	boolVal, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("module config value is not a boolean: %s.%s", moduleName, key)
	}

	return boolVal, nil
}

// GetModuleBoolWithDefault 获取模块布尔配置值，如果不存在则返回默认值
func GetModuleBoolWithDefault(moduleName, key string, defaultValue bool) bool {
	if DefaultConfigManager == nil {
		return defaultValue
	}

	value := DefaultConfigManager.GetModuleConfigWithDefault(moduleName, key, defaultValue)
	boolVal, ok := value.(bool)
	if !ok {
		return defaultValue
	}

	return boolVal
}

// IsConfigManagerInitialized 检查配置管理器是否已初始化
func IsConfigManagerInitialized() bool {
	return DefaultConfigManager != nil
}

// GetConfigManager 获取当前默认配置管理器实例
func GetConfigManager() ConfigManager {
	return DefaultConfigManager
}

// ExportAllConfigs 导出所有配置项为JSON对象，保留层级结构（使用默认配置管理器）
func ExportAllConfigs() (map[string]any, error) {
	if DefaultConfigManager == nil {
		return nil, fmt.Errorf("default config manager not initialized")
	}

	return DefaultConfigManager.ExportAllConfigs()
}
