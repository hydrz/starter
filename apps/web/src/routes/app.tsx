import {
  Outlet,
  createFileRoute,
  useRouterState,
} from "@tanstack/react-router";
import {
  Activity,
  FileCode,
  Globe,
  LayoutDashboard,
  Megaphone,
} from "lucide-react";
import { AppShell, type AppNavItem } from "../components/layout/AppShell";
import { useGetHealth } from "../api/generated/system/system";
import * as m from "../paraglide/messages";

export const Route = createFileRoute("/app")({
  component: AppLayoutRoute,
});

function getNavItems(): Array<AppNavItem> {
  return [
    { label: m.nav_overview(), to: "/app/overview", icon: LayoutDashboard },
    {
      label: m.nav_announcements(),
      to: "/app/announcements",
      icon: Megaphone,
    },
    {
      label: m.nav_observability(),
      to: "/app/observability",
      icon: Activity,
    },
    {
      label: m.nav_api_docs(),
      to: "/api/docs",
      icon: FileCode,
      external: true,
    },
  ];
}

function getSecondaryNavItems(): Array<AppNavItem> {
  return [
    { label: m.nav_back_to_landing(), to: "/", icon: Globe, exact: true },
  ];
}

function getPageTitle(path: string) {
  if (path.includes("/announcements")) {
    return m.console_page_announcements();
  }
  if (path.includes("/observability")) {
    return m.console_page_observability();
  }
  return m.console_page_overview();
}

function ServiceStatusBadge() {
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
    <div className="hidden sm:flex items-center gap-2 rounded-full border border-border/60 bg-muted/40 px-3 py-1 text-xs text-muted-foreground">
      <span
        className={`h-2 w-2 rounded-full ${
          serviceAvailable ? "bg-emerald-500 animate-pulse" : "bg-amber-500"
        }`}
      />
      <span>
        {serviceAvailable ? m.status_api_online() : m.status_api_connecting()}
      </span>
    </div>
  );
}

function AppLayoutRoute() {
  const currentPath = useRouterState({
    select: (state) => state.location.pathname,
  });

  return (
    <AppShell
      navItems={getNavItems()}
      secondaryNavItems={getSecondaryNavItems()}
      pageTitle={getPageTitle(currentPath)}
      headerActions={<ServiceStatusBadge />}
    >
      <Outlet />
    </AppShell>
  );
}
