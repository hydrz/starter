# 身份平台契约映射

- **状态**：Active
- **负责人**：API Guild
- **最后复审**：2026-09-26
- **复审周期**：180 天

## 目的与当前边界

本图谱规定身份平台未来切片怎样从 TypeSpec 进入 Go 和 Web，并记录本阶段已准备的公共错误响应。它不是认证 API；本阶段没有新增认证路由、请求模型或前端调用。

## 映射规则

| 层 | 权威输入 | 产物/责任 | 禁止事项 |
| --- | --- | --- | --- |
| HTTP 设计 | `packages/contracts/features/<feature>/` TypeSpec | operation、输入/输出模型和声明的失败响应 | 在 OpenAPI 或生成 Go/TS 文件中补写业务路由 |
| 通用失败 | `packages/contracts/common/responses.tsp` | `ApiError { code, message }` 与可复用响应模型 | 将内部诊断、token、供应商原文写入 `message` |
| OpenAPI | TypeSpec emitter | `spec/generated/` 统一与模块规范 | 手工修改生成物 |
| Go transport | generated OpenAPI + ogen | `internal/api/<feature>api/` 强类型 Handler | 在生成 Handler 中实现业务逻辑 |
| Go domain | transport adapter + domain port | `internal/<feature>/` 用例与领域模型 | 直接向 Web 或其他领域泄露 store/ogen 类型 |
| persistence | Goose schema + `db/queries/` | `internal/store/` sqlc 类型与查询 | 运行时拼接用户 SQL、修改已共享 migration |
| Web | generated OpenAPI + Orval | `apps/web/src/api/generated/` TanStack hooks | 手写重复 endpoint/client 形状 |

## 已准备的错误响应

所有公开失败响应使用同一 `ApiError` 体，并通过 TypeSpec 声明。具体机器 `code` 由对应 feature 稳定定义，`message` 面向调用方；日志保存诊断上下文。

| HTTP | TypeSpec 模型 | 适用的未来场景 |
| --- | --- | --- |
| 400 | `BadRequestResponse` | 输入或状态转换不合法 |
| 401 | `UnauthorizedResponse` | 无凭据、无效或过期 access token |
| 403 | `ForbiddenResponse` | 已认证但 subject/domain/object-action 不获授权 |
| 404 | `NotFoundResponse` | 资源不存在或按公开策略隐藏 |
| 409 | `ConflictResponse` | 唯一性冲突、刷新令牌重用或并发状态冲突 |
| 412 | `PreconditionFailedResponse` | ETag、版本或显式前置条件失败 |
| 429 | `TooManyRequestsResponse` | 登录、恢复或 webhook 接收的速率限制 |
| 503 | `ServiceUnavailableResponse` | 临时依赖不可用或工作系统维护 |

Future auth routes must select relevant models from this common module, be imported from `main.tsp`, and run `pnpm generate`. Adding a model alone intentionally does not alter current OpenAPI operations.

## Future feature-to-storage mapping

| Feature | TypeSpec feature module | Go domain boundary | Goose/sqlc responsibility | Integration gate |
| --- | --- | --- | --- | --- |
| account and sessions | `features/auth/` | `internal/identity` | accounts, credentials, opaque refresh-token family/session rows | Ed25519/EdDSA `kid` rotation contract plus access/refresh semantics approved |
| OAuth/WebAuthn | `features/auth/` | `internal/identity` adapters | external identities, credential metadata/challenges | provider configuration complete and callback threat model reviewed |
| organizations | `features/organizations/` | `internal/organization` | organizations, memberships, invitations | identity account identifiers stable |
| authorization | route-local declaration or protected feature | `internal/authorization` | policy/version data only when persistence is required | subject/domain/object-action contract agreed |
| notifications | `features/notifications/` | `internal/notification` | notification intent, audience/preferences | intent definition separate from channels |
| delivery/outbox | no direct public route required initially | `internal/delivery` | outbox, attempts, idempotency/claim fields | atomic write and retry semantics tested |
| billing/entitlements | `features/billing/` | `internal/billing` | Stripe event ledger, subscription/entitlement projections | verified webhook contract and authorization boundary agreed |

Future identity adapters issue only `EdDSA` JWTs. They must put `config.AuthConfig.ActiveKID` in the protected `kid` header and sign with `SigningPrivateKey`; validation must reject any algorithm other than EdDSA, a missing/unknown `kid`, or a token that fails under `VerificationPublicKeys[kid]`. Rotation retains multiple public keys in that keyset while exactly one matching private key/active `kid` signs new access tokens.

## Compatibility gates

1. Public operation IDs and published fields remain stable; breaking changes need a version or compatible migration window.
2. TypeSpec is edited before all generated outputs. `pnpm generate` is the only supported path to OpenAPI, ogen, and Orval updates.
3. Each schema change is a new forward-only Goose migration plus sqlc query changes. No endpoint is considered integrated until generated drift and database checks pass.
4. Auth, Stripe, SMTP, Casbin, and worker configuration only expresses setup capability; it does not authorize or instantiate a feature by itself.

## Related material

- [Contract development guide](../development/contracts.md)
- [Contract and data standards](../standards/contracts-and-data.md)
- [Identity platform architecture](../architecture/identity-platform.md)
- [Implementation ledger](identity-platform-ledger.md)
