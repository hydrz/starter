# 架构决策记录

ADR 记录影响多个模块、难以撤销或需要保存取舍背景的决策。普通实现细节不需要 ADR。

## 索引

| 编号 | 决策 | 状态 | 决策日期 |
| --- | --- | --- | --- |
| [0001](0001-commit-generated-contract-artifacts.md) | 提交契约生成物 | 已接受 | 2026-09-24 |
| [0002](0002-forward-only-production-migrations.md) | 生产数据库只向前迁移 | 已接受 | 2026-09-24 |
| [0003](0003-jwt-access-and-opaque-refresh-sessions.md) | JWT access 与不透明 refresh 会话 | 已接受 | 2026-09-26 |
| [0004](0004-casbin-domain-rbac-and-three-part-authorization.md) | Casbin 域 RBAC 与三段授权 | 已接受 | 2026-09-26 |
| [0005](0005-separate-notification-intent-from-delivery.md) | 分离通知意图与交付 | 已接受 | 2026-09-26 |
| [0006](0006-stripe-payments-subscriptions-and-entitlements.md) | Stripe 支付、订阅与 entitlement 投影 | 已接受 | 2026-09-26 |
| [0007](0007-postgres-outbox-and-in-process-worker.md) | PostgreSQL outbox 与进程内 worker | 已接受 | 2026-09-26 |
| [0008](0008-mfa-enforcement-and-oauth-account-linking-policy.md) | MFA 强制策略与 OAuth 账户关联策略 | 已接受 | 2026-09-26 |

## 决策状态定义

ADR 必须且只能使用以下 5 种状态，清晰表达决策的有效性：

| 状态 | 决策效力 | 适用场景 | 状态流转规则 |
| --- | --- | --- | --- |
| **`提议`** (Proposed) | **尚未生效**<br>方案正在团队评审讨论中，不具备架构约束力。 | 提出重大技术选型或架构调整建议时。 | 评审通过转为 `已接受`；否决转为 `已拒绝`。不得在提议状态下合并生产代码。 |
| **`已接受`** (Accepted) | **当前有效基准**<br>团队达成共识并通过评审，代码实现必须严格遵循。 | 成为现行架构事实的决策。 | 架构演进时可被新 ADR 转换为 `被替代`，或不再适用时转为 `已废弃`。 |
| **`被替代`** (Superseded) | **已失去效力**<br>决策曾被执行，但已被后续更高编号的新 ADR 废止或替换。 | 随着技术发展或业务规模扩大推翻旧决策。 | **必须**在头部显式注明被哪个 ADR 替代（如 `被替代：ADR-0005`），保留历史不可修改。 |
| **`已拒绝`** (Rejected) | **不予采纳**<br>方案经过正式讨论评估后决定不采用。 | 备选方案探索完成但不符合团队当前约束。 | 记录未采纳的真实考量（如成本、复杂度、技术成熟度），防止团队在未来重复辩论相同提案。 |
| **`已废弃`** (Deprecated) | **失效退役**<br>所依赖的外部条件、业务形态或技术栈已不复存在。 | 相关组件彻底下线、架构重构后历史决策自然失效。 | 保留历史备查，但不再对新开发具有约束力。 |

## 创建与流转流程

1. 复制 [`docs/templates/adr.md`](../templates/adr.md)；
2. 使用下一个四位编号和简短 kebab-case 名称；
3. 在做出不可逆实现之前发起评审；
4. 接受后更新本索引；
5. 已接受 ADR 不重写历史，使用新 ADR 将其标记为被替代。
