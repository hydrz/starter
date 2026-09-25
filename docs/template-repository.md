# GitHub 模板仓库指南

- **状态**：Active
- **负责人**：Developer Experience
- **最后复审**：2026-09-25
- **复审周期**：90 天

## 创建流程

1. 在 `hydrz/starter` 页面选择 **Use this template**，创建一个不名为 `starter` 的新仓库。
2. 等待 `Initialize template repository` GitHub Actions 工作流完成。
3. 确认 `main` 中出现 `chore: initialize project from template` 提交。
4. 检查 Go module、pnpm package scope、API 标题、Compose 项目、镜像和本地数据库名称。
5. 按[首次启用清单](../.github/README.md)配置 CODEOWNERS、分支保护、安全地址和发布权限。

## 自动命名规则

| 目标 | 默认值 | 示例：`acme/order-console` |
| --- | --- | --- |
| Go module | `github.com/<owner>/<repository>` | `github.com/acme/order-console` |
| pnpm scope | 仓库名的小写 kebab-case | `@order-console/web` |
| 项目 slug | 仓库名的小写 kebab-case | `order-console` |
| 展示名 | 按 `-`、`_`、`.` 分词后首字母大写 | `Order Console` |

项目 slug 会用于 Compose project、容器镜像以及本地 PostgreSQL 的数据库、用户和开发密码。这些值只是本地默认值，生产环境必须使用密钥管理平台提供独立凭据。

## 手动执行

若新仓库禁用了 Actions，在业务修改之前执行：

```bash
node tools/initialize-template.mjs --repository acme/order-console
git add --all
git commit -m "chore: initialize project from template"
git push
```

可使用 `--display-name "Order Console"` 改写展示名，使用 `--module example.com/acme/order-console` 改写 Go module。脚本会拒绝初始化源模板 `hydrz/starter`，防止意外改写模板本身。

## 安全与失败处理

- 工作流仅授予 `contents: write`，不使用 PAT 或外部密钥；
- 源模板中的 job 条件会阻止对 `hydrz/starter` 执行初始化；
- 初始化提交会删除 marker、一次性 workflow 和初始化脚本，防止重复改名；
- 若分支保护禁止 `GITHUB_TOKEN` 推送，先在初始化完成后再启用保护，不要为脚本配置长期高权限 PAT。
