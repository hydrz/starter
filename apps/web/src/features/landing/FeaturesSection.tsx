import {
  Activity,
  Code2,
  Cpu,
  Database,
  Layers,
  ShieldCheck,
} from "lucide-react";

export function FeaturesSection() {
  const features = [
    {
      icon: Code2,
      badge: "Contract-First",
      title: "契约驱动与单一真理源",
      description:
        "基于 TypeSpec 声明式定义 HTTP API 与数据模型。编译期自动产出 OpenAPI 3.0，并驱动两端代码生成，杜绝文档与实现脱节。",
    },
    {
      icon: Cpu,
      badge: "High Performance",
      title: "高性能 Go 模块化后端",
      description:
        "Go 1.27 + Chi 路由，采用 Context-Driven Transactor 事务模式与严格清晰的分层架构，支持高并发并便于单体向分布式演进。",
    },
    {
      icon: Layers,
      badge: "Modern Frontend",
      title: "现代化 React 19 客户端",
      description:
        "采用 React 19、Vite 与 TanStack Router（类型安全代码级路由）、TanStack Query 缓存、Tailwind CSS v4 与 shadcn/ui 组件范式。",
    },
    {
      icon: ShieldCheck,
      badge: "End-to-End Safety",
      title: "端到端全链路类型安全",
      description:
        "后端的改动即刻在前端 Orval 自动生成的 React Query Hooks 与 TypeScript 类型上生效，静态编译期捕获字段不匹配错误。",
    },
    {
      icon: Database,
      badge: "SQL-First Data",
      title: "SQL-First 强类型数据层",
      description:
        "拒绝黑盒 ORM。基于显式 SQL 与 Goose 版本化迁移，sqlc 在编译阶段严格验证 SQL 语法与类型，pgx 原生连接池保障吞吐。",
    },
    {
      icon: Activity,
      badge: "Single Binary Delivery",
      title: "单二进制与极简运维",
      description:
        "Vite 构建的前端静态资源通过 go:embed 直接内嵌进单一 Go 可执行文件。生产环境零 Node.js 运行时依赖，一个文件随处运行。",
    },
  ];

  return (
    <section id="features" className="py-20 border-t border-border/40">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <span className="text-xs font-semibold uppercase tracking-wider text-accent-foreground dark:text-accent">
            CAPABILITIES & ARCHITECTURE
          </span>
          <h2 className="mt-2 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            全栈核心能力与工程基线
          </h2>
          <p className="mt-3 text-base text-muted-foreground">
            涵盖设计、编码、测试到打包交付的全生命周期最佳实践与
            <strong className="text-foreground font-semibold">能力模块</strong>
            。
          </p>
        </div>

        <div className="mt-14 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => {
            const Icon = feature.icon;
            return (
              <div
                key={feature.title}
                className="group relative rounded-2xl border border-border/80 bg-card p-7 text-card-foreground shadow-xs transition-all hover:border-border hover:shadow-md dark:bg-card/60 dark:hover:bg-card"
              >
                <div className="flex items-center justify-between">
                  <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary/10 text-accent-foreground dark:text-accent transition-colors group-hover:bg-primary group-hover:text-primary-foreground">
                    <Icon size={20} />
                  </span>
                  <span className="rounded-full bg-muted/80 px-2.5 py-0.5 text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
                    {feature.badge}
                  </span>
                </div>
                <h3 className="mt-5 text-lg font-semibold tracking-tight text-foreground">
                  {feature.title}
                </h3>
                <p className="mt-2.5 text-xs leading-relaxed text-muted-foreground">
                  {feature.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
