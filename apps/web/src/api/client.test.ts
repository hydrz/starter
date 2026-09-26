import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  __resetAuthSessionForTests,
  getAccessToken,
  setIssuedTokens,
} from "../lib/auth-session";
import { customClient, HttpError } from "./client";

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("customClient (access-token coordinator)", () => {
  beforeEach(() => {
    __resetAuthSessionForTests();
    vi.restoreAllMocks();
  });

  afterEach(() => {
    __resetAuthSessionForTests();
  });

  it("does not attach an Authorization header when there is no token", async () => {
    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(jsonResponse(200, { ok: true }));

    await customClient("/api/system/health");

    expect(fetchSpy).toHaveBeenCalledTimes(1);
    const [, init] = fetchSpy.mock.calls[0];
    const headers = new Headers(init?.headers);
    expect(headers.has("Authorization")).toBe(false);
    expect(init?.credentials).toBe("include");
  });

  it("attaches Authorization: Bearer <token> once a token is stored", async () => {
    setIssuedTokens({
      accessToken: "token-abc",
      tokenType: "Bearer",
      expiresAt: new Date(Date.now() + 60_000).toISOString(),
    });

    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(jsonResponse(200, { ok: true }));

    await customClient("/api/auth/me");

    const [, init] = fetchSpy.mock.calls[0];
    const headers = new Headers(init?.headers);
    expect(headers.get("Authorization")).toBe("Bearer token-abc");
  });

  it("on a 401 from an authenticated call, refreshes once and retries with the new token", async () => {
    setIssuedTokens({
      accessToken: "expired-token",
      tokenType: "Bearer",
      expiresAt: new Date(Date.now() - 1_000).toISOString(),
    });

    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        jsonResponse(401, { code: "unauthorized", message: "expired" }),
      )
      .mockResolvedValueOnce(
        jsonResponse(200, {
          accessToken: "fresh-token",
          tokenType: "Bearer",
          expiresAt: new Date(Date.now() + 60_000).toISOString(),
        }),
      )
      .mockResolvedValueOnce(jsonResponse(200, { data: "ok" }));

    const result = await customClient<{ data: unknown }>("/api/auth/me");

    expect(fetchSpy).toHaveBeenCalledTimes(3);
    expect(fetchSpy.mock.calls[1][0]).toBe("/api/auth/refresh");
    const retryHeaders = new Headers(fetchSpy.mock.calls[2][1]?.headers);
    expect(retryHeaders.get("Authorization")).toBe("Bearer fresh-token");
    expect(getAccessToken()).toBe("fresh-token");
    expect(result.data).toEqual({ data: "ok" });
  });

  it("dedupes concurrent 401s into a single refresh call", async () => {
    setIssuedTokens({
      accessToken: "expired-token",
      tokenType: "Bearer",
      expiresAt: new Date(Date.now() - 1_000).toISOString(),
    });

    let refreshCalls = 0;
    const fetchSpy = vi
      .spyOn(globalThis, "fetch")
      .mockImplementation(
        async (input: RequestInfo | URL, init?: RequestInit) => {
          const url = typeof input === "string" ? input : input.toString();
          if (url === "/api/auth/refresh") {
            refreshCalls += 1;
            return jsonResponse(200, {
              accessToken: "fresh-token",
              tokenType: "Bearer",
              expiresAt: new Date(Date.now() + 60_000).toISOString(),
            });
          }
          // First call per request path returns 401, retries (with the
          // refreshed token) succeed.
          const headers = new Headers(init?.headers);
          if (headers.get("Authorization") === "Bearer expired-token") {
            return jsonResponse(401, {
              code: "unauthorized",
              message: "expired",
            });
          }
          return jsonResponse(200, { data: "ok" });
        },
      );

    const [a, b] = await Promise.all([
      customClient("/api/resource/a"),
      customClient("/api/resource/b"),
    ]);

    expect(refreshCalls).toBe(1);
    expect((a as { data: unknown }).data).toEqual({ data: "ok" });
    expect((b as { data: unknown }).data).toEqual({ data: "ok" });
    fetchSpy.mockRestore();
  });

  it("clears the token and redirects to /sign-in when refresh itself fails", async () => {
    setIssuedTokens({
      accessToken: "expired-token",
      tokenType: "Bearer",
      expiresAt: new Date(Date.now() - 1_000).toISOString(),
    });

    vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        jsonResponse(401, { code: "unauthorized", message: "expired" }),
      )
      .mockResolvedValueOnce(
        jsonResponse(401, { code: "unauthorized", message: "refresh failed" }),
      );

    const assignSpy = vi
      .spyOn(window.location, "assign")
      .mockImplementation(() => undefined);

    await expect(customClient("/api/auth/me")).rejects.toBeInstanceOf(
      HttpError,
    );

    expect(getAccessToken()).toBeNull();
    expect(assignSpy).toHaveBeenCalledWith("/sign-in");
  });

  it("does not attempt a refresh for an anonymous 401 (e.g. wrong sign-in password)", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      jsonResponse(401, {
        code: "invalid_credentials",
        message: "wrong password",
      }),
    );

    await expect(customClient("/api/auth/sign-in")).rejects.toBeInstanceOf(
      HttpError,
    );

    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });
});
