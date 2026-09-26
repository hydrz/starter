import { createFileRoute } from "@tanstack/react-router";
import { Announcements } from "../../../features/announcements/Announcements";

export const Route = createFileRoute("/app/$orgSlug/announcements")({
  component: Announcements,
});
