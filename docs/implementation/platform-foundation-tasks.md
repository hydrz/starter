# 平台底座任务调度手册

- **状态**：Active
- **负责人**：Platform Engineering
- **最后复审**：2026-09-26
- **复审周期**：90 天

本文件是调度 LLM Agent 完成脚手架底座增强的唯一任务清单。每个任务卡可独立派发给一个 Agent。本项目是脚手架：只实现通用能力与扩展点，严禁写入任何具体产品（SaaS 业务、短剧、AI 应用）的业务逻辑。

## 1. 调度规则

1. 一个 Agent 只领取一个任务卡；只在“依赖”全部为 `Done` 后派发。
2. 分支命名：`feat/<任务ID小写>-<短名>`，例如 `feat/b1-otp-delivery`。每个任务一个 PR，PR 标题用 Conventional Commits。
3. 派发给 Agent 的提示词固定为：

   ```text
   你在仓库 starter 中工作。先完整阅读 AGENTS.md 与 docs/implementation/platform-foundation-tasks.md 的“全局约束”一节，
   然后只执行任务卡 <任务ID>。严格限定在任务卡“修改范围”内改动；超出范围的问题记录到 PR 描述的“遗留问题”，不要顺手修。
   完成后运行“完成条件”中的全部命令，全部通过后提交并推送到 <分支名>，创建 PR，PR 描述按任务卡“验收”逐条给出证据。
   最后在本文件“任务状态”表中把该任务改为 Review，并填入 PR 链接。
   ```

4. 并行派发时，同一“冲突组”的任务不得同时进行（它们修改同一文件）。
5. 评审 Agent 使用 `.agents/skills/review-change/SKILL.md`，逐条核对任务卡“验收”；不满足任一条即退回。
6. 合并后由调度者把状态改为 `Done`。

## 2. 全局约束（每个任务都必须遵守）

- 遵守 [AGENTS.md](../../AGENTS.md) 全部护栏；生成物只能通过 `pnpm generate` 更新。
- 新表、新查询按 [change-database-schema](../../.agents/skills/change-database-schema/SKILL.md) 流程，新 migration 使用下一个序号，禁止修改已有 migration。
- 新 HTTP 接口按 [change-api-contract](../../.agents/skills/change-api-contract/SKILL.md) 流程，先改 TypeSpec。
- 新能力必须“默认关闭或零配置可运行”：未配置相关环境变量时服务仍能启动，行为与现状一致。
- 新环境变量只能在 `internal/platform/config` 中读取与校验，并同步到 `.env.example`（附注释）；禁止在业务包中调用 `os.Getenv`。
- 外部服务（存储、支付、短信、LLM、监控）一律先定义 Go 接口（port），再提供一个真实适配器和一个内存/Noop 适配器；单测只用内存适配器。
- 时间一律经 `Clock` 接口注入；日志一律用注入的 `*slog.Logger`。
- 每个新包必须有单元测试；修复 bug 必须先写复现测试。
- 行为或架构选择有取舍时，新增 ADR（`docs/adr/NNNN-*.md`，并登记到 [ADR 索引](../adr/README.md)）。
- 新文档登记到 [文档目录](../documentation-catalog.md)。
- 完成条件：

  ```bash
  pnpm generate
  pnpm check
  pnpm test
  pnpm test:race
  ```

## 3. 任务状态

