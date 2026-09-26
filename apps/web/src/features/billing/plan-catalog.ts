/**
 * Client-side plan catalog for the billing page's checkout buttons
 * (DESIGN.md §6.2 `/app/$orgSlug/billing`).
 *
 * `priceKey` is a server-owned, deployment-specific opaque string (see
 * `internal/billing/catalog.go`'s `Catalog`/`CatalogEntry` and
 * `docs/adr/0006-stripe-payments-subscriptions-and-entitlements.md`). The
 * backend deliberately exposes NO "list available plans" endpoint — the
 * real mapping from price key to Stripe price ID/mode/feature key lives in
 * the `BILLING_PRICE_CATALOG` server config, set per deployment and never
 * returned to clients.
 *
 * This file is therefore UI copy ONLY, not fetched data: a small, readable
 * list of the price keys this page offers as checkout buttons, with a
 * display name and a human price string. These entries MUST be kept in
 * sync by hand with whatever `BILLING_PRICE_CATALOG` the deployment
 * actually configures — there is no way to verify that from the frontend
 * alone (clicking a button for a priceKey the deployment hasn't configured
 * just surfaces the backend's "unknown price key" error via the normal
 * `ApiError` path, it doesn't corrupt anything).
 *
 * `pro_monthly`/`pro_yearly` below are placeholders matching the example
 * key named in `internal/billing/catalog.go`'s doc comment — replace them
 * (and the catalog) with your deployment's real plan lineup.
 */
export interface PlanCatalogEntry {
  priceKey: string;
  name: string;
  priceDisplay: string;
  description: string;
}

export const PLAN_CATALOG: Array<PlanCatalogEntry> = [
  {
    priceKey: "pro_monthly",
    name: "Pro",
    priceDisplay: "$29 / mo",
    description: "Billed monthly, cancel anytime.",
  },
  {
    priceKey: "pro_yearly",
    name: "Pro (annual)",
    priceDisplay: "$290 / yr",
    description: "Two months free compared to monthly billing.",
  },
];
