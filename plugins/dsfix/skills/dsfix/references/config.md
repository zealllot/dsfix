# `.dsfix.yaml` 配置参考

```yaml
deepsource:
  # 推荐通过 DEEPSOURCE_API_TOKEN 环境变量提供，不要写到这里
  api_token: ""

repository:
  # VCS 组织/用户
  owner: ""
  # 仓库名
  name: ""

filter:
  # 类别白名单（留空 = 全部）
  # 可选: Bug Risk, Anti-pattern, Security, Performance, Typecheck, Style, Documentation
  categories: []

  # 严重程度白名单（留空 = 全部）
  # 可选: critical, major, minor
  severities: []

  # 单次 sync 最多拉多少 issue（留空 = 不限）
  limit:

  # 路径 glob 过滤（支持 ** 跨目录）
  # 例：
  #   paths_include: ["internal/**", "cmd/**"]
  #   paths_exclude: ["vendor/**", "**/*_test.go"]
  paths_include: []
  paths_exclude: []

verify:
  # 修复后用来验证编译/通过的命令。留空 → AI 自己判断
  # 例: "go build ./...", "pnpm typecheck", "make lint"
  command: ""
```

## 环境变量覆盖

`DEEPSOURCE_API_TOKEN` 总是优先于 yaml 里的 `api_token`。如果 yaml 里写了 token 但环境变量为空，dsfix 会打印一行 stderr 警告（提示这是 secret）。

## Glob 语法

- `*` 匹配单个路径段内任意字符（不跨 `/`）
- `**` 匹配 0 个或多个路径段（跨 `/`）
- `?` 匹配单个字符
- `[abc]`、`[!abc]` 字符类

例子：

| pattern | matches |
|---|---|
| `internal/**` | `internal/foo.go`、`internal/a/b.go` |
| `**/*_test.go` | `foo_test.go`、`a/b/foo_test.go` |
| `vendor/**` | 任何 `vendor/` 子树文件 |
| `*.go` | 仅根目录的 `.go` 文件（不含子目录） |

## 多个项目

每个项目独立一份 `.dsfix.yaml`（不要共享），但可以共用同一个 `DEEPSOURCE_API_TOKEN` 环境变量。
