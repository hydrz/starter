import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangle } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import {
  getListInvitationsQueryKey,
  useCreateInvitation,
  useListInvitations,
  useRevokeInvitation,
} from "../../api/generated/organizations/organizations";
import { OrganizationRole } from "../../api/generated/model";
import type { OrganizationInvitation } from "../../api/generated/model";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { ConfirmDialog } from "../../components/layout/ConfirmDialog";
import {
  DataTable,
  type DataTableColumn,
} from "../../components/layout/DataTable";
import { EmptyState } from "../../components/layout/EmptyState";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "../../components/ui/form";
import { Input } from "../../components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../components/ui/select";
import { toast } from "../../components/ui/toast";
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

const invitableRoles = [
  OrganizationRole.admin,
  OrganizationRole.member,
  OrganizationRole.viewer,
] as const;

function buildInviteSchema() {
  return z.object({
    email: z.email(m.invitations_validation_email()),
    role: z.enum(invitableRoles),
  });
}

type InviteForm = z.infer<ReturnType<typeof buildInviteSchema>>;

/** 邀请管理（DESIGN.md §6.2）：待处理邀请列表、发起邀请、撤销邀请。 */
export function InvitationsPage() {
  const { organizationId } = useOrgContext();
  const queryClient = useQueryClient();
  const [pendingRevoke, setPendingRevoke] =
    useState<OrganizationInvitation | null>(null);

  const invitationsQuery = useListInvitations(organizationId);
  const invitations =
    invitationsQuery.data?.status === 200 ? invitationsQuery.data.data : [];

  const invalidate = () =>
    queryClient.invalidateQueries({
      queryKey: getListInvitationsQueryKey(organizationId),
    });

  const schema = buildInviteSchema();
  const form = useForm<InviteForm>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", role: OrganizationRole.member },
  });

  const createInvitation = useCreateInvitation({
    mutation: {
      onSuccess: async () => {
        await invalidate();
        toast.success(m.invitations_create_success());
        form.reset({ email: "", role: OrganizationRole.member });
      },
      onError: (error) => {
        const message = getErrorMessage(error);
        form.setError("root", { type: "server", message });
        toast.error(message);
      },
    },
  });

  const revokeInvitation = useRevokeInvitation({
    mutation: {
      onSuccess: async () => {
        await invalidate();
        toast.success(m.invitations_revoke_success());
        setPendingRevoke(null);
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  const onSubmit = form.handleSubmit((values) =>
    createInvitation.mutate({ organizationId, data: values }),
  );

  const columns: Array<DataTableColumn<OrganizationInvitation>> = [
    {
      id: "email",
      header: m.invitations_table_email(),
      cell: (invitation) => (
        <span className="font-mono text-xs">{invitation.email}</span>
      ),
    },
    {
      id: "role",
      header: m.invitations_table_role(),
      cell: (invitation) => roleLabel(invitation.role),
    },
    {
      id: "expires",
      header: m.invitations_table_expires(),
      cell: (invitation) => (
        <span className="font-mono text-xs text-muted-foreground">
          {new Date(invitation.expiresAt).toLocaleDateString()}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      className: "text-right",
      cell: (invitation) => (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="text-destructive hover:text-destructive"
          onClick={() => setPendingRevoke(invitation)}
        >
          {m.invitations_revoke_action()}
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <Form {...form}>
        <form
          onSubmit={onSubmit}
          className="flex flex-col gap-3 rounded-lg border border-border p-4 sm:flex-row sm:items-end"
        >
          <FormField
            control={form.control}
            name="email"
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormLabel>{m.invitations_create_email_label()}</FormLabel>
                <FormControl>
                  <Input
                    type="email"
                    placeholder="teammate@example.com"
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="role"
            render={({ field }) => (
              <FormItem className="sm:w-40">
                <FormLabel>{m.invitations_create_role_label()}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {invitableRoles.map((role) => (
                      <SelectItem key={role} value={role}>
                        {roleLabel(role)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button type="submit" disabled={createInvitation.isPending}>
            {createInvitation.isPending
              ? m.invitations_create_submitting()
              : m.invitations_create_submit()}
          </Button>
        </form>
        {form.formState.errors.root && (
          <Alert variant="destructive">
            <AlertTriangle />
            <AlertDescription>
              {form.formState.errors.root.message}
            </AlertDescription>
          </Alert>
        )}
      </Form>

      {invitationsQuery.isError && (
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>
            {getErrorMessage(invitationsQuery.error)}
          </AlertDescription>
        </Alert>
      )}

      {!invitationsQuery.isError && (
        <DataTable
          columns={columns}
          data={invitations}
          getRowId={(invitation) => invitation.id}
          isLoading={invitationsQuery.isPending}
          emptyState={
            <EmptyState
              title={m.invitations_empty_title()}
              description={m.invitations_empty_description()}
              className="rounded-none border-0 border-t border-border"
            />
          }
        />
      )}

      <ConfirmDialog
        open={pendingRevoke !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingRevoke(null);
          }
        }}
        title={m.invitations_revoke_confirm_title({
          email: pendingRevoke?.email ?? "",
        })}
        description={m.invitations_revoke_confirm_description()}
        confirmLabel={m.invitations_revoke_action()}
        pending={revokeInvitation.isPending}
        onConfirm={() => {
          if (pendingRevoke) {
            revokeInvitation.mutate({
              organizationId,
              invitationId: pendingRevoke.id,
            });
          }
        }}
      />
    </div>
  );
}
