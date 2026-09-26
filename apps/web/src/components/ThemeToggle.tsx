import { Laptop, Moon, Sun } from "lucide-react";
import { type Theme, useUIStore } from "../stores/ui-store";

export function ThemeToggle({ className = "" }: { className?: string }) {
  const { theme, setTheme } = useUIStore();

  const options: Array<{ value: Theme; label: string; icon: typeof Sun }> = [
    { value: "light", label: "浅色模式", icon: Sun },
    { value: "system", label: "跟随系统", icon: Laptop },
    { value: "dark", label: "深色模式", icon: Moon },
  ];

  return (
    <div
      role="radiogroup"
      aria-label="选择主题外观"
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