| ID | 名称 | 依赖 | 冲突组 | 状态 | PR |
| --- | --- | --- | --- | --- | --- |
| B1 | 修复 Email OTP 投递 | - | notify | Todo | |
| B2 | 可信代理与客户端 IP | - | http | Todo | |
| B3 | 配置入口收敛 | - | main | Todo | |
| F1 | 通用 outbox 分发与重试 | B1 | notify | Todo | |
| F2 | 模块注册机制 | B3 | main, http | Todo | |
| F3 | 仓储事务 helper | - | repo | Todo | |
| F4 | 真实数据库集成测试 | F3 | repo | Todo | |
| F5 | 可观测性 | F2 | http, main | Todo | |
| F6 | 通用 HTTP 限流 | B2, F2 | http | Todo | |
| C1 | 后台作业与定时任务 | F1 | notify | Todo | |
| C2 | 对象存储与上传 | F2 | - | Todo | |
| C3 | 积分账本与额度 | F2 | billing | Todo | |
| C4 | 支付提供方抽象 | C3 | billing | Todo | |
| C5 | 多客户端认证 | F2 | auth | Todo | |
| C6 | 流式响应约定 | F2 | - | Todo | |
| C7 | 审计日志 | F2 | - | Todo | |
| C8 | 通知渠道扩展 | F1 | notify | Todo | |
| C9 | 平台超级管理员 | C7 | auth | Todo | |
| C10 | 账号注销与数据导出 | C7 | auth | Todo | |
| D1 | 完整切片生成器 | F2 | tools | Todo | |
| D2 | 模块裁剪命令 | F2 | tools | Todo | |
| D3 | E2E 测试 | F4 | - | Todo | |
| D4 | 部署与迁移加固 | - | - | Todo | |
| D5 | 着陆页预渲染 | - | web | Todo | |

## 4. 任务卡

### B1 修复 Email OTP 投递

- **修改范围**：`internal/notification/`、`internal/delivery/template.go`、`internal/delivery/templates/`、相关测试。
- **步骤**：
  1. 写失败测试：`notification.Service.Handle` 收到 topic `auth.email_otp_requested` 时应渲染并通过 SMTP 通道发送。
  2. 在 `notification/domain.go` 与 `delivery/template.go` 增加该 topic 常量及对应 kind；新增 `auth_email_otp.html` / `auth_email_otp.txt` 模板（展示验证码与有效期）。
  3. 在 `Handle` 的 switch 中路由到该处理；payload 结构与 `internal/auth/otp.go` 写入的保持一致。
  4. 新增一个测试，遍历 `internal/auth` 中所有 `outboxTopic*` 常量，断言 notification 都能处理（防止再次断链）。
- **验收**：新测试在修复前失败、修复后通过；auth 模块产生的每个 topic 都有处理器和模板。

### B2 可信代理与客户端 IP

- **修改范围**：`internal/platform/config/`、`internal/platform/httpserver/`、`internal/auth/httpcontext.go`、`internal/billing/webhookhttp.go`、`.env.example`、`docs/deployment.md`。
- **步骤**：
  1. 新增配置 `TRUSTED_PROXIES`（逗号分隔 CIDR，默认空）。
  2. 移除全局 `middleware.RealIP`；新增 `httpserver.ClientIP` 中间件：仅当 `RemoteAddr` 落在可信 CIDR 内时，才从 `X-Forwarded-For` 自右向左取第一个非可信地址，结果写入 context。
  3. 提供 `httpserver.ClientIPFromContext(ctx)`；auth 和 billing 限流改用它。
  4. 测试：不可信来源伪造 XFF 不生效；可信来源多级 XFF 解析正确。
- **验收**：未配置 `TRUSTED_PROXIES` 时忽略所有转发头；deployment 文档说明反向代理场景如何配置。

### B3 配置入口收敛

- **修改范围**：`internal/platform/config/`、`apps/server/main.go`、`.env.example`。
- **步骤**：
  1. 把 `BILLING_PRICE_CATALOG`、`BILLING_SUCCESS_URL`、`BILLING_CANCEL_URL`、`BILLING_PORTAL_RETURN_URL`、`AUTH_TOTP_ISSUER` 移入 `config.Config` 并加校验；删除 `main.go` 中的 `envOrDefault`。
  2. 新增 `APP_NAME`（默认 `Starter`）、`APP_BASE_URL`（默认 `http://localhost:8080`）；billing 默认回跳 URL、TOTP issuer、邮件模板 AppName 均由它们推导。
  3. 删除未被使用的 `AUTHORIZATION_MODEL_PATH`、`AUTHORIZATION_POLICY_PATH`（配置、测试、`.env.example`）。
