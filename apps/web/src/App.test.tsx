import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App";

describe("App", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
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

  it("navigates to the app console when clicking launch button", async () => {
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
      expect(screen.getByText("概览看板")).toBeInTheDocument();
      expect(screen.getByText("公告管理")).toBeInTheDocument();
      expect(screen.getByText("系统观测")).toBeInTheDocument();
    });
  });
});
