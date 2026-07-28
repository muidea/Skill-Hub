# magicCommon/event 模块功能说明

## 概述

`magicCommon/event` 是一个基于 Go 语言实现的高性能、线程安全的事件发布-订阅系统。该模块提供了灵活的事件匹配机制、同步/异步投递能力、上下文传递支持以及类型安全的泛型辅助函数，适用于微服务架构中的事件驱动通信。

## 核心设计理念

1. **发布-订阅模式**：解耦事件生产者与消费者
2. **有序分 lane 调度**：保持同步事件语义，同时将串行边界收敛到 lane
3. **通配符匹配**：支持灵活的事件路由规则
4. **类型安全**：提供泛型辅助函数确保编译时类型检查
5. **错误恢复**：内置 panic 恢复机制，保证系统稳定性

## 目录结构

```
magicCommon/event/
├── event.go      # 事件和结果的基础实现、泛型辅助函数
├── hub.go        # 事件中心核心实现、观察者模式
├── hub_test.go   # 事件中心测试用例
└── event_test.go # 事件和辅助函数测试用例
```

## 核心接口

### Event 接口

```go
type Event interface {
    ID() string                    // 事件ID
    Source() string                // 事件源标识
    Destination() string           // 事件目标标识
    LaneKey() string               // 事件顺序 lane 标识，默认等于 Destination
    Header() Values                // 事件头部（元数据）
    Context() context.Context      // 事件上下文
    BindContext(ctx context.Context) // 绑定上下文
    BindLaneKey(laneKey string)    // 绑定顺序 lane
    Data() any                     // 事件数据
    SetData(key string, val any)   // 设置附加数据
    GetData(key string) any        // 获取附加数据
    Match(pattern string) bool     // 匹配事件模式
}
```

### Result 接口

```go
type Result interface {
    Error() *cd.Error              // 获取错误信息
    Set(data any, err *cd.Error)   // 设置结果和错误
    Get() (any, *cd.Error)         // 获取结果和错误
    SetVal(key string, val any)    // 设置键值对结果
    GetVal(key string) any         // 获取键值对结果
}
```

### Observer 接口

```go
type Observer interface {
    ID() string                    // 观察者ID
    Notify(event Event, result Result) // 事件通知回调
}
```

### Hub 接口

```go
type Hub interface {
    Subscribe(eventID string, observer Observer)   // 订阅事件
    Unsubscribe(eventID string, observer Observer) // 取消订阅
    Post(event Event)                              // 异步发送事件（无返回）
    Send(event Event) Result                       // 同步发送事件（有返回）
    Terminate()                                    // 终止事件中心
}
```

### SimpleObserver 接口

```go
type SimpleObserver interface {
    Observer
    Subscribe(eventID string, observerFunc ObserverFunc) // 订阅事件（函数回调）
    Unsubscribe(eventID string)                          // 取消订阅
}
```

## 核心实现

### Values 类型

`Values` 是一个 `map[string]any` 的别名，提供类型安全的访问方法：

```go
type Values map[string]any

func (s Values) Set(key string, value any)
func (s Values) Get(key string) any
func (s Values) GetString(key string) string
func (s Values) GetInt(key string) int
func (s Values) GetBool(key string) bool
```

### 事件匹配模式

支持 MQTT 风格的通配符匹配：

| 通配符 | 说明 | 示例 |
|--------|------|------|
| `+` | 单级通配符，匹配一个非空段 | `/user/+` 匹配 `/user/123` |
| `#` | 多级通配符，匹配零个或多个段 | `/user/#` 匹配 `/user/123/profile` |
| `:id` | 参数通配符，与 `+` 相同 | `/user/:id` 匹配 `/user/123` |

**匹配规则**：
- 路径使用 `/` 分隔
- 通配符必须匹配非空段（`#` 除外，它可以匹配空段后的内容）
- 匹配算法支持复杂嵌套模式

### hubImpl 实现

