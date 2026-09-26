import { useGetHealth } from "../api/generated/system/system";
import * as m from "../paraglide/messages";

/** 顶栏服务状态徽标，从 `routes/app.tsx` 抽出，供 `routes/app/$orgSlug.tsx` 复用。 */
export function ServiceStatusBadge() {
  const health = useGetHealth({
    query: {
      refetchInterval: 30_000,
      retry: 1,
    },
  });
  const serviceAvailable = Boolean(
    health.data?.data &&
    "status" in health.data.data &&
    health.data.data.status === "ok",
  );

  return (
    <div className="hidden sm:flex items-center gap-2 rounded-full border border-border/60 bg-muted/40 px-3 py-1 text-xs text-muted-foreground">
      <span
        className={`h-2 w-2 rounded-full ${
          serviceAvailable ? "bg-emerald-500 animate-pulse" : "bg-amber-500"
        }`}
      />
      <span>
        {serviceAvailable ? m.status_api_online() : m.status_api_connecting()}
      </span>
    </div>
  );
}
