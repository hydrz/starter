import type { OAuthProvider } from "../api/generated/model";

/**
 * Disambiguates "OAuth sign-in callback" from "OAuth account-link callback"
 * on `/auth/callback/{google,github}` (DESIGN.md §6.2 security page).
 *
 * Both flows redirect the browser to the *same* provider-registered
 * callback URL and the provider only ever echoes back `code`/`state` — it
 * has no notion of "this was a link, not a sign-in". The backend does know
 * (state carries `intent`/`linking_user_id`, see `internal/auth/oauth.go`),
 * but that's only resolved once we call the matching
 * `/oauth/{provider}/callback` vs `/oauth/{provider}/link/callback`
 * endpoint — the frontend has to pick the right one *before* making that
 * call. So the page that starts a link flow leaves a one-shot breadcrumb in
 * `sessionStorage` right before the full-page redirect, and the callback
 * page consumes (reads + clears) it to decide which mutation to fire.
 * sessionStorage (not localStorage) is deliberate: it's scoped to this tab
 * and this flow can't legitimately span tabs.
 */

const STORAGE_KEY_PREFIX = "starter.oauth_link_pending.";

function storageKey(provider: OAuthProvider): string {
  return `${STORAGE_KEY_PREFIX}${provider}`;
}

/** Call right before redirecting the browser to the provider's authorization URL for a link flow. */
export function markOAuthLinkPending(provider: OAuthProvider): void {
  try {
    sessionStorage.setItem(storageKey(provider), "1");
  } catch {
    // sessionStorage unavailable (private mode, blocked storage, SSR):
    // the callback page falls back to treating this as a sign-in attempt,
    // which fails safely (a link mutation would just also fail).
  }
}

/**
 * Reads and clears the breadcrumb for `provider`. Returns true exactly once
 * per `markOAuthLinkPending` call — a page refresh on the callback route
 * after that won't re-trigger a link attempt.
 */
export function consumeOAuthLinkPending(provider: OAuthProvider): boolean {
  try {
    const key = storageKey(provider);
    const pending = sessionStorage.getItem(key) === "1";
    sessionStorage.removeItem(key);
    return pending;
  } catch {
    return false;
  }
}
