# ADR-0001：提交契约生成物

- 状态：已接受
- 日期：2026-09-24

## 背景

TypeSpec 会派生 OpenAPI 文档、Go 服务端接口与 TypeScript 客户端。团队需要决定这些生成物是只在构建时生成，还是一并纳入版本控制。

## 决策

将以下生成物提交到 Git：

- `spec/generated/openapi.yaml`；
- `internal/api/openapi.gen.go`；
- `apps/web/src/api/generated/`。

生成物顶部或所在目录必须明确禁止手工修改。所有修改从 TypeSpec 或生成配置开始，并由 `pnpm generate` 统一生成。`pnpm check:generated` 会重新生成并检查 Git diff，阻止源契约与生成物漂移。

## 结果

优点：

- Pull Request 可直接审阅 API 兼容性及前后端影响；
- Go 与 Web 构建不要求重复安装所有生成工具；
- 可以清楚发现生成器升级造成的变化。

代价：

- 提交内容会增加；
- 生成器版本必须固定，否则容易产生无关 diff；
- 合并冲突必须通过重新生成解决，不能手工编辑生成文件。
