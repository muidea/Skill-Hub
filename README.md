# Skill Hub

一款专为 AI 时代开发者设计的"技能（Prompt/Script）生命周期管理工具"。它旨在解决 AI 指令碎片化、跨工具同步难、缺乏版本控制等痛点。

## 核心理念

- **Git 为中心**：所有技能存储在Git仓库中，作为单一可信源
- **一键分发**：将技能快速应用到不同的AI工具
- **闭环反馈**：将项目中的手动修改反馈回技能仓库

## 功能特性

- ✅ **技能管理**：创建、查看、启用、禁用技能
- ✅ **变量支持**：技能模板支持变量替换
- ✅ **标记块技术**：非侵入式修改目标配置文件
- ✅ **差异检测**：自动检测手动修改
- ✅ **反馈闭环**：将修改反向更新到技能仓库
- ✅ **原子操作**：安全的文件写入和备份
- ✅ **跨平台**：支持Linux、macOS、Windows

## 快速开始

### 安装

```bash
# 从源码编译
git clone <repository-url>
cd skill-hub
make build
sudo make install

# 或直接使用二进制
./skill-hub --help
```

### 基本使用

1. **初始化工作区**
   ```bash
   skill-hub init
   ```

2. **查看可用技能**
   ```bash
   skill-hub list
   ```

3. **在当前项目启用技能**
   ```bash
   skill-hub use git-expert
   ```

4. **应用技能到项目**
   ```bash
   skill-hub apply
   ```

5. **检查技能状态**
   ```bash
   skill-hub status
   ```

6. **反馈手动修改**
   ```bash
   skill-hub feedback git-expert
   ```

## 命令参考

| 命令 | 描述 | 示例 |
|------|------|------|
| `init` | 初始化Skill Hub工作区 | `skill-hub init` |
| `list` | 列出所有可用技能 | `skill-hub list` |
| `use` | 在当前项目启用技能 | `skill-hub use git-expert` |
| `apply` | 将技能应用到项目 | `skill-hub apply --dry-run` |
| `status` | 检查技能状态 | `skill-hub status` |
| `feedback` | 反馈手动修改 | `skill-hub feedback git-expert` |
| `update` | 更新技能仓库 | `skill-hub update` |
| `search` | 搜索GitHub技能 | `skill-hub search ai` |

## 技能规范

### 目录结构
```
/skills
  /git-expert
    ├── skill.yaml       # 技能元数据
    ├── prompt.md        # 核心指令 (支持Go Template语法)
    └── scripts/         # (可选) 伴随执行的脚本
```

### skill.yaml 格式
```yaml
id: "git-expert"
name: "Git 提交专家"
version: "1.0.0"
author: "dev-team"
description: "根据变更自动生成符合 Conventional Commits 规范的说明"
tags: ["git", "workflow"]
compatibility:
  cursor: true
  claude_code: false
variables:
  - name: "LANGUAGE"
    default: "zh-CN"
    description: "输出语言"
dependencies: []
```

### 模板变量
在 `prompt.md` 中使用 Go Template 语法：
```markdown
# 技能说明
语言: {{.LANGUAGE}}
```

## 架构设计

```
Data Layer (Git)
    ↓
Logic Layer (Go CLI)
    ↓
Application Layer (Adapters)
    ↓
Target Tools (Cursor, Claude, etc.)
```

### 核心组件
- **CLI框架**: Cobra + Viper
- **Git引擎**: go-git (纯Go实现)
- **模板引擎**: text/template
- **文件适配器**: Cursor (.cursorrules), Claude (config.json)
- **状态管理**: JSON状态文件

## 开发

### 项目结构
```
skill-hub/
├── cmd/skill-hub/          # 程序入口
├── internal/               # 私有逻辑
│   ├── adapter/            # 工具适配器
│   ├── cli/                # Cobra命令定义
│   ├── config/             # 配置管理
│   ├── engine/             # 核心引擎
│   ├── git/                # Git操作封装
│   ├── state/              # 状态管理
│   └── ui/                 # 终端交互
├── pkg/spec/               # 公共定义
└── go.mod
```

### 构建
```bash
make build    # 编译
make test     # 运行测试
make lint     # 代码检查
make release  # 跨平台发布
```

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 贡献

欢迎提交Issue和Pull Request！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启Pull Request