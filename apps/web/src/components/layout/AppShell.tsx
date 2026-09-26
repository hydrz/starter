import type { LucideIcon } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import { ChevronLeft, ChevronRight, Menu } from "lucide-react";
import { LanguageSwitcher } from "../LanguageSwitcher";
import { ThemeToggle } from "../ThemeToggle";
import { UserAvatarMenu } from "../UserAvatarMenu";
import { Button } from "../ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "../ui/sheet";
import { useUIStore } from "../../stores/ui-store";
import { OrgSwitcher, type OrgSwitcherOrganization } from "./OrgSwitcher";
import * as m from "../../paraglide/messages";

export interface AppNavItem {
  label: string;
  to: string;
  icon: LucideIcon;
  /** 精确匹配当前路径高亮；默认使用前缀匹配 */
  exact?: boolean;
  /** 外部链接（如 API 文档）：渲染为新标签页打开的 <a>，不参与路由高亮 */
  external?: boolean;
}

export interface AppShellProps {
  /** 侧边栏导航项，数据驱动——Stage 3/4 直接扩充这个数组即可，无需改动 AppShell 本身 */
  navItems: Array<AppNavItem>;
  /** 侧边栏底部的次要链接（例如"返回前台"） */
  secondaryNavItems?: Array<AppNavItem>;
  organizations?: Array<OrgSwitcherOrganization>;
  activeOrganizationId?: string;
  pageTitle: ReactNode;
  breadcrumb?: ReactNode;
  headerActions?: ReactNode;
  children: ReactNode;
}

function isActive(pathname: string, item: AppNavItem) {
  if (item.exact) {
    return pathname === item.to;
  }
  return pathname === item.to || pathname.startsWith(`${item.to}/`);
}

function NavList({
  navItems,
  secondaryNavItems,
  pathname,
  collapsed,
  onNavigate,
}: {
  navItems: Array<AppNavItem>;
  secondaryNavItems: Array<AppNavItem>;
  pathname: string;
  collapsed: boolean;
  onNavigate?: () => void;
}) {
  return (
    <nav aria-label={m.nav_console_aria()}>
      {navItems.map((item) => {
        const Icon = item.icon;
        if (item.external) {
          return (
            <a
              key={item.to}
              href={item.to}
              target="_blank"
              rel="noreferrer"
              className="nav-item"
            >
              <div className="flex items-center gap-2.5">
                <Icon size={15} />
                {!collapsed && <span>{item.label}</span>}
              </div>
            </a>
          );
        }
        const active = isActive(pathname, item);
        return (
          <Link
            key={item.to}
            to={item.to}
            onClick={onNavigate}
            className={`nav-item ${active ? "active" : ""}`}
          >
            <div className="flex items-center gap-2.5">
              <Icon size={15} />
              {!collapsed && <span>{item.label}</span>}
            </div>
            {!collapsed && active && <span className="nav-dot" />}
          </Link>
        );
      })}

      {secondaryNavItems.length > 0 && (
        <div className="mt-4 border-t border-white/10 pt-4">
          {secondaryNavItems.map((item) => {
            const Icon = item.icon;
            if (item.external) {
              return (
                <a
                  key={item.to}
                  href={item.to}
                  target="_blank"
                  rel="noreferrer"
                  className="nav-item text-neutral-400 hover:text-white"
                >
                  <div className="flex items-center gap-2.5">
                    <Icon size={15} />
                    {!collapsed && <span>{item.label}</span>}
                  </div>
                </a>
              );
            }
            return (
              <Link
                key={item.to}
                to={item.to}
                onClick={onNavigate}
                className="nav-item text-neutral-400 hover:text-white"
              >
                <div className="flex items-center gap-2.5">
                  <Icon size={15} />
                  {!collapsed && <span>{item.label}</span>}
                </div>
              </Link>
            );
          })}
        </div>
      )}
    </nav>
  );
}

