# skill-hub v0.10.3

## 问题修复

- 修复归档完整性校验：移除已废弃且无消费者的 `compatibility` 元数据不再被误判为核心信息丢失。
- 继续保护必需 frontmatter、显式版本、一级/二级章节及技能资源，避免不完整更新覆盖归档内容。
- 补齐 `skill-hub-skill-authoring` 的权威来源核对与归档完整性说明，并升级至 `1.0.6`。

## 全局技能维护

- 发布包中的 workflow skills 可安全归档到默认技能仓库，再通过 `apply --global --force` 恢复 Skill-Hub 托管清单。
- 避免旧版升级造成的非托管目录在后续信息刷新时被不必要地阻断。

## 验证

- `make lint`
- `make test`
- `make build VERSION=0.10.3`
