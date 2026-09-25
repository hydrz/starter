# Starter 企业应用模板

这个 GitHub Template Repository 提供一套面向企业内部运营管理平台的工程化基线。最终交付形态是一个包含前端静态资源的 Go 二进制文件，并通过 Docker Compose 部署。

## 从模板创建项目

在 GitHub 上选择 **Use this template** 并指定新仓库名。首次 `main` push 会触发一次性初始化工作流，根据 `owner/repository` 替换 Go module、pnpm package scope、Compose/image/database 标识和展示名称，然后删除初始化工作流。

初始化完成前不要开始业务开发。等待 `Initialize template repository` 工作流提交 `chore: initialize project from template`，再按[模板仓库指南](docs/template-repository.md)验证结果。

- [工程蓝图](docs/engineering-blueprint.md)
- [分阶段实施路线](docs/delivery-roadmap.md)
- [文档中心](docs/README.md)
- [本地开发指南](docs/development/getting-started.md)
- [契约开发指南](docs/development/contracts.md)
- [数据库开发指南](docs/development/database.md)
- [纵向切片开发指南](docs/development/vertical-slice.md)
- [模板仓库指南](docs/template-repository.md)
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
