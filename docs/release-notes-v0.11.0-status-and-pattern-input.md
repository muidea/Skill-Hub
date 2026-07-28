# skill-hub v0.11.0

## 功能优化

- `skill-hub status --global` 的文本状态表格现在显示每个全局技能的版本。
- 所有支持 `--pattern` 的命令现在也接受一个精确位置 ID，包括 `list`、`status`、`apply`、`feedback` 和 `validate`；既有的 `use` 与 `remove` 行为保持一致。
- 位置参数中的 glob 仍被拒绝，必须使用 `--pattern '<glob>'`，避免 shell 展开改变批量匹配语义。

## 验证

- `make lint`
- `make test`
- `make build VERSION=0.11.0`
