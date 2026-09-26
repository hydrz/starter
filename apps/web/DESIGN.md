# 前端设计系统（DESIGN.md）

本文件是 `apps/web` 的权威设计与前端架构规范。所有页面、组件与交互必须遵循本文件；本文件与代码不一致时，以本文件为准，发现不一致应修正代码或提交 PR 修订本文件（勿默默绕过）。

- **状态**：Active
- **负责人**：Product Web
- **最后复审**：2026-09-26

## 1. 设计语言：Cloudflare 式技术美学

参考 Cloudflare dash/marketing 站的整体气质，而非逐像素复刻：

- **Landing**：高对比度、开发者向文案，粗体大字号标题，充足的垂直节奏；顶部深色/浅色可切换导航；Hero 区域使用几何网格/等高线背景装饰而非摄影图；Feature 区块用图标 + 一句话卡片网格；下方放"架构图/工作原理"分区；CTA 按钮使用高饱和主色，配次要的 outline 按钮；页脚信息密集但分组清晰。
- **Dashboard**：左侧固定/可折叠侧边栏（分组导航 + 组织切换器置顶）；顶部面包屑 + 页面标题 + 操作区；内容区偏向信息密度（数据表格、统计卡片），不做花哨留白；ID、token、API key、时间戳一律等宽字体；空状态统一样式（图标 + 一句话 + 主操作按钮）；危险操作（删除组织、撤销 API key）需二次确认对话框。
- 两者共享同一套 design tokens 与组件库，不是两套视觉语言，只是信息密度不同。

## 2. Design Tokens

沿用现有 Tailwind v4 CSS-first token 架构（`apps/web/src/styles.css` 的 `@theme` + CSS 变量），**替换配色为 Cloudflare 式配色**，不改变 token 命名体系（`--color-primary`、`--color-background` 等），保持与现有 `components/ui/*` 的兼容性。

### 2.1 颜色

| Token                        | 浅色模式                    | 深色模式  | 用途                                 |
| ---------------------------- | --------------------------- | --------- | ------------------------------------ |
| `--color-background`         | `#FFFFFF`                   | `#0A0E11` | 页面底色                             |
| `--color-foreground`         | `#0B1418`                   | `#E9EDEF` | 正文文字                             |
| `--color-card`               | `#FFFFFF`                   | `#111820` | 卡片/面板底色                        |
| `--color-primary`            | `#F6821F`（Cloudflare 橙）  | `#FB9B3C` | 主操作、品牌强调                     |
| `--color-primary-foreground` | `#FFFFFF`                   | `#0A0E11` | 主按钮文字                           |
| `--color-secondary`          | `#0051C3`（深蓝，辅助强调） | `#3B82F6` | 次要强调、链接                       |
| `--color-muted`              | `#F4F6F7`                   | `#161D24` | 次级背景（侧边栏、表头）             |
| `--color-muted-foreground`   | `#5B6B73`                   | `#8B9BA3` | 次要文字                             |
| `--color-border`             | `#E2E8EA`                   | `#1F2932` | 分割线、输入框边框                   |
| `--color-destructive`        | `#D92D20`                   | `#F97066` | 危险操作                             |
| `--color-success`            | `#12B76A`                   | `#32D583` | 成功态（entitlement 生效、支付成功） |
| `--color-warning`            | `#F79009`                   | `#FDB022` | 警告态（试用期、待处理邀请）         |

深色模式通过 `:root[data-theme="dark"]` 与 `prefers-color-scheme` 双通道支持（沿用现有 `ThemeToggle`/`ui-store` 的实现方式，不重新发明主题切换机制）。

### 2.2 字体

- **界面正文**：`Inter`（已引入）为主，中文回退 `"Noto Sans SC"`；
- **等宽（ID/密钥/时间戳/代码）**：新增 `"JetBrains Mono", ui-monospace, monospace`，定义为 `--font-mono` token，`components/ui/code.tsx`（新增）统一消费；
- 字号阶梯沿用 Tailwind 默认 scale（`text-xs` 到 `text-5xl`），Landing 标题最大用到 `text-6xl`/`text-7xl`（响应式降级）。

### 2.3 间距与圆角

- 间距沿用 Tailwind 4px 网格，不新增自定义 spacing scale；
- 圆角：卡片/输入框 `--radius: 0.5rem`（8px），按钮 `0.375rem`（6px），徽章/标签 `9999px`（胶囊），与 Cloudflare 中等圆角风格一致（不用现在过大的圆角）。

