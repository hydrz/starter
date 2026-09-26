import * as m from "../../paraglide/messages";

export function TechStackSection() {
  const groups = [
    {
      category: m.landing_tech_group_contracts(),
      items: [
        { name: "TypeSpec", role: m.landing_tech_role_typespec() },
        { name: "OpenAPI 3.0", role: m.landing_tech_role_openapi() },
        { name: "Scalar", role: m.landing_tech_role_scalar() },
        { name: "ogen", role: m.landing_tech_role_ogen() },
        { name: "Orval", role: m.landing_tech_role_orval() },
      ],
    },
    {
      category: m.landing_tech_group_frontend(),
      items: [
        { name: "React 19", role: m.landing_tech_role_react() },
        { name: "TypeScript", role: m.landing_tech_role_typescript() },
        {
          name: "TanStack Router",
          role: m.landing_tech_role_tanstack_router(),
        },
        { name: "TanStack Query", role: m.landing_tech_role_tanstack_query() },
        { name: "Tailwind CSS v4", role: m.landing_tech_role_tailwind() },
        { name: "shadcn/ui", role: m.landing_tech_role_shadcn() },
        { name: "Zod & Hook Form", role: m.landing_tech_role_zod_rhf() },
      ],
    },
    {
      category: m.landing_tech_group_backend(),
      items: [
        { name: "Go 1.27", role: m.landing_tech_role_go() },
        { name: "Chi Router", role: m.landing_tech_role_chi() },
        { name: "PostgreSQL 17", role: m.landing_tech_role_postgres() },
        { name: "sqlc", role: m.landing_tech_role_sqlc() },
        { name: "pgx/v5", role: m.landing_tech_role_pgx() },
        { name: "Goose", role: m.landing_tech_role_goose() },
      ],
    },
    {
      category: m.landing_tech_group_delivery(),
      items: [
        { name: "Go embed", role: m.landing_tech_role_go_embed() },
        {
          name: "Docker Compose",
          role: m.landing_tech_role_docker_compose(),
        },
        {
          name: "pnpm Workspaces",
          role: m.landing_tech_role_pnpm_workspaces(),
        },
        { name: "Vitest & Testify", role: m.landing_tech_role_vitest() },
        { name: "GitHub Actions", role: m.landing_tech_role_actions() },
        { name: "Dependabot", role: m.landing_tech_role_dependabot() },
      ],
    },
  ];

  return (
    <section id="tech-stack" className="py-20 border-t border-border/40">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <span className="text-xs font-semibold uppercase tracking-wider text-accent-foreground dark:text-accent">
            {m.landing_tech_eyebrow()}
          </span>
          <h2 className="mt-2 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            {m.landing_tech_title()}
          </h2>
          <p className="mt-3 text-base text-muted-foreground">
            {m.landing_tech_subtitle()}
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
