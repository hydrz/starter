import { Link, createFileRoute } from "@tanstack/react-router";
import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { z } from "zod";
import { useConfirmEmailVerification } from "../api/generated/auth/auth";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Button } from "../components/ui/button";
import { AuthLayout } from "../features/auth/AuthLayout";
import { getErrorMessage } from "../lib/errors";
import * as m from "../paraglide/messages";

const searchSchema = z.object({
  token: z.string().optional(),
});

export const Route = createFileRoute("/verify-email")({
  validateSearch: searchSchema,
  component: VerifyEmailPage,
});

function VerifyEmailPage() {
  const { token } = Route.useSearch();
  const [state, setState] = useState<"pending" | "success" | "error">(
    "pending",
  );
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const requestedRef = useRef(false);

  const confirmVerification = useConfirmEmailVerification({
    mutation: {
      onSuccess: () => setState("success"),
      onError: (error) => {
        setErrorMessage(getErrorMessage(error));
        setState("error");
      },
    },
  });

  useEffect(() => {
    if (!token || requestedRef.current) {
      return;
    }
    requestedRef.current = true;
    confirmVerification.mutate({ data: { token } });
    // 仅在挂载时触发一次；`confirmVerification` 每次渲染都是新引用，
    // 不应作为依赖项，否则会不断重复提交验证请求。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  if (!token) {
    return (
      <AuthLayout title={m.auth_verify_email_title()}>
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>
            {m.auth_verify_email_missing_token()}
          </AlertDescription>
        </Alert>
        <Link to="/sign-in">
          <Button className="mt-5 w-full">
            {m.auth_verify_email_go_to_sign_in()}
          </Button>
        </Link>
      </AuthLayout>
    );
  }

  if (state === "pending") {
    return (
      <AuthLayout title={m.auth_verify_email_title()}>
        <div className="flex items-center justify-center gap-2 py-6 text-sm text-muted-foreground">
          <Loader2 className="animate-spin" size={16} />
          <span>{m.auth_verify_email_pending()}</span>
        </div>
      </AuthLayout>
    );
  }

  if (state === "success") {
    return (
      <AuthLayout title={m.auth_verify_email_success_title()}>
        <Alert variant="success">
          <CheckCircle2 />
          <AlertDescription>
            {m.auth_verify_email_success_body()}
          </AlertDescription>
        </Alert>
        <Link to="/sign-in">
          <Button className="mt-5 w-full">
            {m.auth_verify_email_go_to_sign_in()}
          </Button>
        </Link>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout title={m.auth_verify_email_error_title()}>
      <Alert variant="destructive">
        <AlertTriangle />
        <AlertDescription>
          {errorMessage ?? m.auth_verify_email_error_body()}
        </AlertDescription>
      </Alert>
      <Link to="/sign-in">
        <Button className="mt-5 w-full">
          {m.auth_verify_email_go_to_sign_in()}
        </Button>
      </Link>
    </AuthLayout>
  );
}
