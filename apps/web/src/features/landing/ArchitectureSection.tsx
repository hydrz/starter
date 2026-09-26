import {
  ArrowRight,
  CheckCircle2,
  Database,
  FileCode2,
  Globe,
  Server,
} from "lucide-react";

export function ArchitectureSection() {
  const steps = [
    {
      step: "01",
      icon: FileCode2,
      name: "契约中心 (TypeSpec)",
      desc: "在 packages/contracts/ 中使用声明式语法定义 RESTful 接口与请求响应数据模型",
      tag: "Single Source of Truth",
    },
    {
      step: "02",
      icon: Globe,
      name: "OpenAPI 3.0 编译",
      desc: "自动编译输出标准 OpenAPI 3.0 规范，并注入 Scalar 离线交互式 API 参考文档",
      tag: "Autonomous Compilation",
    },
    {
      step: "03",
      icon: Server,
      name: "双端代码同步生成",
      desc: "Go 后端通过 oapi-codegen 派生 Server 接口；前端通过 Orval 派生 React Query Hooks",
      tag: "Zero-Drift Code Gen",
    },
    {
      step: "04",
      icon: Database,
      name: "SQL-First 数据持久化",
      desc: "sqlc 解析原生 SQL 生成 Go 数据模型；Goose 管理受控增量数据库表结构迁移",
      tag: "Type-Checked SQL",
    },
  ];

  return (
    <section
      id="architecture"
      className="py-20 border-t border-border/40 bg-muted/20"
    >
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <span className="text-xs font-semibold uppercase tracking-wider text-accent-foreground dark:text-accent">
            ARCHITECTURE PIPELINE
          </span>
          <h2 className="mt-2 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            契约优先的设计与编译管线
          </h2>
          <p className="mt-3 text-base text-muted-foreground">
            拒绝手工维护 API 与重复编写联调代码，以
            <strong className="text-foreground font-semibold">
              {" "}
              契约中心{" "}
            </strong>
            为核心驱动全栈自动化流转。
          </p>
        </div>

        {/* Pipeline Steps Grid */}
        <div className="mt-14 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {steps.map((item, index) => {
            const Icon = item.icon;
            return (
              <div
                key={item.step}
                className="relative flex flex-col justify-between rounded-xl border border-border/80 bg-card p-6 text-card-foreground shadow-xs transition-transform hover:-translate-y-1 dark:bg-card/70"
              >
                <div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-mono font-bold text-muted-foreground/80">
                      {item.step}
                    </span>
                    <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary/10 text-accent-foreground dark:text-accent">
                      <Icon size={16} />
                    </span>
                  </div>
                  <h3 className="mt-4 text-sm font-semibold text-foreground">
                    {item.name}
                  </h3>
                  <p className="mt-2 text-xs leading-relaxed text-muted-foreground">
                    {item.desc}
                  </p>
                </div>

                <div className="mt-5 border-t border-border/60 pt-3">
                  <span className="inline-flex items-center gap-1 text-[10px] font-medium text-emerald-600 dark:text-emerald-400">
                    <CheckCircle2 size={11} />
                    <span>{item.tag}</span>
                  </span>
                </div>

                {index < steps.length - 1 && (
                  <div
                    className="hidden lg:block absolute -right-3 top-1/2 -translate-y-1/2 z-10 text-muted-foreground/40"
                    aria-hidden="true"
                  >
                    <ArrowRight size={14} />
                  </div>
                )}
              </div>
            );
          })}
        </div>

        {/* Technical Flow Visualization Box */}
        <div className="mt-10 rounded-2xl border border-border/80 bg-card p-6 sm:p-8 shadow-xs dark:bg-card/40">
          <div className="flex flex-col sm:flex-row items-center justify-between gap-4 border-b border-border/60 pb-5">
            <div>
              <h4 className="text-sm font-semibold text-foreground">
                单一制品交付流水线 (Single Binary Deployment)
              </h4>
              <p className="mt-0.5 text-xs text-muted-foreground">
                前端静态产物自动构建至 internal/platform/webui/dist，并通过 Go
                embed 打包进单二进制
              </p>
            </div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span className="inline-flex h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
              <span>零 Node.js 运行时</span>
            </div>
          </div>

          <div className="mt-5 grid grid-cols-1 md:grid-cols-3 gap-4 text-xs">
            <div className="rounded-lg border border-border/60 bg-muted/40 p-4">
              <span className="font-semibold text-foreground">
                1. 前端编译打包
              </span>
              <p className="mt-1 text-muted-foreground">
                pnpm build:web 生成高压缩 Vite 生产静态资产
              </p>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/40 p-4">
              <span className="font-semibold text-foreground">
                2. Go 静态内嵌
              </span>
              <p className="mt-1 text-muted-foreground">
                go:embed 将 webui 资源直接嵌入编译结果二进制中
              </p>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/40 p-4">
              <span className="font-semibold text-foreground">
                3. 极速容器运行
              </span>
              <p className="mt-1 text-muted-foreground">
                scratch / alpine 基础镜像轻量封装，Docker Compose 一键启动
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
