# 身份平台实施台账

- **状态**：Active
- **负责人**：Platform Engineering
- **最后复审**：2026-09-26
- **复审周期**：90 天

## 目的

本台账是身份平台跨工作流的集成清单与证据入口。架构选择以[身份平台架构](../architecture/identity-platform.md)和 ADR 为准；HTTP 细节以 TypeSpec 为准。本台账不替代它们。

## 状态规则

`Planned` 表示尚未开始；`In progress` 表示该工作流正在修改其拥有的边界；`Ready for integration` 表示该工作流的产物和验证证据已就绪，仍须由后续工作流通过集成门；`Integrated` 表示已被依赖方实际采用。只有显式记录证据才能提升状态。

## 工作流状态

| 工作流 | 所有者 | 初始状态 | 当前状态 | 集成门 | 产物 |
| --- | --- | --- | --- | --- | --- |
| A — Foundation, governance, configuration, shared infrastructure | Platform Architecture | In progress | Integrated | 主 Agent 已复核并合入；后续工作流只通过公开 config/TypeSpec 边界接入 | 配置加载器、通用错误响应、ADR、架构/契约图谱 |
| B — Identity and sessions | Identity | Planned | Planned | A 已就绪；认证契约和 schema 评审 | 账户、会话、JWT access/opaque refresh 实现 |
| C — Organizations and authorization | Authorization | Planned | Planned | B account identity 稳定；三段授权输入评审 | 组织、成员关系、Casbin model/policy/adapters |
| D — Delivery and outbox | Messaging | Planned | Planned | A 已就绪；通知 intent 和事务边界评审 | outbox、worker、SMTP delivery adapter |
| E — Stripe and entitlements | Billing | Planned | Planned | C 授权边界确认；webhook 安全评审 | Stripe projection、订阅、entitlement |
| F — Product UI | Product Web | Planned | Planned | B/C 生成的公开契约与授权语义稳定 | Orval client usage、账户/组织界面 |

## A 的接口决策

- `config.Load(os.LookupEnv)` returns `config.Config` or a `*config.ValidationError`; validation messages include environment variable names only, never values.
- Current startup reads `DatabaseURL`、`Address` 和 `AutoMigrate` from this config while retaining defaults: database URL points at the local starter database, address is `:8080`, and auto migration is enabled except for exact `AUTO_MIGRATE=false`.
- Auth, OAuth (Google/GitHub), WebAuthn, SMTP, Stripe, authorization, and worker configurations are parsed but do not initialize any future service. Every optional group is disabled when entirely absent; partial groups fail with missing names. The auth group requires `AUTH_JWT_ACTIVE_KID`, `AUTH_JWT_SIGNING_PRIVATE_KEY`, `AUTH_JWT_VERIFICATION_KEYSET`, `AUTH_JWT_ISSUER`, `AUTH_JWT_ACCESS_TTL`, and `AUTH_REFRESH_TOKEN_TTL` together.
- Future issuers use `AuthConfig.SigningPrivateKey` for EdDSA and write `AuthConfig.ActiveKID` into the protected JWT `kid`; future verifiers reject absent/unknown `kid` and use only `AuthConfig.VerificationPublicKeys[kid]`. The active public key must match the signing private key; the keyset may retain prior public keys for rotation. Key material is unpadded base64url in environment variables, and config errors name variables only.
- `WorkerConfig` defaults to a one-second poll interval and batch size 50; it only enables when `WORKER_ENABLED=true`.
- Shared TypeSpec errors now cover 401, 403, 409, 412, and 429 in addition to existing 400, 404, and 503. No feature route consumes them yet.
- `compose.yaml` is intentionally unchanged because current compose only needs PostgreSQL/current app startup; optional integrations have no runtime dependency until their owners implement them.

## Verification evidence

| Date | Scope | Command | Outcome |
| --- | --- | --- | --- |
| 2026-09-26 | A | `gofmt -w apps/server/main.go internal/platform/config/*.go` | Passed; Go source formatted. |
| 2026-09-26 | A | `go test ./apps/server ./internal/platform/config` | Passed. |
| 2026-09-26 | A | `pnpm install --frozen-lockfile` | Passed; installed the lockfile-pinned workspace dependencies required by TypeSpec generation. |
| 2026-09-26 | A | `pnpm generate` | Passed after dependency installation; regenerated only official contract/server/web/sqlc outputs. |
| 2026-09-26 | A | `go test ./...` | Passed. |
| 2026-09-26 | A | `pnpm check:format` | Passed. |
| 2026-09-26 | A | `pnpm check:docs` | Passed for 40 Markdown files. |
| 2026-09-26 | A | `pnpm check:generated` | Expected non-zero in an uncommitted worktree because it verifies generated artifacts have no Git diff; the displayed diff exactly matches the TypeSpec common-response addition. |
| 2026-09-26 | A | `pnpm check` | Stopped at `check:generated` for the same expected uncommitted generated-artifact diff; remaining static sub-gates were run independently. |
| 2026-09-26 | A | `pnpm check:go && pnpm check:sql && pnpm check:web && pnpm check:skills && pnpm check:actions` | Passed. |
| 2026-09-26 | A | `git diff --check` | Passed. |
| 2026-09-26 | A | `go test -race ./...` | Blocked by local toolchain: `-race requires cgo; enable cgo by setting CGO_ENABLED=1`. Normal `go test ./...` passed. |
| 2026-09-26 | A | `pnpm test:race` | Not run separately because it is the same Go race command and would fail for the same local CGO prerequisite. |
| 2026-09-26 | A | `go test ./... && pnpm check:format && pnpm check:go && pnpm check:sql && pnpm check:web && pnpm check:skills && pnpm check:actions` | Passed. |
| 2026-09-26 | A | `pnpm check:docs && git diff --check` | Passed after final documentation/config updates. |
| 2026-09-26 | A | `gofmt -w internal/platform/config/config.go internal/platform/config/config_test.go && go test ./internal/platform/config` | Passed after replacing the provisional shared-secret config with the Ed25519 active-`kid` signing and verification-keyset interface. |
| 2026-09-26 | A | `pnpm check:docs && git diff --check` | Passed after Ed25519 documentation updates. |
| 2026-09-26 | A integration | `go test ./apps/server ./internal/platform/config` | Passed in the main worktree after independent review. |
| 2026-09-26 | A integration | `pnpm generate && pnpm check:go && pnpm check:sql && pnpm check:web && pnpm check:skills` | Passed in the main worktree; generated diffs are derived only from the common TypeSpec response additions. |
| 2026-09-26 | A integration | `pnpm check:docs && pnpm check:format && pnpm check:actions && git diff --check` | Passed in the main worktree; corrected the contract-map Markdown table structure during review. |

## Integration checklist for later owners

- [ ] Run `pnpm generate` after adding or changing TypeSpec endpoint declarations; inspect and commit generated diffs instead of editing them.
- [ ] Add new Goose migrations only; pair data access changes with sqlc generation and tests.
- [ ] Use config values only in composition roots/adapters; do not log secret-bearing config fields.
- [ ] Preserve notification intent versus delivery attempt as separate domain concepts.
- [ ] Verify Stripe webhooks before changing entitlement state, and do not use Stripe as an authorization engine.
- [ ] Record exact commands and outcomes in this ledger before requesting integration.
