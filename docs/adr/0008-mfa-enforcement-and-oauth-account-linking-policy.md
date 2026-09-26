# ADR-0008：MFA 强制策略与 OAuth 账户关联策略

- 状态：已接受
- 日期：2026-09-26

## 背景

工作流 F 在既有的 `internal/auth.Service`（[ADR-0003](0003-jwt-access-and-opaque-refresh-sessions.md)）之上新增了 Email OTP、TOTP/MFA、Google/GitHub OAuth 与 WebAuthn/Passkey。三个问题在 ADR-0003 中未覆盖，需要显式决策：

1. 用户开启 TOTP 后，是否应让已经签发的 access/refresh token 立即失效？
2. 已认证账号如何将一个 OAuth 身份"关联"到自己名下，才能避免被跨账号劫持？
3. OAuth 账户解析（sign-in / sign-up）能否以邮箱作为匹配或合并依据？

## 决策

### MFA 强制策略：仅向前生效，不追溯

- `TOTPFactor` 的 `status` 只有在从 `pending` 变为 `verified`（`ConfirmTOTPEnrollment` 成功）之后，才会被 `Service.checkMFARequired` 计入；
- `checkMFARequired` 只在**新的**主认证请求（`PasswordSignIn`、`CompleteOAuthSignIn`、`FinishWebAuthnAuthentication`）里被调用；它从不撤销、过期或标记已经签发的 refresh session/access token；
- 因此启用 TOTP 之后，已经登录的浏览器标签页或已经缓存的 refresh session 会继续有效，直到其自然过期或用户主动 `SignOut`/`RevokeSession`；只有**下一次**从空会话重新登录才会被要求提供第二因子；
- 这与密码重置（`ConfirmPasswordReset` 会主动撤销该用户的全部 refresh family）刻意不同：密码泄露是需要立即遏制的凭据事件，而"用户决定加固账号"不是需要立即遏制的事件。

### OAuth 账户关联策略：仅显式、经过近期重新认证的会话可以关联

- 账户解析（sign-in 与 link）永远以 `(provider, provider_subject)` 作为唯一身份键（见 `db/migrations/00005_create_mfa_oauth_webauthn.sql` 的 `oauth_accounts_provider_subject_key` 唯一约束），从不使用 email 匹配或合并；
- `CompleteOAuthSignIn` 在找不到既有 `(provider, subject)` 记录时，总是创建一个新账号，其 `users.email` 是基于 `provider + uuid` 生成的合成值，绝不直接复用 provider 报告的邮箱（两个不同 `provider_subject` 完全可能报告相同邮箱，例如共享邮箱或历史脏数据）；provider 报告的邮箱仅作为 `oauth_accounts.email` 展示字段保存；
- 将 OAuth 提供方关联到**已存在**账号是一个独立操作（`BeginOAuthLink`/`LinkOAuthAccount`），要求：（a）调用方已认证（`userID` 来自已验证的 access token）；（b）该 access token 是"近期"签发的——`BeginOAuthLink` 拒绝任何签发时间早于 `oauthLinkReauthWindow`（5 分钟）之前的 token，返回 `ErrReauthenticationRequired`；（c）授权 state 在服务端记录了发起关联时的 `linking_user_id`，回调时必须与当前认证用户完全一致，否则拒绝（`ErrOAuthStateInvalid`），从而防止"未认证或认证到另一账号的回调"悄悄挂到别的账号上；
- 5 分钟窗口是"近期重新认证"的近似实现：它复用了 access token 短生命周期这一既有事实，而不是引入一个新的、独立的 step-up 认证流程。这是一个已知的简化——参见实施台账 F 部分"未能完整测试的不变量"一节。

## 结果

- 好处：MFA 采用方式对现有会话零副作用，运维无需协调"强制下线"；OAuth 关联在结构上不可能把身份挂错账号，即使 state/PKCE 泄露也仅暴露给发起该 state 的同一账号；账户解析永远不会因为供应商方邮箱数据的重复或过期而把两个人的账号合并。
- 代价：一个已经登录很久的会话，在攻击者获得密码后仍可完成一次不带 MFA 的登录（直到该会话过期）——这是"向前生效"策略的直接后果，运维如需更强保证应结合 `RevokeAllForUser`/更短的 refresh TTL；"近期重新认证"用 access token 签发时间近似，而非真正的 step-up 挑战。
