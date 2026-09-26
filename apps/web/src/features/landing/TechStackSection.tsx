export function TechStackSection() {
  const groups = [
    {
      category: "协议与契约层",
      items: [
        { name: "TypeSpec", role: "声明式 API 契约语言" },
        { name: "OpenAPI 3.0", role: "业界标准接口规范" },
        { name: "Scalar", role: "100% 离线内嵌交互式文档" },
        { name: "oapi-codegen", role: "Go 服务端契约脚手架" },
        { name: "Orval", role: "React Query Hooks 自动生成" },
      ],
    },
    {
      category: "现代化前端栈",
      items: [
        { name: "React 19", role: "最新现代化 UI 渲染引擎" },
        { name: "TypeScript", role: "端到端全链路类型安全" },
        { name: "TanStack Router", role: "100% 类型化文件与代码路由" },
        { name: "TanStack Query", role: "强大的异步状态管理与缓存" },
        { name: "Tailwind CSS v4", role: "超快下一代原子化样式引擎" },
        { name: "shadcn/ui", role: "基于 Radix/CVA 的可复用组件" },
        { name: "Zod & Hook Form", role: "模式驱动的前端表单校验" },
      ],
    },
    {
      category: "Go 后端与数据层",
      items: [
        { name: "Go 1.27", role: "编译型高并发服务端运行时" },
        { name: "Chi Router", role: "极轻量、符合标准库习惯的 HTTP 路由" },
        { name: "PostgreSQL 17", role: "现代化关系型数据引擎" },
        { name: "sqlc", role: "SQL 编译期验证与 Go 代码生成" },
        { name: "pgx/v5", role: "极速高性能原生数据库驱动池" },
        { name: "Goose", role: "显式正向版本化数据库迁移" },
      ],
    },
    {
      category: "工程基线与交付",
      items: [
        { name: "Go embed", role: "前端静态产物完全嵌入单二进制" },
        { name: "Docker Compose", role: "开箱即用的一键本地环境" },
        { name: "pnpm Workspaces", role: "高性能 Monorepo 依赖管理" },
        { name: "Vitest & Testify", role: "前后端单元与集成测试套件" },
        { name: "GitHub Actions", role: "严格的分支保护与自动化流水线" },
        { name: "Dependabot", role: "每周依赖安全与小版本自动更新" },
      ],
    },
  ];

  return (
    <section id="tech-stack" className="py-20 border-t border-border/40">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <span className="text-xs font-semibold uppercase tracking-wider text-accent-foreground dark:text-accent">
            MODERN TECH STACK
          </span>
          <h2 className="mt-2 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            精心甄选的现代化全栈技术选型
          </h2>
          <p className="mt-3 text-base text-muted-foreground">
            拒绝过度工程，聚焦于开发者生产力（DX）、长期可维护性与交付确定性。
          </p>
        </div>

        <div className="mt-14 grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
          {groups.map((group) => (
            <div
              key={group.category}
              className="rounded-2xl border border-border/80 bg-card p-6 text-card-foreground shadow-xs dark:bg-card/60"
            >
              <h3 className="text-sm font-semibold tracking-tight text-foreground border-b border-border/60 pb-3">
                {group.category}
              </h3>
              <ul className="mt-4 space-y-3.5">
                {group.items.map((tech) => (
                  <li key={tech.name} className="flex flex-col">
                    <span className="text-xs font-medium text-foreground">
                      {tech.name}
                    </span>
                    <span className="text-[11px] text-muted-foreground">
                      {tech.role}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
