import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, KeyRound, Monitor } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import {
  getListAPIKeysQueryKey,
  getListSessionsQueryKey,
  useCreateAPIKey,
  useListAPIKeys,
  useListSessions,
  useRevokeAPIKey,
  useRevokeSession,
} from "../../api/generated/auth/auth";
import type { APIKey, Session } from "../../api/generated/model";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { Code } from "../../components/ui/code";
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
import { toast } from "../../components/ui/toast";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";
import { OneTimeSecretDialog } from "./OneTimeSecretDialog";

function formatDate(value?: string): string {
  return value ? new Date(value).toLocaleString() : "—";
}

/**
 * `/app/account/sessions`（DESIGN.md §6.2）：活跃会话 + API Key 两个区块。
 * 放在同一路由（而不是拆成两个子路由）——两者都是"账户级别的凭据管理"，
 * 内容量都不大，拆两个路由反而增加一次导航跳转，不符合这里的信息密度。
 */
export function SessionsPage() {
  return (
    <div className="space-y-10">
      <SessionsSection />
      <ApiKeysSection />
    </div>
  );
}

function SessionsSection() {
  const queryClient = useQueryClient();
  const [pendingRevoke, setPendingRevoke] = useState<Session | null>(null);

  const sessionsQuery = useListSessions();
  const sessions =
    sessionsQuery.data?.status === 200 ? sessionsQuery.data.data : [];

  const revokeSession = useRevokeSession({
    mutation: {
      onSuccess: async () => {
        await queryClient.invalidateQueries({
          queryKey: getListSessionsQueryKey(),
        });
        toast.success(m.account_sessions_revoke_success());
        setPendingRevoke(null);
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  const columns: Array<DataTableColumn<Session>> = [
    {
      id: "device",
      header: m.account_sessions_table_device(),
      cell: (session) => (
        <span className="flex items-center gap-2">
          <Monitor size={14} className="shrink-0 text-muted-foreground" />
          <span className="truncate text-xs">
            {session.userAgent ?? m.account_sessions_unknown_device()}
          </span>
        </span>
      ),
    },
    {
      id: "ip",
      header: m.account_sessions_table_ip(),
      cell: (session) => (
        <span className="font-mono text-xs text-muted-foreground">
          {session.ipAddress ?? "—"}
        </span>
      ),
    },
    {
      id: "lastUsed",
      header: m.account_sessions_table_last_used(),
      cell: (session) => (
        <span className="font-mono text-xs text-muted-foreground">
          {formatDate(session.lastUsedAt)}
        </span>
      ),
    },
    {
      id: "created",
      header: m.account_sessions_table_created(),
      cell: (session) => (
        <span className="font-mono text-xs text-muted-foreground">
          {formatDate(session.createdAt)}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      className: "text-right",
      cell: (session) => (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="text-destructive hover:text-destructive"
          onClick={() => setPendingRevoke(session)}
        >
          {m.account_sessions_revoke_action()}
        </Button>
      ),
    },
  ];

  return (
    <section className="space-y-4">
      <div>
        <h3 className="text-sm font-semibold text-foreground">
          {m.account_sessions_heading()}
        </h3>
        <p className="text-xs text-muted-foreground">
          {m.account_sessions_description()}
        </p>
      </div>

      {sessionsQuery.isError && (
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>
            {getErrorMessage(sessionsQuery.error)}
          </AlertDescription>
        </Alert>
      )}

      {!sessionsQuery.isError && (
        <DataTable
          columns={columns}
          data={sessions}
          getRowId={(session) => session.id}
          isLoading={sessionsQuery.isPending}
          emptyState={
            <EmptyState
              icon={Monitor}
              title={m.account_sessions_empty_title()}
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
        title={m.account_sessions_revoke_confirm_title({
          device:
            pendingRevoke?.userAgent ??
            pendingRevoke?.ipAddress ??
            m.account_sessions_unknown_device(),
        })}
        description={m.account_sessions_revoke_confirm_description()}
        confirmLabel={m.account_sessions_revoke_action()}
        pending={revokeSession.isPending}
        onConfirm={() => {
          if (pendingRevoke) {
            revokeSession.mutate({ id: pendingRevoke.id });
          }
        }}
      />
    </section>
  );
}

function buildCreateApiKeySchema() {
  return z.object({
    name: z
      .string()
      .min(1, m.account_api_keys_validation_name_required())
      .max(120, m.account_api_keys_validation_name_too_long()),
  });
}

type CreateApiKeyForm = z.infer<ReturnType<typeof buildCreateApiKeySchema>>;

function ApiKeysSection() {
  const queryClient = useQueryClient();
  const [pendingRevoke, setPendingRevoke] = useState<APIKey | null>(null);
  // 明文密钥只在内存里短暂存在——弹窗关闭（`OneTimeSecretDialog` 的
  // `onAcknowledge`）立刻清空，绝不重新展示（DESIGN.md §8 一次性展示基线）。
  const [revealedSecret, setRevealedSecret] = useState<{
    name: string;
    key: string;
  } | null>(null);

  const apiKeysQuery = useListAPIKeys();
  const apiKeys =
    apiKeysQuery.data?.status === 200 ? apiKeysQuery.data.data : [];

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: getListAPIKeysQueryKey() });

  const schema = buildCreateApiKeySchema();
  const form = useForm<CreateApiKeyForm>({
    resolver: zodResolver(schema),
    defaultValues: { name: "" },
  });

  const createApiKey = useCreateAPIKey({
    mutation: {
      onSuccess: async (res) => {
        if (res.status === 201) {
          await invalidate();
          setRevealedSecret({ name: res.data.name, key: res.data.key });
          form.reset({ name: "" });
        }
      },
      onError: (error) => {
        const message = getErrorMessage(error);
        form.setError("root", { type: "server", message });
        toast.error(message);
      },
    },
  });

  const revokeApiKey = useRevokeAPIKey({
    mutation: {
      onSuccess: async () => {
        await invalidate();
        toast.success(m.account_api_keys_revoke_success());
        setPendingRevoke(null);
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  const onSubmit = form.handleSubmit((values) =>
    createApiKey.mutate({ data: values }),
  );

  const columns: Array<DataTableColumn<APIKey>> = [
    {
      id: "name",
      header: m.account_api_keys_table_name(),
      cell: (key) => <span className="text-xs font-medium">{key.name}</span>,
    },
    {
      id: "prefix",
      header: m.account_api_keys_table_prefix(),
      cell: (key) => (
        <Code copyable={false} className="text-[11px]">
          {key.prefix}…
        </Code>
      ),
    },
    {
      id: "lastUsed",
      header: m.account_api_keys_table_last_used(),
      cell: (key) => (
        <span className="font-mono text-xs text-muted-foreground">
          {formatDate(key.lastUsedAt)}
        </span>
      ),
    },
    {
      id: "status",
      header: m.account_api_keys_table_status(),
      cell: (key) =>
        key.revokedAt ? (
          <span className="text-xs text-destructive">
            {m.account_api_keys_status_revoked()}
          </span>
        ) : (
          <span className="text-xs text-success">
            {m.account_api_keys_status_active()}
          </span>
        ),
    },
    {
      id: "actions",
      header: "",
      className: "text-right",
      cell: (key) =>
        key.revokedAt ? null : (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={() => setPendingRevoke(key)}
          >
            {m.account_api_keys_revoke_action()}
          </Button>
        ),
    },
  ];

  return (
    <section className="space-y-4">
      <div>
        <h3 className="text-sm font-semibold text-foreground">
          {m.account_api_keys_heading()}
        </h3>
        <p className="text-xs text-muted-foreground">
          {m.account_api_keys_description()}
        </p>
      </div>

      <Form {...form}>
        <form
          onSubmit={onSubmit}
          className="flex flex-col gap-3 rounded-lg border border-border p-4 sm:flex-row sm:items-end"
        >
          <FormField
            control={form.control}
            name="name"
            render={({ field }) => (
              <FormItem className="flex-1">
                <FormLabel>{m.account_api_keys_create_name_label()}</FormLabel>
                <FormControl>
                  <Input
                    placeholder={m.account_api_keys_create_name_placeholder()}
                    {...field}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <Button type="submit" disabled={createApiKey.isPending}>
            {createApiKey.isPending
              ? m.account_api_keys_create_submitting()
              : m.account_api_keys_create_submit()}
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

      {apiKeysQuery.isError && (
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>
            {getErrorMessage(apiKeysQuery.error)}
          </AlertDescription>
        </Alert>
      )}

      {!apiKeysQuery.isError && (
        <DataTable
          columns={columns}
          data={apiKeys}
          getRowId={(key) => key.id}
          isLoading={apiKeysQuery.isPending}
          emptyState={
            <EmptyState
              icon={KeyRound}
              title={m.account_api_keys_empty_title()}
              description={m.account_api_keys_empty_description()}
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
        title={m.account_api_keys_revoke_confirm_title({
          name: pendingRevoke?.name ?? "",
        })}
        description={m.account_api_keys_revoke_confirm_description()}
        confirmLabel={m.account_api_keys_revoke_action()}
        pending={revokeApiKey.isPending}
        onConfirm={() => {
          if (pendingRevoke) {
            revokeApiKey.mutate({ id: pendingRevoke.id });
          }
        }}
      />

      <OneTimeSecretDialog
        open={revealedSecret !== null}
        onAcknowledge={() => setRevealedSecret(null)}
        title={m.account_api_keys_reveal_title({
          name: revealedSecret?.name ?? "",
        })}
        description={m.account_api_keys_reveal_description()}
      >
        <Code className="w-full justify-between text-sm">
          {revealedSecret?.key ?? ""}
        </Code>
      </OneTimeSecretDialog>
    </section>
  );
}
