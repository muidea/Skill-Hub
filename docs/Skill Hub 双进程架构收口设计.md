# Skill Hub 双进程架构收口设计

**状态：已确认，待分阶段实施。** 本文是当前架构收口的唯一目标说明；它覆盖现有功能的等价迁移，不授权增加平台集成、认证授权或新的业务能力。

## 1. 决策与范围

运行形态固定为：

```text
skill-hub   = Cobra CLI 客户端
skill-hubd  = 唯一的本地 daemon、HTTP API、Web UI 与 framework 服务宿主
```

删除 `skill-hub serve` 及其全部子命令，不保留兼容别名或由 CLI 转调 daemon 的过渡实现。服务启动、停止、注册与状态管理均由 `skill-hubd` 提供或由系统服务管理。发布包、安装器和升级流程必须将两个二进制视作同一版本的原子产物。

本次迁移必须保持 Cobra 命令（除 `serve`）、参数、文本/JSON 输出、退出码、`/api/v1/*` HTTP 合同、Web UI、Git 与状态语义，以及 `secret-key` 的远端 push 保护语义不变。不得引入 magicPlatform、身份认证、授权、租户、审计、数据库迁移、新 API 版本或新业务命令。

## 2. 目标分层

```text
application/skill-hub/cmd/       CLI 二进制入口
application/skill-hubd/cmd/      daemon 二进制入口
internal/clis/skill-hub/         Cobra 适配器
internal/services/skill-hubd/    daemon 进程生命周期
internal/runtime/remote/         服务可用时的远程执行
internal/runtime/local/          服务不可用时的嵌入式执行
internal/adapters/hubclient/     HTTP 协议客户端
internal/modules/blocks/         单一资源或技术能力 owner
internal/modules/application/    跨能力用例编排、HTTP 入站适配器
internal/pkg/*port/              窄接口、稳定 DTO 与纯 helper
```

HTTP client 已收口至 `internal/adapters/hubclient`，旧 `internal/modules/blocks/hubclient` 已删除。`internal/adapter` 仅保留本地备份清理 helper；不得恢复双写或旧 client 路径。

### 2.1 职责边界

- `internal/clis/skill-hub` 只处理命令注册、参数解析、交互、输出、退出码和运行时选择；不得直接访问 Git、状态仓库或 Module/Block 的内部实现。
- `internal/services/skill-hubd` 仅处理配置、日志、监听器与 `Startup/Run/Shutdown`。它不是业务 facade、Service 注册表或 repository provider。
- `runtime/remote` 负责发现与健康检查 daemon，并经 `hubclient` 调用既有 HTTP API；`runtime/local` 在 daemon 不可用时显式装配本地能力，不启动 HTTP、不扫描 framework 的全局 plugin 注册。
- Block 只拥有单一资源或技术能力的状态；Application Module 编排跨 owner 用例。跨 owner 的运行期协作仅通过 owner 定义的 typed EventHub 合同或稳定 port，不能继续经宽泛 `runtime.Service` 绕过边界。

## 3. 请求与生命周期

CLI 对每次可桥接的操作先做有界的 daemon 可用性检查：成功则进入 remote runtime，失败则进入 local runtime；两条路径必须共享相同的命令 DTO、结果模型、错误码和渲染器。

```text
Cobra command → runtime selector → remote → hubclient → skill-hubd API
                                  └→ local  → explicit local lifecycle → use case
```

`skill-hubd` 显式装配所需 Initiator、Block 与 Application Module。入口负责选择运行单元；`Setup` 完成接线和路由注册，监听器只在路由就绪后接受请求；关闭时先停止新输入，再按逆序释放资源，并保证幂等。使用 EventHub 的 Module/Block 必须在自身 `biz` 内嵌共享 Base Biz；HTTP service、`module.go` 和进程 service 不持有 Hub 或订阅。

## 4. 分阶段收口

1. **兼容基线**：为 Cobra、JSON/文本输出、HTTP API、Web UI、`secret-key`、服务管理和 remote/local 降级建立测试与快照；列出所有 `serve`、旧 runtime 和旧 adapter 调用点。
2. **双二进制骨架**：按已验证的 magicCommon/framework 与可选 magicEngine 版本加入最小依赖；创建 `skill-hubd` 入口和 process service，但不改变对外合同。
3. **能力边界迁移**：为仓库、项目状态/应用/反馈/状态、全局分发等建立窄 port；将资源 owner 收口为 Block，将跨能力工作流收口为 Application Module；逐步移除 `runtime.Service` 调用。
4. **服务与客户端迁移**：将既有 HTTP 路由迁到 Application service，保持路径与 payload 不变；实现 remote/local selector 和 `hubclient`，确保两条路径结果等价。
5. **移除 `serve`**：将服务注册和进程管理迁到 `skill-hubd` 或系统服务配置；删除 CLI `serve`、内嵌 HTTP server、相关文档和测试；升级逻辑改为管理 `skill-hubd`。
6. **清理与发布**：删除旧 ServeMux、旧 runtime facade、过渡 adapter 和无调用空壳；完成结构扫描、全量测试、文档更新和常规发布流程。

每一阶段均可独立发布；只有其兼容基线通过，才允许进入下一阶段。

## 5. 验收标准

- `skill-hub` 帮助中不存在 `serve`；`skill-hubd` 是唯一监听 HTTP 的二进制。
- 已有非服务命令在 daemon 可用和不可用时均保持相同可观察结果；状态枚举继续使用 `synced`、`modified`、`outdated`、`diverged`、`missing`、`conflict`、`orphaned`、`unavailable`，全局聚合额外允许 `mixed`。
- `/api/v1/*` 合同、Web UI 与 `secret-key` 的只读/push 保护回归通过。
- 生产代码不再引用旧 `runtime.Service`、CLI 内嵌 server 或废弃 adapter 路径；局部 runtime 不使用 plugin-manager 扫描。
- 结构、单元和端到端验证通过：`make lint`、`make test`，以及 `~/codespace/venv/bin/python3 -m pytest -p no:rerunfailures tests/e2e -c tests/e2e/pytest.ini`。

## 6. 后续但不属于本次

完成等价迁移后，才单独设计 magicPlatform 接入、身份认证、权限、租户与审计。该阶段可复用 `skill-hubd`、Application Module、port 和 EventHub 边界，但不得反向改变本次已经冻结的 CLI/API 兼容合同。
