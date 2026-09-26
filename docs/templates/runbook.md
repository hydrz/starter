# [事件或恢复任务名称]

> **说明**：从本模板创建新 Runbook 时，保存至 `docs/runbooks/<kebab-case-name>.md`（如 `docs/runbooks/redis-failover.md`），并在 [`docs/runbooks/README.md`](../runbooks/README.md) 与 [`docs/documentation-catalog.md`](../documentation-catalog.md) 中登记。生效后删除本说明引用块。

- **状态**：Draft（可选值：Draft | Active | Superseded | Deprecated，详见 [文档规范与生命周期](../documentation-policy.md)）
- **负责人**：[值班团队或角色]
- **最后复审**：YYYY-MM-DD
- **复审周期**：90 天

## 触发条件与影响

[说明告警、用户症状、影响范围和不适用场景。]

## 前置安全检查

- [权限、备份、审批和不可逆操作确认。]

## 诊断

1. [先执行只读、低风险检查。]
2. [记录关键时间、请求 ID、版本和观测结果。]

## 缓解与恢复

1. [最小风险的缓解步骤。]
2. [恢复步骤及每步停止条件。]

## 验证

[给出指标、健康检查和用户路径的成功标准。]

## 回退与升级

[说明何时停止、如何回退、联系哪个角色以及需要携带哪些证据。]

## 事后工作

- [事件记录、数据核对、永久修复和本文更新。]
