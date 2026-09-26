import { Link } from "@tanstack/react-router";
import { ArrowRight, BookOpen } from "lucide-react";
import { GithubIcon } from "../../components/GithubIcon";
import { LanguageSwitcher } from "../../components/LanguageSwitcher";
import { ThemeToggle } from "../../components/ThemeToggle";
import * as m from "../../paraglide/messages";

export function LandingNavbar() {
  return (
    <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/80 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-8">
          <Link
            to="/"
            className="flex items-center gap-2.5 font-semibold text-foreground transition-opacity hover:opacity-90"
            aria-label={m.landing_nav_home_aria()}
          >
            <span className="flex h-7 w-7 items-center justify-center rounded-full border border-border/80 bg-primary/10 text-xs font-bold text-accent-foreground dark:text-accent">
              E
            </span>
            <span className="text-base tracking-tight font-semibold">
              {m.app_brand()}
            </span>
          </Link>

          <nav className="hidden md:flex items-center gap-6 text-xs font-medium text-muted-foreground">
            <a
              href="#features"
              className="transition-colors hover:text-foreground"
            >
              {m.landing_nav_features()}
            </a>
            <a
              href="#architecture"
              className="transition-colors hover:text-foreground"
            >
              {m.landing_nav_architecture()}
            </a>
            <a
              href="#tech-stack"
              className="transition-colors hover:text-foreground"
            >
              {m.landing_nav_tech_stack()}
            </a>
            <a
              href="#quick-start"
              className="transition-colors hover:text-foreground"
            >
              {m.landing_nav_quick_start()}
            </a>
            <a
              href="/api/docs"
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1 transition-colors hover:text-foreground"
            >
              <span>{m.landing_nav_api_docs()}</span>
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
            aria-label={m.landing_nav_github_aria()}
          >
            <GithubIcon size={15} />
          </a>

          <LanguageSwitcher className="hidden sm:inline-flex" />
          <ThemeToggle />

          <Link
            to="/app"
            className="inline-flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3.5 text-xs font-medium text-primary-foreground shadow-xs transition-colors hover:opacity-95"
          >
            <span>{m.landing_nav_console_cta()}</span>
            <ArrowRight size={13} />
          </Link>
        </div>
      </div>
    </header>
  );
}