`hubImpl` 是事件中心的核心实现，具有以下特点：

1. **线程安全**：使用读写锁保护内部数据结构
2. **异步处理**：通过 `execute.Execute` 管理协程池
3. **通道通信**：使用 action channel 处理订阅/发布操作
4. **优雅关闭**：支持 `Terminate()` 方法安全关闭所有协程

**运行语义**：
- `Post()` 是异步投递，如果内部 channel 在超时窗口内无法接收，当前实现会记录告警并放弃这次投递，而不是无限阻塞调用方。
- `Send()` 是同步投递，如果内部 channel 在超时窗口内无法接收，会返回超时结果。
- `Send()` 和 `Post()` 都按 `LaneKey()` 做顺序调度；同一个 lane 内严格顺序，不同 lane 之间允许并行。
- `LaneKey()` 默认等于 `Destination()`，因此旧代码在不显式设置 lane 时行为保持不变。
- `Terminate()` 是幂等且并发安全的；关闭阶段如果内部执行器在等待窗口内没有排空，会记录告警而不是无限等待。
- 事件匹配缓存按 `eventID + destination` 维度缓存，避免不同 destination 间误复用观察者列表。
- 默认应用关闭路径现在会先让 service 结束，再关闭 `BackgroundRoutine`，最后调用 `EventHub.Terminate()`，避免 hub 在 service 已经退出后继续接收新工作。
- 观察者匹配仍按 `destination` 完成，`lane` 只决定调度顺序域，不参与 observer 路由。
- `lane` 建议使用有界 key，不要把每次请求的随机 ID 直接作为 lane，避免长期累积过多内部 channel。

**内部数据结构**：
- `event2Observer`：事件ID到观察者列表的映射
- `laneKey2ActionChannel`：lane 到处理通道的映射
- `hubActionChannel`：中心操作通道

### simpleObserver 实现

`simpleObserver` 是简化观察者实现，允许使用函数回调：

```go
type ObserverFunc func(Event, Result)

observer := NewSimpleObserver("my-observer", hub)
observer.Subscribe("/user/+", func(event Event, result Result) {
    // 处理事件
})
```

如果需要把“观察者逻辑名称”和“destination 匹配模式”分开，可以使用：

```go
observer := NewSimpleObserverWithMatchID(
    "base-observer",
    "/internal/modules/kernel/base/#",
    hub,
)
```

## 工厂函数

### 创建事件中心

```go
// 创建事件中心，capacitySize 指定执行器容量
func NewHub(capacitySize int) Hub

// 创建带可选内部缓冲和执行器配置的事件中心
func NewHubWithOptions(capacitySize int, opts ...HubOption) Hub
```

### 创建事件

```go
// 创建基础事件
func NewEvent(id, source, destination string, header Values, data any) Event

// 创建带上下文的事件
func NewEventWithContext(id, source, destination string, header Values, context context.Context, data any) Event
```

### 创建结果

```go
// 创建结果对象
func NewResult(id, source, destination string) Result
```

### 创建简化观察者

```go
// 创建简化观察者
func NewSimpleObserver(id string, hub Hub) SimpleObserver

// 创建带独立 destination 匹配模式的简化观察者
func NewSimpleObserverWithMatchID(id, matchID string, hub Hub) SimpleObserver
```

### 顺序 lane 示例

```go
ev := NewEvent("/value/filter", "public", "/internal/modules/kernel/base", nil, payload)
ev.BindLaneKey("/internal/modules/kernel/base/app-1/goods")

// 同一个 lane 内保持顺序；不同 lane 可并行
result := hub.Send(ev)
```

## 泛型辅助函数

### 类型安全的结果转换

```go
// 将 Result.Get() 的值转换为指定类型
func GetAs[T any](r Result) (T, *cd.Error)

// 将 Result.GetVal() 的值转换为指定类型
func GetValAs[T any](r Result, key string) (T, bool)
```

