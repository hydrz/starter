import { Link } from "@tanstack/react-router";
import { BookOpen } from "lucide-react";
import { useGetHealth } from "../../api/generated/system/system";
import { GithubIcon } from "../../components/GithubIcon";
import * as m from "../../paraglide/messages";

export function LandingFooter() {
  const health = useGetHealth({
    query: {
      refetchInterval: 30_000,
      retry: 1,
    },
  });
  const serviceAvailable = Boolean(
    health.data?.data &&
    "status" in health.data.data &&
    health.data.data.status === "ok",
  );

  return (
    <footer className="border-t border-border/40 bg-background py-12 text-xs text-muted-foreground">
      <div className="mx-auto flex max-w-7xl flex-col items-center justify-between gap-6 px-4 sm:flex-row sm:px-6 lg:px-8">
        <div className="flex items-center gap-3">
          <span className="flex h-6 w-6 items-center justify-center rounded-full border border-border/80 bg-primary/10 text-[11px] font-bold text-accent-foreground dark:text-accent">
            E
          </span>
          <span className="font-semibold text-foreground">{m.app_brand()}</span>
          <span>{m.landing_footer_tagline()}</span>
        </div>

        <div className="flex flex-wrap items-center gap-6">
          <Link to="/app" className="transition-colors hover:text-foreground">
            {m.landing_footer_console()}
          </Link>
          <a
            href="/api/docs"
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1 transition-colors hover:text-foreground"
          >
            <span>{m.landing_footer_api_docs()}</span>
            <BookOpen size={11} />
          </a>
          <a
            href="https://github.com/hydrz/starter"
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1 transition-colors hover:text-foreground"
          >
            <span>GitHub</span>
            <GithubIcon size={12} />
          </a>
        </div>

        <div className="flex items-center gap-2">
          <span
            className={`h-2 w-2 rounded-full ${
              serviceAvailable ? "bg-success" : "bg-warning"
            }`}
          />
          <span>
            {serviceAvailable
              ? m.landing_footer_status_online()
              : m.landing_footer_status_checking()}
          </span>
        </div>
      </div>
    </footer>
  );
}
