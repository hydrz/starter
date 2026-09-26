/* eslint-disable react-refresh/only-export-components */
import {
  Link,
  Navigate,
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  useRouterState,
} from "@tanstack/react-router";
import {
  Activity,
  ChevronLeft,
  ChevronRight,
  FileCode,
  Globe,
  LayoutDashboard,
  Megaphone,
} from "lucide-react";
import { useGetHealth } from "./api/generated/client";
import { ThemeToggle } from "./components/ThemeToggle";
import { UserAvatarMenu } from "./components/UserAvatarMenu";
import { Button } from "./components/ui/button";
import { Announcements } from "./features/announcements/Announcements";
import { LandingPage } from "./features/landing/LandingPage";
import { ObservabilityPage } from "./features/observability/ObservabilityPage";
import { useUIStore } from "./stores/ui-store";

const modules = [
  { name: "契约中心", detail: "TypeSpec 驱动的 API 设计", state: "已接入" },
  { name: "数据访问", detail: "PostgreSQL · sqlc · pgx", state: "已接入" },
  { name: "交付流水线", detail: "单二进制 · Docker Compose", state: "已接入" },
];

function AppLayout() {
  const { sidebarCollapsed, toggleSidebar } = useUIStore();
  const currentPath = useRouterState({
    select: (state) => state.location.pathname,
  });

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

  const getPageTitle = (path: string) => {
    if (path.includes("/announcements")) {
      return "公告通知";
    }
    if (path.includes("/observability")) {
      return "系统观测";
    }
    return "概览看板";
  };

  return (
    <div
      className="shell"
      style={{
        gridTemplateColumns: sidebarCollapsed ? "72px 1fr" : "240px 1fr",
      }}
    >
      <aside
        className="sidebar"
        style={{
          width: sidebarCollapsed ? "72px" : "240px",
          padding: sidebarCollapsed ? "24px 12px" : "24px 18px",
        }}
      >
        <div className="flex items-center justify-between pb-6">
          <Link
            to="/app/overview"
            className="flex items-center gap-2.5 font-semibold text-foreground transition-opacity hover:opacity-90"
            aria-label="Starter 控制台 首页"
          >
            <span className="brand-mark">E</span>
            {!sidebarCollapsed && (
              <span className="text-white font-semibold">Starter</span>
            )}
          </Link>
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleSidebar}
            aria-label={sidebarCollapsed ? "展开侧边栏" : "折叠侧边栏"}
            className="h-7 w-7 text-muted-foreground hover:text-white"
          >
            {sidebarCollapsed ? (
              <ChevronRight size={16} />
            ) : (
              <ChevronLeft size={16} />
            )}
          </Button>
        </div>

        <nav aria-label="控制台导航">
          <Link
            to="/app/overview"
            className={`nav-item ${
              currentPath === "/app" || currentPath === "/app/overview"
                ? "active"
                : ""
            }`}
          >
            <div className="flex items-center gap-2.5">
              <LayoutDashboard size={15} />
              {!sidebarCollapsed && <span>概览</span>}
            </div>
            {!sidebarCollapsed &&
              (currentPath === "/app" || currentPath === "/app/overview") && (
                <span className="nav-dot" />
              )}
          </Link>

          <Link
            to="/app/announcements"
            className={`nav-item ${
              currentPath === "/app/announcements" ? "active" : ""
            }`}
          >
            <div className="flex items-center gap-2.5">
              <Megaphone size={15} />
              {!sidebarCollapsed && <span>公告管理</span>}
            </div>
            {!sidebarCollapsed && currentPath === "/app/announcements" && (
              <span className="nav-dot" />
            )}
          </Link>

          <Link
            to="/app/observability"
            className={`nav-item ${
              currentPath === "/app/observability" ? "active" : ""
            }`}
          >
            <div className="flex items-center gap-2.5">
              <Activity size={15} />
              {!sidebarCollapsed && <span>系统观测</span>}
            </div>
            {!sidebarCollapsed && currentPath === "/app/observability" && (
              <span className="nav-dot" />
            )}
          </Link>

          <a
            className="nav-item"
            href="/api/docs"
            target="_blank"
            rel="noreferrer"
          >
            <div className="flex items-center gap-2.5">
              <FileCode size={15} />
              {!sidebarCollapsed && <span>API 契约</span>}
            </div>
          </a>

          <Link
            to="/"
            className="nav-item text-neutral-400 hover:text-white mt-4 border-t border-white/10 pt-4"
          >
            <div className="flex items-center gap-2.5">
              <Globe size={15} />
              {!sidebarCollapsed && <span>返回前台</span>}
            </div>
          </Link>
        </nav>

        <div className="sidebar-footer">
          <span className={serviceAvailable ? "pulse" : "pulse unavailable"} />
          {!sidebarCollapsed && (
            <span>{serviceAvailable ? "API 在线可用" : "等待服务就绪"}</span>
          )}
        </div>
      </aside>

      <main style={{ gridColumn: 2 }}>
        <header className="topbar">
          <div>
            <span className="eyebrow">STARTER · CONSOLE</span>
            <div className="flex items-center gap-2 mt-1">
              <h1 className="text-xl font-bold tracking-tight">
                Starter 控制台
              </h1>
              <span className="text-muted-foreground text-sm font-normal">
                /
              </span>
              <span className="text-sm font-medium text-muted-foreground">
                {getPageTitle(currentPath)}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <div className="hidden sm:flex items-center gap-2 rounded-full border border-border/60 bg-muted/40 px-3 py-1 text-xs text-muted-foreground">
              <span
                className={`h-2 w-2 rounded-full ${
                  serviceAvailable
                    ? "bg-emerald-500 animate-pulse"
                    : "bg-amber-500"
                }`}
              />
              <span>{serviceAvailable ? "API 服务就绪" : "正在连接服务"}</span>
            </div>

            <ThemeToggle />
            <UserAvatarMenu />
          </div>
        </header>

        <Outlet />
      </main>
    </div>
  );
}

