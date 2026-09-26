import { Link } from "@tanstack/react-router";
import { ArrowRight, BookOpen } from "lucide-react";
import { GithubIcon } from "../../components/GithubIcon";
import { ThemeToggle } from "../../components/ThemeToggle";

export function LandingNavbar() {
  return (
    <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-8">
          <Link
            to="/"
            className="flex items-center gap-2.5 font-semibold text-foreground transition-opacity hover:opacity-90"
            aria-label="Starter 首页"
          >
            <span className="flex h-7 w-7 items-center justify-center rounded-full border border-border/80 bg-primary/10 text-xs font-bold text-accent-foreground dark:text-accent">
              E
            </span>
            <span className="text-base tracking-tight font-semibold">
              Starter
            </span>
          </Link>

          <nav className="hidden md:flex items-center gap-6 text-xs font-medium text-muted-foreground">
            <a
              href="#features"
              className="transition-colors hover:text-foreground"
            >
              核心能力
            </a>
            <a
              href="#architecture"
              className="transition-colors hover:text-foreground"
            >
              契约架构
            </a>
            <a
              href="#tech-stack"
              className="transition-colors hover:text-foreground"
            >
              技术选型
            </a>
            <a
              href="#quick-start"
              className="transition-colors hover:text-foreground"
            >
              快速开始
            </a>
            <a
              href="/api/docs"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1 transition-colors hover:text-foreground"
            >
              <span>API 文档</span>
              <BookOpen size={12} />
            </a>
          </nav>
        </div>

        <div className="flex items-center gap-3">
          <a
            href="https://github.com/hydrz/starter"
            target="_blank"
            rel="noreferrer"
            className="hidden sm:flex h-8 w-8 items-center justify-center rounded-lg border border-border/60 text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label="GitHub 仓库"
          >
            <GithubIcon size={15} />
          </a>

          <ThemeToggle />

          <Link
            to="/app"
            className="inline-flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3.5 text-xs font-medium text-primary-foreground shadow-xs transition-colors hover:opacity-95"
          >
            <span>Starter 控制台</span>
            <ArrowRight size={13} />
          </Link>
        </div>
      </div>
    </header>
  );
}