- **验收**：`grep -rn "os.Getenv" apps internal --include=*.go` 只剩 config 包；config 测试覆盖新增字段。

### F1 通用 outbox 分发与重试

- **修改范围**：`internal/delivery/`、`internal/notification/`、`db/migrations/`、`db/queries/outbox.sql`、`apps/server/main.go`、ADR。
- **步骤**：
  1. 在 `delivery` 中新增 `Dispatcher`：`Register(topic string, h Handler)`；Worker 只依赖 Dispatcher。未注册 topic 直接进入死信，不重试。
  2. 把“确认 / 重试”从 `notification.Service` 上移到 Worker：处理成功即 `MarkProcessed`；失败则按指数退避（基础 5s，上限 1h，带抖动）`Retry`；`attempts >= maxAttempts`（默认 10，可配 `WORKER_MAX_ATTEMPTS`）时标记死信。
  3. 新 migration：`outbox_events` 增加 `dead_lettered_at timestamptz` 与部分索引；claim 查询排除已死信的事件。
  4. `notification.Service` 只保留业务处理，改为按 topic 向 Dispatcher 注册。
  5. 新增 ADR：outbox 重试与死信策略。
- **验收**：单测覆盖成功、可重试失败、超过上限转死信、未知 topic 转死信四种路径；Worker 不再 import notification。

### F2 模块注册机制

- **修改范围**：新增 `internal/platform/module/`；`apps/server/main.go`；`internal/platform/httpserver/handler.go`；各领域包新增 `module.go`；TypeSpec 公共装饰器；`tools/generate-server.mjs`；ADR。
- **步骤**：
  1. 定义接口：

     ```go
     type Module interface {
         Name() string
         Enabled(cfg config.Config) bool
         Init(ctx context.Context, deps Deps) error        // Deps 含 pool、queries、transactor、logger、clock、config、authorizer、dispatcher
         Routes(r chi.Router, mw Middlewares)              // mw 提供 Authenticate、RequirePermission
         OutboxHandlers() map[string]delivery.Handler
         Shutdown(ctx context.Context) error
     }
     ```

  2. auth、organization、announcement、billing、notification 各实现一个 `Module`；`main.go` 只负责加载配置、创建 Deps、按顺序注册模块列表、启动与关停。
  3. `httpserver.NewHandler` 改为接收 `[]Module`，只挂载全局中间件、文档、健康检查、webui。
  4. 权限声明移入 TypeSpec：新增装饰器生成 OpenAPI 扩展 `x-permission: "<resource>:<action>"`；`tools/generate-server.mjs` 额外生成每个 api 包的 `permissions_gen.go`（operationID → 权限）；RequirePermission 中间件按 operationID 查表，删除 handler.go 中手写的权限路由。
  5. ADR：模块注册与契约内权限声明。
- **验收**：`main.go` 少于 150 行；新增模块只需新建包并在模块列表加一行；现有全部 handler 测试通过；权限表由生成器产出，`check:generated` 覆盖该文件。

### F3 仓储事务 helper

- **修改范围**：`internal/platform/database/`、所有 `internal/*/postgres.go` 及 `internal/auth/{oauth,totp,webauthn}.go` 中的仓储。
- **步骤**：在 `database` 包新增 `func Queries(ctx context.Context, q *store.Queries) *store.Queries`（有事务则 `WithTx`）；替换所有重复的 `q(ctx)` 实现。
- **验收**：`grep -rn "TxFromContext" internal | grep -v platform/database` 结果为空；行为不变。

### F4 真实数据库集成测试

