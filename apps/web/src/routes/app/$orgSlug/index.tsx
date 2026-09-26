import { createFileRoute } from "@tanstack/react-router";
import { OrgOverviewPage } from "../../../features/overview/OverviewPage";

export const Route = createFileRoute("/app/$orgSlug/")({
  component: OrgOverviewPage,
});
