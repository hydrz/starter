import { createFileRoute } from "@tanstack/react-router";
import { ObservabilityPage } from "../../features/observability/ObservabilityPage";

export const Route = createFileRoute("/app/observability")({
  component: ObservabilityPage,
});
