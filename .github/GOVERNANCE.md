# GitHub 仓库治理

## 稳定检查名称

在 `main` 分支保护中要求以下检查通过：

- `Quality`
- `Database`
- `Build`

同时启用：

- 合并前至少一次批准；
- 要求 CODEOWNERS 审核；
- 新提交后撤销旧批准；
- 要求对话全部解决；
- 禁止直接 push 和强制 push；
- 管理员遵守相同规则。

工作流内部只调用根目录已有命令，不复制 lint、测试或构建实现。Job 名称属于分支保护接口，修改时必须同步仓库设置和本文档。

## 首次启用清单

1. 将 `CODEOWNERS` 中的 `@example/*` 替换为真实且拥有写权限的 GitHub 团队；
2. 将安全 Issue 联系地址中的示例仓库替换为当前仓库；
3. 在 Repository Settings 中启用上述分支保护；
4. 确认 GitHub Actions 仅具有默认只读权限，发布工作流单独申请 `contents: write`；
5. 验证 Dependabot 能访问私有 registry；需要凭据时使用 Dependabot secrets；
6. 创建测试 tag 验证 draft release，再删除测试 release 和 tag。

在完成第 1、2 步前，不应启用 CODEOWNERS 必需审核或公开 Issue 表单。

## Action 供应链

工作流中的第三方 Action 固定到完整 commit SHA，并在行尾标注对应主版本。Dependabot 提议升级后，应核对 Action 官方发布说明和 commit 来源，再接受新的 SHA。

## 发布边界

推送符合 `vMAJOR.MINOR.PATCH` 的 tag 会验证代码，构建 Linux amd64 单二进制及校验和，推送对应版本和 `latest` GHCR 镜像，并创建 **draft release**。草稿仍需负责人核对 release notes、migration 和镜像后手工发布；工作流不会自动部署到任何环境。
