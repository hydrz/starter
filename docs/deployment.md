# 交付与运维指南

- **状态**：Active
- **负责人**：Platform Engineering
- **最后复审**：2026-09-27
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
2. 应用启动时自动执行嵌入的 Goose migrations（可通过 `AUTO_MIGRATE=false` 禁用）；
3. 应用通过自身的 `healthcheck` 子命令探测 readiness。

任何 migration 失败都会阻止应用启动。

## 配置

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `DATABASE_URL` | 本地开发连接串 | 数据库连接串（支持在 query 中微调连接池）；云数据库或生产环境直接注入 |
| `PORT` | `8080` | 服务内部监听端口（各云平台与容器标准变量） |
| `APP_PORT` | `8080` | Compose 宿主机发布端口（仅当宿主机 8080 冲突时指定） |
| `AUTO_MIGRATE` | `true` | 进程启动时是否自动执行待处理 migration（生产多副本集群推荐独立 `migrate` 一次性任务并设为 `false`，见下文“数据库迁移”） |
| `HTTP_ADDRESS` | 无 | 可选：完整监听地址（默认 `:PORT`） |
| `SHUTDOWN_TIMEOUT` | `10s` | 收到 SIGTERM/SIGINT 后等待在途 HTTP 请求、outbox worker 与连接池关闭的总预算 |

默认凭据只适用于本地。共享环境使用 secret manager 或受控环境注入，不将 `.env`、连接串或凭据提交到 Git。

## 健康语义

- `/api/healthz` 是 liveness，只表示 HTTP 进程能够响应；
- `/api/readyz` 是 readiness，会探测 PostgreSQL，依赖不可用时返回 `503 not_ready`；
- 未知 `/api/*` 返回 404，不进入 SPA fallback；
- 非 API 且没有扩展名的路径回退到 `index.html`，支持前端 history routing；
- 带扩展名但不存在的静态资源返回 404，带 hash 的 `/assets/*` 使用一年 immutable cache。

## 容器安全

运行镜像基于 `scratch`，只包含静态 Go 二进制与 CA 证书，并使用非 root UID/GID `65532`。Compose 进一步启用只读根文件系统、`no-new-privileges` 和受限 `/tmp` tmpfs。应用端口默认只绑定 loopback；公网入口应由受管理的 TLS reverse proxy 提供。

## 数据库迁移

迁移执行前会先在数据库上取得 PostgreSQL advisory lock（会话级、绑定专用连接），随后才开始运行 Goose。任何数量的应用副本同时执行 `AUTO_MIGRATE=true` 都是安全的：只有一个进程真正执行 migration，其余进程阻塞等待，等锁释放后看到 schema 已是最新，直接继续启动。

生产环境推荐流程仍然是**独立 `migrate` 一次性任务**，并把应用副本设为 `AUTO_MIGRATE=false`：

```bash
## 一次性迁移任务（发布流水线中执行，执行完即退出）
docker run --rm ghcr.io/<owner>/<repo>:vMAJOR.MINOR.PATCH /app/server migrate

## 应用副本保持 AUTO_MIGRATE=false，只消费已就绪的 schema
```

约定：

- 迁移只向前推进（[ADR-0002](adr/0002-forward-only-production-migrations.md)），生产禁止任何形式的 Goose `down`；
- `migrate` 子命令与非 `AUTO_MIGRATE` 副本共享同一连接串；任务失败即终止发布，不启动新版本副本；
- 执行迁移前确认已有备份（见 [PostgreSQL 备份与恢复 Runbook](runbooks/postgres-backup-restore.md)）。

## 优雅关停

进程收到 SIGTERM/SIGINT 后按固定顺序退出，总预算由 `SHUTDOWN_TIMEOUT`（默认 `10s`）控制：

1. **停止接收请求**：HTTP listener 立即关闭，就绪探针随之失败；
2. **等待在途 HTTP 请求**：`server.Shutdown` 等待活动请求完成或超时；
3. **停止作业/worker**：outbox worker 完成当前批次后退出，避免与仍在写库的请求竞争；
4. **关闭连接池**：以上全部静默后才释放数据库连接。

容器编排需给足宽限期：Compose 中应用的 `stop_grace_period: 30s` 已大于默认 `SHUTDOWN_TIMEOUT`；Kubernetes 等平台应保证 `terminationGracePeriodSeconds` ≥ `SHUTDOWN_TIMEOUT`。

## 环境分层与 staging

配置读取遵循固定顺序：进程环境变量覆盖 `.env` 文件，`.env` 缺失时回落到内置默认值。分层约定：

| 层 | 用途 | 变量来源 |
| --- | --- | --- |
| 本地开发 | `pnpm dev` / `pnpm db:*` | `.env`（从 `.env.example` 复制，全部为安全默认值） |
| staging | 预生产验证，连接独立数据库 | 编排平台的 secrets/环境注入，敏感值永不入库 |
| production | 面向真实流量 | 同 staging；仅额外启用支付、SMTP 等外部集成 |

staging 与 production 必须使用相同的镜像构建流程（同一 release workflow 产物），仅允许环境变量与外部资源不同；禁止在 staging 手工修补数据库 schema——任何变更必须走新 migration 进入该环境，保持与 production 的迁移历史一致。staging 数据库按与生产相同的备份策略定期快照。

## 镜像发布

推送 `vMAJOR.MINOR.PATCH` 标签会触发 [release workflow](../.github/workflows/release.yml)：

1. 运行 `pnpm check`、`pnpm test`、`pnpm build` 全量门禁；
2. 构建单架构候选镜像并使用 Trivy 扫描；存在 HIGH 或 CRITICAL 漏洞时**发布失败并阻断**；
3. 扫描通过后构建并推送 `linux/amd64` 与 `linux/arm64` 多架构镜像（附 SBOM 证明），并上传二进制与 SHA-256 校验和到 draft release。

镜像 digest 是部署的事实依据：staging 与 production 部署时引用 digest 而非 `latest`。

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
