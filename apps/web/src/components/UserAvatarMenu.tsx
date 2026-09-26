import { ChevronDown, ExternalLink, ShieldCheck, User } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu";
import * as m from "../paraglide/messages";

export function UserAvatarMenu() {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="flex items-center gap-2 rounded-full border border-border/80 bg-background/80 p-1 pr-2.5 text-xs font-medium text-foreground transition-colors hover:bg-muted/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label={m.user_menu_aria()}
        >
          <span className="relative flex h-7 w-7 items-center justify-center rounded-full bg-primary text-[11px] font-semibold text-primary-foreground">
            AD
            <span className="absolute -bottom-0.5 -right-0.5 h-2 w-2 rounded-full bg-emerald-500 ring-2 ring-background" />
          </span>
          <span className="hidden sm:inline">Admin</span>
          <ChevronDown size={13} className="text-muted-foreground" />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="font-normal">
          <p className="text-xs font-semibold text-foreground">
            Admin Developer
          </p>
          <p className="truncate text-[11px] text-muted-foreground">
            admin@starter.dev
          </p>
          <div className="mt-1.5 inline-flex items-center gap-1 rounded bg-muted/80 px-1.5 py-0.5 text-[10px] text-muted-foreground">
            <ShieldCheck size={11} className="text-emerald-500" />
            <span>{m.user_menu_role()}</span>
          </div>
        </DropdownMenuLabel>

        <DropdownMenuSeparator />

        <DropdownMenuItem className="text-muted-foreground">
          <User size={13} />
          <span>{m.user_menu_profile()}</span>
        </DropdownMenuItem>
        <DropdownMenuItem asChild className="text-muted-foreground">
          <a href="/api/docs" target="_blank" rel="noreferrer">
            <span className="flex-1">{m.user_menu_api_docs()}</span>
            <ExternalLink size={12} />
          </a>
        </DropdownMenuItem>

        <DropdownMenuSeparator />

        <DropdownMenuItem className="text-muted-foreground hover:text-destructive">
          <span>{m.user_menu_switch_workspace()}</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
