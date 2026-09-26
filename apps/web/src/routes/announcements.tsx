import { createFileRoute, redirect } from "@tanstack/react-router";

/** 向后兼容重定向：/announcements -> /app/announcements（沿用旧 router.tsx 的行为）。 */
export const Route = createFileRoute("/announcements")({
  beforeLoad: () => {
    throw redirect({ to: "/app/announcements" });
  },
});
