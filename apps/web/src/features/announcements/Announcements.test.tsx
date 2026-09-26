import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { Announcements } from "./Announcements";

describe("Announcements feature", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });
    vi.restoreAllMocks();
  });

  function renderFeature() {
    return render(
      <QueryClientProvider client={queryClient}>
        <Announcements />
      </QueryClientProvider>,
    );
  }

  it("renders empty state when there are no announcements", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          items: [],
          total: 0,
          limit: 10,
          offset: 0,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    renderFeature();

    expect(screen.getByText("正在加载公告…")).toBeInTheDocument();
    await waitFor(() => {
      expect(
        screen.getByText("还没有公告，请创建第一条。"),
      ).toBeInTheDocument();
    });
  });

  it("renders list items when announcements exist", async () => {
    const mockItems = [
      {
        id: "11111111-1111-1111-1111-111111111111",
        title: "系统升级通知",
        content: "今晚进行数据库迁移",
        status: "published",
        createdAt: "2026-09-25T10:00:00Z",
        updatedAt: "2026-09-25T10:00:00Z",
      },
      {
        id: "22222222-2222-2222-2222-222222222222",
        title: "草稿通知",
        content: "待定内容",
        status: "draft",
        createdAt: "2026-09-25T11:00:00Z",
        updatedAt: "2026-09-25T11:00:00Z",
      },
    ];

    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          items: mockItems,
          total: 2,
          limit: 10,
          offset: 0,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    renderFeature();

    await waitFor(() => {
      expect(screen.getByText("系统升级通知")).toBeInTheDocument();
      expect(screen.getByText("今晚进行数据库迁移")).toBeInTheDocument();
      expect(screen.getByText("已发布")).toBeInTheDocument();
      expect(screen.getByText("草稿通知")).toBeInTheDocument();
      expect(screen.getByText("草稿")).toBeInTheDocument();
    });
  });

  it("shows validation errors when submitting empty form", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({ items: [], total: 0, limit: 10, offset: 0 }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );

    const user = userEvent.setup();
    renderFeature();

    const submitBtn = screen.getByRole("button", { name: "创建公告" });
    await user.click(submitBtn);

    await waitFor(() => {
      expect(screen.getByText("请输入公告标题")).toBeInTheDocument();
      expect(screen.getByText("请输入公告内容")).toBeInTheDocument();
    });
  });

  it("handles API error responses and displays server error message", async () => {
    vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ items: [], total: 0, limit: 10, offset: 0 }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            code: "invalid_request",
            message: "标题长度不符合要求",
          }),
          { status: 400, headers: { "Content-Type": "application/json" } },
        ),
      );

    const user = userEvent.setup();
    renderFeature();

    await user.type(
      screen.getByPlaceholderText("例如：计划维护通知"),
      "测试公告",
    );
    await user.type(
      screen.getByPlaceholderText("填写需要发布的内容"),
      "测试内容详情",
    );

    const submitBtn = screen.getByRole("button", { name: "创建公告" });
    await user.click(submitBtn);

    await waitFor(() => {
      expect(screen.getByText("标题长度不符合要求")).toBeInTheDocument();
    });
  });
});
