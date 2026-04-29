---
description: Run the DSFix workflow — list pending DeepSource issue types, pick one, batch-fix, commit per type.
argument-hint: "[shortcode]"
allowed-tools: Bash(python3 *), Bash(git *), Bash(echo *), Read, Edit, Grep, Glob, Task
---

按 dsfix 工作流修复 DeepSource issue。

先定义 alias 简化后续调用：

```bash
alias dsfix="python3 ${CLAUDE_PLUGIN_ROOT}/skills/dsfix/scripts/dsfix.py"
```

## 流程

如果 `$ARGUMENTS` 为空：
1. 跑 `dsfix start`，把任务表完整展示给用户。
2. 等用户输入序号或 shortcode。
3. 拿到 shortcode 之后继续。

如果 `$ARGUMENTS` 已经是一个 shortcode，直接用它。

## 批量修复

1. `dsfix batch <SHORTCODE>` — 输出待修位置列表 + verify 命令。
2. **修复每一处**（不要漏）。同一文件按行号降序，避免行号偏移。
3. 运行 dsfix 输出里指定的 verify 命令。失败就修，直到通过。
4. 简要总结：涉及文件、改了什么 pattern。
5. 展示 dsfix 输出里的建议 commit message。
6. 让用户确认：
   - 确认 → `dsfix complete-batch`（自动 stage + commit）
   - 拒绝 → `dsfix skip-batch`（自动 `git checkout --` 回退）

## 大批量

50+ 处的话考虑用 Task tool 委派给 `dsfix-fixer` subagent 做修复 + verify 循环，主对话只负责确认 + commit。

详细规则：见 `${CLAUDE_PLUGIN_ROOT}/skills/dsfix/references/workflow.md`。
