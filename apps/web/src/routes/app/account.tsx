import {
  Outlet,
  createFileRoute,
  useNavigate,
  useRouterState,
} from "@tanstack/react-router";
import { KeyRound, LayoutDashboard, ShieldCheck } from "lucide-react";
import { useListUserOrganizations } from "../../api/generated/organizations/organizations";
import { AppShell, type AppNavItem } from "../../components/layout/AppShell";
import type { OrgSwitcherOrganization } from "../../components/layout/OrgSwitcher";
import { ServiceStatusBadge } from "../../components/ServiceStatusBadge";
import * as m from "../../paraglide/messages";

function getNavItems(): Array<AppNavItem> {
  return [
    {
      label: m.nav_account_overview(),
      to: "/app/account",
      icon: LayoutDashboard,
      exact: true,
    },
    {
      label: m.nav_account_sessions(),
      to: "/app/account/sessions",
      icon: KeyRound,
    },
    {
      label: m.nav_account_security(),
      to: "/app/account/security",
      icon: ShieldCheck,
    },
  ];
}

function getPageTitle(pathname: string) {
  if (pathname.includes("/account/sessions")) {
    return m.console_page_account_sessions();
  }
  if (pathname.includes("/account/security")) {
    return m.console_page_account_security();
  }
  return m.console_page_account_overview();
}

/**
 * `/app/account` 布局路由（DESIGN.md §6.2）：与 `routes/app/$orgSlug.tsx`
 * 相同的 `AppShell` 壳层，但没有组织上下文——账户设置是跨组织的。仍然把
 * `useListUserOrganizations` 的结果喂给组织切换器（不传 `activeOrganizationId`，
 * `OrgSwitcher` 会退化展示第一个组织），让用户能从账户页直接跳回某个组织，
 * 而不是必须先点 Logo 回 `/app` 再选一次。
 */
function AccountLayoutRoute() {
  const navigate = useNavigate();
  const currentPath = useRouterState({
    select: (state) => state.location.pathname,
  });

  const organizationsQuery = useListUserOrganizations();
  const organizations =
    organizationsQuery.data?.status === 200 ? organizationsQuery.data.data : [];
  const switcherOrganizations: Array<OrgSwitcherOrganization> =
    organizations.map((org) => ({
      id: org.id,
      name: org.name,
      slug: org.slug,
    }));

  return (
    <AppShell
      navItems={getNavItems()}
      organizations={switcherOrganizations}
      onSelectOrganization={(org) =>
        void navigate({ to: "/app/$orgSlug", params: { orgSlug: org.slug } })
      }
      onCreateOrganization={() => void navigate({ to: "/app" })}
      pageTitle={getPageTitle(currentPath)}
      headerActions={<ServiceStatusBadge />}
    >
      <Outlet />
    </AppShell>
  );
}

export const Route = createFileRoute("/app/account")({
  component: AccountLayoutRoute,
});
