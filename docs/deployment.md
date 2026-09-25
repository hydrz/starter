# 交付与运维指南

- **状态**：Active
- **负责人**：Platform Engineering
- **最后复审**：2026-09-25
- **复审周期**：90 天

## 制品模型

生产构建先运行 Vite，再使用 `production` build tag 将 `internal/platform/webui/dist` 嵌入 Go。最终运行时不需要 Node.js，也不需要单独的静态文件目录。

```bash
pnpm build
./dist/server
```

默认 Go 构建使用轻量开发占位页面，以便单元测试不依赖预先生成的前端目录；生产二进制必须通过根 `pnpm build` 或 Dockerfile 构建。`pnpm test:embed` 会使用真实 Vite 产物验证生产 embed 和 SPA fallback。

## Compose 部署顺序

```bash
cp .env.example .env
docker compose build
docker compose up --detach
docker compose ps
```

Compose 按以下顺序运行：

1. PostgreSQL 通过 `pg_isready`；
2. 同一应用镜像以 `migrate` 子命令执行嵌入的 Goose migrations；
3. migration 成功退出后启动应用；
4. 应用通过自身的 `healthcheck` 子命令探测 readiness。

任何 migration 失败都会阻止应用启动。不要通过跳过 migration job 强行启动依赖新 schema 的版本。

## 配置

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `DATABASE_URL` | 本地开发连接串 | 应用与 migration 数据库连接；生产环境必须注入 |
| `HTTP_ADDRESS` | `:8080` | 进程监听地址 |
| `HEALTHCHECK_URL` | `http://127.0.0.1:8080/api/readyz` | 二进制 healthcheck 子命令目标 |
| `APP_PORT` | `8080` | Compose 主机绑定端口 |
| `APP_VERSION` | `dev` | Compose 镜像标签和构建版本 |
| `POSTGRES_*` | `enterprise` 本地值 | Compose PostgreSQL 初始化参数 |

默认凭据只适用于本地。共享环境使用 secret manager 或受控环境注入，不将 `.env`、连接串或凭据提交到 Git。

## 健康语义

- `/api/healthz` 是 liveness，只表示 HTTP 进程能够响应；
- `/api/readyz` 是 readiness，会探测 PostgreSQL，依赖不可用时返回 `503 not_ready`；
- 未知 `/api/*` 返回 404，不进入 SPA fallback；
- 非 API 且没有扩展名的路径回退到 `index.html`，支持前端 history routing；
- 带扩展名但不存在的静态资源返回 404，带 hash 的 `/assets/*` 使用一年 immutable cache。

## 容器安全

运行镜像基于 `scratch`，只包含静态 Go 二进制与 CA 证书，并使用非 root UID/GID `65532`。Compose 进一步启用只读根文件系统、`no-new-privileges` 和受限 `/tmp` tmpfs。应用端口默认只绑定 loopback；公网入口应由受管理的 TLS reverse proxy 提供。

## 日志与版本

应用向 stdout 输出 JSON 日志，容器平台负责采集、保留和脱敏。构建时注入 version、commit 和 build date，并在启动日志输出。不得记录数据库连接串、请求凭据、公告正文或个人信息。

## 发布

`vMAJOR.MINOR.PATCH` tag 触发发布工作流，产出：

- Linux amd64 单二进制；
- SHA-256 校验文件；
- `ghcr.io/<owner>/<repo>:vMAJOR.MINOR.PATCH` 镜像；
- 同版本的 `latest` 镜像；
- 等待人工确认的 draft GitHub Release。

正式发布前核对 release notes、migration、备份状态和镜像 digest。当前工作流不自动部署环境。

## 停止和恢复

```bash
docker compose down
```

该命令保留 PostgreSQL volume。禁止在共享环境运行 `docker compose down --volumes`。数据库备份与恢复遵循[PostgreSQL 备份与恢复 Runbook](runbooks/postgres-backup-restore.md)。
