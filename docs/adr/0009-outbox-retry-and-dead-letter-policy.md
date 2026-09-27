# ADR-0009：Outbox 重试与死信策略

- 状态：已接受
- 日期：2026-09-27
- 决策者：Architecture
- 被替代：无

## 背景

[ADR-0007](0007-postgres-outbox-and-in-process-worker.md) 确立了 PostgreSQL outbox 与进程内 worker。早期实现把"确认 / 重试"职责放在消费者（`notification.Service`）中：每个消费者自行调用 `MarkProcessed` / `Retry`、自行计算退避，且失败的事件会无限重试，没有任何终止状态。多个消费者接入后，重试逻辑必然重复实现且行为不一致；未知 topic 的事件则永远滞留在队列中被反复 claim。

## 决策

把 outbox 事件的生命周期管理上移到 `delivery.Worker`，消费者只表达业务结果：

- **Dispatcher 注册制**：`delivery.Dispatcher` 按 topic 注册 `Handler`。Worker 只依赖 Dispatcher；`notification.Service` 等消费者在启动时按 topic 注册，不再参与任何 outbox 状态转换。
- **成功即确认**：handler 返回 nil，Worker 立即 `MarkProcessed`。
- **可重试失败走指数退避**：handler 返回错误时按基础 5s、上限 1h 的指数退避加全抖动（full jitter）`Retry`；`attempts` 达到上限（默认 10，`WORKER_MAX_ATTEMPTS` 可配）的事件进入死信。
- **未知 topic 直接死信**：未注册 topic 的事件不重试，立即死信并保留 `last_error` 供排查。
- **死信是终态**：`outbox_events.dead_lettered_at` 非空的事件被 claim 查询的部分索引排除，永不再被领取；死信处理（人工重放、告警）由后续可观测性任务承接。

## 备选方案

### 各消费者自带重试策略

实现简单、初期迁移少，但策略漂移不可避免，且无法统一设置重试上限，未选择。

### 独立死信队列表

便于集中管理与重放，但增加一次事务写入与两套状态的一致性问题；`dead_lettered_at` 加 `last_error` 已满足当前排查需求，未选择。

## 结果

所有消费者获得一致的重试与终止语义，事件不再无限重试；代价是消费者失去了对自身 topic 的重试节奏控制——需要定制节奏的 topic 将来可在 Handler 契约上扩展。死信数量目前只能通过数据库查询观测，告警接入依赖 F5 可观测性任务。
