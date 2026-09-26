# PostgreSQL 备份与恢复

- **状态**：Active
- **负责人**：Data Platform
- **最后复审**：2026-09-25
- **复审周期**：90 天

## 适用范围与安全要求

本手册描述 Compose 部署的逻辑备份流程。生产执行前必须确认目标环境、备份存储权限、可用空间、恢复时间目标和变更审批。示例文件可能包含敏感业务数据，必须加密存储并限制访问。

## 创建备份

```bash
mkdir -p backups
docker compose exec -T postgres \
  pg_dump --username="starter" --dbname="starter" --format=custom \
  > "backups/starter-$(date -u +%Y%m%dT%H%M%SZ).dump"
```

记录应用版本、最新 Goose migration 版本、数据库版本、文件大小和 SHA-256。将备份复制到 Compose 主机之外的受控存储，并按组织策略验证加密和保留期。

## 验证备份

不要把“命令成功退出”当作可恢复证明。至少在隔离数据库中执行：

```bash
pg_restore --list backups/<backup-file>.dump >/dev/null
sha256sum backups/<backup-file>.dump
```

定期进行完整恢复演练并验证关键记录数量和应用读取路径。

## 恢复前检查

1. 停止应用写入并记录事件开始时间；
2. 确认备份来源、SHA-256、数据库主版本和 migration 版本；
3. 对当前数据库再创建一份应急备份；
4. 优先恢复到新的数据库实例，验证后切换连接；
5. 明确失败时的回退目标和负责人。

## 恢复

以下命令会覆盖目标数据库对象，只能对已确认的隔离或恢复目标执行：

```bash
cat backups/<backup-file>.dump | docker compose exec -T postgres \
  pg_restore --username="starter" --dbname="starter" \
  --clean --if-exists --no-owner --no-privileges
```

恢复后运行 `pnpm db:status`。若备份 schema 早于当前应用要求，先评估再执行向前 migration；不要修改或回退已经发布的 migration 历史。

## 验证与升级

验证数据库日志、migration 状态、公告列表读取、创建/更新事务以及关键记录数量。发现版本不兼容、校验失败或数据缺失时立即停止切换，保留日志与恢复目标，并升级给 Data Platform。恢复完成后记录实际 RPO/RTO、执行者、校验结果和后续改进项。
