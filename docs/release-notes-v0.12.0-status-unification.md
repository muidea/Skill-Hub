# skill-hub v0.12.0

## 功能优化

- `skill-hub status` 与 `skill-hub status --global` 统一使用 `synced`、`modified`、`outdated`、`diverged`、`missing`、`conflict`、`orphaned`、`unavailable` 状态值。
- `status --global` 按 skill 聚合展示多个 Agent；Agent 状态不一致时显示 `mixed` 并输出逐 Agent 明细。
- 状态 JSON 统一提供 `scope`、`status` 与 `reason`；新增 `legacy_status` 以便旧脚本迁移。

## 兼容性

- 旧项目状态与旧服务响应会在读取时自动映射到规范状态，例如 `Synced` / `ok` 映射为 `synced`，`Outdated` / `stale` 映射为 `outdated`。
- `legacy_status` 保留原有项目或全局状态名称，供迁移期间使用。

## 验证

- `make lint`
- `make test`
- `make build VERSION=0.12.0`
