# 架构决策记录

ADR 记录影响多个模块、难以撤销或需要保存取舍背景的决策。普通实现细节不需要 ADR。

## 索引

| 编号 | 决策 | 状态 |
| --- | --- | --- |
| [0001](0001-commit-generated-contract-artifacts.md) | 提交契约生成物 | 已接受 |
| [0002](0002-forward-only-production-migrations.md) | 生产数据库只向前迁移 | 已接受 |

## 创建方式

1. 复制 [`docs/templates/adr.md`](../templates/adr.md)；
2. 使用下一个四位编号和简短 kebab-case 名称；
3. 在做出不可逆实现之前发起评审；
4. 接受后更新本索引；
5. 已接受 ADR 不重写历史，使用新 ADR 将其标记为被替代。
