import {
  clearAccessToken,
  getAccessToken,
  setIssuedTokens,
} from "../lib/auth-session";
import type { AccessTokenResponse } from "./generated/model";

export interface ApiErrorPayload {
  code: string;
  message: string;
}

export class HttpError<T = ApiErrorPayload> extends Error {
  readonly status: number;
  readonly data: T;

  constructor(status: number, data: T, message?: string) {
    const displayMessage =
      message ??
      (typeof data === "object" && data && "message" in data
        ? String((data as { message: unknown }).message)
        : `HTTP request failed with status ${status}`);
    super(displayMessage);
    this.name = "HttpError";
    this.status = status;
    this.data = data;
  }
}

const REFRESH_URL = "/api/auth/refresh";
const SIGN_IN_PATH = "/sign-in";

// 并发 401 去重：多个请求同时过期时，只发起一次 refresh 调用，
// 其余请求共享同一个 in-flight promise（标准的 refresh-token dedupe 模式）。
let refreshInFlight: Promise<boolean> | null = null;

async function performSilentRefresh(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        // 走原生 fetch 而非生成的 `refreshAccessToken` hook：后者从
        // `api/generated/auth/auth.ts` 导入 `customClient`（本文件），
        // 反向导入会形成 client.ts <-> auth.ts 的模块循环引用。
        const res = await fetch(REFRESH_URL, {
          method: "POST",
          credentials: "include",
        });
        if (!res.ok) {
          clearAccessToken();
          return false;
        }
        const data = (await res.json()) as AccessTokenResponse;
        setIssuedTokens(data);
        return true;
      } catch {
        clearAccessToken();
        return false;
      }
    })().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

function redirectToSignIn(): void {
  if (typeof window === "undefined") {
    return;
  }
  if (window.location.pathname === SIGN_IN_PATH) {
    return;
  }
  window.location.assign(SIGN_IN_PATH);
}

function withAuthHeader(headers: HeadersInit | undefined): {
  headers: Headers;
  hadExplicitAuth: boolean;
} {
  const merged = new Headers(headers);
  const hadExplicitAuth = merged.has("Authorization");
  const token = getAccessToken();
  if (token && !hadExplicitAuth) {
    merged.set("Authorization", `Bearer ${token}`);
  }
  return {
    headers: merged,
    hadExplicitAuth: hadExplicitAuth || Boolean(token),
  };
}

async function performFetch(
  url: string,
  options: RequestInit | undefined,
  headers: Headers,
): Promise<Response> {
  return fetch(url, {
    ...options,
    headers,
    // 认证接口（/api/auth/*）依赖 HttpOnly refresh cookie，
    // 所有请求都必须带上 credentials 才能让浏览器发送/接收该 cookie。
    credentials: "include",
  });
}

export const customClient = async <T>(
  url: string,
  options?: RequestInit,
): Promise<T> => {
  const { headers, hadExplicitAuth } = withAuthHeader(options?.headers);
  let res = await performFetch(url, options, headers);

  // 只在"这是一个带着 access token 发出的已认证请求"时才尝试静默刷新，
  // 避免匿名请求（如密码登录时输错密码返回的 401）被误判为会话过期，
  // 从而触发不必要的 refresh 调用甚至把用户重定向离开登录页。
  if (res.status === 401 && hadExplicitAuth) {
    const refreshed = await performSilentRefresh();
    if (refreshed) {
      const retry = withAuthHeader(options?.headers);
      res = await performFetch(url, options, retry.headers);
    } else {
      redirectToSignIn();
    }
  }

  const isNoContent = [204, 205, 304].includes(res.status);
  const text = isNoContent ? null : await res.text();
  let bodyData: unknown = null;
  if (text) {
    try {
      bodyData = JSON.parse(text);
    } catch {
      bodyData = text;
    }
  }

  if (!res.ok) {
    throw new HttpError(res.status, bodyData as ApiErrorPayload);
  }

  return {
    data: bodyData,
    status: res.status,
    headers: res.headers,
  } as T;
};
