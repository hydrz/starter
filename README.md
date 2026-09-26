# Starter · 现代化全栈应用开发模板 (Full-Stack Starter Kit)

Starter 是一个高生产力、现代化且生产就绪的**全栈 Web 应用与 SaaS 开发模板**。基于 **Go 1.27 + React 19 + TypeSpec 契约优先 + PostgreSQL**，前端静态资源直接内嵌至单一 Go 二进制文件中，零 Node.js 运行时依赖，提供极致的开发体验（DX）与单文件极简交付。

## 核心特性

- **端到端强类型契约**：以 TypeSpec 作为 API 单一真理源，一处修改自动生成 OpenAPI 3.0 规范、Go 后端强类型接口模型与自动校验（`ogen`）、前端 React Query 客户端请求 Hooks（`Orval`）以及完全离线内嵌的交互式 Scalar 文档。
- **现代化前端 (React 19)**：基于 React 19、TypeScript、Vite、TanStack Router（类型安全代码级路由）、TanStack Query（异步数据流管理）、Tailwind CSS v4、shadcn/ui、React Hook Form 与 Zod。
- **高可靠模块化 Go 后端**：基于 Go 1.27 与 Chi 路由，采用清晰的模块化单体架构与 Context-Driven Transactor 事务模式，业务逻辑、数据访问与传输层严格解耦。
- **SQL 优先数据层**：PostgreSQL 配合 Goose 显式版本化数据库迁移，sqlc 在编译期对 SQL 进行类型验证并生成高性能 Go 数据访问代码，采用 pgx 原生连接池。
- **单二进制交付**：前端构建产物通过 `go:embed` 嵌入到单一二进制文件中，生产环境免配 Nginx 与 Node.js 运行时，配合轻量级多阶段 Docker 构建与 Docker Compose 即可一键拉起。
- **开箱即用的工程基线**：内置全套自动化质量门禁（`pnpm check`）、防漂移校验、CI/CD 自动化流水线、Dependabot 依赖自动更新及 Agent Skills 工作流支持。

## 从模板创建项目

在 GitHub 上选择 **Use this template** 并指定新仓库名。首次 `main` push 会触发一次性初始化工作流，根据 `owner/repository` 自动替换 Go module、pnpm package scope、Compose/image/database 标识和展示名称，然后自动清理初始化工作流。

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
| 协议与契约 | TypeSpec、OpenAPI 3.0、Scalar、ogen、Orval |
| 前端栈 | React 19、TypeScript、Vite、TanStack Query、TanStack Router、Tailwind CSS v4、shadcn/ui、Zod |
| 后端栈 | Go 1.27、Chi、Context-Driven Transactor |
| 数据持久层 | PostgreSQL 17、sqlc、pgx/v5、Goose |
| 工程与交付 | pnpm Workspaces、Go embed、Docker Buildx、Docker Compose、GitHub Actions |

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
