import { createFileRoute } from "@tanstack/react-router";
import { SettingsPage } from "../../../features/organizations/SettingsPage";

export const Route = createFileRoute("/app/$orgSlug/settings")({
  component: SettingsPage,
});
