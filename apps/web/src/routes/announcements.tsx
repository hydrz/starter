import { createFileRoute, redirect } from "@tanstack/react-router";

/**
 * 向后兼容重定向：/announcements -> /app（沿用旧 router.tsx 的行为）。
 * Stage 3 起公告页是组织范围路由（`/app/$orgSlug/announcements`），
 * 这里没有可用的组织上下文，先落到组织选择/自动跳转页，由用户/`/app`
 * 的单组织自动跳转逻辑决定去哪个组织。
 */
export const Route = createFileRoute("/announcements")({
  beforeLoad: () => {
    throw redirect({ to: "/app" });
  },
});