- **修改范围**：新增 `internal/platform/testdb/`；各领域包新增 `*_integration_test.go`；`package.json`；`.github/workflows/ci.yml`；`docs/standards/testing.md`。
- **步骤**：
  1. 用 testcontainers-go 启动 `postgres:17`，执行全部 goose migration，每个测试用独立 schema 或事务回滚隔离；无 Docker 时 `t.Skip`。
  2. 构建标签 `integration`；新增脚本 `pnpm test:integration`。
  3. 至少覆盖：每个 `db/queries/*.sql` 文件的主路径；outbox claim 并发（两个 worker 不重复领取）；Transactor 回滚。
  4. CI 新增 `integration` job。
- **验收**：CI 中 integration job 通过；testing 规范说明何时写集成测试。

### F5 可观测性

- **修改范围**：新增 `internal/platform/observability/`；config；httpserver 中间件；Worker；`apps/web/src/lib/`；`.env.example`；新增 runbook。
- **步骤**：
  1. OpenTelemetry：`OTEL_EXPORTER_OTLP_ENDPOINT` 配置时启用 trace 导出（HTTP 入口、pgx 查询、outbox 处理、外部 HTTP 客户端）；未配置时 Noop。
  2. Prometheus：`METRICS_ADDRESS`（默认空=关闭）单独端口暴露 `/metrics`，包含 HTTP 延迟/状态码、pgx 连接池、outbox 积压与死信数、Go runtime。
  3. 日志：slog handler 自动附加 `trace_id`、`request_id`；`LOG_LEVEL`、`LOG_FORMAT(json|text)` 配置。
  4. 错误上报：定义 `ErrorReporter` 接口，提供 Sentry 适配器（`SENTRY_DSN`）与 Noop；Recoverer 与 5xx 走它。前端同理，`VITE_SENTRY_DSN` 为空时不加载。
  5. runbook：如何查看指标、trace 与死信。
- **验收**：全部未配置时行为与现状一致；配置后有集成测试或手工证据截图写入 PR。

### F6 通用 HTTP 限流

- **修改范围**：新增 `internal/platform/ratelimit/`；config；httpserver；`db/migrations/`；TypeSpec 装饰器（与 F2 同一生成器）。
- **步骤**：
  1. 接口 `Limiter.Allow(ctx, key string, limit int, window time.Duration) (bool, retryAfter time.Duration, error)`；提供内存实现与 PostgreSQL 实现（固定窗口计数表 + 定期清理），`RATE_LIMIT_BACKEND=memory|postgres`。
  2. 全局默认策略按客户端 IP；已认证请求按用户 ID；API key 请求按 key。TypeSpec 可用 `x-rate-limit` 覆盖单个操作。
  3. 超限返回统一错误体与 `Retry-After`。
  4. OTP 与 webhook 限流迁移到该组件。
- **验收**：多实例共享 postgres 后端的集成测试通过；统一错误响应已加入契约。

### C1 后台作业与定时任务

- **修改范围**：新增 `internal/platform/jobs/`；`db/migrations/`；`db/queries/jobs.sql`；Module 接口扩展；TypeSpec（作业查询接口）；ADR。
- **步骤**：
  1. 新表 `jobs`：`id, org_id, kind, payload, status(queued|running|succeeded|failed|cancelled), progress(0-100), result jsonb, error, attempts, max_attempts, run_at, locked_until, created_by, timestamps`。使用 `FOR UPDATE SKIP LOCKED` 领取，心跳续租 `locked_until`。
  2. API：`jobs.Enqueue(ctx, kind, payload, opts)`（可在 Transactor 事务内调用）、`Register(kind, handler)`、handler 可调用 `ReportProgress`；支持取消。
  3. 定时任务：`Module.Schedules()` 返回 `(name, cron表达式, kind)`；用 advisory lock 保证多实例只触发一次。
  4. 通用接口：`GET /api/organizations/{organizationId}/jobs/{id}` 查询状态与进度；前端提供 `useJob(id)` 轮询 hook。
  5. 并发度配置 `JOBS_CONCURRENCY`；关停时等待进行中的作业或释放租约。
