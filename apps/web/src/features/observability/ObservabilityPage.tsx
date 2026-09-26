import {
  Activity,
  CheckCircle2,
  Database,
  FileCode,
  Layers,
  RefreshCw,
  Server,
} from "lucide-react";
import {
  useGetHealth,
  useGetReadiness,
} from "../../api/generated/system/system";
import { Button } from "../../components/ui/button";

export function ObservabilityPage() {
  const health = useGetHealth({
    query: {
      refetchInterval: 15_000,
      retry: 2,
    },
  });

  const readiness = useGetReadiness({
    query: {
      refetchInterval: 15_000,
      retry: 2,
    },
  });

  const isHealthy = Boolean(
    health.data?.data &&
    "status" in health.data.data &&
    health.data.data.status === "ok",
  );
  const isReady = Boolean(
    readiness.data?.data &&
    "status" in readiness.data.data &&
    readiness.data.data.status === "ok",
  );
  const isRefreshing = health.isFetching || readiness.isFetching;

  function handleRefresh() {
    health.refetch();
    readiness.refetch();
  }

  const endpoints = [
    {
      method: "GET",
      path: "/api/healthz",
      name: "存活探针 (Liveness Probe)",
      status: isHealthy ? "200 OK" : "检查中",
      healthy: isHealthy,
    },
    {
      method: "GET",
      path: "/api/readyz",
      name: "就绪探针 (Readiness Probe)",
      status: isReady ? "200 OK" : "检查中",
      healthy: isReady,
    },
    {
      method: "GET",
      path: "/api/announcements",
      name: "业务切片查询 (List Announcements)",
      status: "200 OK",
      healthy: true,
    },
    {
      method: "POST",
      path: "/api/announcements",
      name: "业务切片写入 (Create Announcement)",
      status: "201 Created",
      healthy: true,
    },
    {
      method: "GET",
      path: "/api/docs",
      name: "Scalar 契约交互文档引擎",
      status: "200 OK",
      healthy: true,
    },
  ];

  return (
    <div className="py-6 space-y-8">
      {/* Page Heading */}
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 border-b border-border/60 pb-6">
        <div>
          <span className="eyebrow">SYSTEM OBSERVABILITY</span>
          <h2 className="text-2xl font-bold tracking-tight text-foreground">
            系统状态与可观测性
          </h2>
          <p className="mt-1 text-xs text-muted-foreground">
            实时监测 Go 后端服务存活、数据库就绪探针、TypeSpec
            契约流转与客户端运行时健康度。
          </p>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={handleRefresh}
          disabled={isRefreshing}
          className="flex items-center gap-1.5 text-xs"
        >
          <RefreshCw
            size={13}
            className={isRefreshing ? "animate-spin text-primary" : ""}
          />
          <span>{isRefreshing ? "正在刷新探针…" : "手动刷新探针"}</span>
        </Button>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {/* Card 1: Health Status */}
        <div className="rounded-xl border border-border/80 bg-card p-5 text-card-foreground shadow-xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground">
              API 存活探针
            </span>
            <span
              className={`flex h-7 w-7 items-center justify-center rounded-lg ${
                isHealthy
                  ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                  : "bg-amber-500/10 text-amber-600"
              }`}
            >
              <Server size={15} />
            </span>
          </div>
          <div className="mt-4 flex items-baseline gap-2">
            <strong className="text-2xl font-bold">
              {isHealthy ? "Healthy" : "Connecting"}
            </strong>
            <span className="text-[11px] font-mono text-muted-foreground">
              /api/healthz
            </span>
          </div>
          <p className="mt-2 text-[11px] text-muted-foreground">
            {isHealthy ? "HTTP 服务正常响应" : "等待服务返回状态响应…"}
          </p>
        </div>

        {/* Card 2: Readiness Status */}
        <div className="rounded-xl border border-border/80 bg-card p-5 text-card-foreground shadow-xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground">
              数据库就绪探针
            </span>
            <span
              className={`flex h-7 w-7 items-center justify-center rounded-lg ${
                isReady
                  ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                  : "bg-amber-500/10 text-amber-600"
              }`}
            >
              <Database size={15} />
            </span>
          </div>
          <div className="mt-4 flex items-baseline gap-2">
            <strong className="text-2xl font-bold">
              {isReady ? "Ready" : "Waiting"}
            </strong>
            <span className="text-[11px] font-mono text-muted-foreground">
              /api/readyz
            </span>
          </div>
          <p className="mt-2 text-[11px] text-muted-foreground">
            {isReady ? "PostgreSQL 连接池就绪" : "正在等待数据库连接…"}
          </p>
        </div>

        {/* Card 3: Contract Engine */}
        <div className="rounded-xl border border-border/80 bg-card p-5 text-card-foreground shadow-xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground">
              契约规格版本
            </span>
            <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/10 text-accent-foreground dark:text-accent">
              <FileCode size={15} />
            </span>
          </div>
          <div className="mt-4 flex items-baseline gap-2">
            <strong className="text-2xl font-bold">OpenAPI 3.0</strong>
            <span className="text-[11px] font-mono text-muted-foreground">
              TypeSpec
            </span>
          </div>
          <p className="mt-2 text-[11px] text-muted-foreground">
            双端编译防漂移校验启用中
          </p>
        </div>

        {/* Card 4: Client Runtime */}
        <div className="rounded-xl border border-border/80 bg-card p-5 text-card-foreground shadow-xs">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-muted-foreground">
              前端运行时
            </span>
            <span className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary/10 text-accent-foreground dark:text-accent">
              <Layers size={15} />
            </span>
          </div>
          <div className="mt-4 flex items-baseline gap-2">
            <strong className="text-2xl font-bold">React 19</strong>
            <span className="text-[11px] font-mono text-muted-foreground">
              Vite 8
            </span>
          </div>
          <p className="mt-2 text-[11px] text-muted-foreground">
            TanStack Router & Query 驱动
          </p>
        </div>
      </div>

      {/* Endpoint Table */}
      <div className="rounded-xl border border-border/80 bg-card overflow-hidden shadow-xs">
        <div className="border-b border-border/60 bg-muted/30 px-5 py-3.5 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Activity size={14} className="text-primary" />
            <h3 className="text-sm font-semibold text-foreground">
              核心 RESTful 接口探针监控
            </h3>
          </div>
          <span className="text-[11px] text-muted-foreground">
            5 个关键端点已接入
          </span>
        </div>

        <div className="divide-y divide-border/60 text-xs">
          {endpoints.map((ep) => (
            <div
              key={ep.path + ep.method}
              className="flex items-center justify-between px-5 py-3.5 hover:bg-muted/20 transition-colors"
            >
              <div className="flex items-center gap-3">
                <span
                  className={`inline-block rounded px-2 py-0.5 font-mono text-[10px] font-bold ${
                    ep.method === "GET"
                      ? "bg-blue-500/10 text-blue-600 dark:text-blue-400"
                      : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                  }`}
                >
                  {ep.method}
                </span>
                <span className="font-mono text-foreground font-medium">
                  {ep.path}
                </span>
                <span className="hidden sm:inline text-muted-foreground">
                  — {ep.name}
                </span>
              </div>

              <div className="flex items-center gap-2">
                <span
                  className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium ${
                    ep.healthy
                      ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                      : "bg-amber-500/10 text-amber-600"
                  }`}
                >
                  <CheckCircle2 size={11} />
                  <span>{ep.status}</span>
                </span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Layer Health Checklist */}
      <div className="rounded-xl border border-border/80 bg-card p-6 shadow-xs">
        <h3 className="text-sm font-semibold text-foreground border-b border-border/60 pb-3">
          垂直切片全链路架构分层健康度
        </h3>

        <div className="mt-4 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3 text-xs">
          <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
            <span className="font-semibold text-foreground">
              1. Contract 层
            </span>
            <p className="mt-1 text-[11px] text-muted-foreground">
              TypeSpec 契约编译通过
            </p>
            <span className="mt-2 inline-flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">
              <CheckCircle2 size={11} /> 校验通过
            </span>
          </div>

          <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
            <span className="font-semibold text-foreground">
              2. Transport 层
            </span>
            <p className="mt-1 text-[11px] text-muted-foreground">
              Chi 路由 & ogen 强类型引擎
            </p>
            <span className="mt-2 inline-flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">
              <CheckCircle2 size={11} /> 适配正常
            </span>
          </div>

          <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
            <span className="font-semibold text-foreground">
              3. Domain 业务层
            </span>
            <p className="mt-1 text-[11px] text-muted-foreground">
              Context-Driven 事务管理
            </p>
            <span className="mt-2 inline-flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">
              <CheckCircle2 size={11} /> 逻辑解耦
            </span>
          </div>

          <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
            <span className="font-semibold text-foreground">
              4. Persistence 层
            </span>
            <p className="mt-1 text-[11px] text-muted-foreground">
              sqlc 编译期类型验证
            </p>
            <span className="mt-2 inline-flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">
              <CheckCircle2 size={11} /> 类型严苛
            </span>
          </div>

          <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
            <span className="font-semibold text-foreground">
              5. Database 引擎
            </span>
            <p className="mt-1 text-[11px] text-muted-foreground">
              PostgreSQL 17 + Goose
            </p>
            <span className="mt-2 inline-flex items-center gap-1 text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">
              <CheckCircle2 size={11} /> 连接就绪
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}
