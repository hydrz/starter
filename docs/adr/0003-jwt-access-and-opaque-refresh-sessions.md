# ADR-0003：JWT access 与不透明 refresh 会话

- 状态：已接受
- 日期：2026-09-26
- 决策者：Architecture
- 被替代：无

## 背景

身份平台需要让浏览器和 API 客户端持有短期访问凭据，同时允许会话续期、撤销、设备管理和令牌重用检测。完全无状态的长寿命 JWT 无法可靠支持这些服务器端安全控制。

## 决策

使用短期 **Ed25519 / EdDSA** 签名 JWT 作为 access token，使用随机生成、仅以安全哈希保存的不透明 refresh token。每个 access JWT 必须在受保护 header 中携带 `kid`，并由配置的 active key 签名；验证器只能依据 `kid` 从配置的 public-key keyset 选取验证密钥，且必须拒绝缺失或未知 `kid`。refresh token 属于可旋转的 token family/session；刷新时轮换，并对重用执行家族撤销或等价的安全处置。JWT 只携带访问所需的最小稳定 claims，不承载可频繁变化的授权或 entitlement 真相。

密钥轮换先将新 public key 加入 verification keyset，部署所有验证器，再以该 key 的 `kid` 和 private key 作为 active signing key；旧 public key 保留到其可能签发的全部 access token 都过期后才删除。配置接口为 `AUTH_JWT_ACTIVE_KID`、`AUTH_JWT_SIGNING_PRIVATE_KEY`、`AUTH_JWT_VERIFICATION_KEYSET`、issuer 与两个 TTL；private key 和每个 keyset value 都是无填充 base64url 编码的 Ed25519 原始 key bytes。加载器只在所有 auth 变量均存在时启用 auth，且错误仅报告变量名。

认证 API 在 TypeSpec 定义后才生成 OpenAPI、ogen 与 Orval 产物。账户、凭据和会话使用前向 Goose migration 和 sqlc 查询实现。

## 备选方案

### HMAC/共享密钥 JWT

单一共享 secret 无法提供非对称验证边界，并使轮换期的 key identity 与 verifier keyset 语义不明确，未选择。

### 双 JWT

实现表面简单，但长寿命 refresh JWT 难以逐个撤销、轮换和安全地检测重放，未选择。

### 服务端 session ID 作为全部访问凭据

支持撤销，但会让每个受保护请求都依赖 session lookup，且失去 JWT 适合的短期无状态访问边界，未选择为默认方案。

## 结果

获得短期 access token 与有状态续期控制的组合，并支持明确 `kid` 的零停机 key rotation；代价是增加 refresh-token 存储、哈希、轮换、异常重用处理和 keyset 运维。本 ADR 只定义配置准备接口，不交付认证端点或 schema。

Limen 与 Authula 的公开架构是会话与身份边界的设计影响来源；本仓库不复制其代码。
