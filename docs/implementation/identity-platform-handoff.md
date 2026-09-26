# 身份平台实施交接清单

- **状态**：Active
- **负责人**：Platform Engineering
- **最后复审**：2026-09-26
- **复审周期**：每次交接

本文件用于在新会话恢复身份平台实施；架构决策以 ADR、架构文档、契约图谱和实施台账为准。

## 已进入 PR 的范围

当前 PR：[hydrz/starter#17](https://github.com/hydrz/starter/pull/17)，功能分支 `feat/identity-platform-foundation` 包含以下已验证 workstream：

| Workstream | 提交 | 交付 |
| --- | --- | --- |
| A — Foundation | `1336f6f` | 配置加载器、公共 TypeSpec 错误响应、ADR、架构文档、契约图谱与实施台账。 |
| B — Identity | `2444864` | Argon2id 密码、Ed25519 access JWT、opaque refresh family、防重放、API key、认证 TypeSpec/ogen/Orval。 |
| D — Delivery | `4b65633` | PostgreSQL outbox worker、SMTP/Noop channel、HTML/Text 模板、notification intent 到 delivery 的路由。 |
| C — Organization + RBAC | `c72befe`, `83aec7d` | 个人组织自动创建、成员/邀请、Casbin domain RBAC、公告租户化、组织 TypeSpec/ogen/Orval。 |

功能分支已通过：

```bash
pnpm check
```

```bash
pnpm test
```

`pnpm test:race` 尚未通过：本地 Go 工具链要求 `CGO_ENABLED=1` 才支持 `-race`，需要具备 C 编译环境后再执行。

## PR 与合并状态

- PR #17 已推送并已开启 Auto-fix/CI 监控。
- 当前仓库不允许 GitHub auto-merge；PR 创建时 `reviewDecision=REVIEW_REQUIRED`。
- 等 CI 和审查要求满足后，需要有权限的用户在 GitHub 手动合并。
- PR 合并前不要清理 `feat/identity-platform-foundation` 分支。

## 主分支与功能分支

当前会话工作目录位于 `feat/identity-platform-foundation`。为创建 PR，本地 `main` 已安全恢复到 `origin/main` 的 `0c57756`；完整 A–D/C 成果均保留在 PR 分支中。

```bash
git status --short --branch
```

```bash
git log --oneline origin/main..HEAD
```

## 未完成的第二波 worktree（严禁删除）

### Workstream E — Stripe 支付、订阅与权益

- 路径：`C:\Users\nan\github.com\hydrz\starter\.claude\worktrees\agent-a202de09c8133c81f`
- 分支：`worktree-agent-a202de09c8133c81f`
- 基线：`83aec7d`
- 状态：子 Agent 因上游网关 502 中断，未完成独立复核或提交。
- 已报告但尚未采信的工作：`00004_create_billing_and_entitlements.sql`、`db/queries/billing.sql`、`internal/billing/`、billing TypeSpec。
- 已知收尾编译问题：
  1. `internal/billing/handler.go` 缺少 `strings` import；
  2. `internal/billing/service.go` 有未使用 `time` import，且 `handleCheckoutSessionCompleted` 中 `rec` 未使用。

恢复时先检查 diff 与状态，修复编译，运行生成/测试/全量门禁，补 Ledger 并提交。主 Agent 必须审查：

- webhook 读取原始 body、先验签再解析、请求体有上限；
- `stripe_webhook_events` 使用 event ID 唯一约束实现幂等；
- Checkout redirect 不直接发放 entitlement；
- 价格、金额、customer ID、成功/取消 URL 不可由客户端任意指定；
- billing endpoint 同时经过 membership、Casbin 与 billing-account 归属验证。

### Workstream F — OTP、TOTP、OAuth、Passkey

- 路径：`C:\Users\nan\github.com\hydrz\starter\.claude\worktrees\agent-af8348bbdda8716d2`
- 分支：`worktree-agent-af8348bbdda8716d2`
- 基线：`83aec7d`
- 状态：子 Agent 同样因上游网关 502 中断，未完成独立复核或提交。
- 已报告但尚未采信的工作：`00005_create_advanced_auth_credentials.sql`、OTP/OAuth/WebAuthn 查询、`internal/auth` 扩展、auth TypeSpec 扩展。
- 中断前最后行动：修改 `internal/notification/service.go` 的 `handleAuthEvent` 中 `kind` 字段；必须审查该变动是否符合 delivery/notification 分层和模板路由。

恢复后必须复核：

- OTP/TOTP/OAuth/Passkey 成功路径均调用 Workstream B 的同一 session/refresh issuer；
- OTP、恢复码、OAuth state、PKCE verifier 与 WebAuthn challenge 均有服务端存储、过期和单次消费；
- TOTP secret 加密存储，恢复码只展示一次且仅存 hash；
- OAuth 身份唯一键为 `(provider, provider_subject)`，不得按邮箱自动关联；
- Passkey 注册需要既有认证，discoverable 登录成功后走统一 session 创建。

## 接手标准操作程序

1. 首先阅读：
   - [身份平台架构](../architecture/identity-platform.md)
   - [身份平台实施台账](identity-platform-ledger.md)
   - [身份平台契约映射](identity-platform-contract-map.md)
   - [ADR 索引](../adr/README.md)
   - `C:\Users\nan\.claude\plans\snuggly-painting-hejlsberg.md`
2. 不信任子 Agent 自述，先逐个检查 E/F：

```bash
git -C "C:\Users\nan\github.com\hydrz\starter\.claude\worktrees\agent-a202de09c8133c81f" status --short
```

```bash
git -C "C:\Users\nan\github.com\hydrz\starter\.claude\worktrees\agent-a202de09c8133c81f" log --oneline -5
```

```bash
git -C "C:\Users\nan\github.com\hydrz\starter\.claude\worktrees\agent-af8348bbdda8716d2" status --short
```

```bash
git -C "C:\Users\nan\github.com\hydrz\starter\.claude\worktrees\agent-af8348bbdda8716d2" log --oneline -5
```

3. 每个 workstream 必须先在自己的 worktree 完成并提交，再基于已提交的最新 main rebase；不要将子 Agent 未提交改动复制到主工作树。
4. 生成代码冲突时，修改 TypeSpec/SQL SSOT 后运行 `pnpm generate`；禁止手工解决 `internal/api/`、`internal/store/`、`spec/generated/`、`apps/web/src/api/generated/` 的内容冲突。
5. 合并每个 workstream 后，在功能分支重新运行：

```bash
pnpm check
```

```bash
pnpm test
```

6. E/F 合并后再启动 Workstream G：前端 access-token coordinator、登录/组织/安全设置/计费 UI 和真实浏览器 E2E 验证。

## 清理边界

- PR 合并后，可清理已纳入 PR 的 A/B/C/D 临时 worktree 与相应已合并分支。
- 严禁清理 `agent-a202de09c8133c81f` 与 `agent-af8348bbdda8716d2`，它们持有未提交的 E/F 工作。
- 确认远程 `main` 已包含 PR #17 后，才可删除 `feat/identity-platform-foundation` 的远程/本地分支和 A–D/C 临时 worktree。
