# 身份平台云端交接清单

- **状态**：Active
- **负责人**：Platform Engineering
- **最后复审**：2026-09-26
- **复审周期**：每次交接

本文件面向新的云端会话。它只依赖已推送的 Git 仓库和 `main` 分支；不要求本机 worktree、临时分支、会话记录或尚未提交的文件存在。

## 已合并基线

[PR #17](https://github.com/hydrz/starter/pull/17) 已合并。云端会话应从当前 `main` 开始：

```bash
git switch main
git pull --ff-only origin main
```

基线提交为：

```text
6c45a9e feat: add identity platform foundation (#17)
```

该提交已包含并经验证：

| Workstream | 已交付能力 |
| --- | --- |
| A — Foundation | 分组配置、Ed25519 JWT keyset、公共 TypeSpec 错误响应、ADR、架构文档、Ledger、契约图谱。 |
| B — Identity | Argon2id 密码、短期 Ed25519 access JWT、opaque refresh family、防重放、API key、认证 API。 |
| C — Organization + RBAC | 自动个人组织、成员和邀请、Casbin domain RBAC、公告租户化、组织 API。 |
| D — Delivery | PostgreSQL outbox worker、SMTP/Noop channel、HTML/Text 模板、notification intent 到 delivery 的路由。 |

基线验证已通过：

```bash
pnpm check
```

```bash
pnpm test
```

`pnpm test:race` 需要启用 CGO 且具备 C 编译环境；在满足该前提的云端环境中应补跑：

```bash
CGO_ENABLED=1 pnpm test:race
```

## 云端会话必须先阅读的权威文件

按以下顺序读取，所有新实现必须与其一致：

1. [身份平台架构](../architecture/identity-platform.md)
2. [身份平台实施台账](identity-platform-ledger.md)
3. [身份平台契约映射](identity-platform-contract-map.md)
4. [ADR 索引](../adr/README.md)，尤其 ADR-0003 至 ADR-0007
5. [`AGENTS.md`](../../AGENTS.md)

这些文件已定义：单 Go 二进制约束、TypeSpec/Goose/sqlc 的 SSOT 边界、JWT/refresh 模型、Casbin 三段授权、delivery 与 notification 分层、Stripe entitlement 投影，以及工作流验证要求。

## 下一步：重新启动第二波 workstream

此前本地会话曾启动 Stripe 和高级认证实现，但它们在 **提交前** 被上游运行时中断。它们未进入 `main`、未进入任何可依赖的远程分支，云端会话不应假设这些工作存在或尝试恢复本地临时目录。

应在当前 `main` 创建新的隔离分支/worktree，重新实施并验证下列两个 workstream。两个实现可并行，但共享文件由主 Agent 串行整合；每个完成后必须先提交，再成为后续工作的基线。

### E — Stripe 支付、订阅与权益

目标：支持一次性 Checkout 与循环订阅，但把业务能力统一投影为组织级 entitlement。

**应新增或扩展的边界**：

- 新迁移和 sqlc 查询：`billing_accounts`、`checkout_sessions`、`subscriptions`、`one_time_purchases`、`stripe_webhook_events`、`entitlements`；金额必须使用最小货币单位整数与 ISO 币种；Stripe event ID 必须唯一。
- `internal/billing/` 是唯一可导入 Stripe SDK 的领域包；其他模块只能依赖窄的 `EntitlementReader`。
- TypeSpec `packages/contracts/features/billing/`：组织 billing summary、创建 Checkout session、创建 Customer Portal session。
- 组织作用域 billing route 必须同时经过：身份认证 → 当前 membership → Casbin domain permission → billing account 归属复核。
- 服务器端 catalog 只接受受控 `priceKey`，再映射到 Stripe Price ID；浏览器不得传入金额、Stripe customer/price/subscription ID 或任意 success/cancel URL。
- Webhook 使用独立 raw-body handler：先限制 body 大小、验证 `Stripe-Signature`，再解析；Webhook 是 entitlement/subscription 状态的唯一来源。Checkout 回跳不能直接授予权益。
- 用唯一 event ID 的持久化记录保证幂等；瞬态持久化失败必须返回 5xx 以触发 Stripe 重试，或进入有证据的重试流程。
- 计费事件如需通知，只写 notification/outbox intent，绝不直接发 SMTP。

**主 Agent 必须验收的安全不变量**：

- 重复或乱序 webhook 不会重复授予、错误撤销或丢失权益；
- 取消订阅不会错误撤销已经获得的一次性权益；
- 任意跨组织 customer/portal/checkout 访问被拒绝；
- 不记录 webhook 原文、支付密钥或支付卡数据到日志和业务表；
- 不让前端回跳、HTTP 状态或浏览器数据成为付款完成的事实来源。

### F — Email OTP、TOTP、OAuth 与 Passkey

目标：在现有 `internal/auth.Service` 上扩展认证方式；所有成功路径必须复用 Workstream B 的同一 JWT/refresh family 签发逻辑。

**应新增或扩展的边界**：

- 新的 forward-only migration 与 sqlc 查询：email OTP challenges、TOTP factor、recovery codes、MFA challenge、OAuth account、OAuth authorization state、WebAuthn credential、WebAuthn challenge。
- Email OTP：代码只存 HMAC digest，具备过期、尝试次数、单次消费、按邮箱/IP 限流；请求对未知邮箱保持不可枚举语义；发送走现有 outbox/notification/delivery 管道。
- TOTP：使用标准实现；secret 以独立对称密钥进行 AES-GCM 加密存储；两步 enrollment；recovery code 仅展示一次、只存 hash、单次消费。是否启用后强制 MFA 必须成为显式文档化决策。
- Google/GitHub OAuth：持久化一次性 state、nonce 和 PKCE verifier；严格固定回调 origin；身份唯一键为 `(provider, provider_subject)`，绝不能按邮箱自动关联。链接已有账号必须在已认证且近期重认证的会话中显式执行。
- Passkey：采用成熟纯 Go WebAuthn 库；challenge 服务端持久化、短时有效、单次消费；注册必须要求已认证会话；discoverable 登录成功后调用既有统一 session issuer。
- 扩展现有 TypeSpec `packages/contracts/features/auth/`，再执行 `pnpm generate`；不得手写 ogen/Orval 文件。

**主 Agent 必须验收的安全不变量**：

- OTP/TOTP/OAuth/Passkey 没有平行 JWT、refresh token 或 session 逻辑；
- challenge/state/recovery code 均具备服务端持久化、过期和原子单次消费；
- TOTP secret 从不明文存储；
- OAuth provider email 不是账号关联依据；
- Passkey 注册未认证时被拒绝；
- WebAuthn 验证 origin、RP ID、challenge、credential 归属和 sign counter。

## 云端实施与整合协议

1. 从 `main` 创建 feature 分支，或由主 Agent 创建隔离 worktree；不可依赖本机临时分支名。
2. 先修改 TypeSpec/SQL 源文件，再执行生成：

```bash
pnpm generate
```

3. 严禁手改以下生成路径：
   - `spec/generated/`
   - `internal/api/`
   - `internal/store/`
   - `apps/web/src/api/generated/`
4. 每个 workstream 完成前必须更新：
   - [身份平台实施台账](identity-platform-ledger.md)
   - [身份平台契约映射](identity-platform-contract-map.md)
   - 需要长期决策时新增/更新 ADR
5. 每个 workstream 必须在独立分支中提交；主 Agent 审阅 diff、重跑生成与测试后再合并。
6. 合并后必须在最新集成分支运行：

```bash
pnpm check
```

```bash
pnpm test
```

7. E/F 都完成并合并后，才能开始 Workstream G：前端 access-token coordinator、认证/组织/安全设置/计费 UI，以及真实浏览器端到端验证。

## 云端运行注意事项

- 配置使用 `.env.example` 中定义的变量；不要提交实际密钥或生产连接串。
- 完整认证、SMTP、OAuth、WebAuthn、Stripe 验证需由云端/CI 注入测试凭据；缺少可选服务时应覆盖“disabled”配置路径。
- Stripe 测试使用独立 test-mode key 和测试 webhook secret；不混用生产凭据。
- WebAuthn 需要可验证的 HTTPS origin（本地开发例外按配置与浏览器规则处理）。
- 在云端实现中不创建依赖本机路径、个人 worktree、交互式终端状态或未提交文件的文档和脚本。
