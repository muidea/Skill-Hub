# v0.13.2 daemon 服务管理

## 主要变更

- 新增 `skill-hubd service install|start|stop|restart|status|uninstall`，在 Linux 上管理当前用户的 systemd daemon 单元。
- `service install` 支持 `--host`、`--port`、`--secret-key` 和 `--no-start`；`status --json` 可用于自动化检查。
- 不恢复已移除的 `skill-hub serve` 或 `~/.skill-hub/services.json` 注册表模型。

## 安装可靠性

- 安装器将 `skill-hub` 和 `skill-hubd` 作为同一事务替换；任一步失败会回滚已替换的二进制。
- 安装完成后同时验证 CLI 与 daemon 的版本，避免两个二进制版本不一致。

## 验证

- `make lint`
- `go test ./... --count=1`
- `make release-all`
