import { Laptop, Moon, Sun } from "lucide-react";
import * as m from "../paraglide/messages";
import { type Theme, useUIStore } from "../stores/ui-store";

export function ThemeToggle({ className = "" }: { className?: string }) {
  const { theme, setTheme } = useUIStore();

  const options: Array<{
    value: Theme;
    label: string;
    icon: typeof Sun;
  }> = [
    { value: "light", label: m.theme_light(), icon: Sun },
    { value: "system", label: m.theme_system(), icon: Laptop },
    { value: "dark", label: m.theme_dark(), icon: Moon },
  ];

  return (
    <div
      role="radiogroup"
      aria-label={m.theme_switch_aria()}
      className={`inline-flex items-center gap-0.5 rounded-lg border border-border/60 bg-muted/60 p-0.5 text-muted-foreground backdrop-blur-xs ${className}`}
    >
      {options.map(({ value, label, icon: Icon }) => {
        const active = theme === value;
        return (
          <button
            key={value}
            type="button"
            role="radio"
            aria-checked={active}
            aria-label={label}
            title={label}
            onClick={() => setTheme(value)}
            className={`flex h-7 w-7 items-center justify-center rounded-md text-xs font-medium transition-colors ${
              active
                ? "bg-background text-foreground shadow-xs"
                : "hover:text-foreground"
            }`}
          >
            <Icon size={14} />
          </button>
        );
      })}
    </div>
  );
}
