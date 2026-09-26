import { Building2, ChevronsUpDown, Plus } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu";
import * as m from "../../paraglide/messages";

export interface OrgSwitcherOrganization {
  id: string;
  name: string;
  slug: string;
}

export interface OrgSwitcherProps {
  organizations: Array<OrgSwitcherOrganization>;
  activeOrganizationId?: string;
  onSelect?: (organization: OrgSwitcherOrganization) => void;
  onCreate?: () => void;
  collapsed?: boolean;
}

// TODO(stage-3): 接入真实组织 API（当前使用占位数据，UI 已完整可用）。

/** 侧边栏顶部的组织切换下拉（DESIGN.md §1/§3/§6.2）。 */
export function OrgSwitcher({
  organizations,
  activeOrganizationId,
  onSelect,
  onCreate,
  collapsed = false,
}: OrgSwitcherProps) {
  const active =
    organizations.find((org) => org.id === activeOrganizationId) ??
    organizations[0];

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={m.org_switcher_aria()}
          className="flex w-full items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-2.5 py-2 text-left text-xs font-medium text-white transition-colors hover:bg-white/10"
        >
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-md bg-primary/20 text-primary">
            <Building2 size={13} />
          </span>
          {!collapsed && (
            <>
              <span className="min-w-0 flex-1 truncate">
                {active?.name ?? m.org_switcher_placeholder()}
              </span>
              <ChevronsUpDown size={13} className="shrink-0 opacity-60" />
            </>
          )}
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuLabel>{m.org_switcher_personal()}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {organizations.map((org) => (
          <DropdownMenuItem
            key={org.id}
            onSelect={() => onSelect?.(org)}
            className="justify-between"
          >
            <span className="truncate">{org.name}</span>
            {org.id === active?.id && (
              <span className="h-1.5 w-1.5 rounded-full bg-primary" />
            )}
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={() => onCreate?.()}>
          <Plus size={13} />
          <span>{m.org_switcher_create()}</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
