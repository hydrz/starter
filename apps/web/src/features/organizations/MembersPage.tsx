import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { useState } from "react";
import {
  getListMembersQueryKey,
  useListMembers,
  useRemoveMember,
  useUpdateMemberRole,
} from "../../api/generated/organizations/organizations";
import { OrganizationRole } from "../../api/generated/model";
import type { OrganizationMembership } from "../../api/generated/model";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { toast } from "../../components/ui/toast";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../components/ui/select";
import { Button } from "../../components/ui/button";
import { ConfirmDialog } from "../../components/layout/ConfirmDialog";
import {
  DataTable,
  type DataTableColumn,
} from "../../components/layout/DataTable";
import { EmptyState } from "../../components/layout/EmptyState";
import { useCurrentUser } from "../../lib/auth-session";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";
import { useOrgContext } from "./OrgContext";

function roleLabel(role: OrganizationRole) {
  switch (role) {
    case OrganizationRole.owner:
      return m.role_owner();
    case OrganizationRole.admin:
      return m.role_admin();
    case OrganizationRole.member:
      return m.role_member();
    case OrganizationRole.viewer:
      return m.role_viewer();
  }
}

const roleOptions = Object.values(OrganizationRole);

/**
 * 成员管理（DESIGN.md §6.2）：列表、行内角色调整、移除成员。
 *
 * "最后一个 owner 不能被降级/移除"是后端强制的业务规则
 * （`internal/organization` 的 `ErrLastOwner`），前端不重复实现这条规则，
 * 只是把后端返回的 `ApiError.message` 原样通过 toast 展示出来。
 */
export function MembersPage() {
  const { organizationId } = useOrgContext();
  const { identity } = useCurrentUser();
  const queryClient = useQueryClient();
  const [pendingRemoval, setPendingRemoval] =
    useState<OrganizationMembership | null>(null);

  const membersQuery = useListMembers(organizationId);
  const members =
    membersQuery.data?.status === 200 ? membersQuery.data.data : [];

  const invalidate = () =>
    queryClient.invalidateQueries({
      queryKey: getListMembersQueryKey(organizationId),
    });

  const updateRole = useUpdateMemberRole({
    mutation: {
      onSuccess: async () => {
        await invalidate();
        toast.success(m.members_role_update_success());
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  const removeMember = useRemoveMember({
    mutation: {
      onSuccess: async () => {
        await invalidate();
        toast.success(m.members_remove_success());
        setPendingRemoval(null);
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  const columns: Array<DataTableColumn<OrganizationMembership>> = [
    {
      id: "email",
      header: m.members_table_email(),
      cell: (member) => (
        <span className="flex items-center gap-2">
          <span className="font-mono text-xs">
            {member.email ?? member.userId}
          </span>
          {member.userId === identity?.id && (
            <span className="rounded bg-muted px-1.5 py-0.5 text-[10px] font-medium text-muted-foreground">
              {m.members_you_badge()}
            </span>
          )}
        </span>
      ),
    },
    {
      id: "role",
      header: m.members_table_role(),
      cell: (member) => (
        <Select
          value={member.role}
          onValueChange={(value) =>
            updateRole.mutate({
              organizationId,
              userId: member.userId,
              data: { role: value as OrganizationRole },
            })
          }
          disabled={updateRole.isPending}
        >
          <SelectTrigger className="h-8 w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {roleOptions.map((role) => (
              <SelectItem key={role} value={role}>
                {roleLabel(role)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ),
    },
    {
      id: "joined",
      header: m.members_table_joined(),
      cell: (member) => (
        <span className="font-mono text-xs text-muted-foreground">
          {new Date(member.createdAt).toLocaleDateString()}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      className: "text-right",
      cell: (member) => (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="text-destructive hover:text-destructive"
          onClick={() => setPendingRemoval(member)}
        >
          {m.members_remove_action()}
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      {membersQuery.isError && (
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>
            {getErrorMessage(membersQuery.error)}
          </AlertDescription>
        </Alert>
      )}

      {!membersQuery.isError && (
        <DataTable
          columns={columns}
          data={members}
          getRowId={(member) => member.id}
          isLoading={membersQuery.isPending}
          emptyState={
            <EmptyState
              title={m.members_empty_title()}
              description={m.members_empty_description()}
              className="rounded-none border-0 border-t border-border"
            />
          }
        />
      )}

      <ConfirmDialog
        open={pendingRemoval !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingRemoval(null);
          }
        }}
        title={m.members_remove_confirm_title({
          email: pendingRemoval?.email ?? pendingRemoval?.userId ?? "",
        })}
        description={m.members_remove_confirm_description()}
        confirmLabel={m.members_remove_action()}
        pending={removeMember.isPending}
        onConfirm={() => {
          if (pendingRemoval) {
            removeMember.mutate({
              organizationId,
              userId: pendingRemoval.userId,
            });
          }
        }}
      />
    </div>
  );
}
