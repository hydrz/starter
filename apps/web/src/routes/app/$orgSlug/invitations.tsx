import { createFileRoute } from "@tanstack/react-router";
import { InvitationsPage } from "../../../features/organizations/InvitationsPage";

export const Route = createFileRoute("/app/$orgSlug/invitations")({
  component: InvitationsPage,
});
