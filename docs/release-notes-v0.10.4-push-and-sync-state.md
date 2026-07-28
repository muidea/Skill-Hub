# skill-hub v0.10.4

## 问题修复

- 修复 `skill-hub push`：默认仓库工作区干净但本地提交领先远端时，现在会识别并推送这些已提交内容。
- 推送预览与 JSON 输出新增待推送状态及 `ahead` / `behind` 计数，服务模式与本地模式保持一致。
- `skill-hub repo sync` 现在先检查分支关系：本地领先时明确报告 `local_ahead`，分叉或脏工作区阻塞时返回可操作状态，不会隐式推送或创建合并提交。

## 验证

- `make lint`
- `make test`
- `make build VERSION=0.10.4`
