# magicCommon/execute 模块说明

## 概述

`magicCommon/execute` 提供一个轻量的并发执行器，用于限制同时运行的任务数量，并为上层模块提供统一的任务提交和等待能力。`event.Hub` 和 `task.BackgroundRoutine` 都依赖这层执行器。

## 核心能力

- 限制并发任务数，避免无界 goroutine 膨胀
- 捕获任务 panic，避免任务异常直接杀死调用链
- 提供兼容的 `Wait()` 行为
- 提供显式结果的 `WaitTimeout()`，让调用方能判断是否真正等到空闲
- 提供 `Idle()` 和 `WaitContext()`，便于组件关闭阶段按状态或上下文控制等待

## 核心接口

```go
type Execute struct {
    // 内部包含并发计数和容量控制
}

func NewExecute(capacitySize int) Execute
func (s *Execute) Run(funcPtr func())
func (s *Execute) Wait()
func (s *Execute) Idle() bool
func (s *Execute) WaitTimeout(timeout time.Duration) bool
func (s *Execute) WaitContext(ctx context.Context) bool
```

## 行为语义

### Run

- `Run()` 会提交一个任务并立即返回。
- 执行器通过内部容量通道限制并发度。
- 任务执行过程中如果 panic，会被捕获并记录日志。

### Wait

- `Wait()` 保持历史兼容行为。
- 当前实现等价于 `WaitTimeout(5 * time.Second)`。
- 如果 5 秒内任务没有全部完成，`Wait()` 会直接返回，不会继续阻塞。

### WaitTimeout

- `WaitTimeout(timeout)` 会等待已提交任务清空。
- 返回 `true` 表示在超时前已经空闲。
- 返回 `false` 表示超时返回，此时执行器里可能仍有任务在运行。
- `timeout <= 0` 表示无限等待，直到全部任务完成。

### Idle / WaitContext

- `Idle()` 用于快速判断当前是否还有活动任务。
- `WaitContext(ctx)` 在等待任务排空时同时监听外部取消。
- 如果组件关闭流程本身已经由 `context.Context` 驱动，优先使用 `WaitContext()`。

## 设计约束

`Execute` 当前同时服务两类上层模块：

1. **短生命周期任务**
   例如普通异步回调、批量通知、后台短任务。

2. **长生命周期任务**
   例如 `event.Hub` 内部的常驻 action loop。

这也是 `Wait()` 仍保留超时返回语义的原因。对包含常驻任务的调用方，如果需要明确判断等待结果，应优先使用 `WaitTimeout()`。

## 推荐使用方式

### 普通场景

```go
exec := execute.NewExecute(32)
exec.Run(func() {
    // do work
})

ok := exec.WaitTimeout(2 * time.Second)
if !ok {
    // 明确处理“仍有任务未完成”的情况
}
```

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

if !exec.WaitContext(ctx) {
    // ctx 先结束，执行器可能仍有任务未排空
}
```

### 上层组件关闭阶段

如果执行器用于组件关闭流程，不要把 `Wait()` 当成“必然已经全部停止”的信号。应当：

1. 先阻止新任务继续进入
2. 再调用 `WaitTimeout()`
3. 根据返回值记录日志或进入降级处理

`event.Hub.Terminate()` 当前就是按这个语义处理的。
