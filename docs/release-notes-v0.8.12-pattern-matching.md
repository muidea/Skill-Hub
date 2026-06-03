# v0.8.12 Glob Pattern Support

## 变更摘要

- 新增基于 Go `path.Match` 的 glob pattern 语法，匹配技能 `ID` 字段（与历史 `use <id>` / `status <id>` / `apply <id>` / `feedback <id>` / `validate <id>` 的精确 ID 语义保持一致）：
  - `*` 匹配零或多个任意字符
  - `?` 匹配恰好一个任意字符
  - `[abc]` 字符类（否定用 `[^abc]`）
  - `**` 匹配全部技能
- 接受 pattern 的命令扩展为 variadic positional 形式：
  - `skill-hub list [pattern...]`
  - `skill-hub use <pattern> [pattern...]`
  - `skill-hub status [pattern...]`
  - `skill-hub apply [pattern...]`
  - `skill-hub feedback [pattern...]`
  - `skill-hub validate [pattern...]`
- 单字面量参数（无通配符）保持原有精确 ID 行为，`use <id>` / `status <id>` / `apply <id>` / `feedback <id>` / `validate <id>` 完全向后兼容。
- 单独的 `*` 显式拒绝（`ErrInvalidInput`），避免歧义；如需匹配全部请使用 `**`。
- `use <pattern...>` 命中 0 个技能时**静默通过**，不报错。
- `apply` / `feedback` / `validate` 命中多个技能时**逐个处理，单个失败不影响后续**，最后打印成功/失败汇总。
- 服务化路径：新增 `POST /api/v1/skills/find` HTTP 端点，CLI 优先走服务桥接；服务不可用时回退到本地 `multirepo.Manager.FindSkillsByPatterns`。
- 跨仓库场景下，pattern 一次性返回 `(Repository::ID)` 去重后的命中集合；选择冲突仍走 `chooseSkillCandidate` 交互。

## 验证

- `pkg/utils/glob_test.go` 覆盖：精确匹配、`*` / `?` / `[abc]` / `[^abc]` / `**`、单独 `*` 拒绝、非法括号拒绝、空 pattern。
- `internal/multirepo/manager_test.go::TestManager_FindSkillsByPatterns` 覆盖两仓库 prefix / single-char / `**` / 空集 / 仓库过滤场景。
- `internal/cli/status_test.go` / `list_test.go` 新增 `hasWildcard`、`compilePatterns`、`filterStatusSummaryByIDs` 单元测试。
- `internal/cli/service_mode_test.go` 中 `fakeServiceBridgeClient` 扩展 `FindSkillsByPatterns` 桩，已有的服务模式测试继续通过。
- `tests/e2e/test_skill_pattern_match.py` 覆盖 `list 'magic*'` / `list '**'` / 单独 `*` 拒绝 / `status 'magic*' --json`。
- 发布前需通过：
  - `make lint`
  - `go test ./... --count=1`
  - `go build -o bin/skill-hub ./application/skill-hub/cmd`
  - `./bin/skill-hub list 'magic*'`
  - `./bin/skill-hub list '**'`
  - `./bin/skill-hub list '*'`（预期失败，错误码 `ErrInvalidInput`）
  - `./bin/skill-hub use 'no-such-prefix-xyz*'`（预期静默通过）
  - `./bin/skill-hub apply 'magic*'`
  - `./bin/skill-hub feedback 'magic*' --dry-run`
  - `pytest -p no:rerunfailures tests/e2e -c tests/e2e/pytest.ini -k pattern`
