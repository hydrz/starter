/* eslint-disable react-refresh/only-export-components -- 同时导出 Provider 组件与 useOrgContext hook，符合仓库其它 context/provider 文件的既有模式（如 ui/form.tsx）。 */
import { createContext, useContext, type ReactNode } from "react";
import type { OrganizationRole } from "../../api/generated/model";

export interface OrgContextValue {
  organizationId: string;
  slug: string;
  name: string;
  /** 当前登录用户在这个组织里的角色（来自 `useListUserOrganizations`）。 */
  role: OrganizationRole;
}

const OrgContext = createContext<OrgContextValue | null>(null);

export function OrgProvider({
  value,
  children,
}: {
  value: OrgContextValue;
  children: ReactNode;
}) {
  return <OrgContext.Provider value={value}>{children}</OrgContext.Provider>;
}

/**
 * 读取 `routes/app/$orgSlug.tsx` 布局路由解析好的组织上下文
 * （URL 里的 `$orgSlug` 已经解析为真实的 `organizationId`）。
 * 只能在该路由子树内使用（Members/Invitations/Settings/Overview/Announcements）。
 */
export function useOrgContext(): OrgContextValue {
  const ctx = useContext(OrgContext);
  if (!ctx) {
    throw new Error(
      "useOrgContext must be used within the /app/$orgSlug route subtree",
    );
  }
  return ctx;
}