- **验收**：单测与集成测试覆盖领取互斥、失败重试、超时续租失效后被重新领取、cron 单实例触发；附一个示例 kind（`system.noop`），不写任何产品业务。

### C2 对象存储与上传

- **修改范围**：新增 `internal/storage/`；config；`db/migrations/`；TypeSpec `features/files`；前端 `apps/web/src/lib/upload.ts`；ADR。
- **步骤**：
  1. 接口 `ObjectStore`：`PresignPut`、`PresignGet`、`Head`、`Delete`、分片上传（Create/PresignPart/Complete/Abort）。适配器：S3 兼容（`STORAGE_S3_*`，兼容 MinIO、R2、OSS、COS）与本地磁盘（开发用，经 Go 服务代理）。
  2. 表 `files`：`id, org_id, owner_id, key, content_type, size, checksum, status(pending|ready|deleted), metadata jsonb, timestamps`。
  3. 接口：创建上传（返回预签名 URL 或分片计划）→ 客户端直传 → 完成确认（服务端 Head 校验大小/类型）→ 获取下载 URL。权限走 Casbin `files:*`。
  4. 上传完成写 outbox 事件 `files.uploaded`，供后续模块（转码、解析）订阅。
  5. 前端提供带进度与分片续传的上传 hook。
  6. `compose.yaml` 增加可选 MinIO 服务（profile `storage`）。
- **验收**：本地磁盘与 MinIO 两种适配器均有集成测试；大小与 content-type 白名单可配置。

### C3 积分账本与额度

- **修改范围**：新增 `internal/credits/`；`internal/billing/`；`db/migrations/`；TypeSpec；ADR。
- **步骤**：
  1. 复式账本：表 `credit_accounts(org_id, balance)` 与只追加的 `credit_ledger(id, account_id, delta, reason, reference_type, reference_id, idempotency_key unique, created_at)`；余额更新与流水写入同一事务，余额不得为负（行锁）。
  2. API：`Grant`、`Reserve`（预扣）、`Commit`、`Release`、`Balance`；全部幂等。
  3. 用量计量：`usage_events(org_id, meter, quantity, occurred_at, idempotency_key)` 与按周期汇总查询。
  4. 权益与额度中间件：`RequireEntitlement(featureKey)`、`RequireCredits(meter, estimate)`；可在 TypeSpec 用 `x-entitlement` 声明。
  5. Stripe 一次性购买与订阅续费事件发放积分（映射在价格目录配置中）。
  6. 接口：余额、流水分页、用量汇总；前端账单页展示。
- **验收**：并发扣减集成测试无超扣；重复幂等键不重复记账；账本流水之和与余额一致的校验测试。

### C4 支付提供方抽象

- **修改范围**：`internal/billing/`；config；`db/migrations/`；ADR。
- **步骤**：
  1. 把现有 `Gateway` 泛化为 `PaymentProvider`：`CreateCheckout`、`CreatePortal`（可选能力）、`VerifyWebhook`、`ParseEvent → 统一领域事件`。
  2. Stripe 作为第一个实现；表结构增加 `provider` 列，外部 ID 改为 `(provider, external_id)` 唯一。
  3. 预留并实现接口骨架与测试替身，供后续接入微信支付、支付宝、Apple/Google 内购；webhook 路由改为 `/api/billing/webhooks/{provider}`。
- **验收**：Stripe 全部现有测试通过；新增一个内存 provider 走通下单→回调→发放权益/积分的完整测试。

### C5 多客户端认证

- **修改范围**：`internal/auth/`；config；TypeSpec `features/auth`；`db/migrations/`；httpserver（CORS）。
- **步骤**：
  1. 刷新令牌双模式：请求头 `X-Client-Type: web|native`；native 时 refresh token 在响应体返回、刷新接口从请求体读取；web 保持 HttpOnly cookie。
  2. CORS：`CORS_ALLOWED_ORIGINS` 配置，默认空=不下发 CORS 头。
  3. 手机号登录：`SmsSender` 接口（Noop + 一个真实适配器骨架）、手机号 OTP，复用 email OTP 的限流与挑战表设计；`users` 增加 `phone`、`phone_verified_at`。
  4. OAuth provider 注册表化：新增 provider 只需实现 `OAuthClient` 并在配置启用；补充微信开放平台（网站应用）适配器。
  5. 设备/会话列表与单设备登出接口。
