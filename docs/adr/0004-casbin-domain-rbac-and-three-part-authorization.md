# ADR-0004：Casbin 域 RBAC 与三段授权

- 状态：已接受
- 日期：2026-09-26
- 决策者：Architecture
- 被替代：无

## 背景

多组织产品需要在组织边界内表达成员角色和资源访问，而系统级角色或单纯 URL 权限无法表达租户隔离。授权还必须避免将账单状态、认证身份和业务操作混为一层。

## 决策

采用 Casbin domain RBAC。每个授权请求明确提供 **subject**、**domain**、**object/action**：subject 是已认证的账户或服务身份，domain 通常是组织，object/action 是受保护资源及操作。组织成员关系决定 subject 是否可在 domain 中被评估；Casbin policy 决定允许的 object/action。路由与领域层都必须通过该统一授权边界，而不是分散比较角色字符串。

Casbin model 和 policy 的配置路径必须成对出现；实际策略、组织表和 middleware 留给授权工作流实现。计费 entitlement 可以作为业务操作的额外前置条件，但不替代组织内授权。

## 备选方案

### 应用各处硬编码角色检查

初期编写快，但难以审计、复用和保持组织隔离一致性，未选择。

### 仅系统范围 RBAC

不能可靠表达同一用户在不同组织中的不同角色，未选择。

## 结果

授权输入和审计边界明确，代价是需要维护 domain 解析、成员关系和 policy 测试。后续实现必须从 TypeSpec 定义公开的 401/403 语义，并通过前向 migration/sqlc 管理持久化状态。

Authula 的公开授权边界是设计影响来源；本仓库不复制其代码。
