import { renderToStaticMarkup } from "react-dom/server";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import App from "./App";

describe("App", () => {
  it("renders the foundation dashboard", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const markup = renderToStaticMarkup(
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>,
    );

    expect(markup).toContain("工程控制台");
    expect(markup).toContain("纵向业务切片");
    expect(markup).toContain("契约中心");
    expect(markup).toContain("运营公告");
  });
});
