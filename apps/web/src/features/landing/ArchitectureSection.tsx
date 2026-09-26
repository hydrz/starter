import {
  ArrowRight,
  CheckCircle2,
  Database,
  FileCode2,
  Globe,
  Server,
} from "lucide-react";
import * as m from "../../paraglide/messages";

export function ArchitectureSection() {
  const steps = [
    {
      step: "01",
      icon: FileCode2,
      name: m.landing_architecture_step1_name(),
      desc: m.landing_architecture_step1_desc(),
      tag: m.landing_architecture_step1_tag(),
    },
    {
      step: "02",
      icon: Globe,
      name: m.landing_architecture_step2_name(),
      desc: m.landing_architecture_step2_desc(),
      tag: m.landing_architecture_step2_tag(),
    },
    {
      step: "03",
      icon: Server,
      name: m.landing_architecture_step3_name(),
      desc: m.landing_architecture_step3_desc(),
      tag: m.landing_architecture_step3_tag(),
    },
    {
      step: "04",
      icon: Database,
      name: m.landing_architecture_step4_name(),
      desc: m.landing_architecture_step4_desc(),
      tag: m.landing_architecture_step4_tag(),
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
            {m.landing_architecture_eyebrow()}
          </span>
          <h2 className="mt-2 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            {m.landing_architecture_title()}
          </h2>
          <p className="mt-3 text-base text-muted-foreground">
            {m.landing_architecture_subtitle_prefix()}{" "}
            <strong className="text-foreground font-semibold">
              {m.landing_architecture_subtitle_strong()}
            </strong>{" "}
            {m.landing_architecture_subtitle_suffix()}
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
                  <span className="inline-flex items-center gap-1 text-[10px] font-medium text-success">
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
                {m.landing_architecture_pipeline_title()}
              </h4>
              <p className="mt-0.5 text-xs text-muted-foreground">
                {m.landing_architecture_pipeline_desc()}
              </p>
            </div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span className="inline-flex h-2 w-2 rounded-full bg-success animate-pulse" />
              <span>{m.landing_architecture_pipeline_badge()}</span>
            </div>
          </div>

          <div className="mt-5 grid grid-cols-1 md:grid-cols-3 gap-4 text-xs">
            <div className="rounded-lg border border-border/60 bg-muted/40 p-4">
              <span className="font-semibold text-foreground">
                {m.landing_architecture_pipeline_step1_title()}
              </span>
              <p className="mt-1 text-muted-foreground">
                {m.landing_architecture_pipeline_step1_desc()}
              </p>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/40 p-4">
              <span className="font-semibold text-foreground">
                {m.landing_architecture_pipeline_step2_title()}
              </span>
              <p className="mt-1 text-muted-foreground">
                {m.landing_architecture_pipeline_step2_desc()}
              </p>
            </div>
            <div className="rounded-lg border border-border/60 bg-muted/40 p-4">
              <span className="font-semibold text-foreground">
                {m.landing_architecture_pipeline_step3_title()}
              </span>
              <p className="mt-1 text-muted-foreground">
                {m.landing_architecture_pipeline_step3_desc()}
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
