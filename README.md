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

## 使用方法

### 1. 同步 Issues

```bash
dsfix sync
```

### 2. 查看状态

```bash
dsfix status   # 总体进度
dsfix list     # 按类型分组查看待处理 issues
```

### 3. 批量修复（推荐）

查看有哪些类型的 issues：
```bash
dsfix list
```

批量修复同一类型的所有 issues：
```bash
dsfix batch SCC-S1039           # 修复所有该类型 issues
dsfix batch SCC-S1039 -l 10     # 限制只处理 10 个
```

Cascade 会一次性修复所有位置，确认后：
```bash
dsfix complete-batch            # 一次性提交所有修复
dsfix skip-batch                # 跳过并恢复所有修改
```

### 4. 单个修复

```bash
dsfix next                      # 获取下一个任务
# Cascade 修复后
dsfix complete                  # 提交
dsfix skip                      # 跳过并恢复
```

## 工作流程

### 批量修复流程（推荐，效率高）

```
1. dsfix sync                   # 拉取 issues
2. dsfix list                   # 查看各类型数量
3. dsfix batch <shortcode>      # 批量获取同类型任务
4. [Cascade 批量修复]
5. dsfix complete-batch         # 一次性提交
6. 重复 3-5 直到完成
```

### 单个修复流程

```
1. dsfix sync
2. dsfix next
3. [Cascade 修复]
4. dsfix complete
5. 重复 2-4
```

## 与 Cascade 集成

**批量修复：**
> 运行 `dsfix batch SCC-S1039` 获取任务，然后帮我批量修复

**单个修复：**
> 运行 `dsfix next` 获取下一个任务，然后帮我修复

Cascade 会：
1. 读取任务信息
2. 修复所有位置
3. 展示修改内容和 commit message
4. 等待你确认
5. 确认后自动提交

## 命令参考

| 命令 | 说明 |
|------|------|
| `dsfix init` | 初始化配置文件 |
| `dsfix sync` | 从 DeepSource 同步 issues |
| `dsfix status` | 查看任务统计 |
| `dsfix list` | 按类型分组查看待处理 issues |
| `dsfix next` | 输出下一个任务 |
| `dsfix batch <shortcode>` | 批量输出同类型任务 |
| `dsfix batch <shortcode> -l N` | 限制批量数量 |
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
