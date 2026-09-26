import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";
import { performSilentRefresh } from "../api/client";
import { getGetCurrentIdentityQueryOptions } from "../api/generated/auth/auth";
import { getAccessToken } from "../lib/auth-session";
import { isUnauthenticatedError } from "../lib/errors";
import { queryClient } from "../lib/query";
import * as m from "../paraglide/messages";

/**
 * `/app` 路由树的认证守卫（DESIGN.md §6.2，本文件不再渲染 `AppShell`——
 * 壳层下沉到 `routes/app/$orgSlug.tsx`，因为侧边栏导航/组织切换器需要
 * 已解析的组织上下文，而 `/app`（组织选择）本身还没有组织）。
 *
 * `beforeLoad` 对整个 `/app/*` 子树只跑一次（父路由先于子路由匹配）：
 * - access-token 协调器只在内存中保存 token，刷新页面后内存状态丢失，
 *   所以先尝试一次静默 refresh（HttpOnly cookie）判断会话是否仍然有效，
 *   而不是直接放行导致下游组件对着一个"注定 401"的状态渲染。
 * - 静默 refresh 后内存里仍然没有 token，说明确实未登录（无论是真的没有
 *   会话，还是后端不可达都无法建立已认证状态），直接跳转登录页，不发起
 *   任何业务请求。
 * - token 存在时，再用 `useGetCurrentIdentity` 对应的 query 确认这个
 *   token 仍然有效；仅在明确收到 401（`HttpError`）时才跳转登录页，
 *   其余错误（网络错误等）继续抛出，交给 `AppErrorBoundary`，
 *   不用一个笼统的 try/catch 把"后端暂时不可用"误判成"未登录"。
 */
export const Route = createFileRoute("/app")({
  beforeLoad: async () => {
    if (!getAccessToken()) {
      await performSilentRefresh();
    }

    if (!getAccessToken()) {
      throw redirect({ to: "/sign-in" });
    }

    try {
      await queryClient.ensureQueryData(getGetCurrentIdentityQueryOptions());
    } catch (error) {
      if (isUnauthenticatedError(error)) {
        throw redirect({ to: "/sign-in" });
      }
      throw error;
    }
  },
  pendingComponent: AppRoutePending,
  component: () => <Outlet />,
});

function AppRoutePending() {
  return (
    <div className="flex min-h-screen items-center justify-center gap-2 text-sm text-muted-foreground">
      <Loader2 className="h-4 w-4 animate-spin" />
      <span>{m.app_session_loading()}</span>
    </div>
  );
}
