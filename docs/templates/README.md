# 文档模板

模板用于保持必要信息一致，而不是要求所有文档拥有相同章节。

## 模板与保存位置映射

从模板创建新文档时，**不得在 `docs/templates/` 原地编辑**，必须将模板复制到对应的目标目录并按规范重命名，填写元数据并在索引中登记：

| 模板 | 目标保存路径与命名规则 | 对应登记索引 | 适用生命周期状态 |
| --- | --- | --- | --- |
| [ADR 模板](adr.md) | `docs/adr/NNNN-<kebab-case-title>.md`<br>（如 `docs/adr/0003-multi-tenant-isolation.md`，使用 4 位递增编号） | [`docs/adr/README.md`](../adr/README.md) | `提议`、`已接受`、`被替代`、`已拒绝`、`已废弃` |
| [功能开发指南模板](feature-guide.md) | `docs/development/<kebab-case-name>.md`<br>（如 `docs/development/audit-logging.md`） | [`docs/documentation-catalog.md`](../documentation-catalog.md) 与 [`docs/README.md`](../README.md) | `Draft`、`Active`、`Superseded`、`Deprecated` |
| [Runbook 模板](runbook.md) | `docs/runbooks/<kebab-case-name>.md`<br>（如 `docs/runbooks/redis-failover.md`） | [`docs/runbooks/README.md`](../runbooks/README.md) 与 [`docs/documentation-catalog.md`](../documentation-catalog.md) | `Draft`、`Active`、`Superseded`、`Deprecated` |
| [发布说明模板](release-notes.md) | GitHub Release 描述，或归档于 `docs/releases/v<MAJOR>.<MINOR>.<PATCH>.md` | GitHub Releases 与 [交付与运维指南](../deployment.md) | 每次发布生成 |

## 使用步骤

1. **复制模板**：从 `docs/templates/` 复制对应模板文件至目标路径，按小写 `kebab-case` 规范命名（ADR 必须带 4 位递增序号前缀）；
2. **清除占位符**：删除文档中所有中括号 `[...]` 提示说明，替换为实际内容；
3. **补充必需元数据**：按[文档规范与生命周期](../documentation-policy.md)填写状态、负责人、最后复审日期和复审周期；
4. **登记索引**：在上方映射表对应的索引文件（如 `docs/adr/README.md`、`docs/runbooks/README.md`、`docs/documentation-catalog.md`）中追加新条目；
5. **门禁验证**：运行 `pnpm check:docs` 确保文档只有一个一级标题且相对链接均有效。

模板自身保持通用性，不直接参与文档生命周期流转。
