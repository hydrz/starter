# Agent Skills 规范与实现

- **状态**：Active
- **负责人**：Developer Experience
- **最后复审**：2026-09-25
- **复审周期**：90 天

本文冻结仓库级 Agent Skills 的位置、职责和质量标准，并记录当前已交付的 Skill 集合。

## 1. 结论

仓库共享的 Skill 必须放在仓库根目录的 `.agents/skills/`，而不是 `skills/`、`.codex/skills/` 或 `docs/skills/`：

```text
.agents/
└── skills/
    └── <skill-name>/
        ├── SKILL.md              # 必需：触发描述与工作流
        ├── agents/
        │   └── openai.yaml       # 可选：展示、调用策略和工具依赖
        ├── scripts/              # 可选：确定性、可重复执行的脚本
        ├── references/           # 可选：按需读取的规范与背景资料
        └── assets/               # 可选：要复制或变换的模板、静态资源
```

Codex 会从当前工作目录向仓库根目录逐级扫描 `.agents/skills`。因此，本仓库默认只在根目录建立一套团队共享 Skill；只有当子模块存在明确不同的触发范围时，才在子目录增加同名层级。

用户个人、跨仓库复用的 Skill 应放在 `$HOME/.agents/skills/`，不提交到本仓库。需要跨仓库分发多个 Skills、MCP 连接或 hooks 时，应升级为 Plugin，而不是把通用工具强行放进项目仓库。

## 2. Skill 与其他机制的边界

| 机制 | 应承载的内容 | 不应承载的内容 |
| --- | --- | --- |
| `AGENTS.md` | 长期有效的仓库约束、编码规则、验证命令 | 某一类任务的完整操作手册 |
| Agent Skill | 可重复任务的步骤、决策点、输入输出和安全边界 | 所有任务都必须遵守的全局规则 |
| pnpm script / shell / Go 工具 | 确定性执行、生成、检查与自动修复 | 需要语义判断的流程说明 |
| MCP server | 实时数据、鉴权、授权和受控外部操作 | 项目工作流与评审方法 |
| Plugin | 跨仓库分发的一组 Skills、MCP、hooks 与资源 | 仅适用于当前仓库的轻量流程 |

本仓库的 Skills 只编排现有 SSOT、文档和脚本，不复制实现逻辑。例如，契约 Skill 应调用 `pnpm generate` 和 `pnpm check:generated`，而不是在 Skill 内重新实现代码生成。

## 3. `SKILL.md` 约定

每个 Skill 是一个独立目录，并且必须包含大写文件名 `SKILL.md`。本仓库采用比通用规范更严格的最小 frontmatter：

```markdown
---
name: change-api-contract
description: Update TypeSpec API contracts and regenerate verified Go and TypeScript clients when an API shape or endpoint changes.
---

Follow these steps ...
```

### 3.1 名称

- 使用小写字母、数字和连字符；目录名必须与 `name` 一致；
- 名称应体现一个明确动作，例如 `change-api-contract`，避免 `platform-helper` 之类宽泛名称；
- 同一发现范围内不得复用 `name`。Codex 不会合并同名 Skills，重复命名会导致调用歧义。

### 3.2 描述与触发

- `description` 是隐式触发的主要依据，必须同时写清“做什么”和“何时使用”；
- 把核心用例和触发词放在描述开头，保持简洁，避免因 Skill 较多、描述被截断而失去辨识度；
- 应写清边界，避免多个 Skills 同时匹配同一请求；
- 高风险或只允许显式执行的 Skill，可在 `agents/openai.yaml` 设置 `policy.allow_implicit_invocation: false`。

### 3.3 正文

- 使用祈使句，明确输入、前置条件、步骤、输出与成功标准；
- 明确哪些事实不得推断，以及何时询问、停止或拒绝；
- 显式保留用户指令优先级，不让 Skill 覆盖用户当前任务中的明确约束；
- 只记录 Agent 需要的非显然知识；已有规范使用链接引用，避免复制后漂移；
- 正文保持短小。大量策略、示例或 schema 下沉至 `references/`，并注明何时读取。

