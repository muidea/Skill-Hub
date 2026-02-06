# Skill Hub

一款专为 AI 时代开发者设计的"技能（Prompt/Script）生命周期管理工具"。它旨在解决 AI 指令碎片化、跨工具同步难、缺乏版本控制等痛点。

## 🚀 快速开始

### 一键安装（Linux/macOS）
```bash
curl -s https://raw.githubusercontent.com/muidea/skill-hub/main/scripts/download-latest.sh | bash
```

### 基本使用流程
```bash
# 1. 初始化工作区
skill-hub init

# 2. 查看可用技能
skill-hub list

# 3. 启用技能并设置目标
skill-hub use git-expert --target open_code

# 4. 应用技能到项目
skill-hub apply

# 5. 检查状态
skill-hub status
```

## 核心理念

- **Git 为中心**：所有技能存储在Git仓库中，作为单一可信源
- **一键分发**：将技能快速应用到不同的AI工具
- **闭环反馈**：将项目中的手动修改反馈回技能仓库

## 功能特性

- **技能管理**：创建、查看、启用、禁用技能
- **变量支持**：技能模板支持变量替换
- **跨工具同步**：支持 Cursor、Claude Code、OpenCode 等AI工具
- **版本控制**：基于Git的技能版本管理
- **差异检测**：自动检测手动修改并支持反馈
- **安全操作**：原子文件写入和备份机制

## 安装方式

### 方式一：使用预编译二进制（推荐）

1. **访问 [GitHub Releases](https://github.com/muidea/skill-hub/releases)** 页面
2. **下载对应平台的压缩包**：
   - Linux: `skill-hub-linux-amd64.tar.gz` 或 `skill-hub-linux-arm64.tar.gz`
   - macOS: `skill-hub-darwin-amd64.tar.gz` 或 `skill-hub-darwin-arm64.tar.gz`
   - Windows: `skill-hub-windows-amd64.tar.gz` 或 `skill-hub-windows-arm64.tar.gz`

3. **解压并安装**：

   **Linux/macOS**:
   ```bash
   # 下载并解压
   tar -xzf skill-hub-linux-amd64.tar.gz
   
   # 安装到系统路径
   sudo cp skill-hub /usr/local/bin/
   
   # 或直接运行
   ./skill-hub --help
   ```

   **Windows**:
   ```powershell
   # 解压后将 skill-hub.exe 添加到系统 PATH
   # 或在解压目录中运行
   .\skill-hub.exe --help
   ```

### 方式二：从源码编译
```bash
git clone https://github.com/muidea/skill-hub.git
cd skill-hub
make build
sudo make install
```

## 命令参考

| 命令 | 描述 | 示例 |
|------|------|------|
| `init` | 初始化Skill Hub工作区 | `skill-hub init [git-url]` |
| `list` | 列出所有可用技能 | `skill-hub list` |
| `use` | 在当前项目启用技能 | `skill-hub use git-expert --target open_code` |
| `set-target` | 设置项目首选目标 | `skill-hub set-target open_code` |
| `apply` | 将技能应用到项目 | `skill-hub apply --dry-run` |
| `status` | 检查技能状态 | `skill-hub status` |
| `feedback` | 反馈手动修改 | `skill-hub feedback git-expert` |
| `update` | 更新技能仓库 | `skill-hub update` |
| `remove` | 从项目移除技能 | `skill-hub remove git-expert` |
| `git` | Git仓库操作 | `skill-hub git --help` |

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
name: "git-expert"
version: "1.0.0"
description: "Git 提交专家"
author: "dev-team"
tags: ["git", "workflow"]
preferred_target: cursor
targets:
  cursor: true
  claude_code: true
  open_code: true
variables:
  project_name: "{{ .ProjectName }}"
  language: "{{ .Language }}"
content: |
  # 技能内容...
  # 支持Go Template语法: {{.project_name}}, {{.language}}
```

### 示例技能
项目包含三个高质量的技能示例：
- **golang-best-practices**: Go语言最佳实践和代码规范
- **react-typescript**: React + TypeScript开发最佳实践  
- **docker-devops**: Docker容器化和DevOps最佳实践

## 支持的AI工具

| 工具 | 支持状态 | 配置文件位置 |
|------|----------|--------------|
| **Cursor** | ✅ 完全支持 | `~/.cursor/rules` 或项目级 `.cursorrules` |
| **Claude Code** | ✅ 完全支持 | `~/.claude/config.json` 或项目级 `.clauderc` |
| **OpenCode** | ✅ 完全支持 | `~/.config/opencode/skills/` 或项目级 `.agents/skills/` |

## 构建和发布

### 本地构建
```bash
# 开发构建
make build

# 发布构建（所有平台）
make release-all VERSION=1.0.0

# 查看帮助
make help
```

### 自动发布
项目使用GitHub Actions实现自动发布，创建git标签时自动构建并发布预编译二进制。

#### 使用发布脚本：
```bash
./scripts/create-release.sh
```

## 贡献指南

欢迎提交Issue和Pull Request！

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启Pull Request

### 开发要求
- 遵循现有代码风格
- 添加适当的测试
- 更新相关文档
- 确保向后兼容性

## CI/CD状态

[![CI](https://github.com/muidea/skill-hub/actions/workflows/ci.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/ci.yml)
[![Release](https://github.com/muidea/skill-hub/actions/workflows/release.yml/badge.svg)](https://github.com/muidea/skill-hub/actions/workflows/release.yml)

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 问题反馈

如遇到问题或有功能建议，请：
1. 查看现有Issue是否已解决
2. 创建新的Issue，详细描述问题
3. 提供复现步骤和环境信息