import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  RouterProvider,
  createMemoryHistory,
  createRootRoute,
  createRouter,
} from "@tanstack/react-router";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { OrganizationRole } from "../../api/generated/model";
import * as m from "../../paraglide/messages";
import { OrgOverviewPage } from "../overview/OverviewPage";
import { InvitationsPage } from "./InvitationsPage";
import { MembersPage } from "./MembersPage";
import { OrgProvider } from "./OrgContext";

const testOrg = {
  organizationId: "00000000-0000-0000-0000-000000000001",
  slug: "acme",
  name: "Acme",
  role: OrganizationRole.owner,
};

/**
 * 这几个页面本质上是 `/app/$orgSlug/*` 路由树下的组件，正常只能通过一次
 * 完整的登录 + 路由守卫才能到达。沙箱里没有可用的后端，无法走真实登录
 * 流程手动验证"加载态/错误态渲染正常、不留白屏、不抛未捕获异常"，所以
 * 用组件测试模拟同样的三态场景（DESIGN.md §8）：网络失败时展示错误态而
 * 不是白屏或未处理的 Promise rejection。
 */
function renderWithProviders(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <OrgProvider value={testOrg}>{ui}</OrgProvider>
    </QueryClientProvider>,
  );
}

/**
 * `OrgOverviewPage` 用 `<Link>` 做快捷入口，`<Link>` 需要一个真实的
 * router 上下文才能解析 `to`（`useLinkProps` 依赖 `RouterProvider`）。
 * 这里搭一个只有一个根路由的最小内存路由器，只为满足这个依赖，
 * 不测试路由本身。
 */
function renderWithRouterAndProviders(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const rootRoute = createRootRoute({
    component: () => (
      <QueryClientProvider client={queryClient}>
        <OrgProvider value={testOrg}>{ui}</OrgProvider>
      </QueryClientProvider>
    ),
  });
  const router = createRouter({
    routeTree: rootRoute,
    history: createMemoryHistory({ initialEntries: ["/"] }),
  });
  return render(<RouterProvider router={router} />);
}

describe("org-scoped dashboard pages: loading/error three-state", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("MembersPage surfaces a network error instead of a blank page", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("network down"));

    renderWithProviders(<MembersPage />);

    await waitFor(() => {
      expect(screen.getByText("network down")).toBeInTheDocument();
    });
  });

  it("InvitationsPage renders the create-invitation form even while the list is loading", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(
      () => new Promise(() => undefined), // never resolves: stays "loading"
    );

    renderWithProviders(<InvitationsPage />);

    expect(
      await screen.findByRole("button", {
        name: m.invitations_create_submit(),
      }),
    ).toBeInTheDocument();
  });

  it("OrgOverviewPage renders without throwing when member/invitation/announcement requests fail", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(
      async () =>
        new Response(JSON.stringify({ code: "internal", message: "boom" }), {
          status: 500,
          headers: { "Content-Type": "application/json" },
        }),
    );

    renderWithRouterAndProviders(<OrgOverviewPage />);

    expect(await screen.findByText("Acme")).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getAllByText("boom").length).toBeGreaterThan(0);
    });
  });
});
