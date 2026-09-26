import { zodResolver } from "@hookform/resolvers/zod";
import { AlertTriangle, Check, Fingerprint, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import {
  useBeginOAuthLink,
  useBeginTOTPEnrollment,
  useBeginWebAuthnRegistration,
  useConfirmTOTPEnrollment,
  useFinishWebAuthnRegistration,
  useListOAuthAccounts,
} from "../../api/generated/auth/auth";
import { OAuthProvider } from "../../api/generated/model";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../components/ui/card";
import { Code } from "../../components/ui/code";
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
import { markOAuthLinkPending } from "../../lib/oauth-link";
import * as m from "../../paraglide/messages";
import { OneTimeSecretDialog } from "./OneTimeSecretDialog";
import {
  decodeCreationOptions,
  encodeRegistrationCredential,
} from "./webauthn";

/** `/app/account/security`（DESIGN.md §6.2）：TOTP、Passkey、已关联 OAuth 账号三个区块。 */
export function SecurityPage() {
  return (
    <div className="max-w-2xl space-y-6">
      <TotpCard />
      <PasskeyCard />
      <OAuthAccountsCard />
    </div>
  );
}

// --- TOTP --------------------------------------------------------------

function buildTotpConfirmSchema() {
  return z.object({
    code: z.string().length(6, m.account_security_totp_validation_code()),
  });
}

type TotpConfirmForm = z.infer<ReturnType<typeof buildTotpConfirmSchema>>;

/**
 * TOTP 卡片：本地维护一个"没有查询恢复现有 TOTP 状态"的简化前提——后端目前
 * 没有暴露"当前账户是否已启用 TOTP"的独立查询端点（`useGetCurrentIdentity`
 * 的 `Identity` 类型里也没有这个字段），所以这里始终展示"开始注册"入口；
 * 已经注册过的账户再次点击会走同样的 begin/confirm 流程重新注册一个新的
 * factor，这与后端的幂等语义一致（begin 每次都会创建新 factor）。
 */
function TotpCard() {
  const [factorId, setFactorId] = useState<string | null>(null);
  const [secret, setSecret] = useState<string | null>(null);
  const [otpauthUrl, setOtpauthUrl] = useState<string | null>(null);
  const [recoveryCodes, setRecoveryCodes] = useState<Array<string> | null>(
    null,
  );

  const beginEnrollment = useBeginTOTPEnrollment({
    mutation: {
      onSuccess: (res) => {
        if (res.status === 200) {
          setFactorId(res.data.factorId);
          setSecret(res.data.secret);
          setOtpauthUrl(res.data.otpauthUrl);
        }
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  const schema = buildTotpConfirmSchema();
  const form = useForm<TotpConfirmForm>({
    resolver: zodResolver(schema),
    defaultValues: { code: "" },
  });

  const confirmEnrollment = useConfirmTOTPEnrollment({
    mutation: {
      onSuccess: (res) => {
        if (res.status === 200) {
          setRecoveryCodes(res.data.recoveryCodes);
          setFactorId(null);
          setSecret(null);
          setOtpauthUrl(null);
          form.reset({ code: "" });
        }
      },
      onError: (error) => {
        const message = getErrorMessage(error);
        form.setError("root", { type: "server", message });
        toast.error(message);
      },
    },
  });

  const onSubmit = form.handleSubmit((values) => {
    if (!factorId) {
      return;
    }
    confirmEnrollment.mutate({ data: { factorId, code: values.code } });
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <ShieldCheck size={16} />
          {m.account_security_totp_heading()}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          {m.account_security_totp_description()}
        </p>

        {!factorId && (
          <Button
            type="button"
            onClick={() => beginEnrollment.mutate()}
            disabled={beginEnrollment.isPending}
          >
            {beginEnrollment.isPending
              ? m.account_security_totp_begin_pending()
              : m.account_security_totp_begin_action()}
          </Button>
        )}

        {factorId && secret && (
          <div className="space-y-4 rounded-lg border border-border p-4">
            <div className="space-y-1.5">
              <p className="text-xs font-medium text-foreground">
                {m.account_security_totp_scan_instructions()}
              </p>
              {/* 没有可用的轻量二维码库（package.json 未引入任何 QR 依赖），
                  按 DESIGN.md 要求不为此新增依赖，退化为可复制的密钥文本。 */}
              <Code className="w-full justify-between text-xs" copyable>
                {secret}
              </Code>
              {otpauthUrl && (
                <p className="break-all font-mono text-[11px] text-muted-foreground">
                  {otpauthUrl}
                </p>
              )}
            </div>

            <Form {...form}>
              <form onSubmit={onSubmit} className="flex items-end gap-3">
                <FormField
                  control={form.control}
                  name="code"
                  render={({ field }) => (
                    <FormItem className="flex-1">
                      <FormLabel>
                        {m.account_security_totp_confirm_code_label()}
                      </FormLabel>
                      <FormControl>
                        <Input
                          inputMode="numeric"
                          maxLength={6}
                          className="font-mono"
                          placeholder="123456"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Button type="submit" disabled={confirmEnrollment.isPending}>
                  {confirmEnrollment.isPending
                    ? m.account_security_totp_confirm_submitting()
                    : m.account_security_totp_confirm_submit()}
                </Button>
              </form>
              {form.formState.errors.root && (
                <Alert variant="destructive" className="mt-3">
                  <AlertTriangle />
                  <AlertDescription>
                    {form.formState.errors.root.message}
                  </AlertDescription>
                </Alert>
              )}
            </Form>
          </div>
        )}
      </CardContent>

      <OneTimeSecretDialog
        open={recoveryCodes !== null}
        onAcknowledge={() => setRecoveryCodes(null)}
        title={m.account_security_totp_recovery_title()}
        description={m.account_security_totp_recovery_description()}
      >
        <div className="grid grid-cols-2 gap-2">
          {(recoveryCodes ?? []).map((code) => (
            <Code key={code} className="justify-center text-xs">
              {code}
            </Code>
          ))}
        </div>
      </OneTimeSecretDialog>
    </Card>
  );
}

// --- Passkeys ------------------------------------------------------------

/**
 * Passkey 卡片：真实调用浏览器 WebAuthn API（`navigator.credentials.create`），
 * 不是 mock。沙箱里的无头 Chromium 大概率没有可用的 platform authenticator，
 * 完整注册仪式在这里跑不通——按钮的 loading/错误态、以及
 * begin -> decode -> `navigator.credentials.create` -> encode -> finish
 * 这条链路本身，已经用 `webauthn.test.ts` 的编解码往返测试 +
 * `pnpm --filter @starter/web check` 的类型检查验证过。
 */
function PasskeyCard() {
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const beginRegistration = useBeginWebAuthnRegistration();
  const finishRegistration = useFinishWebAuthnRegistration();

  const isPending = beginRegistration.isPending || finishRegistration.isPending;

  const handleRegister = async () => {
    setError(null);
    setSuccess(false);
    try {
      if (
        typeof navigator === "undefined" ||
        !navigator.credentials ||
        typeof navigator.credentials.create !== "function"
      ) {
        setError(m.account_security_passkey_unsupported());
        return;
      }

      const beginRes = await beginRegistration.mutateAsync();
      if (beginRes.status !== 200) {
        setError(getErrorMessage(beginRes.data));
        return;
      }

      const creationOptions = decodeCreationOptions(beginRes.data.publicKey);
      const credential = await navigator.credentials.create(creationOptions);
      if (!credential) {
        setError(m.account_security_passkey_cancelled());
        return;
      }

      const credentialJson = encodeRegistrationCredential(
        credential as PublicKeyCredential,
      );
      const finishRes = await finishRegistration.mutateAsync({
        data: { credential: credentialJson },
      });
      if (finishRes.status === 204) {
        setSuccess(true);
        toast.success(m.account_security_passkey_register_success());
      } else {
        setError(getErrorMessage(finishRes.data));
      }
    } catch (caught) {
      setError(getErrorMessage(caught));
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Fingerprint size={16} />
          {m.account_security_passkey_heading()}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">
          {m.account_security_passkey_description()}
        </p>

        <Button
          type="button"
          onClick={() => void handleRegister()}
          disabled={isPending}
        >
          {isPending
            ? m.account_security_passkey_register_pending()
            : m.account_security_passkey_register_action()}
        </Button>

        {success && (
          <Alert variant="success">
            <Check />
            <AlertDescription>
              {m.account_security_passkey_register_success()}
            </AlertDescription>
          </Alert>
        )}

        {error && (
          <Alert variant="destructive">
            <AlertTriangle />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
      </CardContent>
    </Card>
  );
}

// --- OAuth accounts --------------------------------------------------------

const linkableProviders = [OAuthProvider.google, OAuthProvider.github] as const;

function providerLabel(provider: OAuthProvider): string {
  switch (provider) {
    case OAuthProvider.google:
      return m.auth_oauth_provider_google();
    case OAuthProvider.github:
      return m.auth_oauth_provider_github();
  }
}

function OAuthAccountsCard() {
  const accountsQuery = useListOAuthAccounts();
  const accounts =
    accountsQuery.data?.status === 200 ? accountsQuery.data.data : [];
  const linkedProviders = new Set(accounts.map((account) => account.provider));

  const beginLink = useBeginOAuthLink({
    mutation: {
      onSuccess: (res, variables) => {
        if (res.status === 200) {
          // 整页导航前留下面包屑，供 `/auth/callback/{provider}` 判定这是
          // 关联流程而非登录流程（见 `lib/oauth-link.ts`）。
          markOAuthLinkPending(variables.provider);
          window.location.assign(res.data.authorizationUrl);
        }
      },
      onError: (error) => toast.error(getErrorMessage(error)),
    },
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">
          {m.account_security_oauth_heading()}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">
          {m.account_security_oauth_description()}
        </p>

        {accountsQuery.isError && (
          <Alert variant="destructive">
            <AlertTriangle />
            <AlertDescription>
              {getErrorMessage(accountsQuery.error)}
            </AlertDescription>
          </Alert>
        )}

        {accountsQuery.isPending && (
          <div className="space-y-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        )}

        {!accountsQuery.isPending && !accountsQuery.isError && (
          <ul className="divide-y divide-border rounded-lg border border-border">
            {linkableProviders.map((provider) => {
              const linkedAccount = accounts.find(
                (account) => account.provider === provider,
              );
              return (
                <li
                  key={provider}
                  className="flex items-center justify-between gap-3 px-4 py-3"
                >
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {providerLabel(provider)}
                    </p>
                    {linkedAccount && (
                      <p className="font-mono text-xs text-muted-foreground">
                        {linkedAccount.email ??
                          m.account_security_oauth_linked_no_email()}
                      </p>
                    )}
                  </div>
                  {linkedAccount ? (
                    <Badge variant="secondary">
                      {m.account_security_oauth_linked_badge()}
                    </Badge>
                  ) : (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={
                        beginLink.isPending || linkedProviders.has(provider)
                      }
                      onClick={() => beginLink.mutate({ provider })}
                    >
                      {m.account_security_oauth_link_action({
                        provider: providerLabel(provider),
                      })}
                    </Button>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
