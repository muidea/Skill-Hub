# v0.13.1 发布构建修复

## 修复

- 将 CLI 的 `service_bridge.go` 纳入版本控制，并修正 `.gitignore` 例外规则，避免 `internal/clis/skill-hub/` 下的新 Go 源文件被误忽略。
- 修复干净检出环境执行 `make release-all` 时 `serviceBridgeClient` 和 `hubClientIfAvailable` 未定义的编译失败。

## 验证

- `go test ./... --count 1`
- `make lint`
- `make release-all`
