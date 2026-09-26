import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { OAuthProvider } from "../../../api/generated/model";
import { OAuthCallbackPage } from "../../../features/auth/OAuthCallbackPage";
import * as m from "../../../paraglide/messages";

const searchSchema = z.object({
  code: z.string().optional(),
  state: z.string().optional(),
});

export const Route = createFileRoute("/auth/callback/google")({
  validateSearch: searchSchema,
  component: GoogleCallbackPage,
});

function GoogleCallbackPage() {
  const { code, state } = Route.useSearch();
  return (
    <OAuthCallbackPage
      provider={OAuthProvider.google}
      providerLabel={m.auth_oauth_provider_google()}
      code={code}
      state={state}
    />
  );
}