## 3. 组件系统

继续 shadcn/ui 模式（Radix UI primitives + `class-variance-authority` + `tailwind-merge`，非独立组件库依赖），扩展 `apps/web/src/components/ui/` 现有的 `button/card/badge/input/textarea` 到完整集合：

`dialog`、`dropdown-menu`、`sheet`（移动端侧边导航/详情面板）、`tabs`、`table`、`select`、`checkbox`、`switch`、`radio-group`、`avatar`、`tooltip`、`toast`（sonner）、`skeleton`、`separator`、`form`（react-hook-form 封装，复用现有 `@hookform/resolvers` + `zod`）、`alert`、`progress`、`breadcrumb`、`pagination`、`code`（等宽文本 + 一键复制）。

业务级复合组件放在 `apps/web/src/components/layout/`（非 `ui/`，因为不是通用原语）：`AppShell`（侧边栏+顶栏骨架）、`OrgSwitcher`、`StatCard`、`EmptyState`、`DataTable`（基于 `ui/table` 的分页/排序封装）、`ConfirmDialog`（危险操作二次确认的统一封装，所有删除/撤销类操作必须复用，不得各自实现确认弹窗）。

新增依赖：`@radix-ui/react-*`（按需，dialog/dropdown-menu/tabs/select/checkbox/switch/radio-group/avatar/tooltip/separator）、`sonner`、`cmdk`（可选，用于后续命令面板，本轮不强制交付）。

## 4. 国际化：inlang + Paraglide JS

- 使用 `@inlang/paraglide-js` 作为编译期 i18n 方案（消息在构建期编译为类型安全的函数，零运行时 bundle 开销），项目元数据存于 `apps/web/project.inlang/settings.json`，消息文件 `apps/web/messages/{locale}.json`（`@inlang/plugin-message-format`）。
- 语言：`en`（源语言/base locale）与 `zh`（本仓库主要使用语言，作为第二语言收录，初期与 `en` 同步维护，不做机器翻译占位）。
- 策略：不使用 URL locale 前缀（`strategy: ["localStorage", "baseLocale"]`），避免与现有 `TanStack Router` 路由树产生地址级别的复杂度；语言切换入口放在顶部导航（Landing）与账户设置（Dashboard），持久化到 `localStorage`。
- 所有新增页面文案必须使用 Paraglide 生成的消息函数（`import * as m from "@/paraglide/messages"`），不得硬编码中/英文字符串；已有页面（announcements/observability/landing）本轮一并迁移。
- `pnpm --filter @starter/web check` 需新增 `pnpm --filter @starter/web paraglide:compile`（或等效构建期钩子）保证消息编译产物随构建生成，不手工维护生成目录（比照仓库其余"生成物不手改"的约定）。

## 5. 路由架构：迁移到 TanStack Router 文件路由

现状 `apps/web/src/router.tsx` 是手写的 `createRoute` 树，随着本轮补齐 20+ 页面会难以维护。改为 `@tanstack/router-plugin`（Vite 插件）驱动的文件路由：`apps/web/src/routes/` 目录，文件名即路由（`__root.tsx`、`index.tsx`、`sign-in.tsx`、`app/$orgSlug/index.tsx` 等），插件在构建期生成 `routeTree.gen.ts`（视为生成物，加入 `.gitignore`，不手动编辑）。这是本轮唯一的架构性变更；其余（TanStack Query、Zustand、react-hook-form、Orval 生成客户端）保持不变。

## 6. 信息架构 / 页面清单

### 6.1 公开路由（未登录）

| 路径                                             | 页面           | 说明                                                           |
| ------------------------------------------------ | -------------- | -------------------------------------------------------------- |
| `/`                                              | Landing        | 重写为 Cloudflare 式营销页                                     |
| `/sign-in`                                       | 登录           | 密码登录 + "使用邮箱验证码登录"入口 + Google/GitHub OAuth 按钮 |
| `/sign-up`                                       | 注册           | 邮箱+密码                                                      |
| `/forgot-password` / `/reset-password`           | 密码重置       | 两步：请求 → 凭 token 设置新密码                               |
| `/verify-email`                                  | 邮箱验证       | 凭 token 完成验证                                              |
| `/otp/verify`                                    | 邮箱验证码登录 | 配合 `/sign-in` 的"验证码登录"分支                             |
| `/mfa`                                           | 二次验证       | 主登录后要求 TOTP/恢复码，完成后签发会话                       |
| `/auth/callback/google`、`/auth/callback/github` | OAuth 回调     | 处理 code 交换与账号关联结果展示                               |
| `/invitations/accept`                            | 接受组织邀请   | 凭邀请 token                                                   |

