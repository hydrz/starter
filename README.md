# 企业应用工程化工具链

这个仓库用于逐步沉淀一套面向企业内部运营管理平台的工程化方案。最终交付形态是一个包含前端静态资源的 Go 二进制文件，并通过 Docker Compose 部署。

## 当前进度

当前完成 **第 9 部分：交付与运维**。Vite 产物已嵌入 Go 单二进制，并通过最小容器、Compose migration job、健康检查和发布制品完成部署闭环。

第 10 部分暂未开始实现；当前先评审 [Agent Skills 规范与实施提案](docs/agent-skills.md)，确认正确的发现目录、职责边界与验证方法。

- [工程蓝图](docs/engineering-blueprint.md)
- [分阶段实施路线](docs/delivery-roadmap.md)
- [文档中心](docs/README.md)
- [本地开发指南](docs/development/getting-started.md)
- [契约开发指南](docs/development/contracts.md)
- [数据库开发指南](docs/development/database.md)
- [纵向切片开发指南](docs/development/vertical-slice.md)
- [工程规范索引](docs/standards/README.md)
- [贡献指南](CONTRIBUTING.md)

## 核心技术栈

| 分层 | 选型 |
| --- | --- |
| 数据持久层 | PostgreSQL、sqlc、pgx、Goose |
| 协议与契约 | TypeSpec、OpenAPI、Scalar、oapi-codegen、Orval |
| 后端 | Go、Chi |
| 前端 | React、TypeScript、Vite、TanStack Query、TanStack Router、Tailwind CSS、shadcn/ui、React Hook Form、Zod、Zustand |
| 工程与交付 | pnpm Workspaces、pnpm scripts、Go embed、Docker Compose |

## 协作方式

每个阶段遵循“提案 → 评审确认 → 实现 → 验证 → 文档固化”的节奏。未经确认，不跨阶段堆叠脚手架或工作流，确保每一步都可审查、可回退。

## 快速开始

```bash
cp .env.example .env
pnpm install
pnpm db:up
pnpm db:migrate
pnpm dev
```

前端默认运行在 `http://127.0.0.1:5173`，并将 `/api` 请求代理到运行在 `http://127.0.0.1:8080` 的 Go 服务。更多命令和环境要求参见[本地开发指南](docs/development/getting-started.md)。

## Compose 部署

```bash
cp .env.example .env
docker compose up --build --detach
docker compose ps
```

应用默认只绑定 `127.0.0.1:8080`。部署、配置、migration 和验证步骤参见[交付与运维指南](docs/deployment.md)。
