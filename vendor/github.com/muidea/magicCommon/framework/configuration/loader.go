package configuration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// TOMLConfigLoader TOML配置加载器
type TOMLConfigLoader struct {
	configDir string
}

// NewTOMLConfigLoader 创建TOML配置加载器
func NewTOMLConfigLoader(configDir string) *TOMLConfigLoader {
	return &TOMLConfigLoader{
		configDir: configDir,
	}
}

// LoadGlobalConfig 加载全局配置
func (l *TOMLConfigLoader) LoadGlobalConfig() (map[string]any, error) {
	globalConfigPath := filepath.Join(l.configDir, "application.toml")

	if _, err := os.Stat(globalConfigPath); os.IsNotExist(err) {
		// 全局配置文件不存在，返回空配置
		return make(map[string]any), nil
	}

	data, err := os.ReadFile(globalConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config file: %w", err)
	}

	var config map[string]any
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse global config file: %w", err)
	}

	return config, nil
}

// LoadModuleConfig 加载模块配置
func (l *TOMLConfigLoader) LoadModuleConfig(moduleName string) (map[string]any, error) {
	configDir := filepath.Join(l.configDir, "config.d")
	moduleConfigPath := filepath.Join(configDir, fmt.Sprintf("%s.toml", moduleName))

	if _, err := os.Stat(moduleConfigPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("module config file not found: %s", moduleConfigPath)
	}

	data, err := os.ReadFile(moduleConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read module config file: %w", err)
	}

	var config map[string]any
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse module config file: %w", err)
	}

	return config, nil
}

// ListModules 列出所有模块
func (l *TOMLConfigLoader) ListModules() ([]string, error) {
	configDir := filepath.Join(l.configDir, "config.d")

	// 检查目录是否存在
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	modules := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".toml") {
			moduleName := strings.TrimSuffix(entry.Name(), ".toml")
			modules = append(modules, moduleName)
		}
	}

	return modules, nil
}

// LoadAllModuleConfigs 加载所有模块配置
func (l *TOMLConfigLoader) LoadAllModuleConfigs() (map[string]map[string]any, error) {
	modules, err := l.ListModules()
	if err != nil {
		return nil, err
	}

	allConfigs := make(map[string]map[string]any)
	for _, module := range modules {
		config, err := l.LoadModuleConfig(module)
		if err != nil {
			// 单个模块加载失败不影响其他模块
			continue
		}
		allConfigs[module] = config
	}

	return allConfigs, nil
}
