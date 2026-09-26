import { createFileRoute } from "@tanstack/react-router";
import { SessionsPage } from "../../../features/account/SessionsPage";

export const Route = createFileRoute("/app/account/sessions")({
  component: SessionsPage,
});
