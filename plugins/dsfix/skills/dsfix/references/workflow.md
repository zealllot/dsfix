# DSFix 详细工作流

## 状态机

每个 task 对应 DeepSource 的一处 occurrence：

```
pending ──(dsfix batch)──> in_progress ──(complete-batch)──> fixed
                                       ──(skip-batch)─────> skipped
                                       ──(中断/重启)──────> pending (via reset-progress)
```

状态持久化在 `<repo>/.dsfix/tasks.json`。

## 命令参考

| 命令 | 作用 |
|------|------|
| `dsfix init` | 写 `.dsfix.yaml` 模板 |
| `dsfix sync` | 从 DeepSource 拉 issues，新增到 store |
| `dsfix status` | 打印各状态的任务数量 |
| `dsfix list` | 按 shortcode 分组列待修 issue |
| `dsfix start` | 给 AI 用的入口 prompt（含选择菜单） |
| `dsfix next` | 输出下一个单任务 prompt（mark in_progress） |
| `dsfix batch <shortcode>` | 输出该 shortcode 的批量修复 prompt（mark all in_progress） |
| `dsfix batch <shortcode> -l N` | 限制只输出前 N 个 |
| `dsfix complete` | 单任务：stage + commit + mark fixed |
| `dsfix complete-batch` | 批量：stage 所有 in_progress 文件 + commit + mark fixed |
| `dsfix skip` | 单任务：`git checkout --` 回退 + mark skipped |
| `dsfix skip-batch` | 批量：回退所有 in_progress 文件 + mark skipped |
| `dsfix reset-progress` | 把 in_progress 全部丢回 pending（用于中断恢复） |
| `dsfix reset` | 把所有任务全部丢回 pending（重新开始） |

## 边界情况处理

### 中断后恢复

如果上一次会话在 `dsfix batch` 之后中断，`tasks.json` 里会留下 in_progress 任务。`dsfix start` 会在表头打印警告。
- 改动还在：用户决定继续修 → 直接修剩下的 → `dsfix complete-batch`
- 改动也丢了：先 `dsfix reset-progress`，然后重新 `dsfix batch <shortcode>`

### 同一文件多个 issue

`dsfix batch` 输出的列表里同一文件多处会按行号**降序**排列。**严格按这个顺序修**，否则前面的修改会让后面的行号失效。

### Verify 命令未配置

`.dsfix.yaml` 里 `verify.command` 留空时，dsfix 会让你「自行判断该项目用什么」。看 Makefile / package.json scripts / 项目语言决定：
- Go: `go build ./...` 或 `go test ./...`
- TypeScript: `pnpm typecheck` / `tsc --noEmit`
- Python: `python -m pyflakes` / `make lint`

### 修复影响了其他文件

`dsfix complete-batch` 默认只 stage 任务记录里的文件。如果你的修复改动了相邻文件（比如改 import path），那些文件不会被 commit。

解决：在确认前先用 `git status` 看一下 untracked + unstaged，提醒用户。或者在 commit 之前手动 `git add <extra-files>`。

### 单任务模式

`dsfix next` / `complete` / `skip` 是单任务模式，主要用于：
- 一次只想修一个特殊 case
- 调试 dsfix 本身

主流程**不**用单任务模式。

## 委派给 dsfix-fixer subagent

适用：一次要修 30+ 处，主对话窗口想保持干净。

主 agent 调用方式：用 Task tool，传入 subagent_type="dsfix-fixer"，prompt 里说明 shortcode + verify 命令 + 仓库路径。subagent 完成后返回总结，主 agent 接着跟用户确认 + commit。
