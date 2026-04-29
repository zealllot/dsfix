# DeepSource API Token 申请

每位使用者各自申请，不要共用 token。

## 步骤

1. 登录 DeepSource：https://app.deepsource.io
2. 打开 Personal Access Tokens 页面：https://app.deepsource.io/settings/tokens
3. 点 **Create Token**，给一个能识别的名字（比如 `dsfix-cli`）。
4. 选权限范围：至少包含读取 issue 的权限（默认就够）。
5. 复制生成的 token —— 关掉页面就再也看不到了。

## 设置环境变量

加到你的 shell 配置（`~/.zshrc` / `~/.bashrc`）：

```bash
export DEEPSOURCE_API_TOKEN="ds_pat_xxxxxxxxxxxx"
```

然后 `source ~/.zshrc` 或重开终端。

## 验证

```bash
echo $DEEPSOURCE_API_TOKEN | head -c 10  # 应该输出前 10 位
```

或直接跑 `dsfix sync`，能拉到 issue 就说明 token 正常。

## 安全注意事项

- **不要把 token 提交到 git**。`.dsfix.yaml` 已经被默认模板的 README 引导加到 `.gitignore`，但 token 推荐放环境变量而不是 yaml 里。
- token 泄露了就立刻去 https://app.deepsource.io/settings/tokens 撤销重发。
- CI/CD 里用项目级 secret，不要用个人 token。
