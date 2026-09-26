import { Outlet, createRootRoute } from "@tanstack/react-router";

/**
 * 全局根布局。文件路由迁移（DESIGN.md §5）后，跨路由的 provider/壳层
 * 挂载点统一放在这里；QueryClientProvider / AppErrorBoundary 仍在
 * `main.tsx` 中挂载（不属于路由树本身）。
 */
export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return <Outlet />;
}
