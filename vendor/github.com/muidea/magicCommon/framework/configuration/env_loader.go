package configuration

import (
	"fmt"
	"os"
	"strings"
)

// EnvConfigLoader 环境变量配置加载器
type EnvConfigLoader struct {
	prefix string
}

// NewEnvConfigLoader 创建环境变量配置加载器
func NewEnvConfigLoader(prefix string) *EnvConfigLoader {
	return &EnvConfigLoader{
		prefix: prefix,
	}
}

// Load 加载环境变量配置
func (l *EnvConfigLoader) Load() (map[string]any, error) {
	config := make(map[string]any)

	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		// 如果设置了前缀，只处理带前缀的环境变量
		if l.prefix != "" && !strings.HasPrefix(key, l.prefix) {
			continue
		}

		// 移除前缀
		if l.prefix != "" {
			key = strings.TrimPrefix(key, l.prefix)
		}

		// 将环境变量名转换为配置键名（下划线转点号，大写转小写）
		configKey := l.normalizeKey(key)
		// 将点号分隔的键名展开为嵌套结构
		l.setNestedValue(config, configKey, l.parseValue(value))
	}

	return config, nil
}

// setNestedValue 设置嵌套配置值
func (l *EnvConfigLoader) setNestedValue(config map[string]any, key string, value any) {
	parts := strings.Split(key, ".")
	current := config

	for i, part := range parts {
		// 如果是最后一个部分，直接设置值
		if i == len(parts)-1 {
			current[part] = value
			return
		}

		// 如果不是最后一个部分，确保下一级映射存在
		if next, exists := current[part]; exists {
			if nextMap, ok := next.(map[string]any); ok {
				current = nextMap
			} else {
				// 如果存在但不是映射，用新的映射替换
				newMap := make(map[string]any)
				current[part] = newMap
				current = newMap
			}
		} else {
			// 如果不存在，创建新的映射
			newMap := make(map[string]any)
			current[part] = newMap
			current = newMap
		}
	}
}

// normalizeKey 规范化键名
func (l *EnvConfigLoader) normalizeKey(key string) string {
	// 将下划线转换为点号
	key = strings.ReplaceAll(key, "_", ".")
	// 转换为小写
	return strings.ToLower(key)
}

// parseValue 解析环境变量值
func (l *EnvConfigLoader) parseValue(value string) any {
	// 尝试解析为布尔值
	if value == "true" || value == "TRUE" {
		return true
	}
	if value == "false" || value == "FALSE" {
		return false
	}

	// 尝试解析为整数（仅当整个字符串都是数字时）
	if intVal, err := parseInt(value); err == nil {
		return intVal
	}

	// 尝试解析为浮点数（仅当整个字符串都是浮点数时）
	if floatVal, err := parseFloat(value); err == nil {
		return floatVal
	}

	// 默认为字符串
	return value
}

// parseInt 尝试解析为整数
func parseInt(s string) (int64, error) {
	var result int64
	n, err := fmt.Sscanf(s, "%d", &result)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("not an integer")
	}
	// 检查是否整个字符串都被解析了
	if fmt.Sprintf("%d", result) != s {
		return 0, fmt.Errorf("not a pure integer")
	}
	return result, nil
}

// parseFloat 尝试解析为浮点数
func parseFloat(s string) (float64, error) {
	var result float64
	n, err := fmt.Sscanf(s, "%f", &result)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("not a float")
	}
	// 检查是否整个字符串都被解析了
	if fmt.Sprintf("%g", result) != s && fmt.Sprintf("%f", result) != s {
		return 0, fmt.Errorf("not a pure float")
	}
	return result, nil
}

// EnvConfigMerger 环境变量配置合并器
type EnvConfigMerger struct {
	envLoader *EnvConfigLoader
}

// NewEnvConfigMerger 创建环境变量配置合并器
func NewEnvConfigMerger(prefix string) *EnvConfigMerger {
	return &EnvConfigMerger{
		envLoader: NewEnvConfigLoader(prefix),
	}
}

// Merge 合并环境变量配置到现有配置中
func (m *EnvConfigMerger) Merge(existingConfig map[string]any) (map[string]any, error) {
	envConfig, err := m.envLoader.Load()
	if err != nil {
		return nil, err
	}

	// 深度合并配置
	mergedConfig := m.deepMerge(existingConfig, envConfig)
	return mergedConfig, nil
}

// deepMerge 深度合并两个配置映射
func (m *EnvConfigMerger) deepMerge(dest, src map[string]any) map[string]any {
	result := make(map[string]any)

	// 先复制目标配置
	for k, v := range dest {
		result[k] = v
	}

	// 然后合并源配置
	for k, srcVal := range src {
		if destVal, exists := result[k]; exists {
			// 如果目标中已存在该键
			if destMap, destOk := destVal.(map[string]any); destOk {
				if srcMap, srcOk := srcVal.(map[string]any); srcOk {
					// 如果都是映射，递归合并
					result[k] = m.deepMerge(destMap, srcMap)
				} else {
					// 如果源不是映射，直接覆盖（环境变量优先级更高）
					result[k] = srcVal
				}
			} else {
				// 如果目标不是映射，直接覆盖
				result[k] = srcVal
			}
		} else {
			// 如果目标中不存在该键，直接添加
			result[k] = srcVal
		}
	}

	return result
}