function OverviewPage() {
  return (
    <>
      <section className="hero">
        <div>
          <p className="kicker">Full-Stack Starter Kit</p>
          <h2>从清晰的边界开始，持续交付可靠的软件。</h2>
          <p className="hero-copy">
            统一 Go 与 React
            的开发入口，为契约、数据和自动化流水线预留稳定边界。
          </p>
        </div>

        <div className="metric">
          <span>当前阶段</span>
          <strong>05</strong>
          <small>纵向业务切片</small>
        </div>
      </section>

      <section className="section" aria-labelledby="modules-title">
        <div className="section-heading">
          <div>
            <span className="eyebrow">SYSTEM FOUNDATION</span>
            <h3 id="modules-title">能力模块</h3>
          </div>
          <span className="section-meta">3 CORE DOMAINS</span>
        </div>

        <div className="cards">
          {modules.map((module, index) => (
            <article key={module.name} className="card">
              <span className="card-number">0{index + 1}</span>
              <div>
                <h4>{module.name}</h4>
                <p>{module.detail}</p>
              </div>
              <span className="card-state">{module.state}</span>
            </article>
          ))}
        </div>
      </section>

      <section className="workflow" aria-labelledby="workflow-title">
        <div className="section-heading">
          <div>
            <span className="eyebrow">DEVELOPMENT LOOP</span>
            <h3 id="workflow-title">研发循环</h3>
          </div>
          <span className="section-meta">REPEATABLE CADENCE</span>
        </div>

        <div className="workflow-row">
          <div className="workflow-step">
            <span>01</span>
            <strong>设计</strong>
          </div>
          <div className="workflow-step">
            <span>02</span>
            <strong>实现</strong>
          </div>
          <div className="workflow-step">
            <span>03</span>
            <strong>验证</strong>
          </div>
          <div className="workflow-step">
            <span>04</span>
            <strong>交付</strong>
          </div>
        </div>
      </section>
    </>
  );
}

const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

// Marketing Landing Page at "/"
const landingRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: LandingPage,
});

// Console Shell Route at "/app"
const appRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/app",
  component: AppLayout,
});

const appIndexRoute = createRoute({
  getParentRoute: () => appRoute,
  path: "/",
  component: OverviewPage,
});

const appOverviewRoute = createRoute({
  getParentRoute: () => appRoute,
  path: "/overview",
  component: OverviewPage,
});

const appAnnouncementsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: "/announcements",
  component: Announcements,
});

const appObservabilityRoute = createRoute({
  getParentRoute: () => appRoute,
  path: "/observability",
  component: ObservabilityPage,
});

// Backward-compatibility redirect: /announcements -> /app/announcements
const legacyAnnouncementsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/announcements",
  component: () => <Navigate to="/app/announcements" />,
});

export const routeTree = rootRoute.addChildren([
  landingRoute,
  appRoute.addChildren([
    appIndexRoute,
    appOverviewRoute,
    appAnnouncementsRoute,
    appObservabilityRoute,
  ]),
  legacyAnnouncementsRoute,
]);

export const router = createRouter({
  routeTree,
  defaultPreload: "intent",
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
