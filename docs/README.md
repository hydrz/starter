# 文档中心

这里是项目文档的统一入口。开始开发前先阅读[本地开发指南](development/getting-started.md)，需要了解整体边界时阅读[工程蓝图](engineering-blueprint.md)。

## 导航

| 分类 | 用途 | 入口 |
| --- | --- | --- |
| 开发指南 | 完成具体开发任务 | [本地开发](development/getting-started.md)、[契约](development/contracts.md)、[数据库](development/database.md)、[纵向切片](development/vertical-slice.md) |
| 工程规范 | 约束代码与评审质量 | [规范索引](standards/README.md) |
| 架构决策 | 记录重要选择及其原因 | [ADR 目录](adr/README.md) |
| 运维手册 | 在明确场景下执行诊断和恢复 | [Runbook 目录](runbooks/README.md) |
| 模板 | 创建结构一致的新文档 | [模板目录](templates/README.md) |
| 路线图 | 了解分阶段交付状态 | [实施路线](delivery-roadmap.md) |
| GitHub 治理 | 配置检查、评审和发布规则 | [仓库治理](../.github/GOVERNANCE.md) |
| 交付与运维 | 构建、部署、配置和运行服务 | [部署指南](deployment.md)、[Runbook 目录](runbooks/README.md) |
| Agent Skills | 规范仓库级 Agent 工作流的位置、边界与验证方式 | [Agent Skills 规范与实现](agent-skills.md) |
| GitHub Template | 从模板创建仓库并自动初始化项目命名 | [模板仓库指南](template-repository.md) |

## 文档治理

- [文档规范与生命周期](documentation-policy.md)
- [文档目录与负责人](documentation-catalog.md)
- [贡献指南](../CONTRIBUTING.md)

发现文档和实现不一致时，优先在修复实现的同一 Pull Request 中更新文档。若无法立即修复，应建立跟踪事项并在目录中将状态标记为需要复审。
