import { Link, useNavigate } from "@tanstack/react-router";
import { AlertTriangle, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useCompleteOAuthSignIn } from "../../api/generated/auth/auth";
import type { OAuthProvider } from "../../api/generated/model";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { getErrorMessage } from "../../lib/errors";
import { setIssuedTokens } from "../../lib/auth-session";
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

  const completeOAuth = useCompleteOAuthSignIn({
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

  useEffect(() => {
    if (!code || !state || requestedRef.current) {
      return;
    }
    requestedRef.current = true;
    completeOAuth.mutate({ provider, data: { code, state } });
    // 仅挂载时触发一次；mutate 引用每次渲染都会变化。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code, state, provider]);

  if (!code || !state) {
    return (
      <AuthLayout title={m.auth_oauth_callback_error_title()}>
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>
            {m.auth_oauth_callback_missing_params()}
          </AlertDescription>
        </Alert>
        <Link to="/sign-in">
          <Button className="mt-5 w-full">
            {m.auth_oauth_callback_retry()}
          </Button>
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
        <Link to="/sign-in">
          <Button className="mt-5 w-full">
            {m.auth_oauth_callback_retry()}
          </Button>
        </Link>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout title={m.auth_oauth_callback_title()}>
      <div className="flex items-center justify-center gap-2 py-6 text-sm text-muted-foreground">
        <Loader2 className="animate-spin" size={16} />
        <span>
          {m.auth_oauth_callback_pending({ provider: providerLabel })}
        </span>
      </div>
    </AuthLayout>
  );
}
