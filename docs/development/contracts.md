# 契约开发指南

## 单一事实来源

HTTP API 在 `packages/contracts/` 中按模块组织（`features/` 与 `common/`），并通过 `main.tsp` 汇聚。OpenAPI、Go 强类型服务端接口与解析器（ogen）、TypeScript 客户端 hooks 都是生成物，不得手工编辑。

```text
packages/contracts/
  ├── common/                 # 通用模型与错误响应
  ├── features/
  │   ├── system/             # 系统存活/就绪探针
  │   └── announcements/      # 业务切片
  └── main.tsp                # 聚合入口
        │
        ├─ TypeSpec ─> spec/generated/
        │                ├── openapi.yaml & openapi.json (统一契约与 Scalar 文档)
        │                └── *.openapi.yaml (模块化 OpenAPI 规格)
        │
        ├─ ogen ────> internal/api/*api/ (模块化强类型 Handler 与自动校验)
        │
        └─ Orval ───> apps/web/src/api/generated/ (按标签拆分的 React Query Hooks)
```

生成物按照 [ADR-0001](../adr/0001-commit-generated-contract-artifacts.md) 纳入版本控制，以便在 Pull Request 中审查契约影响。

## 修改流程

1. 在 `packages/contracts/features/<module>/` 编写契约并由 `main.tsp` 引用；
2. 执行 `pnpm generate`；
3. 在对应的内部模块（如 `internal/<module>`）实现生成的强类型 `Handler` 接口；
4. 在 Web 中只使用 Orval 生成的模块化函数或 hooks；
5. 执行 `pnpm check` 和 `pnpm test`；
6. 审查 TypeSpec、OpenAPI 与两端生成代码的差异后一起提交。

不要在同一个变更中手工修复生成文件。如果生成结果不符合预期，应调整 TypeSpec 或生成配置。

## 命令

| 命令 | 作用 |
| --- | --- |
| `pnpm generate` | 按依赖顺序生成全部契约产物 |
| `pnpm generate:contract` | TypeSpec → 统一及模块化 OpenAPI Spec |
| `pnpm generate:server` | 模块化 OpenAPI → ogen 强类型服务端代码 |
| `pnpm generate:web` | OpenAPI → 按标签拆分的 Fetch/TanStack Query 客户端 |
| `pnpm check:generated` | 重新生成并检查已提交产物是否发生漂移 |

## API 文档

服务启动后可访问：

- Scalar API Reference：`http://127.0.0.1:8080/api/docs`；
- OpenAPI YAML：`http://127.0.0.1:8080/api/openapi.yaml`；
- OpenAPI JSON（兼容端点）：`http://127.0.0.1:8080/api/openapi.json`；
- OpenAPI 源生成物：`spec/generated/openapi.yaml`。

Scalar 交互文档已 100% 离线内嵌在服务二进制中（通过 `/api/docs/scalar.js` 提供资源），完全不依赖任何外部 CDN，支持在企业隔离内网与离线专网中完整浏览。

## 兼容性要求

- 已发布字段不得在同一 API 版本内改变语义；
- 删除字段、收紧类型或新增必填输入属于破坏性变更；
- operation ID 必须稳定，它决定 Go 方法和 TypeScript 函数名称；
- API 路径统一使用 `/api` 前缀，文档与规范端点同样位于该命名空间。
