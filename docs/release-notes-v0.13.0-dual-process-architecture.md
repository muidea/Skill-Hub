# v0.13.0 双进程架构收口

## 主要变更

- 将 `skill-hub` 收口为 Cobra CLI，并新增 `skill-hubd` 作为唯一 daemon、HTTP API 与 Web UI 宿主。
- 删除 `skill-hub serve` 及旧的服务注册实现；发布、安装与升级流程将两个二进制作为同版原子产物处理。
- 将 CLI、HTTP client、运行时、资源 Block 与应用 Module 迁移至正式目录，并使用 focused port 隔离 CLI 与底层 Git、配置、状态和多仓库实现。
- 统一项目与全局 skill 状态枚举及渲染；位置 ID 与 `--pattern` 均可用于相关命令。

## 兼容性与修复

- 保持既有 CLI（除 `serve`）、HTTP `/api/v1/*`、Web UI、`secret-key` push 保护和状态语义兼容。
- 空远端尚未创建 `main` 分支时，push 预检不再误报远端更新错误。
- 更新长期有效文档、安装器、CI 和发布脚本，使其统一使用 `skill-hubd`。

## 验证

- `go test ./... -count=1`
- `make lint`
- `make build`
- `pytest -p no:rerunfailures tests/e2e -c tests/e2e/pytest.ini`
