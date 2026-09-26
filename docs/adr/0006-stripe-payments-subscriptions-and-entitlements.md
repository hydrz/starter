# ADR-0006：Stripe 支付、订阅与 entitlement 投影

- 状态：已接受
- 日期：2026-09-26
- 决策者：Architecture
- 被替代：无

## 背景

产品可能需要一次性支付、订阅及按组织授予能力。将 Stripe customer、price 或 subscription ID 直接用于业务授权会把外部供应商对象、异步 webhook 和产品权限耦合，难以处理失败、重放与未来供应商替换。

## 决策

使用 Stripe 作为通用支付与订阅提供商。服务端验证 webhook 签名，将已验证的 Stripe 事实写入可审计事件账本，并投影为内部 subscription 与 entitlement 状态。业务与授权层查询内部 entitlement，而非直接查询 Stripe。组织 domain RBAC 仍负责谁可进行操作；entitlement 只表达某项能力是否为该付费实体启用。

Stripe 配置是成组可选的 `STRIPE_SECRET_KEY`、`STRIPE_WEBHOOK_SECRET` 和 `STRIPE_PUBLISHABLE_KEY`。本 ADR 不新增 SDK、路由、表或 UI。

## 备选方案

### 在请求路径直接调用 Stripe 决定权限

延迟和可用性依赖外部网络，难以审计历史决定，且无法安全处理 webhook，未选择。

### 将 Stripe ID 作为应用权限

供应商对象与产品能力不同步，不能表达组织角色，未选择。

## 结果

支付、订阅、授权三者保持可演化边界；代价是需要 webhook 验证、去重、投影重放和 entitlement schema。未来实现遵循 TypeSpec、Goose、sqlc 以及 outbox 的集成门。
