# magicCommon/foundation/path 模块说明

## 概述

`foundation/path` 提供两类能力：

- 路径与目录工具函数
- 基于 `fsnotify` 的目录变化监控

适用场景包括目录遍历、文件列表查询、目录复制，以及对目录树中文件创建、修改、删除事件的监听。

## 路径工具

### 常用函数

- `IsDir(path string) bool`
- `IsSubPath(topPath, subPath string) bool`
- `Exist(path string) bool`
- `IsDirEmpty(dirPath string) (bool, error)`
- `SplitParentDir(dirPath string) string`
- `CleanPathContent(dirPath string)`
- `CopyPath(srcPath, dstPath string) error`
- `WalkPath(filePath string) ([]string, error)`
- `ListPath(filePath, filterPattern string, recursive bool) ([]string, error)`

### CopyPath 语义

- 深度复制目录结构和文件内容
- 会保留文件权限和修改时间
- 文件复制使用有限并发 worker
- 符号链接按链接本身复制，不跟随目标

## Monitor

`Monitor` 用于监听目录及其子目录下的文件变化。

### 核心接口

```go
monitor, err := path.NewMonitor(nil)
err = monitor.Start()
err = monitor.AddPath("/tmp/watch")
monitor.AddObserver(observer)
err = monitor.Stop()
```

### 事件类型

- `Create`
- `Modify`
- `Remove`

### 运行语义

- `AddPath()` 会注册目标目录以及其子目录。
- 新建目录时会自动补充对子目录的 watch。
- `AddObserver()` / `RemoveObserver()` 当前是并发安全的。
- 事件通知前会对 observer 列表做快照，避免通知过程中和增删观察者并发竞争。

### 生命周期语义

- `Start()` 只会真正启动一次；重复调用会直接返回。
- `Stop()` 是幂等的。
- `Stop()` 会关闭底层 watcher，并等待内部 goroutine 退出。
- `Stop()` 之后再次 `Start()` 当前会返回错误，而不是重新复用同一个 monitor。

### Ignore 规则

- `NewMonitor(ignores)` 可传入忽略列表。
- 只要路径中包含任一 ignore 子串，就会被忽略。

## 当前限制

- `Monitor` 目前不支持“停止后重新启动”。
- ignore 规则是子串匹配，不是更精细的 glob 或正则。
- 事件分发是进程内回调，不带持久化或重试能力。
