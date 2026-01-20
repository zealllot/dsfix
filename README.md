# DSFix

DeepSource + Windsurf/Cascade 集成工具，自动修复代码质量问题。

## 功能

- 从 DeepSource 拉取代码质量 issues
- 将每个 issue 转化为独立任务
- 在 Windsurf/Cascade 中逐个处理
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

## 使用方法

### 1. 同步 Issues

```bash
dsfix sync
```

从 DeepSource 拉取 issues 并创建本地任务。

### 2. 查看状态

```bash
dsfix status
```

显示任务统计信息。

### 3. 在 Windsurf 中修复

**与 Cascade 配合使用（推荐）**

```bash
dsfix next
```

输出下一个待处理任务的详细信息，Cascade 会自动读取并修复。

修复完成后，确认修改并提交：

```bash
dsfix complete
```

这会自动：
1. Stage 修改的文件
2. 使用建议的 commit message 提交
3. 标记任务为已完成

或者使用自定义 commit message：

```bash
dsfix complete -m "fix: custom commit message"
```

跳过当前任务：

```bash
dsfix skip -R "需要人工判断"
```

### 4. 交互式运行

```bash
dsfix run
```

逐个显示任务，手动确认修复。

### 5. 重置任务

```bash
dsfix reset
```

将所有任务重置为待处理状态。

## 工作流程

```
1. dsfix sync          # 拉取 issues
2. dsfix next          # 获取下一个任务
3. [Cascade 自动修复代码]
4. [确认修改内容]
5. dsfix complete      # 自动 commit 并标记完成
6. 重复 2-5 直到完成
```

## 与 Cascade 集成

在 Windsurf 中，你可以直接告诉 Cascade：

> 运行 `dsfix next` 获取下一个任务，然后帮我修复

Cascade 会：
1. 读取任务信息
2. 查看相关文件
3. 修复问题
4. 等待你确认

确认后运行 `dsfix complete`，自动提交修改。

## 命令参考

| 命令 | 说明 |
|------|------|
| `dsfix init` | 初始化配置文件 |
| `dsfix sync` | 从 DeepSource 同步 issues |
| `dsfix status` | 查看任务统计 |
| `dsfix next` | 输出下一个任务 |
| `dsfix complete` | 提交修复并标记完成 |
| `dsfix complete --no-commit` | 仅标记完成，不提交 |
| `dsfix skip` | 跳过当前任务 |
| `dsfix reset` | 重置所有任务 |
| `dsfix run` | 交互式修复流程 |

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
