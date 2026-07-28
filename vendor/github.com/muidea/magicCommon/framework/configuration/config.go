package configuration

import (
	"time"
)

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	Key      string    // 配置路径 (propertyName 或 moduleName.propertyName)
	OldValue any       // 旧值
	NewValue any       // 新值
	Time     time.Time // 变更时间
}

// ConfigChangeHandler 配置变更处理器
type ConfigChangeHandler func(event ConfigChangeEvent)

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// Get 获取全局配置项
	Get(key string) (any, error)

	// GetWithDefault 获取全局配置项，如果不存在则返回默认值
	GetWithDefault(key string, defaultValue any) any

	// GetModuleConfig 获取模块隔离配置项
	GetModuleConfig(moduleName, key string) (any, error)

	// GetModuleConfigWithDefault 获取模块隔离配置项，如果不存在则返回默认值
	GetModuleConfigWithDefault(moduleName, key string, defaultValue any) any

	// GetSection 获取指定section的配置并反序列化为对象
	GetSection(sectionPath string, target any) error

	// ExportAllConfigs 导出所有配置项为JSON对象，保留层级结构
	ExportAllConfigs() (map[string]any, error)

	// Watch 监听配置变更
	Watch(key string, handler ConfigChangeHandler) error

	// WatchModule 监听模块配置变更
	WatchModule(moduleName, key string, handler ConfigChangeHandler) error

	// WatchSection 监听section配置变更
	WatchSection(sectionPath string, handler ConfigChangeHandler) error

	// Unwatch 取消监听
	Unwatch(key string, handler ConfigChangeHandler) error

	// UnwatchModule 取消模块配置监听
	UnwatchModule(moduleName, key string, handler ConfigChangeHandler) error

	// UnwatchSection 取消section配置监听
	UnwatchSection(sectionPath string, handler ConfigChangeHandler) error

	// Reload 重新加载所有配置
	Reload() error

	// Close 关闭配置管理器
	Close() error

	// IsDebugMode 检查是否启用调试模式
	IsDebugMode() bool

	// SetDebugMode 设置调试模式
	SetDebugMode(debug bool)
}

// ConfigLoader 配置加载器接口
type ConfigLoader interface {
	// LoadGlobalConfig 加载全局配置
	LoadGlobalConfig() (map[string]any, error)

	// LoadModuleConfig 加载模块配置
	LoadModuleConfig(moduleName string) (map[string]any, error)

	// LoadAllModuleConfigs 加载所有模块配置
	LoadAllModuleConfigs() (map[string]map[string]any, error)

	// ListModules 列出所有模块
	ListModules() ([]string, error)
}

// FileWatcher 文件监听器接口
type FileWatcher interface {
	// Watch 监听文件变化
	Watch(filePath string, callback func()) error

	// WatchDirectory 监听目录变化
	WatchDirectory(dirPath string, callback func(filePath string)) error

	// Close 关闭文件监听器
	Close() error
}

// ConfigValidator 配置验证器接口
type ConfigValidator interface {
	// ValidateGlobalConfig 验证全局配置
	ValidateGlobalConfig(config map[string]any) error

	// ValidateModuleConfig 验证模块配置
	ValidateModuleConfig(moduleName string, config map[string]any) error
}

// ConfigOptions 配置选项
type ConfigOptions struct {
	// ConfigDir 配置目录路径
	ConfigDir string

	// WatchInterval 文件监听间隔
	WatchInterval time.Duration

	// EnableHotReload 是否启用热加载
	EnableHotReload bool

	// Validator 配置验证器
	Validator ConfigValidator
}

// DefaultConfigOptions 默认配置选项
func DefaultConfigOptions() *ConfigOptions {
	return &ConfigOptions{
		ConfigDir:       "./config",
		WatchInterval:   time.Second * 5,
		EnableHotReload: true,
		Validator:       &defaultValidator{},
	}
}

// defaultValidator 默认配置验证器
type defaultValidator struct{}

func (v *defaultValidator) ValidateGlobalConfig(config map[string]any) error {
	// 默认实现不进行验证
	return nil
}

func (v *defaultValidator) ValidateModuleConfig(moduleName string, config map[string]any) error {
	// 默认实现不进行验证
	return nil
}
