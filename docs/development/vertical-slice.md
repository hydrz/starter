# 纵向业务切片指南

系统公告与通知 (Announcements) 是仓库中的第一个完整业务切片，用于展示需求如何沿着既定依赖方向落地，而不是作为通用 CRUD 框架。

## 文件映射

| 层 | 公告实现 |
| --- | --- |
| Contract | `packages/contracts/features/announcements/routes.tsp` 中的 `Announcements` 接口 |
| Migration | `db/migrations/00001_create_announcements.sql` |
| Query | `db/queries/announcements.sql` |
| Generated store | `internal/store/announcements.sql.go` |
| Application | `internal/announcement/service.go` |
| Repository adapter | `internal/announcement/postgres.go` |
| Generated server contract | `internal/api/announcementsapi/` |
| HTTP adapter | `internal/announcement/handler.go` |
| Generated client | `apps/web/src/api/generated/announcements/announcements.ts` |
| React feature | `apps/web/src/features/announcements/Announcements.tsx` |

## 依赖方向

```text
HTTP adapter ─> announcement.Service ─> announcement.Repository
                                             ▲
                                             │ implements
                                     PostgresRepository ─> sqlc
```

应用服务只表达公告用例、分页边界和输入不变量，不依赖 HTTP 或 pgx。PostgreSQL adapter 负责在领域模型与 sqlc 类型之间转换。HTTP adapter 负责 JSON、状态码以及 API 错误模型。

## 已验证的边界

- TypeSpec 定义列表、创建、读取、更新和删除契约；
- 列表接口提供有限制的 offset 分页和状态过滤；
- 服务端与数据库同时保护标题、内容和状态不变量；
- 未找到资源返回稳定的 `not_found` 错误代码；
- 非法请求返回 `invalid_request`，未知内部错误不向客户端泄露细节；
- React Hook Form 与 Zod 在提交前校验表单；
- Orval mutation 成功后通过 TanStack Query 失效列表缓存。

## 扩展方式

新增业务模块时复制依赖方向，而不是复制所有物理文件。简单读取可以直接使用 sqlc；只有存在业务不变量或需要隔离存储模型时，才引入应用服务和 repository adapter。跨资源事务应由应用层定义边界，并通过显式 transaction adapter 执行。
