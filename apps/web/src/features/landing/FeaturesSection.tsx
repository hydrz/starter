import {
  Activity,
  Code2,
  Cpu,
  Database,
  Layers,
  ShieldCheck,
} from "lucide-react";
import * as m from "../../paraglide/messages";

export function FeaturesSection() {
  const features = [
    {
      icon: Code2,
      badge: m.landing_feature_contracts_badge(),
      title: m.landing_feature_contracts_title(),
      description: m.landing_feature_contracts_desc(),
    },
    {
      icon: Cpu,
      badge: m.landing_feature_backend_badge(),
      title: m.landing_feature_backend_title(),
      description: m.landing_feature_backend_desc(),
    },
    {
      icon: Layers,
      badge: m.landing_feature_frontend_badge(),
      title: m.landing_feature_frontend_title(),
      description: m.landing_feature_frontend_desc(),
    },
    {
      icon: ShieldCheck,
      badge: m.landing_feature_safety_badge(),
      title: m.landing_feature_safety_title(),
      description: m.landing_feature_safety_desc(),
    },
    {
      icon: Database,
      badge: m.landing_feature_data_badge(),
      title: m.landing_feature_data_title(),
      description: m.landing_feature_data_desc(),
    },
    {
      icon: Activity,
      badge: m.landing_feature_delivery_badge(),
      title: m.landing_feature_delivery_title(),
      description: m.landing_feature_delivery_desc(),
    },
  ];

  return (
    <section id="features" className="py-20 border-t border-border/40">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <span className="text-xs font-semibold uppercase tracking-wider text-accent-foreground dark:text-accent">
            {m.landing_features_eyebrow()}
          </span>
          <h2 className="mt-2 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            {m.landing_features_title()}
          </h2>
          <p className="mt-3 text-base text-muted-foreground">
            {m.landing_features_subtitle_prefix()}{" "}
            <strong className="text-foreground font-semibold">
              {m.landing_features_subtitle_strong()}
            </strong>
            {m.landing_features_subtitle_suffix()}
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
