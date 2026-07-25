# skill-hub v0.10.0

## 破坏性变更

- `use`、`apply`、`status` 和 `remove` 不再接受单 agent 筛选参数；全局操作统一处理已检测或已配置的 agent。

## 新功能

- `skill-hub use <id>` 支持精确启用单个技能；批量 glob 继续使用 `--pattern`。
- `use` 增加 `--repo`、`--dry-run`、`--json`、`--non-interactive` 和 `--var key=value`，支持来源锁定、预览和自动化。
- 项目与全局 use 服务统一使用公共结果模型，CLI 输出统一的逐项结果和汇总。

## 问题修复

- 修复 registry 部分过期时精确 skill ID 被遗漏、重复 pattern 重复启用，以及重新启用时变量被清空的问题。
- 强化归档完整性校验，避免技能更新丢失显式版本、核心 frontmatter、章节或受管资源。

## 技能与文档

- 刷新 `skill-hub-project-usage` 与 `skill-hub-workflow`，同步新的全局使用、来源选择和自动化工作流。
