---
name: dsfix
description: 修复 DeepSource 代码质量 issue —— 从 DeepSource 拉取 issue 列表，按 shortcode 类型批量修复，每个类型独立 commit。当用户提到 DeepSource、要清理代码质量警告、想批量修复同类型问题、或运行 dsfix/dsfix start/dsfix sync 时使用。Fix DeepSource code quality issues by fetching the issue list, batching fixes per shortcode, and committing one type at a time. Use when the user mentions DeepSource, asks to clean up code quality warnings, wants to batch-fix same-type issues, or runs dsfix commands.
---

# DSFix

修复 DeepSource issue 的工作流。每个 issue type（shortcode）一个 commit，让 review 简单。

## 前置检查

每次开工前：

1. **确认 `.dsfix.yaml` 存在**：在仓库根读 `.dsfix.yaml`。如果不存在，运行：
   ```bash
   python3 ${CLAUDE_PLUGIN_ROOT}/skills/dsfix/scripts/dsfix.py init
   ```
   然后让用户填好 `repository.owner` / `repository.name`，`api_token` 推荐用 env var (`DEEPSOURCE_API_TOKEN`)。
2. **确认 token 已设置**：`echo $DEEPSOURCE_API_TOKEN` 非空。否则提示用户去 https://app.deepsource.io/settings/tokens 申请。

为了后续步骤简洁，**给 dsfix 命令起个 alias**，每次会话开始时定义一次：

```bash
alias dsfix="python3 ${CLAUDE_PLUGIN_ROOT}/skills/dsfix/scripts/dsfix.py"
```

或者直接在每次调用时写完整路径。

## 主流程

### 第 1 步：列任务

```bash
dsfix start
```

输出会显示一张表格（Shortcode + 问题描述 + 数量）。**完整地展示给用户看**，让他们选择要修哪一类。

如果输出里说 `No pending tasks`，那 `dsfix start` 已经自动同步过了，告诉用户没有待修任务。

### 第 2 步：批量修复

用户选了序号或 shortcode 之后，运行：

```bash
dsfix batch <SHORTCODE>
```

输出会描述：
- 该 shortcode 的描述
- 所有需要修复的位置（按 file 升序、line 降序排列 —— 同一文件从下往上修，避免行号偏移）
- 该用什么 verify 命令

**关键：你必须修复表格里的每一处，不能只修一部分**。修完后：

1. 运行 verify 命令（dsfix 输出里会写）。失败就修复编译/lint 错误重试。
2. 简要总结改动：涉及哪些文件、改了什么 pattern。
3. 展示 dsfix 输出里的建议 commit message。
4. 让用户确认。

### 第 3 步：提交或回退

- 用户确认 → `dsfix complete-batch`（自动 stage 修改的文件 + git commit）
- 用户拒绝 → `dsfix skip-batch`（自动 `git checkout --` 回退所有改动）

提交完后回到第 1 步循环，直到用户说够了或者所有 shortcode 修完。

## 大批量委派（可选）

如果一次有 50+ 处要修，主对话会很长。把第 2 步的修复循环委派给 `dsfix-fixer` subagent，让它专心修一类，回来汇报。然后主 agent 跟用户确认 + commit。

## 详细参考

- `references/workflow.md` — 完整步骤 + 边界情况
- `references/config.md` — `.dsfix.yaml` 字段说明（filter / verify command / path globs）
- `references/api-token.md` — DeepSource API token 怎么申请
