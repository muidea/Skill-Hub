# v0.8.13 `--pattern` flag migration (cut positional pattern form)

## 变更摘要

- **Breaking**：6 个支持 pattern 的命令（`list` / `use` / `status` / `apply` / `feedback` / `validate`）废弃 v0.8.12 引入的位置参数形式，glob pattern 统一改用 `--pattern` flag：
  - `skill-hub list --pattern <glob>...`
  - `skill-hub use --pattern <id-or-glob>...`
  - `skill-hub status --pattern <id-or-glob>...`
  - `skill-hub apply --pattern <id-or-glob>...`
  - `skill-hub feedback --pattern <id-or-glob>...`
  - `skill-hub validate --pattern <id-or-glob>...`
- **为什么改**：v0.8.12 的位置参数形式 `skill-hub list magic*` 在 shell 启动 CLI 进程前就可能被 shell 提前 glob 展开成 cwd 下的文件名，导致 CLI 收到的是被 shell 改写过的输入。`--pattern` 让 pattern 有明确入口，但带通配符的值仍应引用，例如 `--pattern 'magic*'`，否则 shell 仍可能在 skill-hub 启动前展开。
- **行为**：
  - 任何对位置参数的尝试（如 `list magic*` / `use foo` / `apply magic*`）会被 cobra Args 校验器直接拒绝，并输出简化后的用户提示，例如 `Error: 不再支持位置参数 pattern，请通过 --pattern 传入。请改用：skill-hub list --pattern 'magic*'`。
  - 终端错误输出不再暴露内部 operation/code 前缀（如 `rejectPositionalPattern: INVALID_INPUT -`），也不再为该类错误重复打印 `Error` 或整段 Usage。
  - 单字面量精确 ID 的语义不变：把 `use git-expert` 改成 `use --pattern git-expert` 即可。
  - `--pattern` 仍支持单字面量 / 含通配符 / `**` 三类输入，glob 语法与 v0.8.12 保持一致。
  - `--pattern` 可重复：`--pattern 'magic*' --pattern 'git-*'` 等同多 pattern 并集。
  - `--pattern ''` 显式拒绝（`ErrInvalidInput`）。
  - `feedback` / `validate` 保留 `--all` 入口：`feedback --all` / `feedback --pattern 'magic*'` 二选一，不允许同时给两个。
- **删除**：`internal/cli/pattern_helpers.go` 中的 `detectShellExpansion` / `warnIfShellExpanded` 启发式检测被删除。位置参数不再作为 pattern 入口；未引用通配符导致的 shell 展开仍由用户通过引用参数避免。
- **未变化**：
  - 底层 `multirepo.Manager.FindSkillsByPatterns` 与 `POST /api/v1/skills/find` 服务端点签名不变。
  - `list` / `use` 的服务桥接 / 本地回退路径不变；服务模式优先 `FindSkillsByPatterns`，本地模式直接调 `multirepo.Manager`。
  - `compilePatterns` / `pkg/utils.CompileSkillIDPattern` / `chooseSkillCandidate` / `resolveSkillsByPatterns` 全部保持原样。

## 用户影响

- 现有调用 v0.8.12 位置参数形式的脚本需要改成 `--pattern`：
  - `skill-hub list 'magic*'` → `skill-hub list --pattern 'magic*'`
  - `skill-hub use 'magic*'` → `skill-hub use --pattern 'magic*'`
  - `skill-hub apply 'magic*'` → `skill-hub apply --pattern 'magic*'`
  - `skill-hub status 'magic*' --json` → `skill-hub status --pattern 'magic*' --json`
  - `skill-hub feedback 'magic*' --dry-run` → `skill-hub feedback --pattern 'magic*' --dry-run`
  - `skill-hub validate 'magic*' --links` → `skill-hub validate --pattern 'magic*' --links`
- 单字面量精确 ID 行为保留，但位置参数形式不再接受：`use git-expert` 需改成 `use --pattern git-expert`。
- cwd 存在与 pattern 同名前缀的临时文件时（如 `./magicBase`），应使用 `skill-hub list --pattern 'magic*'`，避免 shell 在 skill-hub 启动前展开 pattern。

## 验证

- `go test ./internal/cli/...`：`patterns_test.go` 覆盖 `readPatternFlag` 三种状态（未设置 / 正常值 / 空元素）、位置参数错误静默 Usage 和 shell 展开提示；`pattern_flag_test.go` 覆盖 6 命令的 cobra Args 拒绝与 RunE 缺 flag 错误；`error_test.go` 覆盖 CLI 用户态错误格式化。
- `internal/cli/service_mode_test.go` 扩展 `TestRunUseViaServiceWithoutLocalConfig` / `TestRunApplyByPatternViaService`，验证 `use` 通过仓库 pattern 查询，项目 `apply` 通过当前项目已启用技能过滤后再按 ID 走单技能端点。
- `tests/e2e/test_skill_pattern_flag.py` 新增 12 个 e2e：单 pattern 过滤、多 pattern 并集、`**`、单独 `*` 拒绝、位置参数拒绝、6 命令 `--pattern` 端到端、`feedback` / `validate` 的 `--all` ↔ `--pattern` 互斥校验。
- 既有 15 个 e2e 文件中 124 个位置参数形式的 `.run(...)` 调用已批量迁移为 `--pattern` 形式（脚本：`/tmp/rewrite_positional.py`，AST-based 改写并保留原格式）。
- 发布前需通过：
  - `make build`
  - `make test`
  - `make lint`
  - `~/codespace/venv/bin/python3 -m pytest -p no:rerunfailures tests/e2e -c tests/e2e/pytest.ini`
  - 手动烟囱（`make build` 之后）：
    - `./bin/skill-hub list --pattern 'magic*'`
    - `./bin/skill-hub list --pattern 'magic*' --pattern 'git-*'`
    - `./bin/skill-hub list 'magic*'`（预期失败，`ErrInvalidInput`，提示 `--pattern`）
    - `./bin/skill-hub use --pattern 'magic*'`
    - `./bin/skill-hub use --pattern git-expert`（精确 ID）
    - `./bin/skill-hub use git-expert`（位置参数 + 字面量，仍被新校验器拒绝，需改 `--pattern`）
    - `./bin/skill-hub apply --pattern 'magic*'`
    - `./bin/skill-hub status --pattern 'magic*' --json`
    - `./bin/skill-hub feedback --pattern 'magic*' --dry-run`
    - `./bin/skill-hub validate --pattern 'magic*'`
  - 在 cwd 有 `magicBase` 等匹配文件时重跑 `./bin/skill-hub list --pattern 'magic*'`，确认 shell 不再拦截，且无任何 shell-expansion 警告。
  - `grep -rn 'warnIfShellExpanded\|detectShellExpansion' internal/` 应返回空。
