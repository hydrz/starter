# 契约与数据规范

## TypeSpec 与 HTTP

- TypeSpec 是 HTTP 契约的唯一事实来源，OpenAPI 和前后端生成代码禁止手改；
- 每个 operation 必须有稳定、能表达动作的 `operationId`；
- 请求输入定义长度、格式、枚举等可声明约束；服务端仍执行实际校验；
- 公开失败响应使用稳定的机器错误码与面向使用者的信息，内部错误细节只进入日志；
- 破坏性契约变更必须升级 API 版本或提供兼容迁移窗口。

## Migration

- 文件名使用递增编号和清晰描述，已经进入共享环境的 migration 永不修改；
- 生产环境只向前修复；`Down` 只服务于发布前验证；
- 破坏性 schema 变更采用 expand/contract，并考虑新旧应用版本并存；
- 约束应尽量由数据库表达，但不得把完整业务流程塞入 trigger；
- 大表 migration 必须评估锁、执行时间、回填和失败恢复方式。

## SQL 与 sqlc

- 查询列名必须显式列出，业务查询禁止 `SELECT *`；
- 所有查询使用 sqlc 命名注释和符合结果基数的命令；
- 参数始终绑定，禁止字符串拼接用户输入；
- 列表查询必须有稳定排序、明确上限，并为实际过滤与排序方式设计索引；
- 事务边界由应用用例决定，不在 repository 内隐藏跨用例事务；
- JSONB 用于真正动态结构，不能替代正常关系建模。

## 变更验证

契约或 SQL 变更执行 `pnpm generate`、`pnpm check:generated`；migration 变更额外执行 `pnpm test:database`。
