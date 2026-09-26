/* eslint-disable react-refresh/only-export-components */
import {
  Link,
  Outlet,
  createRootRoute,
  createRoute,
  createRouter,
  useRouterState,
} from "@tanstack/react-router";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useGetHealth } from "./api/generated/client";
import { Button } from "./components/ui/button";
import { Announcements } from "./features/announcements/Announcements";
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
  const serviceAvailable = health.data?.data?.status === "ok";

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
          padding: sidebarCollapsed ? "28px 12px" : "28px 22px",
        }}
      >
        <div className="flex items-center justify-between pb-8">
          <Link to="/" className="brand !p-0" aria-label="Starter Console 首页">
            <span className="brand-mark">E</span>
            {!sidebarCollapsed && <span>Starter</span>}
          </Link>
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleSidebar}
            aria-label={sidebarCollapsed ? "展开侧边栏" : "折叠侧边栏"}
            className="h-7 w-7 text-muted-foreground hover:text-foreground"
          >
            {sidebarCollapsed ? (
              <ChevronRight size={16} />
            ) : (
              <ChevronLeft size={16} />
            )}
          </Button>
        </div>

        <nav aria-label="主导航">
          <Link
            to="/"
            className={`nav-item ${currentPath === "/" || currentPath === "/overview" ? "active" : ""}`}
          >
            <span>概览</span>
            {(currentPath === "/" || currentPath === "/overview") && (
              <span className="nav-dot" />
            )}
          </Link>
          <Link
            to="/announcements"
            className={`nav-item ${currentPath === "/announcements" ? "active" : ""}`}
          >
            <span>公告</span>
            {currentPath === "/announcements" && <span className="nav-dot" />}
          </Link>
          <a
            className="nav-item"
            href="/api/docs"
            target="_blank"
            rel="noreferrer"
          >
            <span>API 文档</span>
          </a>
        </nav>

        <div className="sidebar-footer">
          <span className={serviceAvailable ? "pulse" : "pulse unavailable"} />
          {!sidebarCollapsed && (
            <span>
              {serviceAvailable ? "API 服务运行正常" : "正在连接 API 服务"}
            </span>
          )}
        </div>
      </aside>

      <main style={{ gridColumn: 2 }}>
        <header className="topbar">
          <div>
            <span className="eyebrow">FULL-STACK STARTER KIT</span>
            <h1>Starter 控制台</h1>
          </div>
          <span className="phase">Phase 05</span>
        </header>

        <Outlet />
      </main>
    </div>
  );
}

function OverviewPage() {
  return (
    <>
      <section className="hero" id="overview">
        <div>
          <p className="kicker">Foundation established</p>
          <h2>
            从清晰的边界开始，
            <br />
            持续交付可靠的软件。
          </h2>
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

      <section className="section" id="architecture">
        <div className="section-heading">
          <div>
            <span className="eyebrow">SYSTEM FOUNDATION</span>
            <h3>能力模块</h3>
          </div>
          <span className="section-meta">3 COMPLETED</span>
        </div>
        <div className="cards">
          {modules.map((module, index) => (
            <article className="card" key={module.name}>
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

      <section className="workflow" id="workflow">
        <span className="eyebrow">DEVELOPMENT LOOP</span>
        <div className="workflow-row">
          {["设计", "实现", "验证", "交付"].map((step, index) => (
            <div className="workflow-step" key={step}>
              <span>{index + 1}</span>
              <strong>{step}</strong>
            </div>
          ))}
        </div>
      </section>
    </>
  );
}

const rootRoute = createRootRoute({
  component: AppLayout,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: OverviewPage,
});

const announcementsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/announcements",
  component: Announcements,
});

export const routeTree = rootRoute.addChildren([
  indexRoute,
  announcementsRoute,
]);

export const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
