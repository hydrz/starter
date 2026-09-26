import { Link } from "@tanstack/react-router";
import {
  ArrowRight,
  BookOpen,
  CheckCircle2,
  Sparkles,
  Terminal,
} from "lucide-react";
import * as m from "../../paraglide/messages";

export function HeroSection() {
  return (
    <section className="relative overflow-hidden pt-12 pb-20 md:pt-20 md:pb-28">
      {/* 几何网格背景装饰（DESIGN.md §1：非摄影图） */}
      <div
        className="pointer-events-none absolute inset-0 -z-20 bg-[linear-gradient(to_right,theme(colors.border/50)_1px,transparent_1px),linear-gradient(to_bottom,theme(colors.border/50)_1px,transparent_1px)] bg-[size:56px_56px] [mask-image:radial-gradient(ellipse_65%_55%_at_50%_0%,black_20%,transparent_75%)]"
        aria-hidden="true"
      />
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
            <span className="font-medium text-foreground">
              {m.landing_hero_badge()}
            </span>
            <span className="text-border">|</span>
            <span>{m.landing_hero_badge_stack()}</span>
          </div>

          {/* Main Headline */}
          <h1 className="text-4xl font-extrabold tracking-tight text-foreground sm:text-5xl md:text-6xl text-balance">
            {m.landing_hero_title_line1()}
            <span className="block mt-1 bg-gradient-to-r from-primary via-orange-500 to-secondary bg-clip-text text-transparent">
              {m.landing_hero_title_line2()}
            </span>
          </h1>

          {/* Subtitle */}
          <p className="mt-6 text-base sm:text-lg leading-relaxed text-muted-foreground text-balance">
            {m.landing_hero_subtitle()}
          </p>

          {/* CTAs */}
          <div className="mt-8 flex flex-wrap items-center justify-center gap-3">
            <Link
              to="/app"
              className="inline-flex h-11 items-center gap-2 rounded-lg bg-primary px-5 text-sm font-medium text-primary-foreground shadow-sm transition-all hover:opacity-95 hover:shadow-md"
            >
              <span>{m.landing_hero_cta_console()}</span>
              <ArrowRight size={15} />
            </Link>

            <a
              href="#quick-start"
              className="inline-flex h-11 items-center gap-2 rounded-lg border border-border bg-background px-5 text-sm font-medium text-foreground shadow-xs transition-colors hover:bg-muted"
            >
              <Terminal size={15} className="text-muted-foreground" />
              <span>{m.landing_hero_cta_quickstart()}</span>
            </a>

            <a
              href="/api/docs"
              target="_blank"
              rel="noreferrer"
              className="inline-flex h-11 items-center gap-2 rounded-lg border border-border/60 bg-transparent px-4 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
            >
              <BookOpen size={15} />
              <span>{m.landing_hero_cta_api_docs()}</span>
            </a>
          </div>

          {/* Core Guarantees Checklist */}
          <div className="mt-12 flex flex-wrap items-center justify-center gap-6 text-xs text-muted-foreground">
            <div className="flex items-center gap-1.5">
              <CheckCircle2 size={14} className="text-success" />
              <span>{m.landing_hero_check_types()}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 size={14} className="text-success" />
              <span>{m.landing_hero_check_binary()}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 size={14} className="text-success" />
              <span>{m.landing_hero_check_sql()}</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
