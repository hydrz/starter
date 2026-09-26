import { Link, createFileRoute, useNavigate } from "@tanstack/react-router";
import { AlertTriangle, CheckCircle2 } from "lucide-react";
import { useState } from "react";
import { z } from "zod";
import { useAcceptInvitation } from "../../api/generated/organizations/organizations";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import { AuthLayout } from "../../features/auth/AuthLayout";
import { useCurrentUser } from "../../lib/auth-session";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";

const searchSchema = z.object({
  token: z.string().optional(),
});

export const Route = createFileRoute("/invitations/accept")({
  validateSearch: searchSchema,
  component: AcceptInvitationPage,
});

function AcceptInvitationPage() {
  const { token } = Route.useSearch();
  const navigate = useNavigate();
  const { isAuthenticated, isLoading } = useCurrentUser();
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [succeeded, setSucceeded] = useState(false);

  const acceptInvitation = useAcceptInvitation({
    mutation: {
      onSuccess: () => setSucceeded(true),
      onError: (error) => setErrorMessage(getErrorMessage(error)),
    },
  });

  if (!token) {
    return (
      <AuthLayout title={m.auth_invite_title()}>
        <Alert variant="destructive">
          <AlertTriangle />
          <AlertDescription>{m.auth_invite_missing_token()}</AlertDescription>
        </Alert>
      </AuthLayout>
    );
  }

  if (isLoading) {
    return <AuthLayout title={m.auth_invite_title()} />;
  }

  if (!isAuthenticated) {
    return (
      <AuthLayout
        title={m.auth_invite_sign_in_required_title()}
        subtitle={m.auth_invite_sign_in_required_body()}
      >
        <div className="grid gap-2.5">
          <Link to="/sign-in">
            <Button className="w-full">{m.auth_invite_sign_in_action()}</Button>
          </Link>
          <Link to="/sign-up">
            <Button variant="outline" className="w-full">
              {m.auth_invite_sign_up_action()}
            </Button>
          </Link>
        </div>
      </AuthLayout>
    );
  }

  if (succeeded) {
    return (
      <AuthLayout title={m.auth_invite_success_title()}>
        <Alert variant="success">
          <CheckCircle2 />
          <AlertDescription>{m.auth_invite_success_body()}</AlertDescription>
        </Alert>
        <Button
          className="mt-5 w-full"
          onClick={() => void navigate({ to: "/app" })}
        >
          {m.auth_invite_success_action()}
        </Button>
      </AuthLayout>
    );
  }

  return (
    <AuthLayout
      title={m.auth_invite_title()}
      subtitle={m.auth_invite_subtitle()}
    >
      {errorMessage ? (
        <Alert variant="destructive" className="mb-4">
          <AlertTriangle />
          <AlertDescription>{errorMessage}</AlertDescription>
        </Alert>
      ) : null}

      <Button
        className="w-full"
        disabled={acceptInvitation.isPending}
        onClick={() => acceptInvitation.mutate({ data: { token } })}
      >
        {acceptInvitation.isPending
          ? m.auth_invite_accept_submitting()
          : errorMessage
            ? m.auth_invite_retry()
            : m.auth_invite_accept_submit()}
      </Button>
    </AuthLayout>
  );
}
