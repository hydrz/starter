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
| B — Identity and sessions | Identity | Planned | Ready for integration | A 已就绪；认证契约和 schema 评审 | 账户、会话、JWT access/opaque refresh 实现 |
| C — Organizations and authorization | Authorization | Planned | Planned | B account identity 稳定；三段授权输入评审 | 组织、成员关系、Casbin model/policy/adapters |
| D — Delivery and outbox | Messaging | Planned | Ready for integration | A/B 已就绪；通知 intent 和 outbox claim 评审 | outbox worker、channel 抽象、SMTP 适配器、邮件模板与通知路由 |
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

## B 的接口决策

- New domain package `internal/auth` (not `internal/identity`, matching the overall plan) exposes `Service` with narrow `UserRepository`, `RefreshTokenRepository`, `OneTimeTokenRepository`, `APIKeyRepository`, `OutboxWriter`, and `Transactor` ports; it never leaks `internal/store` or `internal/api/authapi` types outside `internal/auth/postgres.go` and `internal/auth/handler.go`.
- `auth.Issuer`/`auth.Verifier` issue and verify only EdDSA JWTs using `config.AuthConfig.ActiveKID`/`SigningPrivateKey`/`VerificationPublicKeys`; claims are limited to `sub`, `sid`, `jti`, `iss`, `aud`, `iat`, `nbf`, `exp`, `typ=access`. No role/org/entitlement claim is added, per the frozen contract.
- Passwords use Argon2id PHC hashes (`auth.HashPassword`/`VerifyPassword`) with constant-time comparison and a fixed dummy-hash comparison (`VerifyPasswordOrDummy`) for unknown emails, so sign-in timing does not reveal account existence.
- Refresh, verification, and API-key secrets are never stored raw: `auth.SecretDigester` stores only a versioned HMAC-SHA256 digest, keyed by a dedicated `AuthConfig.SecretDigestPepper` (environment variable `AUTH_SECRET_PEPPER`, part of the existing all-or-none auth group) rather than being derived from the JWT signing key, so rotating the JWT signing key never silently invalidates stored refresh/reset/API-key digests. Digests are recorded in `db/migrations/00002_create_identity_and_delivery.sql` (`refresh_sessions.token_digest`, `one_time_tokens.token_digest`, `api_keys.secret_digest`). Refresh rotation and one-time-token consumption use single conditional `UPDATE ... WHERE ... RETURNING` statements (`db/queries/auth.sql`: `ConsumeRefreshSession`, `ConsumeOneTimeToken`) so there is no check-then-update race; a refresh token that cannot be consumed (already replaced/revoked/expired) but is still a known digest is treated as reuse and revokes its whole family via `RevokeRefreshFamily`.
- Sign-up writes the new `users` row and its first `notification_intents`/`outbox_events` rows (topic `auth.verification_requested`) inside one `database.PgxTransactor` transaction; password reset confirmation revokes every refresh family for the account inside the same transaction as the password update. Auth never calls SMTP inline; it only writes outbox rows for a future delivery worker (workstream D) to claim.
- API keys (`api_keys` table) carry a nullable `organization_id` column with no foreign key yet, so workstream C can add organization scoping later without a forward-only migration that blocks on a not-yet-existing organizations table; the raw key is only returned once, at `CreateAPIKey` time.
- The refresh token itself is delivered only via an HttpOnly/Secure/SameSite=Lax cookie scoped to `Path=/api/auth` (`auth.RefreshCookieName`/`RefreshCookiePath`); no browser storage guidance is implied beyond keeping the access token in memory, which remains a caller (F) concern.
- Public HTTP surface lives at `packages/contracts/features/auth/{models,routes,auth}.tsp`, imported from `main.tsp`, compiled only through `pnpm generate`; all operations are mounted at `/api/auth/*` (`internal/platform/httpserver/handler.go`), reusing A's common `ApiError`/`UnauthorizedResponse`/`ConflictResponse`/`TooManyRequestsResponse` models.
- `httpserver.NewHandler` gained a third, optional `*auth.Service` parameter; when nil (auth disabled), no `/api/auth/*` route is registered and existing health/docs/announcements behavior is unchanged. `auth.AuthenticationMiddleware` only resolves an optional bearer principal into context for the auth handler's own operations (e.g. `GetCurrentIdentity`, session/API-key management); it does not protect any other route, per the integration gate reserving global authorization middleware for workstream C.
- `apps/server/main.go` keeps A's config-driven `cfg.Address`/`cfg.DatabaseURL`/`cfg.AutoMigrate` startup logic unchanged and adds only identity-service wiring: when `cfg.Auth.Enabled`, it builds `auth.Service` via `buildIdentityService` and passes it into `apphttp.NewHandler`. `config.AuthConfig` gained one additional required field, `SecretDigestPepper` (env `AUTH_SECRET_PEPPER`), joining the existing all-or-none auth group; it is independent, high-entropy secret material unrelated to `SigningPrivateKey`, used only to key `auth.SecretDigester`.

