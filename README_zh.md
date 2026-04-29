# DSFix

一个 Claude Code plugin，把 DeepSource 代码质量 issue 按 shortcode 分组，让 Claude 带着你一类一类修，每类一个 commit。

English: [README.md](./README.md)

## 为什么

DeepSource 经常一下子甩几百个 issue 给你。一次修一个 PR 太慢；一次全修又给不出 review-friendly 的 diff。DSFix 按 **shortcode**（比如 `RVV-B0012`）分组，每个 commit 只动一类 issue，跨多少文件都行。

DSFix 是个 Claude Code plugin：装一次，之后跟 Claude 用大白话聊就能跑完整个 sync → 修 → commit 的循环。

## 安装

```
/plugin marketplace add github:zealllot/dsfix
/plugin install dsfix@zealllot-tools
```

需要 Python 3 + `PyYAML`。如果是 PEP 668 系统（新版 macOS / Ubuntu 24+），pip 会拒绝直接装，改用：

```
pipx install PyYAML
# 或者
python3 -m pip install --break-system-packages PyYAML
```

## 配置（每个仓库一次）

1. 申请 DeepSource Personal Access Token：<https://app.deepsource.io/settings/tokens> （每人自己申请，不要共用）。
2. 设环境变量：
   ```bash
   export DEEPSOURCE_API_TOKEN="..."
   ```
3. 在你的项目目录里跟 Claude 说：
   > 跑 dsfix init

   会生成 `.dsfix.yaml` 模板，填好 `repository.owner`、`repository.name`，以及你想加的 filter。

## 使用

跟 Claude 说人话就行：

> 修一下这个项目的 DeepSource 问题

或显式召唤：

> /dsfix

Claude 会：
1. 运行 `dsfix start`，把所有待修 issue 类型列成表展示给你。
2. 等你选序号或 shortcode。
3. 运行 `dsfix batch <shortcode>`，对**每一处**应用相同的修复 pattern。
4. 跑 verify 命令（配置过的 / 自己判断的）。
5. 给你总结改动 + 建议 commit message。
6. 你确认 → `dsfix complete-batch` 自动 stage + commit。你拒绝 → `dsfix skip-batch` 自动 revert。

循环直到修完。

## 配置参考

`.dsfix.yaml` 放在仓库根目录。完整说明：[`plugins/dsfix/skills/dsfix/references/config.md`](./plugins/dsfix/skills/dsfix/references/config.md)。

```yaml
deepsource:
  api_token: ""                  # 推荐用 DEEPSOURCE_API_TOKEN 环境变量

repository:
  owner: ""
  name: ""

filter:
  categories: []                 # Bug Risk / Anti-pattern / Security / Performance / Typecheck / Style / Documentation
  severities: []                 # critical / major / minor
  limit:                         # 最多拉多少；留空 = 不限
  paths_include: []              # glob，支持 **；比如 ["internal/**"]
  paths_exclude: []              # 比如 ["vendor/**", "**/*_test.go"]

verify:
  command: ""                    # 比如 "go build ./..."；留空 = AI 自己判断
```

## 目录结构

- `.claude-plugin/marketplace.json` —— 把这个 repo 注册为 Claude Code marketplace
- `plugins/dsfix/` —— plugin 本体
  - `.claude-plugin/plugin.json` —— plugin manifest
  - `skills/dsfix/SKILL.md` —— skill 入口
  - `skills/dsfix/references/` —— workflow / config / token 申请的详细说明
  - `skills/dsfix/scripts/` —— Python 实现（不再依赖 Go）
  - `commands/dsfix.md` —— `/dsfix` slash command
  - `agents/dsfix-fixer.md` —— 大批量委派用的 subagent

## 架构

`dsfix.py` 暴露一套 CLI，和原 Go 版本子命令一一对应。Claude Code 通过 Bash 调用它。Skill 描述里出现 DeepSource 等关键词时 Claude 会自动激活；slash command 用于显式召唤；subagent 在一次要修 50+ 处时可选用。

## 从 Go 版本迁移

Go binary 在 `v0-go-final` tag 还能用：

```bash
git checkout v0-go-final
go install ./cmd/dsfix
```

`main` 分支这个 commit 之后是纯 plugin 版，Go binary 不再维护。

## License

MIT
