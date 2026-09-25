# 契约开发指南

## 单一事实来源

HTTP API 只在 `packages/contracts/main.tsp` 中定义。OpenAPI、Go 类型与路由接口、TypeScript 请求函数和 TanStack Query hooks 都是生成物，不得手工编辑。

```text
packages/contracts/main.tsp
  └─ TypeSpec ─> spec/generated/openapi.yaml
                   ├─ oapi-codegen ─> internal/api/openapi.gen.go
                   └─ Orval ────────> apps/web/src/api/generated/
```

生成物按照 [ADR-0001](../adr/0001-commit-generated-contract-artifacts.md) 纳入版本控制，以便在 Pull Request 中审查契约影响。

## 修改流程

1. 修改 `packages/contracts/main.tsp`；
2. 执行 `pnpm generate`；
3. 在 `internal/platform/httpserver` 实现生成的 `api.ServerInterface`；
4. 在 Web 中只使用 Orval 生成的函数或 hooks；
5. 执行 `pnpm check` 和 `pnpm test`；
6. 审查 TypeSpec、OpenAPI 与两端生成代码的差异后一起提交。

不要在同一个变更中手工修复生成文件。如果生成结果不符合预期，应调整 TypeSpec 或 `tools/oapi-codegen.yaml`、`apps/web/orval.config.ts`。

## 命令

| 命令 | 作用 |
| --- | --- |
| `pnpm generate` | 按依赖顺序生成全部契约产物 |
| `pnpm generate:contract` | TypeSpec → OpenAPI |
| `pnpm generate:server` | OpenAPI → Go/Chi 服务端代码 |
| `pnpm generate:web` | OpenAPI → Fetch/TanStack Query 客户端 |
| `pnpm check:generated` | 重新生成并检查已提交产物是否发生漂移 |

## API 文档

服务启动后可访问：

- Scalar API Reference：`http://127.0.0.1:8080/api/docs`；
- OpenAPI JSON：`http://127.0.0.1:8080/api/openapi.json`；
- OpenAPI 源生成物：`spec/generated/openapi.yaml`。

Scalar 交互文档已 100% 离线内嵌在服务二进制中（通过 `/api/docs/scalar.js` 提供资源），完全不依赖任何外部 CDN，支持在企业隔离内网与离线专网中完整浏览。

## 兼容性要求

- 已发布字段不得在同一 API 版本内改变语义；
- 删除字段、收紧类型或新增必填输入属于破坏性变更；
- operation ID 必须稳定，它决定 Go 方法和 TypeScript 函数名称；
- API 路径统一使用 `/api` 前缀，文档与规范端点同样位于该命名空间。
