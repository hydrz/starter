# 工程蓝图

## 1. 目标与边界

本方案服务于企业内部运营管理平台，优先保证：

1. **契约唯一**：HTTP API 只在 TypeSpec 中定义，生成物不得手工修改。
2. **SQL 优先**：数据库行为通过显式 SQL 表达，sqlc 负责生成类型安全的 Go 访问层。
3. **单体优先**：默认采用模块化单体，而不是预先拆分微服务。
4. **单一制品**：Vite 产物嵌入 Go 二进制，运行时不依赖 Node.js。
5. **本地与 CI 同构**：开发者执行的 pnpm scripts 也是 CI 调用的入口。
6. **渐进治理**：规范、生成器和检查项随实施阶段逐步加入，避免一次性引入不可维护的复杂度。

### 暂不纳入

- Kubernetes 与多集群发布体系；
- 事件总线、CQRS 等分布式架构；
- 与具体云厂商绑定的基础设施；
- 在业务需求明确前设计通用低代码平台。

## 2. 架构原则

### 2.1 单向依赖

```text
TypeSpec
   └── OpenAPI
       ├── oapi-codegen ──> Go transport types/interfaces
       ├── Orval ─────────> React Query client/hooks
       └── Scalar ────────> API reference

Goose migrations ──> PostgreSQL schema
SQL queries ────────> sqlc ──> pgx-based repository code

React/Vite ──> dist ──> go:embed ──> one executable
```

上游源文件可以生成下游文件；下游文件不得反向成为事实来源。生成代码应具备可重复性，并通过 CI 的 clean-tree 检查防止漂移。

### 2.2 分层职责

| 层 | 职责 | 不应承担 |
| --- | --- | --- |
| Contract | TypeSpec API、共享模型、错误语义 | 业务实现与数据库结构 |
| Transport | Chi 路由、鉴权入口、参数转换、响应映射 | SQL 与核心业务规则 |
| Application | 用例编排、事务边界、授权策略 | HTTP 细节与生成代码修改 |
| Domain | 业务规则、领域值与不变量 | 框架依赖 |
| Repository | sqlc 查询组合、持久化适配 | HTTP 响应构造 |
| Web | 页面、交互、查询与客户端状态 | 手写服务端 DTO 镜像 |

对于简单 CRUD，可以合并 Application 与 Domain 的物理目录，但依赖方向保持不变；只有复杂度出现时才增加抽象。

## 3. 单一事实来源（SSOT）

| 内容 | 唯一事实来源 | 生成/消费结果 |
| --- | --- | --- |
| HTTP API | `spec/main.tsp` | OpenAPI、Go server contract、Web client、Scalar 文档 |
| 数据库结构 | `db/migrations/*.sql` | PostgreSQL schema |
| 数据访问 | `db/queries/*.sql` | sqlc Go 代码 |
| 前端设计令牌 | Web 主题与 CSS variables | Tailwind/shadcn 组件样式 |
| 自动化入口 | 根目录 `package.json` scripts | 本地开发与 GitHub Actions |

当生成结果不符合预期时，应修改源定义或生成配置，而不是直接修补生成文件。

## 4. 目标仓库布局

以下是后续阶段要逐步创建的目标结构，并非要求在第一阶段生成空目录：

```text
.
├── apps/
│   ├── server/               # Go 入口、HTTP 服务和前端 embed
│   └── web/                  # React + Vite 管理端
├── internal/                 # Go 内部业务模块
├── spec/                     # TypeSpec 与生成配置
├── db/
│   ├── migrations/           # Goose migration
│   ├── queries/              # sqlc SQL
│   └── sqlc.yaml
├── packages/                 # 可复用前端包（按需创建）
├── docs/
│   ├── adr/                  # 架构决策记录
│   ├── development/          # 开发与调试说明
│   └── runbooks/             # 运维手册
├── .agents/
│   └── skills/               # Codex 可发现的仓库级 Agent Skills
├── tools/                    # 代码生成与工程脚本
├── compose.yaml
├── go.mod
├── package.json
└── pnpm-workspace.yaml
```

## 5. 关键数据流

### 5.1 契约变更

1. 修改 TypeSpec；
2. 生成并校验 OpenAPI；
3. 由 oapi-codegen 生成后端接口边界；
4. 由 Orval 生成前端请求函数与 TanStack Query hooks；
5. 运行编译、契约漂移检查和相关测试；
6. 提交源文件、配置以及约定纳入版本控制的生成物。

是否提交生成物将在脚手架阶段通过 ADR 明确；无论采用哪种策略，CI 都必须验证生成结果可重复。

### 5.2 数据库变更

1. 创建只向前演进的 Goose migration；
2. 在本地临时数据库执行迁移；
3. 编写或调整查询 SQL；
4. 运行 sqlc 生成；
5. 运行静态检查、repository 测试和迁移往返测试；
6. 禁止通过修改已发布 migration 来改写历史。

### 5.3 构建与运行

1. pnpm 安装固定版本依赖；
2. Vite 构建 Web 静态资源；
3. Go 通过 `embed` 打包前端产物；
4. 构建单一 Linux 二进制；
5. 容器镜像只携带二进制和必要证书；
6. Docker Compose 编排应用与 PostgreSQL，并通过健康检查控制依赖顺序。

## 6. 质量门禁

后续 CI 至少覆盖：

- 格式化、lint 与类型检查；
- Go 单元测试及竞态检查；
- Web 单元测试；
- TypeSpec、OpenAPI、sqlc 与 Orval 生成漂移检查；
- Goose migration 校验；
- 生产构建与容器构建；
- 依赖和镜像安全扫描；
- 提交信息与 Pull Request 模板校验。

质量门禁应优先调用根目录 pnpm scripts，例如 `pnpm check`、`pnpm test`、`pnpm generate` 和 `pnpm build`，避免维护本地与 CI 两套命令。

## 7. 待 ADR 确认的决策

以下事项在对应实施阶段通过架构决策记录确认：

- 生成代码是否提交到 Git；
- API 版本策略与错误响应模型；
- 数据库 migration 的回滚策略；
- 身份认证、会话与权限模型；
- 前端 history fallback 与 API 路径隔离；
- 配置、密钥和审计日志规范；
- release、镜像标签与数据库升级策略。
