# Agent 指令与工程约束 (AGENTS.md)

本文件是面向 AI Agent 的仓库级全局约束、架构事实与执行指引。具体任务流通过上下文指针读取对应 Agent Skill 或开发指南，不在此重复维护。

## 1. 核心架构与事实来源 (SSOT)

- **技术栈**：Go 1.27 + React 19 + TypeSpec + PostgreSQL 17，单二进制嵌入交付（Vite 产物内嵌至 Go 二进制，零 Node.js 运行时依赖）；
- **API 契约**：`packages/contracts/` 下的 TypeSpec 是 HTTP API 的唯一事实来源，生成 OpenAPI 规范、Go 服务端桩代码（`ogen`）与前端请求客户端（`Orval`）；
- **数据持久化**：PostgreSQL 配合 `db/migrations/`（Goose 迁移）与 `db/queries/`（sqlc 生成类型安全 Go 代码），采用 Context-Driven Transactor 事务模式；
- **前后端分界**：后端采用模块化单体，传输层（`internal/api/`）与存储层（`internal/store/`）由工具生成，业务用例位于独立领域包（如 `internal/announcement/`）。

## 2. 硬性护栏与安全边界 (Guardrails)

- **严禁手改生成文件**：包括 `spec/generated/`、`internal/api/`、`internal/store/`、`apps/web/src/api/generated/`，变更必须修改 SSOT 源文件后执行 `pnpm generate`；
- **生产迁移只向前推进 (Forward-only)**：生产数据库严禁回滚已执行迁移（遵循 [ADR-0002](docs/adr/0002-forward-only-production-migrations.md)），变更必须使用新 migration 文件修复；
- **严禁泄露敏感信息**：不得向 Git 提交密钥、生产连接串、认证凭据或本地 `.env` 文件；
- **破坏性操作必须显式确认**：涉及删除数据、重置数据库（`pnpm db:reset`）、标签推送、外部写入或版本发布的动作，必须获得人工明确授权；
- **依赖方向与分层隔离**：应用层不直接暴露 store 或 transport 原始类型，领域模型与外部 DTO 严格解耦。

## 3. 任务路由与上下文指针 (Context Pointers)

根据当前任务分支按需读取对应的权威文档与 Skill，不要凭空推断流程：

- **API 契约变更**：读取 [`.agents/skills/change-api-contract/SKILL.md`](.agents/skills/change-api-contract/SKILL.md)，参考 [`docs/development/contracts.md`](docs/development/contracts.md) 与 [`docs/standards/contracts-and-data.md`](docs/standards/contracts-and-data.md)；
- **数据库表结构或查询变更**：读取 [`.agents/skills/change-database-schema/SKILL.md`](.agents/skills/change-database-schema/SKILL.md)，参考 [`docs/development/database.md`](docs/development/database.md)；
- **纵向全栈业务切片开发**：读取 [`.agents/skills/build-vertical-slice/SKILL.md`](.agents/skills/build-vertical-slice/SKILL.md)，参考 [`docs/development/vertical-slice.md`](docs/development/vertical-slice.md)；
- **代码评审与自检**：读取 [`.agents/skills/review-change/SKILL.md`](.agents/skills/review-change/SKILL.md)，参考 [`docs/standards/code-review.md`](docs/standards/code-review.md)；
- **版本发布与上线准备**：读取 [`.agents/skills/prepare-release/SKILL.md`](.agents/skills/prepare-release/SKILL.md)，参考 [`docs/deployment.md`](docs/deployment.md)；
- **代码风格与规范**：参考 [`docs/standards/go.md`](docs/standards/go.md)（Go 后端）、[`docs/standards/web.md`](docs/standards/web.md)（React 前端）、[`docs/standards/testing.md`](docs/standards/testing.md)（测试）；
- **平台底座增强任务**：按 [`docs/implementation/platform-foundation-tasks.md`](docs/implementation/platform-foundation-tasks.md) 领取单个任务卡执行；
- **架构决策与背景**：参考 [`docs/engineering-blueprint.md`](docs/engineering-blueprint.md) 与 [`docs/adr/README.md`](docs/adr/README.md)；
- **文档创建与模板使用**：按 [`docs/templates/README.md`](docs/templates/README.md) 与 [`docs/documentation-policy.md`](docs/documentation-policy.md) 规定的目标路径保存与索引登记。

## 4. 验证命令与完成条件 (Done Criteria)

任务完成或提交代码前，必须依次通过以下验证闭环：

```bash
pnpm generate         # 若修改了 TypeSpec、SQL 或迁移，重新生成生成物
pnpm check            # 聚合门禁：生成物防漂移、格式、文档、Skills、Actions、Go、SQL、Web 类型与 Lint
pnpm test             # 运行 Go 单元测试与前端 Vitest 测试
pnpm test:race        # Go 竞态检测（涉及并发或后端逻辑变更时必跑）
```

验证结果中不得包含未解决的静态分析错误、生成文件未提交漂移或测试失败。
