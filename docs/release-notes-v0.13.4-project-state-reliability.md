# v0.13.4 Project State Reliability

## 问题修复

- 项目状态保存现在会将项目路径规范化为绝对路径，避免相对路径或目录名被写入 `~/.skill-hub/state.json` 的项目键。
- 项目状态更新使用跨进程文件锁和原子替换，避免 CLI 与 daemon 同时写入时产生损坏的 JSON。
- 状态文件损坏时，`skill-hub status` 会明确提示检查 `~/.skill-hub/state.json` 的 JSON 格式，而不是仅显示“查找项目失败”。

## 用户影响

- 已存在的错误项目键需要一次性迁移到对应的绝对路径键；正常项目命令将继续使用完整路径登记状态。
- `status` 现在能在修复后的状态文件中正确识别项目已启用的技能，并将来源版本更新显示为 `outdated`。

## 测试与验证

- `go test ./... --count=1`
- 发布脚本将继续执行 `make lint`、`make test` 和带版本元数据的构建验证。
