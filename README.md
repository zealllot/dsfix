# DSFix

DeepSource + Windsurf/Cascade 集成工具，自动修复代码质量问题。

## 功能

- 从 DeepSource 拉取代码质量 issues
- 将每个 issue 转化为独立任务
- 在 Windsurf/Cascade 中逐个处理
- 每个修复生成独立 commit
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

**方式一：交互式运行**

```bash
dsfix run
```

逐个显示任务，手动确认修复。

**方式二：与 Cascade 配合使用（推荐）**

```bash
dsfix next
```

输出下一个待处理任务的详细信息，复制到 Cascade 中让 AI 修复。

修复完成后：

```bash
dsfix complete -m "fix(GO-S1000): simplify if-else to switch"
```

或跳过：

```bash
dsfix skip -R "需要人工判断"
```

### 4. 重置任务

```bash
dsfix reset
```

将所有任务重置为待处理状态。

## 工作流程

```
1. dsfix sync          # 拉取 issues
2. dsfix next          # 获取下一个任务
3. [在 Cascade 中修复代码]
4. dsfix complete      # 标记完成并提交
5. 重复 2-4 直到完成
```

## 与 Cascade 集成

在 Windsurf 中，你可以直接告诉 Cascade：

> 运行 `dsfix next` 获取下一个任务，然后帮我修复

Cascade 会：
1. 读取任务信息
2. 查看相关文件
3. 修复问题
4. 等待你确认后提交

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
