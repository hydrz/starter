# 数据库开发指南

## 职责与事实来源

| 内容 | 位置 | 工具 |
| --- | --- | --- |
| Schema 历史 | `db/migrations/*.sql` | Goose |
| 数据访问 SQL | `db/queries/*.sql` | sqlc |
| Go 数据访问代码 | `internal/store/` | sqlc 生成，禁止手工修改 |
| 本地数据库 | `compose.yaml` | PostgreSQL |

数据库 schema 以 Goose migration 为唯一事实来源，不维护独立的 `schema.sql`。sqlc 会直接读取 migration 和查询，生成使用 pgx/v5 的类型安全代码。

## 本地工作流

```bash
cp .env.example .env
pnpm db:up
pnpm db:migrate
pnpm generate:db
pnpm test
```

本地默认连接地址为 `postgres://starter:starter@127.0.0.1:5432/starter?sslmode=disable`。默认凭据只允许用于本地开发，不得复制到共享或生产环境。

## Schema 变更流程

1. 新建递增编号的 `db/migrations/NNNNN_description.sql`；
2. 同时编写 `Up` 与供本地验证使用的 `Down`；
3. 修改 `db/queries/*.sql`；
4. 执行 `pnpm generate:db`，不得直接编辑 `internal/store`；
5. 执行 `pnpm test:database` 验证 `up → down → up`；
6. 执行 `pnpm check`、`pnpm test` 并审查生成代码；
7. migration 发布后永不修改，修复通过新的 migration 完成。

生产环境遵循 [ADR-0002](../adr/0002-forward-only-production-migrations.md) 的只向前迁移策略。

## SQL 约定

- 明确列名，禁止在业务查询中使用 `SELECT *`；
- 查询必须使用 sqlc 命名注释，并选择正确的 `:one`、`:many`、`:exec` 或 `:execrows`；
- 写操作应返回调用方继续处理所需的数据，避免紧随其后的重复查询；
- 时间使用 `timestamptz`，由数据库生成的时间统一使用 `now()`；
- JSONB 只用于结构确实动态的数据，不替代正常关系建模；
- 查询复杂度出现后再建立 repository 包装层，不为简单 CRUD 预建无行为抽象。

## 数据库验证

`pnpm test:database` 会用独立的 Compose 项目和本地 `55432` 端口创建临时数据库、执行全部 migration、回退一步、再次升级并输出状态，最后删除测试 volume，不会复用日常开发数据库。可通过 `POSTGRES_PORT` 覆盖测试端口。
