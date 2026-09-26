import { Languages } from "lucide-react";
import * as m from "../paraglide/messages";
import { getLocale, locales, setLocale } from "../paraglide/runtime";

const labels: Record<(typeof locales)[number], () => string> = {
  en: m.language_en,
  zh: m.language_zh,
};

/**
 * 语言切换控件（DESIGN.md §4）：持久化到 localStorage，切换后 Paraglide
 * runtime 会触发一次页面刷新以应用新语言（`setLocale` 默认行为）。
 */
export function LanguageSwitcher({ className = "" }: { className?: string }) {
  const current = getLocale();

  return (
    <div
      role="radiogroup"
      aria-label={m.language_switch_aria()}
      className={`inline-flex items-center gap-0.5 rounded-lg border border-border/60 bg-muted/60 p-0.5 text-muted-foreground backdrop-blur-xs ${className}`}
    >
      <Languages size={13} className="ml-1 shrink-0" aria-hidden="true" />
      {locales.map((locale) => {
        const active = current === locale;
        return (
          <button
            key={locale}
            type="button"
            role="radio"
            aria-checked={active}
            onClick={() => setLocale(locale)}
            className={`flex h-7 items-center rounded-md px-2 text-xs font-medium transition-colors ${
              active
                ? "bg-background text-foreground shadow-xs"
                : "hover:text-foreground"
            }`}
          >
            {labels[locale]()}
          </button>
        );
      })}
    </div>
  );
}
