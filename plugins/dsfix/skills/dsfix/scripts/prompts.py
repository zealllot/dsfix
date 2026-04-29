"""Prompt generators for the AI assistant. Output is markdown with mixed
Chinese (instructions) and English (structural elements like table headers).

These mirror the Go version's prompt/prompt.go, but cleaned up to assume Claude
Code's native tool use rather than carrying coercive `<!-- AI_INSTRUCTION -->`
directives.
"""

from __future__ import annotations

from io import StringIO

from store import Status, Task


def _verify_line(verify_cmd: str) -> str:
    if not verify_cmd:
        return "运行项目标准 build/lint 命令验证修复（自行判断该项目用什么）。"
    return f"运行 `{verify_cmd}` 验证修复。"


def fix_prompt(t: Task, verify_cmd: str) -> str:
    sb = StringIO()
    w = sb.write
    w("## DeepSource 修复任务\n\n")
    w(f"**Task ID:** `{t.id}`\n")
    w(f"**Issue:** {t.issue.title}\n")
    w(f"**Category:** {t.issue.category}\n")
    w(f"**Severity:** {t.issue.severity}\n")
    w(f"**Shortcode:** {t.issue.shortcode}\n")
    w(f"**Analyzer:** {t.issue.analyzer}\n\n")
    w("### 位置\n")
    w(f"- **文件:** `{t.issue.file_path}`\n")
    w(f"- **行号:** {t.issue.begin_line}-{t.issue.end_line}\n\n")
    w("### 描述\n")
    w(t.issue.description)
    w("\n\n")
    if t.issue.suggestion:
        w("### 建议\n")
        w(t.issue.suggestion)
        w("\n\n")
    w("### 流程\n")
    w("1. 读取上面位置的文件，只修复这一个 issue（最小改动）。\n")
    w(f"2. {_verify_line(verify_cmd)}\n")
    w("3. 展示 diff 和建议的 commit message：\n")
    w(f"   ```\n   {t.commit_message()}\n   ```\n")
    w("4. 让用户确认。\n")
    w("5. 用户确认 → 运行 `dsfix complete`。用户拒绝 → 运行 `dsfix skip`。\n\n")
    return sb.getvalue()


def batch_fix_prompt(tasks: list[Task], verify_cmd: str) -> str:
    if not tasks:
        return "没有待修复的任务。"

    first = tasks[0]
    sb = StringIO()
    w = sb.write
    w("## DeepSource 批量修复任务\n\n")
    w(f"**问题类型:** {first.issue.title}\n")
    w(f"**Shortcode:** {first.issue.shortcode}\n")
    w(f"**Category:** {first.issue.category}\n")
    w(f"**总数:** {len(tasks)} 处\n\n")
    w("### 描述\n")
    w(first.issue.description)
    w("\n\n")
    w("### 待修复位置\n")
    w("任务按 file 升序、line 降序排列 —— 同一文件内从下往上修，避免行号偏移。\n\n")
    w("| # | File | Lines |\n")
    w("|---|------|-------|\n")
    for i, t in enumerate(tasks, 1):
        w(f"| {i} | `{t.issue.file_path}` | {t.issue.begin_line}-{t.issue.end_line} |\n")
    w("\n")
    w("### 流程\n")
    w(f"1. 对上表 **每一处**（共 {len(tasks)} 处）应用相同的修复模式，不要中途停下。\n")
    w("2. 最小改动 —— 只修这一个 issue，不要顺手重构无关代码。\n")
    w(f"3. {_verify_line(verify_cmd)}\n")
    w("4. 简要总结改动（涉及的文件、改了什么）。\n")
    w(f"5. 建议的 commit message：\n")
    w(f"   ```\n   fix({first.issue.shortcode}): {first.issue.title} ({len(tasks)} occurrences)\n   ```\n")
    w("6. 让用户确认。\n")
    w("7. 用户确认 → 运行 `dsfix complete-batch`。用户拒绝 → 运行 `dsfix skip-batch`。\n\n")
    return sb.getvalue()


def start_prompt(tasks: list[Task]) -> str:
    sb = StringIO()
    w = sb.write

    in_progress_count = 0
    groups: dict[str, list[Task]] = {}
    for t in tasks:
        if t.status == Status.IN_PROGRESS:
            in_progress_count += 1
        if t.status in (Status.PENDING, Status.IN_PROGRESS):
            groups.setdefault(t.issue.shortcode, []).append(t)

    if not groups:
        return "✅ 没有待处理的任务！所有 issues 已修复完成。"

    if in_progress_count > 0:
        w(f"⚠️ 有 {in_progress_count} 个任务处于进行中状态（可能是之前中断的）。运行 `dsfix reset-progress` 可重置。\n\n")

    sorted_groups = sorted(groups.items(), key=lambda kv: -len(kv[1]))
    total = sum(len(v) for _, v in sorted_groups)

    w("## 📋 DSFix 任务列表\n\n")
    w(f"**待处理:** {total} 个 issues，{len(sorted_groups)} 种类型\n\n")
    w("| 序号 | Shortcode | 问题描述 | 数量 |\n")
    w("|:----:|-----------|----------|-----:|\n")

    show = sorted_groups[:15]
    for i, (sc, ts) in enumerate(show, 1):
        title = ts[0].issue.title
        if len(title) > 45:
            title = title[:42] + "..."
        w(f"| **{i}** | `{sc}` | {title} | {len(ts)} |\n")

    if len(sorted_groups) > 15:
        w(f"| ... | ... | *还有 {len(sorted_groups) - 15} 种其他类型* | ... |\n")

    w("\n请选择序号或 shortcode，然后运行 `dsfix batch <shortcode>` 开始批量修复。\n")
    return sb.getvalue()


def progress_report(stats: dict[Status, int]) -> str:
    total = sum(stats.values())
    sb = StringIO()
    w = sb.write
    w("## DSFix 进度\n\n")
    w(f"**总任务数:** {total}\n\n")
    w("| 状态 | 数量 |\n")
    w("|------|-----:|\n")
    w(f"| Pending | {stats.get(Status.PENDING, 0)} |\n")
    w(f"| In Progress | {stats.get(Status.IN_PROGRESS, 0)} |\n")
    w(f"| Fixed | {stats.get(Status.FIXED, 0)} |\n")
    w(f"| Skipped | {stats.get(Status.SKIPPED, 0)} |\n")
    w(f"| Failed | {stats.get(Status.FAILED, 0)} |\n")
    if total > 0:
        fixed = stats.get(Status.FIXED, 0)
        w(f"\n**完成度:** {fixed / total * 100:.1f}%\n")
    return sb.getvalue()


def group_stats(tasks: list[Task]) -> str:
    groups: dict[str, list[Task]] = {}
    for t in tasks:
        if t.status == Status.PENDING:
            groups.setdefault(t.issue.shortcode, []).append(t)

    sb = StringIO()
    w = sb.write
    w("## 按类型分组的待修复 issue\n\n")
    w("| Shortcode | Title | Count |\n")
    w("|-----------|-------|------:|\n")
    sorted_groups = sorted(groups.items(), key=lambda kv: -len(kv[1]))
    for sc, ts in sorted_groups:
        if ts:
            w(f"| {sc} | {ts[0].issue.title} | {len(ts)} |\n")
    return sb.getvalue()