- **验收**：native 模式下不设置 cookie；web 模式行为不变；手机号 OTP 有与 email OTP 同等的测试。

### C6 流式响应约定

- **修改范围**：新增 `internal/platform/stream/`；httpserver；`docs/development/streaming.md`；前端 `apps/web/src/lib/sse.ts`。
- **步骤**：
  1. 约定：流式接口不走 ogen，统一挂在 `/api/stream/...`，由模块 `Routes` 注册 raw handler；鉴权、限流、审计复用同一中间件。
  2. 服务端 helper：SSE 写入器（事件、心跳、`Last-Event-ID` 续传、客户端断开取消 context、写超时）。
  3. 事件载荷类型仍在 TypeSpec 中定义为 model 并生成类型，文档说明如何引用。
  4. 前端基于 `fetch` + ReadableStream 的 SSE 客户端（支持 Bearer 头），提供 `useEventStream` hook。
  5. 与 C1 联动：`GET /api/stream/jobs/{id}` 推送作业进度。
- **验收**：helper 单测覆盖心跳、断开取消、续传；开发指南登记到文档目录。

### C7 审计日志

- **修改范围**：新增 `internal/audit/`；`db/migrations/`；TypeSpec；各模块关键写操作。
- **步骤**：
  1. 表 `audit_events(id, org_id, actor_type, actor_id, action, target_type, target_id, ip, user_agent, metadata jsonb, created_at)`，只追加，按时间分区或索引。
  2. `audit.Record(ctx, action, target, metadata)` 在业务事务内调用；actor、IP 从 context 读取。
  3. 接入：登录/登出/MFA 变更、成员与角色变更、邀请、账单操作、API key 变更、文件删除。
  4. 接口：组织内分页查询（权限 `audit:read`）；前端组织设置页新增审计列表。
- **验收**：上述每类操作均有测试断言写入审计记录。

### C8 通知渠道扩展

- **修改范围**：`internal/notification/`；`internal/delivery/`；`db/migrations/`；TypeSpec。
- **步骤**：
  1. 渠道注册表：SMTP 之外新增站内信（`inbox_messages` 表 + 未读数/列表/已读接口）、短信（复用 C5 `SmsSender`）、出站 webhook。
  2. 出站 webhook：`webhook_endpoints(org_id, url, secret, events[], enabled)`，HMAC-SHA256 签名头、重试沿用 F1、投递记录可查询与重放。
  3. 模板按 `topic + channel + locale` 查找，支持 zh/en。
  4. 用户通知偏好表，发送前检查。
- **验收**：每个渠道有单测；出站 webhook 签名可被文档中的示例代码验证。

### C9 平台超级管理员

- **修改范围**：`internal/auth/`、`internal/authorization/`、新增 `internal/admin/`、TypeSpec `features/admin`、前端 `routes/admin/`。
- **步骤**：
  1. 平台级角色 `platform_admin`（Casbin 独立 domain `platform`），通过 CLI 子命令 `server admin grant <email>` 授予。
  2. 接口：用户/组织检索、禁用用户、查看订阅与积分、代登录（签发带 `act` 声明的短期 token，全程写审计）。
  3. 前端独立 `/admin` 路由，仅平台管理员可见。
- **验收**：非平台管理员访问全部 403；代登录产生审计事件且 token 有效期不超过 15 分钟。

### C10 账号注销与数据导出