## 4. 渐进式披露与辅助文件

Skill 使用三层信息加载：

1. 启动时只暴露 `name`、`description` 和路径；
2. 触发后读取完整 `SKILL.md`；
3. 工作流确有需要时才读取 `references/`、执行 `scripts/` 或使用 `assets/`。

因此应遵守：

- `references/` 与 `SKILL.md` 保持一层引用关系，不构造多层跳转；
- 长参考文件提供目录，多个 Skills 共用的事实优先链接现有 `docs/`；
- 仅在必须保证确定性或避免重复编写代码时增加 `scripts/`，并单独测试脚本；
- `assets/` 只存放输出会使用的模板或资源，不作为说明文档目录；
- Skill 目录内不增加 `README.md`、变更日志、安装说明等冗余文件。

`agents/openai.yaml` 不是必需文件。只有需要用户界面元数据、关闭隐式调用或声明 MCP 依赖时才创建；它不能代替 `SKILL.md` 中清晰的工作流说明。

## 5. 安全与可审查性

每个 Skill 必须定义以下安全边界：

- 先读取仓库状态和适用规范，不覆盖用户未提交的修改；
- 生成物必须由 SSOT 和受版本约束的工具产生，禁止手改生成文件；
- destructive migration、发布、回滚、删除数据和外部写操作必须设置确认点；
- 不把密钥、令牌、连接串或生产数据写入 Skill、示例或日志；
- 不绕过 CI、分支保护、CODEOWNERS 或人工审批；
- 失败时保留诊断信息，区分代码错误与环境限制，不伪报验证结果。

Skill 是“如何执行”的指导层，不是新的权限边界。鉴权、授权和外部操作约束仍应由应用、CI、MCP 或平台策略强制执行。

## 6. 创建与验证流程

每个 Skill 按以下顺序交付：

1. 收集 3～5 个真实触发示例以及 2～3 个不应触发的反例；
2. 固定一个任务目标、输入、输出和停止条件；
3. 使用 `$skill-creator` 初始化目录和最小 `SKILL.md`；
4. 复用仓库现有文档与命令，仅按需增加辅助文件；
5. 校验目录、frontmatter、链接和脚本；
6. 用真实请求做正向、反向及边界测试，检查触发精度和执行结果；
7. 在独立上下文中进行一次前向验证，确保 Skill 不依赖创建过程中的隐含信息；
8. 通过代码评审后提交，并随底层流程变化同步维护。

验收至少覆盖：

- **结构**：位置、命名、frontmatter 和引用有效；
- **触发**：该触发时可命中，不相关请求不误触发；
- **流程**：按正确顺序调用仓库命令并遵守确认点；
- **结果**：产生预期文件或评审结果，且质量门禁通过；
- **失败模式**：缺少依赖、环境不可用或输入不足时能安全停止。

## 7. 已实现的 Skills

仓库已交付五个相互边界清晰的 Skills：

1. `change-api-contract`：TypeSpec → OpenAPI → Go/TypeScript 生成物与漂移检查；
2. `change-database-schema`：Goose → SQL 查询 → sqlc，包含 migration 安全检查；
3. `build-vertical-slice`：编排前两者并实现后端、前端与测试；
4. `review-change`：基于仓库规范执行分层评审与质量门禁；
5. `prepare-release`：构建、制品验证和发布准备；生产发布保持人工确认。

每个 Skill 都在 `.agents/skills/<name>/SKILL.md` 中定义触发边界、工作流、停止条件和成功标准，并在 `agents/openai.yaml` 中提供人类可读的展示元数据。`pnpm check:skills` 会校验目录名、frontmatter、描述、一级标题、未完成标记与展示元数据，并已纳入 `pnpm check`。

## 8. 依据

- [OpenAI Codex Skills 文档](https://developers.openai.com/codex/skills/)
- [Agent Skills Specification](https://agentskills.io/specification)
- [OpenAI Skills 示例仓库](https://github.com/openai/skills)
