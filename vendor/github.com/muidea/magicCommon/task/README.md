# magicCommon/task 模块说明

## 概述

`magicCommon/task` 在 `execute.Execute` 之上提供更高层的后台任务调度能力，主要用于：

- 异步执行普通任务
- 同步等待任务完成
- 带超时的同步等待
- 周期性定时任务

`framework/application` 和部分事件、插件场景会通过 `BackgroundRoutine` 使用这层能力。

## 核心接口

```go
type Task interface {
    Run()
}

type BackgroundRoutine interface {
    AsyncTask(task Task) error
    SyncTask(task Task) error
    SyncTaskWithTimeOut(task Task, timeout time.Duration) error
    AsyncFunction(function func()) error
    SyncFunction(function func()) error
    SyncFunctionWithTimeOut(function func(), timeout time.Duration) error
    Timer(ctx context.Context, task Task, intervalValue time.Duration, offsetValue time.Duration) error
    Shutdown(ctx context.Context) bool
}
```

## 行为语义

### AsyncTask / AsyncFunction

- 提交任务后立即返回。
- 任务会先进入后台任务通道，再由内部执行器异步执行。

### SyncTask / SyncFunction

- 等待任务完成。
- 当前实现等价于无限等待的同步任务。

### SyncTaskWithTimeOut / SyncFunctionWithTimeOut

- 等待任务完成直到超时。
- 超时后调用方会返回，但底层任务不会被取消；任务仍可能在后台继续执行。
- 当前实现已经避免了“超时后任务完成再向已关闭 channel 发送”的 panic 风险。

### Timer

- `Timer(ctx, ...)` 会启动一个独立 goroutine。
- 首次执行时间按 `intervalValue` 和 `offsetValue` 计算。
- 之后使用 `Ticker` 按固定周期触发。
- 当 `ctx.Done()` 触发时，后续定时触发会停止。
- 定时触发通过 `AsyncTask()` 进入后台队列，而不是直接在 timer goroutine 中执行。

### Shutdown

- `Shutdown(ctx)` 会停止接收新任务、关闭内部任务队列，并等待已提交任务排空。
- 返回 `true` 表示在 `ctx` 结束前成功排空。
- 返回 `false` 表示 `ctx` 结束后返回，此时可能仍有任务在内部执行器中运行。
- `Shutdown()` 是幂等的。

## 与 execute 的关系

- `BackgroundRoutine` 使用 `execute.Execute` 管理实际并发执行。
- 如果调用方需要显式区分“真正完成”和“等待超时”，应理解：
  - `SyncTaskWithTimeOut()` 只影响等待方
  - 不会中断已经开始运行的任务

## 推荐使用方式

### 普通异步任务

```go
routine := task.NewBackgroundRoutine(32)
_ = routine.AsyncFunction(func() {
    // background work
})
```

### 带超时的同步等待

```go
routine := task.NewBackgroundRoutine(32)
_ = routine.SyncFunctionWithTimeOut(func() {
    // maybe slow work
}, 200*time.Millisecond)

// 超时只表示调用方已返回，不表示任务一定停止
```

### 可取消定时任务

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

routine := task.NewBackgroundRoutine(32)
_ = routine.Timer(ctx, myTask, time.Minute, 0)

// 在组件关闭时停止定时任务
cancel()
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
defer shutdownCancel()
ok := routine.Shutdown(shutdownCtx)
_ = ok
```

## 当前限制

- 任务超时等待不会传播取消信号到任务本身。
- `Timer(ctx, ...)` 依赖调用方传入的 context 控制定时任务退出。
