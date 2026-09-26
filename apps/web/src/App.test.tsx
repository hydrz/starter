import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import { __resetAuthSessionForTests } from "./lib/auth-session";

describe("App", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    __resetAuthSessionForTests();
    window.history.pushState(null, "", "/");
  });

  it("renders the landing page with starter branding and features", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ status: "ok" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(screen.getAllByText("Starter 控制台").length).toBeGreaterThan(0);
      expect(screen.getAllByText("Starter").length).toBeGreaterThan(0);
      expect(screen.getByText("能力模块")).toBeInTheDocument();
      expect(screen.getByText("契约中心 (TypeSpec)")).toBeInTheDocument();
    });
  });

  it("redirects to sign-in when clicking the launch button while signed out", async () => {
    // `/app` 的路由守卫（`routes/app.tsx` `beforeLoad`）会先尝试静默
    // refresh 再确认身份；这里既没有真实的 refresh cookie 也没有内存中的
    // access token，所以无论 mock 返回什么内容，最终都应该落到 /sign-in。
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ status: "ok" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const user = userEvent.setup();
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>,
    );

    const launchButtons = await screen.findAllByRole("link", {
      name: /Starter 控制台/i,
    });
    await user.click(launchButtons[0]);

    await waitFor(() => {
      expect(screen.getByText("登录 Starter 账户以继续。")).toBeInTheDocument();
    });
  });
});
