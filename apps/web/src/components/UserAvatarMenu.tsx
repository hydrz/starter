import { ChevronDown, ExternalLink, ShieldCheck, User } from "lucide-react";
import { useEffect, useRef, useState } from "react";

export function UserAvatarMenu() {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setOpen(false);
      }
    }
    if (open) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [open]);

  return (
    <div className="relative inline-block text-left" ref={containerRef}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 rounded-full border border-border/80 bg-background/80 p-1 pr-2.5 text-xs font-medium text-foreground transition-colors hover:bg-muted/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        aria-expanded={open}
        aria-haspopup="true"
        aria-label="用户账户菜单"
      >
        <span className="relative flex h-7 w-7 items-center justify-center rounded-full bg-primary text-[11px] font-semibold text-primary-foreground">
          AD
          <span className="absolute -bottom-0.5 -right-0.5 h-2 w-2 rounded-full bg-emerald-500 ring-2 ring-background" />
        </span>
        <span className="hidden sm:inline">Admin</span>
        <ChevronDown size={13} className="text-muted-foreground" />
      </button>

      {open && (
        <div
          role="menu"
          aria-orientation="vertical"
          className="absolute right-0 z-50 mt-2 w-56 origin-top-right rounded-xl border border-border/80 bg-card p-2 text-card-foreground shadow-lg backdrop-blur-md focus:outline-none animate-in fade-in-0 zoom-in-95 duration-100"
        >
          <div className="border-b border-border/60 px-2 py-2">
            <p className="text-xs font-semibold">Admin Developer</p>
            <p className="truncate text-[11px] text-muted-foreground">
              admin@starter.dev
            </p>
            <div className="mt-1.5 inline-flex items-center gap-1 rounded bg-muted/80 px-1.5 py-0.5 text-[10px] text-muted-foreground">
              <ShieldCheck size={11} className="text-emerald-500" />
              <span>Full-Stack Workspace</span>
            </div>
          </div>

          <div className="py-1">
            <a
              href="#profile"
              onClick={(e) => {
                e.preventDefault();
                setOpen(false);
              }}
              className="flex items-center gap-2 rounded-lg px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              role="menuitem"
            >
              <User size={13} />
              <span>开发者个人设置</span>
            </a>
            <a
              href="/api/docs"
              target="_blank"
              rel="noreferrer"
              className="flex items-center justify-between rounded-lg px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              role="menuitem"
            >
              <span>Scalar API 交互文档</span>
              <ExternalLink size={12} />
            </a>
          </div>

          <div className="border-t border-border/60 pt-1">
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="flex w-full items-center justify-between rounded-lg px-2.5 py-1.5 text-left text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-destructive"
              role="menuitem"
            >
              <span>切换工作空间</span>
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
