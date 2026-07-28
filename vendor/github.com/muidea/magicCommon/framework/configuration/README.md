# 配置管理框架

基于TOML格式的配置管理框架，支持全局配置和模块隔离配置，提供环境变量注入、热加载和配置变更通知功能。

## 特性

- **TOML格式支持**: 使用TOML作为配置文件格式，支持复杂的数据结构
- **配置隔离**: 全局配置和模块配置完全隔离，避免配置冲突
- **优先级机制**: 环境变量 > 全局配置 > 模块配置
- **热加载**: 配置文件变更时自动重新加载配置
- **事件通知**: 观察者模式支持配置变更通知
- **环境变量注入**: 支持通过环境变量覆盖配置值
- **类型安全**: 提供类型安全的配置获取方法

## 目录结构

```
config/
├── application.toml          # 全局配置文件
└── config.d/                 # 模块配置目录
    ├── payment.toml          # 支付模块配置
    ├── auth.toml             # 认证模块配置
    └── ...                   # 其他模块配置
```

## 快速开始

### 1. 基本使用

```go
package main

import (
    "fmt"
    "magicCommon/framework/configuration"
)

func main() {
    // 初始化配置管理器
    if err := configuration.InitDefaultConfigManager("./config"); err != nil {
        panic(err)
    }

    // 获取全局配置
    port, err := configuration.GetInt("server.port")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Server port: %d\n", port)
    }

    // 获取带默认值的配置
    host := configuration.GetStringWithDefault("server.host", "localhost")
    fmt.Printf("Server host: %s\n", host)
}
```

### 2. 模块配置使用

```go
// 获取模块配置
apiKey, err := configuration.GetModuleString("payment", "api_key")
if err != nil {
    fmt.Printf("Error: %v\n", err)
} else {
    fmt.Printf("Payment API key: %s\n", apiKey)
}

// 获取带默认值的模块配置
timeout := configuration.GetModuleStringWithDefault("payment", "timeout", "30s")
fmt.Printf("Payment timeout: %s\n", timeout)
```

### 3. 配置监听

```go
// 监听全局配置变更
configuration.WatchConfig("server.port", func(event configuration.ConfigChangeEvent) {
    fmt.Printf("Config changed: %s, old: %v, new: %v\n", 
        event.Key, event.OldValue, event.NewValue)
})

// 监听模块配置变更
configuration.WatchModuleConfig("payment", "api_key", func(event configuration.ConfigChangeEvent) {
    fmt.Printf("Module config changed: %s, old: %v, new: %v\n", 
        event.Key, event.OldValue, event.NewValue)
})
```

### 4. 高级使用

```go
// 创建自定义配置管理器
manager, err := configuration.CreateConfigManagerWithDir("./config", true)
if err != nil {
    panic(err)
}
defer manager.Close()

// 直接使用管理器接口
value, err := manager.Get("server.host")
if err != nil {
    fmt.Printf("Error: %v\n", err)
} else {
    fmt.Printf("Server host: %v\n", value)
}
```

## 配置优先级

配置项的优先级从高到低：

1. **环境变量**: 最高优先级，可以覆盖任何配置
2. **全局配置**: `application.toml` 文件中的配置
3. **模块配置**: `config.d/*.toml` 文件中的配置（完全隔离）

### 环境变量命名规则

环境变量会自动转换为配置键名：

- `SERVER_PORT` → `server.port`
- `DATABASE_HOST` → `database.host`
- `APP_NAME` → `app_name`

## 配置文件示例

### 全局配置 (application.toml)

```toml
app_name = "My Application"
version = "1.0.0"

[server]
host = "0.0.0.0"
port = 8080

[database]
host = "localhost"
port = 5432
```

### 模块配置 (config.d/payment.toml)

```toml
api_key = "pk_test_1234567890abcdef"

[gateway]
url = "https://api.payment.com/v1"
timeout = "30s"

[methods]
credit_card = true
paypal = false
```

## API 参考

### 全局配置获取方法

- `GetString(key string) (string, error)`
- `GetStringWithDefault(key, defaultValue string) string`
- `GetInt(key string) (int, error)`
- `GetIntWithDefault(key string, defaultValue int) int`
- `GetBool(key string) (bool, error)`
- `GetBoolWithDefault(key string, defaultValue bool) bool`

### 模块配置获取方法

- `GetModuleString(moduleName, key string) (string, error)`
- `GetModuleStringWithDefault(moduleName, key, defaultValue string) string`

### 监听方法

- `WatchConfig(key string, handler ConfigChangeHandler) error`
- `WatchModuleConfig(moduleName, key string, handler ConfigChangeHandler) error`

### 配置管理器接口

```go
type ConfigManager interface {
    Get(key string) (any, error)
    GetWithDefault(key string, defaultValue any) any
    GetModuleConfig(moduleName, key string) (any, error)
    GetModuleConfigWithDefault(moduleName, key string, defaultValue any) any
    Watch(key string, handler ConfigChangeHandler) error
    WatchModule(moduleName, key string, handler ConfigChangeHandler) error
    Reload() error
    Close() error
}
```

## 热加载机制

配置管理框架支持热加载功能：

1. **文件监听**: 自动监听配置文件的变更
2. **增量加载**: 只重新加载变更的文件
3. **配置验证**: 加载前验证配置的有效性
4. **事件通知**: 配置变更时通知所有监听器
5. **错误恢复**: 单个文件加载失败不影响其他配置

## 最佳实践

1. **配置组织**: 将相关配置组织在同一配置块中
2. **模块隔离**: 不同模块的配置使用独立的配置文件
3. **环境变量**: 敏感信息通过环境变量注入
4. **默认值**: 为可选配置项提供合理的默认值
5. **配置验证**: 实现自定义验证器确保配置有效性

## 故障排除

### 常见问题

1. **配置项找不到**: 检查配置键名是否正确，注意大小写和路径分隔符
2. **模块配置加载失败**: 检查模块配置文件是否存在且格式正确
3. **热加载不工作**: 检查文件权限和监听器是否正常工作
4. **环境变量不生效**: 检查环境变量命名是否符合转换规则

### 调试模式

启用调试日志来查看配置加载过程：

```go
// 设置环境变量
os.Setenv("DEBUG_CONFIG", "true")
```

## 许可证

MIT License