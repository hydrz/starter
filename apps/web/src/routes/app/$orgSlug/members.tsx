import { createFileRoute } from "@tanstack/react-router";
import { MembersPage } from "../../../features/organizations/MembersPage";

export const Route = createFileRoute("/app/$orgSlug/members")({
  component: MembersPage,
});
