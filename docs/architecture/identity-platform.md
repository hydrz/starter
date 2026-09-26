# 身份平台架构

- **状态**：Active
- **负责人**：Architecture
- **最后复审**：2026-09-26
- **复审周期**：180 天

## 目标与边界

身份平台在同一个 Go 1.27 二进制中逐步提供会话、组织内授权、可靠交付和计费能力。它保留现有 Chi、pgxpool、Goose、sqlc、TypeSpec→OpenAPI→ogen 与 React/TanStack 分层；不引入 Node/Bun 服务端运行时，也不以框架快捷方式绕过既有契约、迁移或生成流程。

本阶段只交付共享配置、错误响应契约与治理文档。认证路由、认证表、组织实现、Casbin 策略、SMTP worker、Stripe 接入和产品 UI 均不属于本阶段。

## 组件与依赖图

```text
TypeSpec (packages/contracts) ──> OpenAPI ──> ogen handler / Orval client
          │                                  │
          └── reusable ApiError responses     └── domain adapters (future)

Chi main ──> platform/config ──> current server settings
   │                 └── optional future integration settings
   ├── pgxpool ──> sqlc store ──> PostgreSQL / Goose migrations
   └── future in-process outbox worker ──> delivery channels (SMTP first)

identity/session ──> authorization decision (subject, domain, object/action)
                                 │
organization membership ─────────┘

Stripe webhooks ──> entitlement projection ──> authorization/product checks
```

`packages/contracts/` remains the only source of truth for business REST endpoints. `spec/generated/`, `internal/api/`, and `apps/web/src/api/generated/` are generated artifacts, never implementation surfaces. Database changes remain forward-only Goose migrations with sqlc queries; future domain packages translate generated transport and store types rather than exposing them across layers.

## Identity and authorization model

A future login issues a short-lived **Ed25519 / EdDSA** JWT access token and a long-lived opaque refresh token. Each JWT's protected header carries a required `kid`: issuers sign with the configured active private key, and verifiers resolve only that `kid` in the configured public-key keyset, rejecting missing or unknown keys. Rotation adds a new public key to every verifier first, then promotes its private key and `kid` to active signing; retired public keys remain until their maximum access-token lifetime elapses. The refresh token is stored and rotated server-side; clients never receive a JWT refresh token. Access-token claims identify the subject, while revocation, session rotation, and reuse detection are stateful server concerns.

Authorization is a three-part decision: **subject**, **domain**, and **object/action**. A domain is normally an organization; membership establishes the eligible relationship; Casbin evaluates the requested object/action within that domain. Product entitlement is an input or prerequisite to a business capability, not a replacement for organization authorization.

WebAuthn and Google/GitHub OAuth are optional authentication mechanisms. They feed the same account/session model rather than creating incompatible identity silos.

## Delivery is not notification

A notification is an application-level event addressed to a user or audience. Delivery is an attempt to transmit a rendered notification through a channel. SMTP is the first delivery channel, but later SMS, Telegram, push, or in-app channels may consume the same notification intent. Delivery state, retry policy, provider response, and idempotency belong to the delivery/outbox boundary; business code must not call SMTP inline in a transaction.

## Reliability and payment boundaries

A PostgreSQL outbox row is created in the same transaction as the business state that requires external work. A bounded in-process worker claims rows, performs delivery or webhook side effects, persists outcomes, and retries safely. This preserves one deployable binary while avoiding distributed transactions.

Stripe is a generic payment/subscription/entitlement integration: it maps verified webhook facts into internal subscription and entitlement projections. Stripe objects, webhooks, and price identifiers must not become the authorization model or leak into unrelated product APIs.

## Configuration boundary

`internal/platform/config` is the process configuration seam. It always supplies database URL, HTTP address, and auto-migration behavior compatible with the current server. It also parses grouped future settings for auth, OAuth, WebAuthn, SMTP, Stripe, worker, and authorization without initializing those services. An optional group is disabled when every member is absent; a partial group fails startup and names only missing or invalid environment variable names. The auth group is `AUTH_JWT_ACTIVE_KID`, `AUTH_JWT_SIGNING_PRIVATE_KEY`, `AUTH_JWT_VERIFICATION_KEYSET`, `AUTH_JWT_ISSUER`, `AUTH_JWT_ACCESS_TTL`, and `AUTH_REFRESH_TOKEN_TTL`; all six are required when auth is enabled. Private and public key values are raw Ed25519 key bytes encoded with unpadded base64url; the keyset is a JSON `kid` to encoded-public-key object that includes the active key and any still-valid retired keys. Validation errors never include a key or secret value.

`compose.yaml` requires no change in this phase: it only starts the current app and local PostgreSQL, while optional integrations remain disabled by default and demand no network dependencies.

## Integration gates and ownership

| Workstream | Owner | May start after | Must provide before integration |
| --- | --- | --- | --- |
| A — foundation/governance/config | Platform Architecture | approved architecture | config tests, TypeSpec shared errors, ADRs, ledger evidence |
| B — identity/session | Identity | A ready | TypeSpec auth routes, forward-only schema, rotation/revocation tests |
| C — organizations/Casbin | Authorization | B account/session interfaces agreed | domain membership model, policy model, three-part decision tests |
| D — outbox/delivery | Messaging | A config and schema conventions agreed | outbox migration/sqlc, claim/retry/idempotency tests; SMTP adapter only |
| E — Stripe/entitlements | Billing | C authorization boundary agreed | webhook verification, projections, entitlement integration tests |
| F — product UI | Product Web | B/C public contracts generated | Orval-only clients, user-facing states, no hand-authored endpoint shapes |

No workstream may hand-edit generated artifacts, add an endpoint outside TypeSpec, bypass sqlc/Goose, use environment validation errors that expose secrets, or collapse delivery into notification. Cross-workstream changes pass their owner gate and are recorded in the implementation ledger.

## Related decisions

- [JWT access and opaque refresh sessions](../adr/0003-jwt-access-and-opaque-refresh-sessions.md)
- [Casbin domain RBAC](../adr/0004-casbin-domain-rbac-and-three-part-authorization.md)
- [Delivery and notification separation](../adr/0005-separate-notification-intent-from-delivery.md)
- [Stripe payments, subscriptions, and entitlements](../adr/0006-stripe-payments-subscriptions-and-entitlements.md)
- [PostgreSQL outbox and in-process worker](../adr/0007-postgres-outbox-and-in-process-worker.md)
- [Identity platform implementation ledger](../implementation/identity-platform-ledger.md)
- [Identity platform contract map](../implementation/identity-platform-contract-map.md)
