# skill-hub v0.10.2

## 问题修复

- 修复 `upgrade` 和一键安装脚本会直接覆盖全局托管 skills 目录的问题，避免升级后丢失 `.skill-hub-manifest.json` 并被 `status --global` 误判为 conflict。
- Release 内置 workflow skills 现在只同步到工具无关的专属目录；全局 skills 目录仅由 `use --global` 和 `apply --global` 管理。

## 测试与文档

- 增加回归测试，保证升级器不会因 Codex、OpenCode 或 Claude 的目录配置而写入这些全局目录。
- 刷新安装与命令文档，明确内置 workflow skills 与全局托管 skills 的所有权边界。
