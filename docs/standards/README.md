# 工程规范索引

工程规范服务于三个目标：让正确做法容易执行、让常见错误自动暴露、让必要例外可追踪。规范分为以下部分：

- [Go 工程规范](go.md)
- [Web 工程规范](web.md)
- [契约与数据规范](contracts-and-data.md)
- [测试规范](testing.md)
- [代码评审清单](code-review.md)

## 规范优先级

1. 编译器、生成器与自动检查；
2. 仓库内的 ADR；
3. 本目录中的工程规范；
4. 团队约定与个人偏好。

发生冲突时采用更高优先级规则。确需例外时遵循根目录 `CONTRIBUTING.md` 中的例外流程，而不是修改生成文件或关闭整个检查项。

## 自动化映射

| 规则 | 自动检查 |
| --- | --- |
| 生成物与源文件一致 | `pnpm check:generated` |
| Go 与前端格式 | `pnpm check:format` |
| Go 静态分析 | `pnpm check:go` |
| SQL 与 schema/query 一致性 | `pnpm check:sql` |
| ESLint 与 TypeScript 严格模式 | `pnpm check:web` |
| Go 竞态 | `pnpm test:race` |

`pnpm check` 聚合全部静态代码与生成漂移检查；单元测试与竞态检测由 `pnpm test` 与 `pnpm test:race` 执行。

## 新增与维护规范

新增或修订工程规范时遵守以下约定：

1. **保存位置**：统一保存在 `docs/standards/<topic>.md`（使用小写 `kebab-case` 命名，如 `docs/standards/security.md`）；
2. **必需元数据**：遵循[文档规范与生命周期](../documentation-policy.md)，标题下方必须包含状态（`Draft`、`Active` 等）、负责人、最后复审日期与复审周期；
3. **登记索引**：在本文件顶部的列表加入链接，并在[文档目录与负责人](../documentation-catalog.md)中确认责任团队归属；
4. **自动化优先**：凡是能通过编译器、Linter 或脚本检查的规则，优先落地为 `pnpm check` 的自动化门禁，不在文档中做重复的手工核对约定。
