import { Link } from "@tanstack/react-router";
import {
  AlertTriangle,
  ArrowRight,
  Megaphone,
  Settings,
  UserPlus,
  Users,
} from "lucide-react";
import { useListAnnouncements } from "../../api/generated/announcements/announcements";
import {
  useListInvitations,
  useListMembers,
} from "../../api/generated/organizations/organizations";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Badge } from "../../components/ui/badge";
import { Skeleton } from "../../components/ui/skeleton";
import { EmptyState } from "../../components/layout/EmptyState";
import { StatCard } from "../../components/layout/StatCard";
import { useOrgContext } from "../organizations/OrgContext";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";

/**
 * 组织概览页（DESIGN.md §6.2）：组织基础信息、成员数、待处理邀请数、
 * 最近公告，以及到成员/邀请/设置的快捷入口。entitlement/订阅状态是
 * Stage 4（计费）的范围——这里没有能拿到的真实数据，宁可完全不展示这一
 * 区块，也不用编造的数字占位（DESIGN.md §8 的错误处理基线同理适用于
 * "没有数据"，不是只适用于请求失败）。
 */
export function OrgOverviewPage() {
  const { organizationId, name, slug } = useOrgContext();

  const membersQuery = useListMembers(organizationId);
  const memberCount =
    membersQuery.data?.status === 200
      ? membersQuery.data.data.length
      : undefined;

  const invitationsQuery = useListInvitations(organizationId);
  const pendingInvitationCount =
    invitationsQuery.data?.status === 200
      ? invitationsQuery.data.data.length
      : undefined;

  const announcementsQuery = useListAnnouncements(organizationId, {
    limit: 5,
    offset: 0,
  });
  const recentAnnouncements =
    announcementsQuery.data?.status === 200
      ? announcementsQuery.data.data.items
      : undefined;

  return (
    <div className="space-y-6">
      <section>
        <p className="text-sm text-muted-foreground">
          {m.org_overview_heading_eyebrow()}
        </p>
        <h2 className="text-2xl font-bold tracking-tight text-foreground">
          {name}
        </h2>
        <p className="font-mono text-xs text-muted-foreground">/{slug}</p>
      </section>

      <section className="grid gap-4 sm:grid-cols-3">
        <Link to="/app/$orgSlug/members" params={{ orgSlug: slug }}>
          <StatCard
            label={m.org_overview_members_stat_label()}
            value={
              memberCount ??
              (membersQuery.isError ? "—" : <Skeleton className="h-7 w-10" />)
            }
            hint={m.org_overview_members_stat_hint()}
            icon={Users}
          />
        </Link>
        <Link to="/app/$orgSlug/invitations" params={{ orgSlug: slug }}>
          <StatCard
            label={m.org_overview_invitations_stat_label()}
            value={
              pendingInvitationCount ??
              (invitationsQuery.isError ? (
                "—"
              ) : (
                <Skeleton className="h-7 w-10" />
              ))
            }
            hint={m.org_overview_invitations_stat_hint()}
            icon={UserPlus}
          />
        </Link>
        <Link to="/app/$orgSlug/settings" params={{ orgSlug: slug }}>
          <StatCard
            label={m.org_overview_settings_stat_label()}
            value={<Settings size={20} />}
            hint={m.org_overview_settings_stat_hint()}
          />
        </Link>
      </section>

      <section>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Megaphone size={15} />
            {m.org_overview_recent_announcements_title()}
          </h3>
          <Link
            to="/app/$orgSlug/announcements"
            params={{ orgSlug: slug }}
            className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
          >
            {m.org_overview_recent_announcements_view_all()}
            <ArrowRight size={12} />
          </Link>
        </div>

        {announcementsQuery.isPending && (
          <div className="space-y-2">
            <Skeleton className="h-14 w-full" />
            <Skeleton className="h-14 w-full" />
          </div>
        )}

        {announcementsQuery.isError && (
          <Alert variant="destructive">
            <AlertTriangle />
            <AlertDescription>
              {getErrorMessage(announcementsQuery.error)}
            </AlertDescription>
          </Alert>
        )}

        {recentAnnouncements?.length === 0 && (
          <EmptyState
            icon={Megaphone}
            title={m.org_overview_recent_announcements_empty()}
          />
        )}

        {recentAnnouncements && recentAnnouncements.length > 0 && (
          <ul className="divide-y divide-border rounded-lg border border-border">
            {recentAnnouncements.map((item) => (
              <li
                key={item.id}
                className="flex items-center justify-between gap-3 px-4 py-3"
              >
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium text-foreground">
                    {item.title}
                  </p>
                  <p className="font-mono text-xs text-muted-foreground">
                    {new Date(item.updatedAt).toLocaleDateString()}
                  </p>
                </div>
                <Badge
                  variant={
                    item.status === "published" ? "default" : "secondary"
                  }
                >
                  {item.status === "published"
                    ? m.announcements_status_published()
                    : m.announcements_status_draft()}
                </Badge>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
