import { useSyncExternalStore } from "react";
import { useGetCurrentIdentity } from "../api/generated/auth/auth";
import type { AccessTokenResponse } from "../api/generated/model";

/**
 * 访问令牌协调器（access-token coordinator）。
 *
 * 后端的认证威胁模型刻意不让前端持久化 access token 本身：刷新令牌是
 * HttpOnly cookie（浏览器无法读取，也不需要读取），access token 只通过
 * `Authorization: Bearer` 头下发一次，前端必须自行在内存中保存，
 * 绝不写入 localStorage/sessionStorage（那样会让 XSS 直接窃取长期有效的
 * 凭据）。刷新页面/关闭标签页后内存状态丢失是预期行为——由
 * `apps/web/src/api/client.ts` 的静默 refresh（HttpOnly cookie）重新换发。
 *
 * 见 docs/implementation/identity-platform-ledger.md B 节。
 */

interface TokenState {
  accessToken: string | null;
  /** 过期时间（epoch 毫秒），仅供展示/预判使用，实际过期以服务端 401 为准。 */
  expiresAt: number | null;
}

let state: TokenState = { accessToken: null, expiresAt: null };

type Listener = () => void;
const listeners = new Set<Listener>();

function notify() {
  for (const listener of listeners) {
    listener();
  }
}

/** 当前内存中的 access token，供 `api/client.ts` 的 fetch 拦截逻辑读取。 */
export function getAccessToken(): string | null {
  return state.accessToken;
}

export function getAccessTokenExpiry(): number | null {
  return state.expiresAt;
}

/**
 * 每个"完成登录"的流程（密码登录、邮箱验证码登录、MFA 二次验证、
 * OAuth 回调）在拿到 `AccessTokenResponse` 形状的响应后都必须调用本函数，
 * 把新签发的 access token 存入内存。
 */
export function setIssuedTokens(tokens: AccessTokenResponse): void {
  state = {
    accessToken: tokens.accessToken,
    expiresAt: Date.parse(tokens.expiresAt),
  };
  notify();
}

/** 清空内存中的 access token（登出、refresh 失败时调用）。 */
export function clearAccessToken(): void {
  if (state.accessToken === null && state.expiresAt === null) {
    return;
  }
  state = { accessToken: null, expiresAt: null };
  notify();
}

export function subscribeAuthSession(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function getSnapshot(): string | null {
  return state.accessToken;
}

function getServerSnapshot(): string | null {
  return null;
}

/** React 组件内响应式读取当前 access token（有/无），用于门控请求或 UI。 */
export function useAccessToken(): string | null {
  return useSyncExternalStore(
    subscribeAuthSession,
    getSnapshot,
    getServerSnapshot,
  );
}

/**
 * 包装 `useGetCurrentIdentity`：Stage 3/4 的路由守卫将基于此 hook 判断
 * "是否已登录"并决定重定向目标。本阶段只提供一个合理的占位实现——
 * 仅在内存中存在 access token 时才发起请求，避免每个公开页面都对
 * `/api/auth/me` 发一次注定 401 的请求。
 */
export function useCurrentUser() {
  const token = useAccessToken();
  const query = useGetCurrentIdentity({
    query: {
      enabled: Boolean(token),
      retry: false,
      staleTime: 60_000,
    },
  });

  const identity = query.data?.status === 200 ? query.data.data : undefined;

  return {
    ...query,
    identity,
    isAuthenticated: Boolean(identity),
  };
}

/** 仅供测试使用：把协调器重置为初始状态，避免测试之间互相污染。 */
export function __resetAuthSessionForTests(): void {
  state = { accessToken: null, expiresAt: null };
}
