import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { AlertTriangle } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import {
  getGetOrganizationQueryKey,
  getListUserOrganizationsQueryKey,
  useDeleteOrganization,
  useGetOrganization,
  useUpdateOrganization,
} from "../../api/generated/organizations/organizations";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { ConfirmDialog } from "../../components/layout/ConfirmDialog";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "../../components/ui/form";
import { Input } from "../../components/ui/input";
import { Skeleton } from "../../components/ui/skeleton";
import { toast } from "../../components/ui/toast";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";
import { buildOrgSchema, type OrgFormValues } from "./org-schema";
import { useOrgContext } from "./OrgContext";

/**
 * 组织设置（DESIGN.md §6.2）：名称/slug 编辑 + 危险区（删除组织）。
 * 删除组织的二次确认使用 DESIGN.md §8 要求的"重新输入对象名称"强化模式
 * ——普通的 `ConfirmDialog` 文案已经点名了被删除的组织，这里再加一道
 * 输入校验，因为删除组织是不可逆操作。
 */
export function SettingsPage() {
  const { organizationId, slug: orgSlug } = useOrgContext();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const organizationQuery = useGetOrganization(organizationId);
  const organization =
    organizationQuery.data?.status === 200
      ? organizationQuery.data.data
      : undefined;

  if (organizationQuery.isPending) {
    return (
      <div className="max-w-lg space-y-3">
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-full" />
        <Skeleton className="h-9 w-28" />
      </div>
    );
  }

  if (organizationQuery.isError || !organization) {
    return (
      <Alert variant="destructive">
        <AlertTriangle />
        <AlertDescription>
          {organizationQuery.error
            ? getErrorMessage(organizationQuery.error)
            : m.org_layout_not_found_description()}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="max-w-lg space-y-8">
      <GeneralSettingsForm
        organizationId={organizationId}
        organizationName={organization.name}
        organizationSlug={organization.slug}
        currentOrgSlug={orgSlug}
        onRenamed={(newSlug) =>
          navigate({
            to: "/app/$orgSlug/settings",
            params: { orgSlug: newSlug },
          })
        }
      />

      <DangerZone
        organizationId={organizationId}
        organizationName={organization.name}
        onDeleted={async () => {
          await queryClient.invalidateQueries({
            queryKey: getListUserOrganizationsQueryKey(),
          });
          await navigate({ to: "/app" });
        }}
      />
    </div>
  );
}

function GeneralSettingsForm({
  organizationId,
  organizationName,
  organizationSlug,
  currentOrgSlug,
  onRenamed,
}: {
  organizationId: string;
  organizationName: string;
  organizationSlug: string;
  currentOrgSlug: string;
  onRenamed: (newSlug: string) => void;
}) {
  const queryClient = useQueryClient();
  const schema = buildOrgSchema();
  const form = useForm<OrgFormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: organizationName, slug: organizationSlug },
  });

  // 组织数据首次加载完成后用真实值重置表单默认值（`useGetOrganization`
  // pending 态之前无法拿到 name/slug）。
  useEffect(() => {
    form.reset({ name: organizationName, slug: organizationSlug });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [organizationName, organizationSlug]);

  const updateOrganization = useUpdateOrganization({
    mutation: {
      onSuccess: async (res) => {
        if (res.status === 200) {
          await queryClient.invalidateQueries({
            queryKey: getListUserOrganizationsQueryKey(),
          });
          await queryClient.invalidateQueries({
            queryKey: getGetOrganizationQueryKey(organizationId),
          });
          toast.success(m.settings_save_success());
          if (res.data.slug !== currentOrgSlug) {
            onRenamed(res.data.slug);
          }
        }
      },
      onError: (error) => {
        const message = getErrorMessage(error);
        form.setError("root", { type: "server", message });
        toast.error(message);
      },
    },
  });

  const onSubmit = form.handleSubmit((values) =>
    updateOrganization.mutate({ organizationId, data: values }),
  );

  return (
    <section>
      <h3 className="mb-3 text-sm font-semibold text-foreground">
        {m.settings_general_heading()}
      </h3>
      <Form {...form}>
        <form onSubmit={onSubmit} className="space-y-3.5">
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{m.settings_name_label()}</FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="slug"
            render={({ field }) => (
              <FormItem>
                <FormLabel>{m.settings_slug_label()}</FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          {form.formState.errors.root && (
            <Alert variant="destructive">
              <AlertTriangle />
              <AlertDescription>
                {form.formState.errors.root.message}
              </AlertDescription>
            </Alert>
          )}
          <Button type="submit" disabled={updateOrganization.isPending}>
            {updateOrganization.isPending
              ? m.settings_save_submitting()
              : m.settings_save_submit()}
          </Button>
        </form>
      </Form>
    </section>
  );
}

function DangerZone({
  organizationId,
  organizationName,
  onDeleted,
}: {
  organizationId: string;
  organizationName: string;
  onDeleted: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [retypedName, setRetypedName] = useState("");
  const [mismatch, setMismatch] = useState(false);

  const deleteOrganization = useDeleteOrganization({
    mutation: {
      onSuccess: () => {
        toast.success(m.settings_delete_success());
        setOpen(false);
        onDeleted();
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  return (
    <section className="rounded-lg border border-destructive/40 bg-destructive/5 p-4">
      <h3 className="mb-1 text-sm font-semibold text-destructive">
        {m.settings_danger_heading()}
      </h3>
      <p className="mb-3 text-xs text-muted-foreground">
        {m.settings_danger_description()}
      </p>
      <Button
        type="button"
        variant="destructive"
        onClick={() => {
          setRetypedName("");
          setMismatch(false);
          setOpen(true);
        }}
      >
        {m.settings_delete_button()}
      </Button>

      <ConfirmDialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) {
            setRetypedName("");
            setMismatch(false);
          }
        }}
        title={m.settings_delete_confirm_title({ name: organizationName })}
        description={
          <div className="space-y-3">
            <p>{m.settings_delete_confirm_description()}</p>
            <div>
              <label
                htmlFor="delete-org-confirm-input"
                className="mb-1 block text-xs font-medium text-foreground"
              >
                {m.settings_delete_confirm_retype_label({
                  name: organizationName,
                })}
              </label>
              <Input
                id="delete-org-confirm-input"
                value={retypedName}
                onChange={(event) => {
                  setRetypedName(event.target.value);
                  setMismatch(false);
                }}
                placeholder={organizationName}
                autoComplete="off"
              />
              {mismatch && (
                <p className="mt-1 text-xs text-destructive">
                  {m.settings_delete_confirm_mismatch()}
                </p>
              )}
            </div>
          </div>
        }
        confirmLabel={m.settings_delete_button()}
        pending={deleteOrganization.isPending}
        onConfirm={() => {
          if (retypedName !== organizationName) {
            setMismatch(true);
            return;
          }
          deleteOrganization.mutate({ organizationId });
        }}
      />
    </section>
  );
}