## D 的接口决策

- New package `internal/delivery` provides the outbox worker (`delivery.Worker`), delivery channel abstraction (`delivery.Channel`, `delivery.Message`), standard SMTP adapter (`delivery.SMTPChannel`, `delivery.NoopChannel`), and dual-mode template rendering engine (`delivery.TemplateRenderer`).
- Outbox processor (`delivery.Worker`) periodically claims batches of due events (`available_at <= now() AND processed_at IS NULL AND (claimed_at IS NULL OR claimed_at < now() - timeout)`) using PostgreSQL `SELECT ... FOR UPDATE SKIP LOCKED` (`ClaimOutboxEvents`), parameterized by `config.WorkerConfig.BatchSize` (default 50), `config.WorkerConfig.PollInterval` (default 1s), and `ClaimTimeout` (default 5m). Shutdown is gracefully coordinated via `context.Context`.
- SMTP channel (`delivery.SMTPChannel`) supports standard SMTP, Resend, and Cloudflare Email Routing with proper TLS/STARTTLS handling (direct TLS on port 465, STARTTLS upgrade on port 587, PLAIN authentication, and RFC 2822/5322 MIME `multipart/alternative` formatting with UTF-8 text and HTML). `delivery.NoopChannel` allows development and testing without SMTP credentials.
- Dual-mode email templates for `auth.verification_requested` and `auth.password_reset_requested` use standard library `html/template` and `text/template` embedded via `embed.FS` (`internal/delivery/templates/`).
- Domain package `internal/notification` provides `Service` implementing `delivery.Handler`. When `auth.verification_requested` or `auth.password_reset_requested` outbox events are claimed, it translates them into `notification_intents`, renders dual-mode templates, persists `delivery_messages` with channel `smtp` and unique idempotency key (`outbox:<event-id>:smtp`), and records `delivery_attempts`.
- On delivery success, records attempt status `sent` with provider response and marks outbox event processed (`MarkOutboxEventProcessed`). On delivery failure, records attempt status `failed` with error message, computes exponential backoff (`CalculateBackoff`), and schedules retry (`RetryOutboxEvent`).
- `apps/server/main.go` wires `delivery.Worker` in the background when `cfg.Worker.Enabled || cfg.SMTP.Enabled`, stopping cleanly on SIGTERM/SIGINT within the shutdown timeout.

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
| 2026-09-26 | B | `pnpm install --frozen-lockfile` | Passed; needed once in this fresh worktree before any `pnpm` script could run. |
| 2026-09-26 | B | `pnpm generate` | Passed; compiled `features/auth` TypeSpec, generated `internal/api/authapi`, Orval `apps/web/src/api/generated/auth`, and sqlc `internal/store/{auth,outbox}.sql.go` from the new migration/queries. |
| 2026-09-26 | B | `go build ./...` | Passed. |
| 2026-09-26 | B | `go vet ./...` | Passed. |
| 2026-09-26 | B | `go test ./...` | Passed; `internal/auth` adds 32 unit tests covering Argon2id hashing/verification (including the anti-enumeration dummy-hash path), versioned HMAC secret digesting, EdDSA JWT issuance/verification (wrong audience, unknown `kid`, expired, tampered signature, malformed token), refresh-session rotation, refresh-token reuse triggering family revocation, sign-up/sign-in/API-key/session-listing service behavior, and HTTP-handler adaptation — all against in-memory fakes with an injected fake clock, so no PostgreSQL/SMTP/Stripe/OAuth/WebAuthn dependency is required to run them. |
| 2026-09-26 | B | `go tool sqlc vet -f db/sqlc.yaml` | Passed. |
| 2026-09-26 | B | `gofmt -l apps db internal` (via `node tools/check-gofmt.mjs apps db internal`) | Passed after `gofmt -w` on new/changed Go files. |
| 2026-09-26 | B | `pnpm --filter @starter/contracts format:check` | Passed after `tsp format "**/*.tsp"` reformatted the two new `features/auth` files. |
| 2026-09-26 | B | `pnpm --filter @starter/web format:check` | Passed. |
| 2026-09-26 | B | `pnpm --filter @starter/web check` (eslint + tsc) | Passed. |
| 2026-09-26 | B | `node tools/check-docs.mjs` | Passed for 32 Markdown files. |
| 2026-09-26 | B | `node tools/check-skills.mjs` | Passed for 5 skills. |
| 2026-09-26 | B | `go tool actionlint` | Passed (no findings). |
| 2026-09-26 | B | `git diff --check` (unstaged and staged) | Passed. |
| 2026-09-26 | B | `pnpm check:generated` | Failed as expected in this uncommitted worktree: the reported diff is exactly the new `features/auth` contract's generated OpenAPI/Go/TS/sqlc output. No unexpected or hand-edited generated content is present; this is not evidence that aggregate `pnpm check` passes, since `check:generated` is its first sub-gate. |
| 2026-09-26 | B (review fix) | Rebased onto A's integrated `main` (`config.Load`-driven `apps/server/main.go`, `.env.example`, docs); removed a pre-A env-reading `main.go` this worktree had reintroduced because it branched before A merged. Replaced the JWT-signing-key-derived HMAC pepper (`derivePepper`) with a new required `AuthConfig.SecretDigestPepper`/`AUTH_SECRET_PEPPER` config field so secret-digest rotation is independent of JWT key rotation. | See re-run command rows below. |
| 2026-09-26 | B (review fix) | `go build ./... && go vet ./... && go test ./...` | Passed after the rebase and pepper-config fix. |
| 2026-09-26 | B (review fix) | `pnpm check:format` | Passed. |
| 2026-09-26 | B (review fix) | `git diff --check` | Passed. |
| 2026-09-26 | D | `pnpm install --frozen-lockfile` | Passed; installed dependencies for TypeSpec / contract tools in this worktree. |
| 2026-09-26 | D | `pnpm generate` | Passed; compiled contracts, generated server/web/sqlc queries including new outbox helper queries. |
| 2026-09-26 | D | `go build ./...` | Passed. |
| 2026-09-26 | D | `go vet ./...` | Passed. |
| 2026-09-26 | D | `go test -v ./internal/delivery` | Passed; 15 unit tests covering dual-mode MIME message construction, text-only MIME, non-ASCII header encoding, noop channel, mock SMTP server with plain auth and recipient rejection, HTML/text template rendering, HTML escaping, and outbox worker lifecycle, batching, error continuation, and context cancellation. |
| 2026-09-26 | D | `go test -v ./internal/notification` | Passed; 8 unit tests covering verification intent and message generation, password reset message generation, failure attempt tracking and exponential backoff retry scheduling, idempotency with pre-existing delivery messages, backoff calculation, and unknown topic/channel error handling. |
| 2026-09-26 | D | `go test -count=1 ./...` | Passed across all packages in repository. |
| 2026-09-26 | D | `pnpm check:format` | Passed (gofmt, TypeSpec format, Prettier). |
| 2026-09-26 | D | `pnpm check:docs` | Passed for 40 Markdown files. |
| 2026-09-26 | D | `pnpm check:skills` | Passed for 5 skills. |
| 2026-09-26 | D | `pnpm check:actions` | Passed (actionlint). |
| 2026-09-26 | D | `pnpm check:go` | Passed (go vet). |
| 2026-09-26 | D | `pnpm check:sql` | Passed (sqlc vet). |
| 2026-09-26 | D | `pnpm check:web` | Passed (eslint and tsc). |
| 2026-09-26 | D | `git diff --check` | Passed. |

## Integration checklist for later owners

- [ ] Run `pnpm generate` after adding or changing TypeSpec endpoint declarations; inspect and commit generated diffs instead of editing them.
- [ ] Add new Goose migrations only; pair data access changes with sqlc generation and tests.
- [ ] Use config values only in composition roots/adapters; do not log secret-bearing config fields.
- [ ] Preserve notification intent versus delivery attempt as separate domain concepts.
- [ ] Verify Stripe webhooks before changing entitlement state, and do not use Stripe as an authorization engine.
- [ ] Record exact commands and outcomes in this ledger before requesting integration.
