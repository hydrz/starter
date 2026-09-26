import {
  Outlet,
  createFileRoute,
  useNavigate,
  useRouterState,
} from "@tanstack/react-router";
import type { ReactNode } from "react";
import {
  Activity,
  AlertTriangle,
  CreditCard,
  FileCode,
  Globe,
  LayoutDashboard,
  Megaphone,
  Settings,
  User,
  UserPlus,
  Users,
} from "lucide-react";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { Skeleton } from "../../components/ui/skeleton";
import { useListUserOrganizations } from "../../api/generated/organizations/organizations";
import { AppShell, type AppNavItem } from "../../components/layout/AppShell";
import type { OrgSwitcherOrganization } from "../../components/layout/OrgSwitcher";
import { ServiceStatusBadge } from "../../components/ServiceStatusBadge";
import { OrgProvider } from "../../features/organizations/OrgContext";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";

export const Route = createFileRoute("/app/$orgSlug")({
  component: OrgLayoutRoute,
});

function getNavItems(orgSlug: string): Array<AppNavItem> {
  const base = `/app/${orgSlug}`;
  return [
    { label: m.nav_overview(), to: base, icon: LayoutDashboard, exact: true },
    {
      label: m.nav_announcements(),
      to: `${base}/announcements`,
      icon: Megaphone,
    },
    { label: m.nav_members(), to: `${base}/members`, icon: Users },
    {
      label: m.nav_invitations(),
      to: `${base}/invitations`,
      icon: UserPlus,
    },
    { label: m.nav_billing(), to: `${base}/billing`, icon: CreditCard },
    { label: m.nav_settings(), to: `${base}/settings`, icon: Settings },
    {
      label: m.nav_observability(),
      to: `${base}/observability`,
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
    { label: m.nav_account(), to: "/app/account", icon: User },
    { label: m.nav_back_to_landing(), to: "/", icon: Globe, exact: true },
  ];
}

function getPageTitle(pathname: string) {
  if (pathname.includes("/members")) {
    return m.console_page_members();
  }
  if (pathname.includes("/invitations")) {
    return m.console_page_invitations();
  }
  if (pathname.includes("/billing")) {
    return m.console_page_billing();
  }
  if (pathname.includes("/settings")) {
    return m.console_page_settings();
  }
  if (pathname.includes("/announcements")) {
    return m.console_page_announcements();
  }
  if (pathname.includes("/observability")) {
    return m.console_page_observability();
  }
  return m.console_page_overview();
}

/**
 * `/app/$orgSlug` 布局路由：把 URL 里的 slug 解析为真实的
 * `organizationId`（`useGetOrganization` 只接受 id，不接受 slug——见
 * 生成的 `getOrganization(organizationId: string, ...)` 签名，所以这里
 * 用已经加载的 `useListUserOrganizations` 结果做 slug -> id 的映射，
 * 而不是新增一次请求），渲染带真实组织数据的 `AppShell`，并把解析结果
 * 通过 `OrgProvider` 下发给 Overview/Announcements/Members/Invitations/
 * Settings/Observability 这些子路由。
 */
function OrgLayoutRoute() {
  const { orgSlug } = Route.useParams();
  const navigate = useNavigate();
  const currentPath = useRouterState({
    select: (state) => state.location.pathname,
  });

  const organizationsQuery = useListUserOrganizations();
  const organizations =
    organizationsQuery.data?.status === 200
      ? organizationsQuery.data.data
      : undefined;

  if (organizationsQuery.isPending) {
    return <OrgLayoutSkeleton />;
  }

  if (organizationsQuery.isError) {
    return (
      <OrgLayoutMessage
        icon={<AlertTriangle className="text-destructive" size={20} />}
        title={m.org_layout_error_title()}
        description={getErrorMessage(organizationsQuery.error)}
        action={
          <Button onClick={() => void organizationsQuery.refetch()}>
            {m.org_picker_retry()}
          </Button>
        }
      />
    );
  }

  const activeOrg = organizations?.find((org) => org.slug === orgSlug);

  if (!activeOrg) {
    return (
      <OrgLayoutMessage
        icon={<AlertTriangle className="text-warning" size={20} />}
        title={m.org_layout_not_found_title()}
        description={m.org_layout_not_found_description()}
        action={
          <Button onClick={() => void navigate({ to: "/app" })}>
            {m.org_layout_not_found_action()}
          </Button>
        }
      />
    );
  }

  const switcherOrganizations: Array<OrgSwitcherOrganization> = (
    organizations ?? []
  ).map((org) => ({ id: org.id, name: org.name, slug: org.slug }));

  return (
    <OrgProvider
      value={{
        organizationId: activeOrg.id,
        slug: activeOrg.slug,
        name: activeOrg.name,
        role: activeOrg.role,
      }}
    >
      <AppShell
        navItems={getNavItems(orgSlug)}
        secondaryNavItems={getSecondaryNavItems()}
        organizations={switcherOrganizations}
        activeOrganizationId={activeOrg.id}
        onSelectOrganization={(org) =>
          void navigate({
            to: "/app/$orgSlug",
            params: { orgSlug: org.slug },
          })
        }
        onCreateOrganization={() => void navigate({ to: "/app" })}
        pageTitle={getPageTitle(currentPath)}
        headerActions={<ServiceStatusBadge />}
      >
        <Outlet />
      </AppShell>
    </OrgProvider>
  );
}

function OrgLayoutSkeleton() {
  return (
    <div className="flex min-h-screen">
      <div className="hidden w-60 shrink-0 border-r border-border p-4 md:block">
        <Skeleton className="mb-6 h-8 w-full" />
        <Skeleton className="mb-4 h-9 w-full" />
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, index) => (
            <Skeleton key={index} className="h-8 w-full" />
          ))}
        </div>
      </div>
      <div className="flex-1 p-6">
        <Skeleton className="mb-4 h-8 w-64" />
        <Skeleton className="h-40 w-full" />
      </div>
    </div>
  );
}

function OrgLayoutMessage({
  icon,
  title,
  description,
  action,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  action: ReactNode;
}) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 px-4 text-center">
      {icon}
      <div className="space-y-1.5">
        <h1 className="text-lg font-semibold text-foreground">{title}</h1>
        <Alert variant="default" className="max-w-md text-left">
          <AlertDescription>{description}</AlertDescription>
        </Alert>
      </div>
      {action}
    </div>
  );
}