### 6.2 已登录 Dashboard（`/app` 前缀，`AppShell` 布局）

| 路径                          | 页面              | 说明                                                      |
| ----------------------------- | ----------------- | --------------------------------------------------------- |
| `/app`                        | 组织选择/自动跳转 | 无组织时引导创建，有唯一组织自动进入                      |
| `/app/$orgSlug`               | 组织概览          | entitlement 状态卡、最近公告、快捷入口                    |
| `/app/$orgSlug/announcements` | 公告              | 迁移现有 `features/announcements`，套入新 Shell           |
| `/app/$orgSlug/members`       | 成员管理          | 列表、角色调整、移除                                      |
| `/app/$orgSlug/invitations`   | 邀请管理          | 发起、撤销、待处理列表                                    |
| `/app/$orgSlug/billing`       | 计费              | entitlement/订阅状态、发起 Checkout、Customer Portal 入口 |
| `/app/$orgSlug/settings`      | 组织设置          | 名称/slug、危险区（删除组织）                             |
| `/app/account`                | 账户资料          | 邮箱、修改密码                                            |
| `/app/account/sessions`       | 会话与 API Key    | 活跃会话列表+撤销、API Key 增删                           |
| `/app/account/security`       | 安全设置          | TOTP 启用/关闭+恢复码、Passkey 管理、已关联 OAuth 账号    |

### 6.3 状态页

`404`、`org-not-found`、`unauthorized`（403，如非成员访问组织路径）。

## 7. 目录结构约定

```text
apps/web/src/
  routes/              # 文件路由（新增，见第5节）
  components/ui/        # 通用原语（shadcn 风格，本文件第3节）
  components/layout/     # 业务级复合组件（AppShell/OrgSwitcher/DataTable…）
  features/<feature>/    # 按业务域拆分的页面级组件与本地状态
  lib/                  # 通用工具（现有 query.ts/errors.ts/utils.ts 不变）
  stores/               # zustand 全局状态（现有 ui-store.ts 不变）
  api/generated/         # Orval 生成物，不手改
  paraglide/             # inlang/Paraglide 编译产物，不手改，加入 .gitignore
messages/                # inlang 消息源文件（en.json / zh.json）
project.inlang/          # inlang 项目配置
```

## 8. 交互与可用性基线

- 所有表单：客户端 zod 校验 + 服务端错误映射到对应字段（复用统一 ApiError `{code, message}`，未知 `code` 落到表单级错误提示，不吞掉）；
- 所有异步操作按钮：loading 态禁用 + spinner，不允许重复提交；
- 所有列表：空状态、加载骨架（`ui/skeleton`）、错误态（复用 `AppErrorBoundary`/`lib/errors.ts`）三态齐全；
- 危险操作统一走 `ConfirmDialog`，二次确认文案需包含被操作对象名称（如"确认删除组织 <b>Acme</b>？"），不用通用"确定要删除吗？"；
- 键盘可达性：所有交互元素可 Tab 到达，Dialog/Sheet 打开时焦点陷入（Radix 默认行为，不手工重造）；
- 移动端：Dashboard 侧边栏在窄屏下收起为 `Sheet` 抽屉，Landing 全响应式。

## 9. 本轮交付范围与分期

1. **Stage 1 — 设计系统与基础设施**：token 重写、组件库扩容、inlang/Paraglide 接入、文件路由迁移、`AppShell`/`OrgSwitcher`/`DataTable`/`EmptyState`/`ConfirmDialog`。
2. **Stage 2 — 公开路由/认证流程**：Landing 重写 + 第 6.1 节全部页面。
3. **Stage 3 — Dashboard 核心**：组织概览/成员/邀请/设置，公告页迁移进新 Shell。
4. **Stage 4 — 账户安全与计费**：`/app/account/*` 全部页面 + `/app/$orgSlug/billing`。

各阶段独立提交、独立跑 `pnpm check`/`pnpm test`/`pnpm build`，避免中间状态破坏现有 `pnpm test:embed`（生产 embed 验证）。
