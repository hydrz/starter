/**
 * 预渲染静态页面（D5）：仅在构建期由 `scripts/prerender.mjs` 经 Vite SSR
 * 编译执行，用于生成公开路由的首屏正文；客户端加载后仍由现有 SPA 接管。
 * 独立于 `routes/` 下的交互组件，避免把路由器与查询 Provider 带进构建期。
 */

export function PrerenderedLandingPage() {
  return landingMarkup;
}

const landingMarkup = (
  <div className="min-h-screen bg-background text-foreground">
    <header className="border-b border-border/40 bg-background/80">
      <nav className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <a className="font-semibold text-foreground" href="/">
          Starter
        </a>
        <div className="flex items-center gap-4 text-sm text-muted-foreground">
          <a href="#features">核心能力</a>
          <a href="#architecture">契约架构</a>
          <a href="#quick-start">快速开始</a>
          <a className="font-medium text-foreground" href="/app">
            进入控制台
          </a>
        </div>
      </nav>
    </header>
    <main>
      <section className="mx-auto max-w-4xl px-4 py-24 text-center sm:px-6 lg:px-8">
        <p className="text-sm font-medium text-primary">Starter Kit</p>
        <h1 className="mt-4 text-4xl font-extrabold tracking-tight sm:text-6xl">
          从清晰契约到端到端全栈极速交付
        </h1>
        <p className="mx-auto mt-6 max-w-3xl text-lg leading-relaxed text-muted-foreground">
          面向现代 Web 平台与 SaaS 应用的生产级全栈开发模板。以 TypeSpec
          为单一契约源，自动生成 API、Go 后端与 React Query 强类型客户端。
        </p>
        <div className="mt-8 flex justify-center gap-3">
          <a
            className="rounded-lg bg-primary px-5 py-3 font-medium text-primary-foreground"
            href="/app"
          >
            进入 Starter 控制台
          </a>
          <a
            className="rounded-lg border border-border px-5 py-3 font-medium"
            href="#quick-start"
          >
            快速开始
          </a>
        </div>
      </section>
      <section
        id="features"
        className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8"
      >
        <h2 className="text-3xl font-bold">全栈核心能力与工程基线</h2>
        <div className="mt-8 grid gap-6 md:grid-cols-3">
          <article>
            <h3 className="font-semibold">契约驱动</h3>
            <p className="mt-2 text-muted-foreground">
              TypeSpec 驱动 OpenAPI 与两端代码生成。
            </p>
          </article>
          <article>
            <h3 className="font-semibold">模块化 Go 后端</h3>
            <p className="mt-2 text-muted-foreground">
              高并发、可测试且边界清晰的服务端架构。
            </p>
          </article>
          <article>
            <h3 className="font-semibold">单二进制交付</h3>
            <p className="mt-2 text-muted-foreground">
              前端静态资产嵌入 Go 二进制，运行时无需 Node.js。
            </p>
          </article>
        </div>
      </section>
      <section
        id="architecture"
        className="border-y border-border/40 bg-muted/30 px-4 py-16 sm:px-6 lg:px-8"
      >
        <div className="mx-auto max-w-7xl">
          <h2 className="text-3xl font-bold">契约优先的设计与编译管线</h2>
          <p className="mt-4 text-muted-foreground">
            TypeSpec、OpenAPI、Go、React Query 与 SQL
            在一条可验证的交付链路中协作。
          </p>
        </div>
      </section>
      <section
        id="quick-start"
        className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8"
      >
        <h2 className="text-3xl font-bold">快速开始</h2>
        <pre className="mt-6 overflow-x-auto rounded-lg bg-foreground p-5 text-sm text-background">
          <code>pnpm install{"\n"}pnpm dev</code>
        </pre>
      </section>
    </main>
  </div>
);

interface AuthPageProps {
  title: string;
  subtitle: string;
  submitLabel: string;
  footer: string;
  footerHref: string;
  footerLabel: string;
}

export function PrerenderedAuthPage({
  title,
  subtitle,
  submitLabel,
  footer,
  footerHref,
  footerLabel,
}: AuthPageProps) {
  return (
    <main className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-background px-4 py-12">
      <div
        className="pointer-events-none absolute inset-0 -z-10 bg-[linear-gradient(to_right,theme(colors.border/40)_1px,transparent_1px),linear-gradient(to_bottom,theme(colors.border/40)_1px,transparent_1px)] bg-[size:64px_64px]"
        aria-hidden="true"
      />
      <a
        href="/"
        className="mb-8 flex items-center gap-2.5 font-semibold text-foreground"
      >
        <span className="flex h-7 w-7 items-center justify-center rounded-full border border-border/80 bg-primary/10 text-xs font-bold text-accent-foreground">
          E
        </span>
        <span className="text-base font-semibold tracking-tight">Starter</span>
      </a>
      <section className="w-full max-w-sm rounded-2xl border border-border/80 bg-card p-7 text-card-foreground shadow-md">
        <header className="mb-6 text-center">
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            {title}
          </h1>
          <p className="mt-1.5 text-sm text-muted-foreground">{subtitle}</p>
        </header>
        <form className="grid gap-4" aria-label={title}>
          <label className="grid gap-1.5 text-sm font-medium text-foreground">
            邮箱
            <input
              className="h-10 rounded-md border border-input bg-background px-3"
              type="email"
              autoComplete="email"
            />
          </label>
          <label className="grid gap-1.5 text-sm font-medium text-foreground">
            密码
            <input
              className="h-10 rounded-md border border-input bg-background px-3"
              type="password"
              autoComplete="current-password"
            />
          </label>
          <button
            className="h-10 rounded-md bg-primary px-4 text-sm font-medium text-primary-foreground"
            type="submit"
          >
            {submitLabel}
          </button>
        </form>
      </section>
      <p className="mt-6 text-center text-sm text-muted-foreground">
        {footer}{" "}
        <a
          className="font-medium text-foreground hover:underline"
          href={footerHref}
        >
          {footerLabel}
        </a>
      </p>
    </main>
  );
}
