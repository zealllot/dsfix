# DSFix

DeepSource + Windsurf/Cascade integration tool for automatically fixing code quality issues.

## Features

- Fetch code quality issues from DeepSource
- Convert each issue into an independent task
- Support **batch fixing** of same-type issues (greatly improves efficiency)
- Automatic processing in Windsurf/Cascade
- Auto-generate independent commit for each fix
- Support task filtering (by category, severity)

## Installation

```bash
go install github.com/zealllot/dsfix/cmd/dsfix@latest
```

Or build from source:

```bash
git clone https://github.com/zealllot/dsfix.git
cd dsfix
go build -o dsfix ./cmd/dsfix
```

## Configuration

1. Create DeepSource API Token:
   - Visit https://app.deepsource.io/settings/tokens
   - Create a new Personal Access Token

2. Initialize configuration:

```bash
cd /path/to/your/repo
dsfix init
```

3. Edit `.dsfix.yaml`:

```yaml
deepsource:
  api_token: "your-api-token-here"  # Or set DEEPSOURCE_API_TOKEN env var

repository:
  owner: "your-org"
  name: "your-repo"
  path: ""  # Leave empty to use current directory

filter:
  categories: []  # Leave empty to get all categories
  severities: []  # Leave empty to get all severities
  limit: 100      # Maximum number of issues to fetch
```

## Quick Start

Just tell Cascade:

> Run `dsfix start` to begin fixing

Cascade will:
1. Automatically list all pending issue types
2. Ask which type you want to fix
3. Batch fix after your selection
4. Show changes and wait for your confirmation
5. Auto-commit after confirmation

## Usage

### 1. Sync Issues

```bash
dsfix sync
```

### 2. Start Fixing (Recommended)

```bash
dsfix start
```

AI will automatically list task types, you just need to select which type to fix (enter number or shortcode).

### 3. Manual Batch Fix

```bash
dsfix list                      # View count by type
dsfix batch SCC-S1039           # Fix all issues of this type
dsfix batch SCC-S1039 -l 10     # Limit to 10 issues
```

### 4. Single Fix

```bash
dsfix next                      # Get next task
dsfix complete                  # Commit
dsfix skip                      # Skip and revert
```

## Workflow

### Recommended Flow (Simplest)

```
1. dsfix sync                   # Fetch issues (first time)
2. dsfix start                  # AI lists tasks, you select type
3. [AI batch fixes]
4. [Confirm changes]
5. [AI auto-commits]
6. Repeat 2-5 until done
```

### Manual Batch Flow

```
1. dsfix sync
2. dsfix list
3. dsfix batch <shortcode>
4. [AI fixes]
5. dsfix complete-batch
6. Repeat 3-5
```

## Cascade Integration

**Simplest way:**
> Run `dsfix start` to begin fixing

AI will guide you through the entire process.

## Command Reference

| Command | Description |
|---------|-------------|
| `dsfix init` | Initialize config file |
| `dsfix sync` | Sync issues from DeepSource |
| `dsfix status` | View task statistics |
| `dsfix start` | **Start fixing (Recommended)** |
| `dsfix list` | List pending issues grouped by type |
| `dsfix next` | Output next task |
| `dsfix batch <shortcode>` | Batch output same-type tasks |
| `dsfix complete` | Commit single fix |
| `dsfix complete-batch` | Batch commit all fixes |
| `dsfix skip` | Skip single task and revert |
| `dsfix skip-batch` | Skip batch tasks and revert |
| `dsfix reset` | Reset all tasks |

## Environment Variables

- `DEEPSOURCE_API_TOKEN`: DeepSource API Token

## File Structure

```
.dsfix/
└── tasks.json    # Task storage file

.dsfix.yaml       # Config file
```

## License

MIT

---

# DSFix (中文)

DeepSource + Windsurf/Cascade 集成工具，自动修复代码质量问题。

## 功能

- 从 DeepSource 拉取代码质量 issues
- 将每个 issue 转化为独立任务
- 支持**批量修复**同类型 issues（大幅提升效率）
- 在 Windsurf/Cascade 中自动处理
- 每个修复自动生成独立 commit
- 支持任务过滤（按类别、严重程度）

## 安装

```bash
go install github.com/zealllot/dsfix/cmd/dsfix@latest
```

或从源码构建：

```bash
git clone https://github.com/zealllot/dsfix.git
cd dsfix
go build -o dsfix ./cmd/dsfix
```

## 配置

1. 创建 DeepSource API Token：
   - 访问 https://app.deepsource.io/settings/tokens
   - 创建新的 Personal Access Token

2. 初始化配置：

```bash
cd /path/to/your/repo
dsfix init
```

3. 编辑 `.dsfix.yaml`：

```yaml
deepsource:
  api_token: "your-api-token-here"  # 或设置 DEEPSOURCE_API_TOKEN 环境变量

repository:
  owner: "your-org"
  name: "your-repo"
  path: ""  # 留空使用当前目录

filter:
  categories: []  # 留空获取所有类别
  severities: []  # 留空获取所有严重程度
  limit: 100      # 最大获取数量
```

## 快速开始

只需告诉 Cascade：

> 运行 `dsfix start` 开始修复

Cascade 会：
1. 自动列出所有待处理的 issue 类型
2. 询问你要处理哪个类型
3. 你选择后自动批量修复
4. 展示修改内容，等待你确认
5. 确认后自动提交

## 使用方法

### 1. 同步 Issues

```bash
dsfix sync
```

### 2. 开始修复（推荐）

```bash
dsfix start
```

AI 会自动列出任务类型，你只需选择要处理的类型（输入序号或 shortcode）。

### 3. 手动批量修复

```bash
dsfix list                      # 查看各类型数量
dsfix batch SCC-S1039           # 修复所有该类型 issues
dsfix batch SCC-S1039 -l 10     # 限制只处理 10 个
```

### 4. 单个修复

```bash
dsfix next                      # 获取下一个任务
dsfix complete                  # 提交
dsfix skip                      # 跳过并恢复
```

## 工作流程

### 推荐流程（最简单）

```
1. dsfix sync                   # 首次拉取 issues
2. dsfix start                  # AI 自动列出任务，你选择类型
3. [AI 批量修复]
4. [确认修改]
5. [AI 自动提交]
6. 重复 2-5 直到完成
```

### 手动批量流程

```
1. dsfix sync
2. dsfix list
3. dsfix batch <shortcode>
4. [AI 修复]
5. dsfix complete-batch
6. 重复 3-5
```

## 与 Cascade 集成

**最简单的方式：**
> 运行 `dsfix start` 开始修复

AI 会自动引导你完成整个流程。

## 命令参考

| 命令 | 说明 |
|------|------|
| `dsfix init` | 初始化配置文件 |
| `dsfix sync` | 从 DeepSource 同步 issues |
| `dsfix status` | 查看任务统计 |
| `dsfix start` | **开始修复（推荐）** |
| `dsfix list` | 按类型分组查看待处理 issues |
| `dsfix next` | 输出下一个任务 |
| `dsfix batch <shortcode>` | 批量输出同类型任务 |
| `dsfix complete` | 提交单个修复 |
| `dsfix complete-batch` | 批量提交所有修复 |
| `dsfix skip` | 跳过单个任务并恢复 |
| `dsfix skip-batch` | 跳过批量任务并恢复 |
| `dsfix reset` | 重置所有任务 |

## 环境变量

- `DEEPSOURCE_API_TOKEN`: DeepSource API Token

## 文件结构

```
.dsfix/
└── tasks.json    # 任务存储文件

.dsfix.yaml       # 配置文件
```

## License

MIT
