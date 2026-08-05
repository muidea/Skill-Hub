# v0.13.5 Local Registry Cache

## 问题修复

- 本地重建的仓库技能索引现在写入 `~/.skill-hub/cache/repositories/<repository>/registry.json`，不再改写克隆仓库中受 Git 跟踪的 `registry.json`。
- `skill-hub pull` 和 `skill-hub repo rebuild-index` 会刷新本地缓存，同时保留仓库内索引作为缓存缺失时的兼容读取回退。
- 同步技能仓库不再因工具自身重建索引产生的 `registry.json` 工作区修改而被阻塞。

## 用户影响

- 升级前已产生的 `registry.json` 工作区修改需要一次性执行 `git -C ~/.skill-hub/repositories/<repository> restore registry.json`；后续索引刷新不会再次修改该文件。

## 测试与验证

- `make test`
