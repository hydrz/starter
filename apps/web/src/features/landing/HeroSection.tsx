import { Link } from "@tanstack/react-router";
import {
  ArrowRight,
  BookOpen,
  CheckCircle2,
  Sparkles,
  Terminal,
} from "lucide-react";

export function HeroSection() {
  return (
    <section className="relative overflow-hidden pt-12 pb-20 md:pt-20 md:pb-28">
      {/* Background Glow Effect */}
      <div
        className="pointer-events-none absolute -top-40 left-1/2 -z-10 h-[500px] w-[800px] -translate-x-1/2 rounded-full bg-gradient-to-tr from-accent/20 to-primary/10 blur-3xl"
        aria-hidden="true"
      />

      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-3xl text-center">
          {/* Release Badge */}
          <div className="inline-flex items-center gap-2 rounded-full border border-border/80 bg-muted/60 px-3.5 py-1 text-xs text-muted-foreground backdrop-blur-xs mb-8">
            <Sparkles
              size={13}
              className="text-accent-foreground dark:text-accent"
            />
            <span className="font-medium text-foreground">Starter Kit</span>
            <span className="text-border">|</span>
            <span>Go 1.27 · React 19 · TypeSpec · Single Binary</span>
          </div>

          {/* Main Headline */}
          <h1 className="text-4xl font-extrabold tracking-tight text-foreground sm:text-5xl md:text-6xl text-balance">
            从清晰契约到
            <span className="block mt-1 bg-gradient-to-r from-emerald-600 via-teal-600 to-lime-600 dark:from-emerald-400 dark:via-teal-300 dark:to-lime-400 bg-clip-text text-transparent">
              端到端全栈极速交付
            </span>
          </h1>

          {/* Subtitle */}
          <p className="mt-6 text-base sm:text-lg leading-relaxed text-muted-foreground text-balance">
            面向现代 Web 平台与 SaaS 应用的生产级全栈开发模板。以 TypeSpec
            为单一契约源，编译期自动生成 OpenAPI、Go 后端与 React Query
            强类型客户端，前端产物直接嵌入单二进制，零 Node.js 运行时依赖。
          </p>

          {/* CTAs */}
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <Link
              to="/app"
              className="inline-flex h-11 items-center gap-2 rounded-lg bg-primary px-5 text-sm font-medium text-primary-foreground shadow-sm transition-all hover:opacity-95 hover:shadow-md"
            >
              <span>进入 Starter 控制台</span>
              <ArrowRight size={15} />
            </Link>

            <a
              href="#quick-start"
              className="inline-flex h-11 items-center gap-2 rounded-lg border border-border bg-background px-5 text-sm font-medium text-foreground shadow-xs transition-colors hover:bg-muted"
            >
              <Terminal size={15} className="text-muted-foreground" />
              <span>快速开始</span>
            </a>

            <a
              href="/api/docs"
              target="_blank"
              rel="noreferrer"
              className="inline-flex h-11 items-center gap-2 rounded-lg border border-border/60 bg-transparent px-4 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
            >
              <BookOpen size={15} />
              <span>Scalar API 交互文档</span>
            </a>
          </div>

          {/* Core Guarantees Checklist */}
          <div className="mt-12 flex flex-wrap items-center justify-center gap-6 text-xs text-muted-foreground">
            <div className="flex items-center gap-1.5">
              <CheckCircle2 size={14} className="text-emerald-500" />
              <span>全链路编译期类型安全</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 size={14} className="text-emerald-500" />
              <span>单二进制嵌入免配置交付</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 size={14} className="text-emerald-500" />
              <span>SQL-First 显式数据流</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
