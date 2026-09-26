import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { OAuthProvider } from "../../../api/generated/model";
import { OAuthCallbackPage } from "../../../features/auth/OAuthCallbackPage";
import * as m from "../../../paraglide/messages";

const searchSchema = z.object({
  code: z.string().optional(),
  state: z.string().optional(),
});

export const Route = createFileRoute("/auth/callback/github")({
  validateSearch: searchSchema,
  component: GithubCallbackPage,
});

function GithubCallbackPage() {
  const { code, state } = Route.useSearch();
  return (
    <OAuthCallbackPage
      provider={OAuthProvider.github}
      providerLabel={m.auth_oauth_provider_github()}
      code={code}
      state={state}
    />
  );
}