- **修改范围**：`internal/auth/`、`internal/organization/`、C1 作业、C2 存储、TypeSpec。
- **步骤**：
  1. 数据导出：作业生成 JSON 压缩包写入对象存储，完成后邮件通知下载链接（限时）。
  2. 注销：冷静期（默认 7 天，可配），到期由定时任务匿名化个人字段、撤销会话与 API key；唯一所有者的组织需先转移或删除。
  3. 每个模块通过 Module 接口实现 `ExportUserData`、`EraseUserData`，新模块必须实现。
- **验收**：集成测试覆盖导出内容完整性与注销后无法登录、个人字段已清除。

### D1 完整切片生成器

- **修改范围**：`tools/scaffold-slice.mjs`、`.agents/skills/build-vertical-slice/`、`docs/development/vertical-slice.md`。
- **步骤**：生成器一次产出并注册：TypeSpec（含 `x-permission`）、migration、queries、Go 领域包（domain/ports/service/postgres/handler/module + 单测骨架）、模块列表注册、Casbin 默认策略 migration、前端 feature 目录与路由（列表/详情/表单）、i18n 键。生成后自动运行 `pnpm generate` 并能直接通过 `pnpm check` 与 `pnpm test`。
- **验收**：CI 新增一步：在临时目录复制仓库执行 `pnpm scaffold:slice demo-items` 后运行 `pnpm check && pnpm test`，必须通过。

### D2 模块裁剪命令

- **修改范围**：新增 `tools/remove-module.mjs`、`package.json`、`docs/template-repository.md`。
- **步骤**：`pnpm scaffold:remove <module>` 删除模块的 Go 包、TypeSpec、查询、前端 feature 与路由、模块注册行；migration 不删除，而是生成一个新的 drop migration。支持 `announcement`、`billing`、`credits`、`storage` 等可选模块。
- **验收**：CI 中对每个可选模块执行裁剪后 `pnpm check && pnpm test` 通过。

### D3 E2E 测试

- **修改范围**：新增 `apps/e2e/`（Playwright）、`package.json`、CI。
- **步骤**：compose 启动 PG + 应用（Noop SMTP 改为可查询的内存/文件邮箱，读取验证码）；覆盖：注册→邮箱验证→登录→创建组织→邀请成员→创建公告→（Stripe 模拟 provider）购买→权益生效。
- **验收**：CI `e2e` job 通过，失败时上传 trace。

### D4 部署与迁移加固

- **修改范围**：`db/`、`apps/server/main.go`、`docs/deployment.md`、`compose.yaml`、release workflow。
- **步骤**：
  1. 迁移执行前获取 PostgreSQL advisory lock，多副本 `AUTO_MIGRATE` 安全；文档推荐生产使用独立 `migrate` 一次性任务并将 `AUTO_MIGRATE=false`。
  2. 优雅关停顺序：停止接收请求 → 等待 HTTP → 停止作业/worker → 关闭连接池；关停超时可配。
  3. release 产出多架构镜像（amd64/arm64）与 SBOM，镜像扫描失败阻断发布。
  4. 文档补充 staging 环境与环境变量分层约定。
- **验收**：两个进程同时执行 migrate 的集成测试只有一个实际执行。

### D5 着陆页预渲染

- **修改范围**：`apps/web/`、`internal/platform/webui/`。
- **步骤**：构建期对公开路由（`/`、`/sign-in`、`/sign-up` 及配置列表）预渲染静态 HTML 与 meta（title、description、OG），嵌入二进制；`/app/*` 保持 SPA。生成 `sitemap.xml` 与 `robots.txt`。
- **验收**：`curl /` 返回含正文与 meta 的 HTML；`pnpm test:embed` 通过。

## 5. 推荐派发批次

| 批次 | 可并行任务 |
| --- | --- |
| 1 | B1、B2、B3、F3、D4、D5 |
| 2 | F1、F2、F4 |
| 3 | F5、F6、C1、C2、C3、C5、C6、C7、D1、D2 |
| 4 | C4、C8、C9、C10、D3 |

同一批次内，冲突组相同的任务串行执行。
