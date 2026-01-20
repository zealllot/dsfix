# DSFix

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
  owner: "theplant"
  name: "mcd-website"
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