### 类型安全的事件数据转换

```go
// 将 Event.Data() 的值转换为指定类型
func GetAsFromEvent[T any](e Event) (T, *cd.Error)

// 将 Event.GetData() 的值转换为指定类型
func GetValAsFromEvent[T any](e Event, key string) (T, bool)

// 将 Event.Header() 的值转换为指定类型
func GetHeaderValAsFromEvent[T any](e Event, key string) (T, bool)

// 将 Event.Context() 的值转换为指定类型
func GetContextValAsFromEvent[T any](e Event, key any) (T, bool)
```

## 使用示例

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "github.com/muidea/magicCommon/event"
)

type MyHandler struct {
    id string
}

func (h *MyHandler) ID() string {
    return h.id
}

func (h *MyHandler) Notify(ev event.Event, re event.Result) {
    fmt.Printf("Handler %s received event: %s\n", h.id, ev.ID())
    
    // 获取事件数据
    if data, ok := event.GetValAsFromEvent[string](ev, "data"); ok {
        fmt.Printf("Event data: %s\n", data)
    }
    
    // 设置结果
    if re != nil {
        re.Set("processed", nil)
    }
}

func main() {
    // 创建事件中心
    hub := event.NewHub(10)
    
    // 创建处理器
    handler := &MyHandler{id: "handler-1"}
    
    // 订阅事件
    hub.Subscribe("/user/create", handler)
    
    // 创建事件
    header := event.NewValues()
    header.Set("priority", "high")
    
    ev := event.NewEvent("/user/create", "service-a", "handler-1", header, "user data")
    ev.SetData("data", "additional info")
    
    // 发送事件并获取结果
    result := hub.Send(ev)
    
    // 处理结果
    if result.Error() != nil {
        fmt.Printf("Error: %v\n", result.Error())
    } else {
        if val, err := event.GetAs[string](result); err == nil {
            fmt.Printf("Result: %s\n", val)
        }
    }
    
    // 清理
    hub.Terminate()
}
```

### 使用简化观察者

```go
func main() {
    hub := event.NewHub(10)
    
    // 创建简化观察者
    observer := event.NewSimpleObserver("my-observer", hub)
    
    // 订阅事件（函数回调）
    observer.Subscribe("/order/+", func(ev event.Event, re event.Result) {
        fmt.Printf("Order event: %s\n", ev.ID())
        
        // 获取订单ID（路径参数）
        orderID := strings.Split(ev.ID(), "/")[2]
        fmt.Printf("Order ID: %s\n", orderID)
        
        if re != nil {
            re.Set(map[string]string{"status": "processed"}, nil)
        }
    })
    
    // 发送事件
    ev := event.NewEvent("/order/12345", "payment-service", "my-observer", 
                         event.NewValues(), nil)
    result := hub.Send(ev)
    
    // 处理结果
    if data, err := event.GetAs[map[string]string](result); err == nil {
        fmt.Printf("Processing result: %v\n", data)
    }
    
    hub.Terminate()
}
```

## 设计模式分析

### 1. 观察者模式（发布-订阅）
- **Hub** 作为主题（Subject），管理观察者列表
- **Observer** 作为观察者接口，定义通知方法
- 支持一对多的消息分发

### 2. 命令模式
- **action** 接口定义操作命令
- 通过通道将操作封装为命令对象
- 实现操作队列和异步执行

### 3. 工厂模式
- `NewHub`、`NewEvent`、`NewSimpleObserver` 等工厂函数
- 隐藏具体实现细节，提供统一创建接口

### 4. 泛型编程
- 类型安全的转换函数
- 减少运行时类型断言错误
- 提高代码可读性和安全性

## 性能考虑

1. **并发安全**：使用 `sync.RWMutex` 保护共享数据
2. **异步处理**：避免阻塞调用者，提高吞吐量
3. **通道缓冲**：合理设置通道容量，平衡内存和性能
4. **协程池**：通过 `execute.Execute` 管理协程，避免频繁创建销毁
5. **事件顺序保证**：
   - **同一个观察者的事件顺序保证**：无论是 `Post()` 还是 `Send()` 方法，同一个观察者接收事件的顺序与事件投递顺序一致
   - **不同观察者之间无顺序保证**：不同观察者之间的事件处理可以并行，不保证执行顺序
   - **实现机制**：每个观察者有独立的事件处理通道，通道中的事件按投递顺序顺序执行
   - **性能影响**：顺序执行可能降低吞吐量，但保证了事件处理的确定性

## 错误处理

1. **Panic 恢复**：所有通知调用都包含 recover 机制
2. **错误传递**：通过 `Result` 接口传递处理错误
3. **日志记录**：使用 `log` 包记录异常信息
4. **优雅降级**：单个观察者失败不影响其他观察者

## 测试覆盖

模块包含完整的测试用例：

1. **功能测试**：验证基本功能正确性
2. **匹配测试**：测试通配符匹配算法
3. **并发测试**：验证线程安全性
4. **类型转换测试**：验证泛型辅助函数
5. **事件顺序一致性测试**：验证同一个观察者的事件顺序保证
6. **高并发场景测试**：验证多个发布者同时发送事件的场景
7. **大吞吐场景测试**：验证快速发送大量事件的场景
8. **大量订阅者场景测试**：验证多个观察者订阅相同事件的场景
9. **混合场景测试**：验证并发发布+大量订阅的复杂场景

### 新增压力测试用例说明

#### TestEventOrderConsistency
验证事件顺序一致性特性：
- 同一个观察者通过 `Post()` 方法投递的事件，保证按投递先后顺序进行通知
- 不同观察者之间不保证通知顺序
- 支持同步 `Send()` 和异步 `Post()` 两种方式的事件顺序保证

#### TestHighConcurrency
测试高并发场景：
- 多个发布者协程同时发送事件
- 验证事件处理无丢失、无死锁
- 测试参数：5个发布者，每个发送10个事件，共50个事件

#### TestHighThroughput
测试大吞吐场景：
- 快速发送大量事件（200个事件）
- 测量事件处理吞吐量（events/sec）
- 验证事件顺序正确性

#### TestManySubscribers
测试大量订阅者场景：
- 30个观察者订阅相同事件
- 每个观察者接收15个事件
- 验证所有观察者都收到正确数量的事件

#### TestMixedScenario
测试混合复杂场景：
- 5个发布者并发发送事件
- 20个观察者订阅相同事件
- 每个发布者发送10个事件，共50个事件
- 验证系统在复杂负载下的稳定性和正确性

### 运行测试

运行所有测试：
```bash
cd magicCommon/event
go test -v
```

运行特定测试：
```bash
# 运行事件顺序一致性测试
go test -v -run TestEventOrderConsistency

# 运行并发和大吞吐测试
go test -v -run "TestHighConcurrency|TestHighThroughput|TestManySubscribers|TestMixedScenario"

# 运行所有测试（包括压力测试）
go test -v ./...
```

## 扩展建议

1. **持久化支持**：添加事件持久化存储
2. **重试机制**：实现失败事件的重试策略
3. **监控指标**：添加事件处理统计和监控
4. **分布式支持**：扩展为跨服务的事件总线
5. **序列化优化**：支持多种数据序列化格式

## 注意事项

1. **内存泄漏**：确保及时调用 `Terminate()` 释放资源
2. **死锁风险**：避免在通知回调中执行阻塞操作
3. **通配符性能**：复杂通配符模式可能影响匹配性能
4. **上下文传递**：合理使用上下文传递请求级数据

## 总结

`magicCommon/event` 模块是一个功能完善、设计优雅的事件系统，适用于需要解耦和异步通信的 Go 应用程序。其灵活的匹配机制、类型安全的 API 和健壮的错误处理使其成为构建事件驱动架构的理想选择。
