# 本地 PostgreSQL 恢复

- **状态**：Active
- **负责人**：Developer Experience
- **最后复审**：2026-09-25
- **复审周期**：90 天

## 适用场景

本手册只适用于个人开发环境中的 Compose PostgreSQL。不得将删除 volume 的步骤用于共享或生产数据库。

## 症状

- `pnpm db:up` 后健康检查持续失败；
- 应用报告无法连接默认的 `127.0.0.1:5432`；
- 本地 migration 历史已损坏，且数据无需保留。

## 诊断

```bash
docker compose ps
docker compose logs --tail=100 postgres
pnpm db:status
```

先检查端口占用、`.env` 中的本地凭据以及容器日志。需要保留本地数据时，在删除 volume 前停止并寻求数据库负责人协助。

## 恢复

确认本地数据可以永久删除后执行：

```bash
docker compose down --volumes
pnpm db:up
pnpm db:migrate
pnpm db:status
```

## 验证与升级

`pnpm db:status` 应显示全部 migration 已应用，随后运行 `pnpm test`。若全新 volume 仍无法通过健康检查，保留 `docker compose logs postgres` 输出并升级给 Developer Experience；不要反复删除数据掩盖镜像或 migration 问题。