/**
 * Dashboard 骨架（DESIGN.md §1/§3/§6.2）：左侧可折叠侧边栏（组织切换器置顶
 * + 数据驱动导航）+ 顶部面包屑/标题/操作区。移动端侧边栏收起为 `Sheet` 抽屉。
 */
export function AppShell({
  navItems,
  secondaryNavItems = [],
  organizations = [],
  activeOrganizationId,
  pageTitle,
  breadcrumb,
  headerActions,
  children,
}: AppShellProps) {
  const { sidebarCollapsed, toggleSidebar } = useUIStore();
  const pathname = useRouterState({ select: (state) => state.location.pathname });

  const sidebarBody = (collapsed: boolean, onNavigate?: () => void) => (
    <>
      <div className="flex items-center justify-between pb-4">
        <Link
          to="/app"
          className="flex items-center gap-2.5 font-semibold text-foreground transition-opacity hover:opacity-90"
          aria-label={m.sidebar_home_aria()}
        >
          <span className="brand-mark">E</span>
          {!collapsed && (
            <span className="text-white font-semibold">
              {m.app_brand()}
            </span>
          )}
        </Link>
        {onNavigate === undefined && (
          <Button
            variant="ghost"
            size="icon"
            onClick={toggleSidebar}
            aria-label={
              sidebarCollapsed ? m.sidebar_expand() : m.sidebar_collapse()
            }
            className="h-7 w-7 text-muted-foreground hover:text-white"
          >
            {sidebarCollapsed ? (
              <ChevronRight size={16} />
            ) : (
              <ChevronLeft size={16} />
            )}
          </Button>
        )}
      </div>

      {organizations.length > 0 && (
        <div className="pb-4">
          <OrgSwitcher
            organizations={organizations}
            activeOrganizationId={activeOrganizationId}
            collapsed={collapsed}
          />
        </div>
      )}

      <NavList
        navItems={navItems}
        secondaryNavItems={secondaryNavItems}
        pathname={pathname}
        collapsed={collapsed}
        onNavigate={onNavigate}
      />
    </>
  );

  return (
    <div
      className="shell"
      style={
        {
          "--sidebar-width": sidebarCollapsed ? "72px" : "240px",
        } as CSSProperties
      }
    >
      {/* 桌面端固定侧边栏 */}
      <aside
        className="sidebar hidden md:flex"
        style={{
          width: sidebarCollapsed ? "72px" : "240px",
          padding: sidebarCollapsed ? "24px 12px" : "24px 18px",
        }}
      >
        {sidebarBody(sidebarCollapsed)}
      </aside>

      <main style={{ gridColumn: 2 }}>
        <header className="topbar">
          <div className="flex items-center gap-3">
            {/* 移动端：Sheet 抽屉导航 */}
            <Sheet>
              <SheetTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="md:hidden"
                  aria-label={m.nav_console_aria()}
                >
                  <Menu size={18} />
                </Button>
              </SheetTrigger>
              <SheetContent side="left" className="w-72 bg-[#18221d] p-0">
                <SheetHeader className="sr-only">
                  <SheetTitle>{m.nav_console_aria()}</SheetTitle>
                </SheetHeader>
                <div className="p-5">
                  {sidebarBody(false, () => undefined)}
                </div>
              </SheetContent>
            </Sheet>

            <div>
              {breadcrumb}
              <div className="flex items-center gap-2 mt-1">
                <h1 className="text-xl font-bold tracking-tight">
                  {m.console_title()}
                </h1>
                <span className="text-muted-foreground text-sm font-normal">
                  /
                </span>
                <span className="text-sm font-medium text-muted-foreground">
                  {pageTitle}
                </span>
              </div>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {headerActions}
            <LanguageSwitcher className="hidden sm:inline-flex" />
            <ThemeToggle />
            <UserAvatarMenu />
          </div>
        </header>

        {children}
      </main>
    </div>
  );
}
