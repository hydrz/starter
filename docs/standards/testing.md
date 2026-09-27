# 测试规范

## 测试分层

| 类型 | 目标 | 默认入口 |
| --- | --- | --- |
| 单元测试 | 业务分支、校验、纯转换 | `pnpm test` |
| Transport 测试 | 路由、状态码、序列化、公开错误 | `go test ./internal/platform/httpserver/...` |
| 数据库集成测试 | PostgreSQL 数据访问与事务行为 | `pnpm test:integration` |
| 生成漂移测试 | SSOT 与生成物一致 | `pnpm check:generated` |
| 构建测试 | 生产制品可编译 | `pnpm build` |

## 数据库集成测试

- 数据库集成测试通过 `internal/platform/testdb`（testcontainers-go + `postgres:17`）启动真实 PostgreSQL 并执行全部 goose migration；无 Docker 环境时自动 `t.Skip`，其他启动或迁移错误视为失败。
- 集成测试文件使用 `//go:build integration` 构建标签，入口为 `pnpm test:integration`（等价于 `go test -tags=integration -count=1 ./...`）；普通 `pnpm test` 不依赖 Docker。CI 中由独立的 `integration` job 运行。
- **何时写集成测试**：当行为依赖 PostgreSQL 语义而单元测试无法覆盖时必须写集成测试，包括事务 commit/rollback（PgxTransactor）、并发锁语义（如 outbox `FOR UPDATE SKIP LOCKED` 领取）、约束与幂等（UNIQUE/ON CONFLICT 行为）、以及每个 `db/queries/*.sql` 新增查询的主路径。
- 每个测试通过 `testdb.New(t)` 获得独立、已迁移的数据库实例，测试间互不共享状态；断言对外行为（写入对外可见性、错误映射、行数），不断言内部调用顺序。

## 编写规则

- 测试关注对外行为，不绑定无意义的内部调用顺序；
- 每个测试清楚区分 Arrange、Act、Assert，不引入测试间共享可变状态；
- Go 测试在安全时调用 `t.Parallel()`，临时资源通过 `t.Cleanup()` 释放；
- 缺陷修复必须包含在修复前失败的回归测试；
- 时间、随机数和外部服务在业务层通过窄接口控制，避免 sleep 与不稳定网络；
- 测试数据应最小且表达意图，不复制生产数据。

## 失败处理

不允许通过增加无界重试、延长 sleep 或跳过断言处理 flaky test。应先定位共享状态、时间、顺序或外部依赖，并记录根因。确因环境能力缺失而未运行的检查必须在 Pull Request 中明确标记。

## 覆盖策略

不设置脱离风险的统一覆盖率目标。核心业务不变量、错误映射、权限边界和不可逆数据变更必须覆盖；简单生成代码不重复测试，改由生成漂移和编译检查保护。
