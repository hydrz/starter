import { Link, useNavigate } from "@tanstack/react-router";
import { AlertTriangle, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import {
  useCompleteOAuthLink,
  useCompleteOAuthSignIn,
} from "../../api/generated/auth/auth";
import type { OAuthProvider } from "../../api/generated/model";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { toast } from "../../components/ui/toast";
import { getErrorMessage } from "../../lib/errors";
import { getAccessToken, setIssuedTokens } from "../../lib/auth-session";
import { performSilentRefresh } from "../../api/client";
import { consumeOAuthLinkPending } from "../../lib/oauth-link";
import * as m from "../../paraglide/messages";
import { AuthLayout } from "./AuthLayout";

interface OAuthCallbackPageProps {
  provider: OAuthProvider;
  providerLabel: string;
  code?: string;
  state?: string;
}

/**
 * `/auth/callback/google` 与 `/auth/callback/github` 共享的实现。
 * 两个 provider 除了 provider 常量与展示文案外行为完全一致，避免重复代码。
 *
 * 同时承担两种流程（DESIGN.md §6.2"已关联 OAuth 账号"）：未登录的
 * "使用 OAuth 登录"（`useCompleteOAuthSignIn`）与已登录状态下从
 * `/app/account/security` 发起的"关联 OAuth 账号"（`useCompleteOAuthLink`）。
 * provider 回调本身无法区分这两种意图（见 `lib/oauth-link.ts` 的说明），
 * 这里在挂载时读一次 `consumeOAuthLinkPending` 决定分支。
 */
export function OAuthCallbackPage({
  provider,
  providerLabel,
  code,
  state,
}: OAuthCallbackPageProps) {
  const navigate = useNavigate();
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const requestedRef = useRef(false);
  // 只在挂载时读一次：这是一次性消费（读了就清），不能在重渲染时重复判断。
  const [isLinking] = useState(() => consumeOAuthLinkPending(provider));

  const completeOAuthSignIn = useCompleteOAuthSignIn({
    mutation: {
      onSuccess: (res) => {
        if (res.status === 200) {
          setIssuedTokens(res.data);
          void navigate({ to: "/app" });
          return;
        }
        if (res.status === 202) {
          void navigate({
            to: "/mfa",
            search: { challengeId: res.data.challengeId },
          });
        }
      },
      onError: (error) => setErrorMessage(getErrorMessage(error)),
    },
  });

  const completeOAuthLink = useCompleteOAuthLink({
    mutation: {
      onSuccess: (res) => {
        if (res.status === 204) {
          toast.success(
            m.account_security_oauth_link_success({ provider: providerLabel }),
          );
          void navigate({ to: "/app/account/security" });
        }
      },
      onError: (error) => setErrorMessage(getErrorMessage(error)),
    },
  });

  useEffect(() => {
    if (!code || !state || requestedRef.current) {
      return;
    }
    requestedRef.current = true;

    if (isLinking) {
      void (async () => {
        // 关联流程要求已登录：跳转到 provider 授权页是整页导航，
        // access-token 协调器的内存态会被清空，回跳回来时必须先静默
        // refresh 一次（HttpOnly cookie）才能重新拿到 Authorization 头，
        // 否则 `completeOAuthLink` 会被后端当成未登录请求拒绝。
        if (!getAccessToken()) {
          await performSilentRefresh();
        }
        completeOAuthLink.mutate({ provider, data: { code, state } });
      })();
      return;
    }

    completeOAuthSignIn.mutate({ provider, data: { code, state } });
    // 仅挂载时触发一次；mutate 引用每次渲染都会变化。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code, state, provider, isLinking]);

  const retryTo = isLinking ? "/app/account/security" : "/sign-in";
  const retryLabel = isLinking
    ? m.account_security_oauth_link_retry()
    : m.auth_oauth_callback_retry();

  if (!code || !state) {
    return (
      <AuthLayout title={m.auth_oauth_callback_error_title()}>
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>
            {m.auth_oauth_callback_missing_params()}
          </AlertDescription>
        </Alert>
        <Link to={retryTo}>
          <Button className="mt-5 w-full">{retryLabel}</Button>
        </Link>
      </AuthLayout>
    );
  }

  if (errorMessage) {
    return (
      <AuthLayout title={m.auth_oauth_callback_error_title()}>
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>{errorMessage}</AlertDescription>
        </Alert>
        <Link to={retryTo}>
          <Button className="mt-5 w-full">{retryLabel}</Button>
        </Link>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout title={m.auth_oauth_callback_title()}>
      <div className="flex items-center justify-center gap-2 py-6 text-sm text-muted-foreground">
        <Loader2 className="animate-spin" size={16} />
        <span>
          {isLinking
            ? m.account_security_oauth_link_pending({ provider: providerLabel })
            : m.auth_oauth_callback_pending({ provider: providerLabel })}
        </span>
      </div>
    </AuthLayout>
  );
}
