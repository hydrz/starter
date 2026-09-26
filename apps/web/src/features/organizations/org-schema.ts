import { z } from "zod";
import * as m from "../../paraglide/messages";

/**
 * 组织名称/slug 的客户端校验规则，创建组织（`OrgPickerPage`）与组织设置
 * （`SettingsPage`）共用。长度上限与后端 `CreateOrganizationInput`/
 * `UpdateOrganizationInput` 一致（见生成的 model 类型），slug 格式规则是
 * 纯前端 UX 约束（后端只校验长度），真正的唯一性/合法性以服务端响应为准。
 */
export function buildOrgSchema() {
  return z.object({
    name: z
      .string()
      .trim()
      .min(1, m.org_validation_name_required())
      .max(120, m.org_validation_name_max()),
    slug: z
      .string()
      .trim()
      .min(1, m.org_validation_slug_required())
      .max(100, m.org_validation_slug_max())
      .regex(/^[a-z0-9]+(-[a-z0-9]+)*$/, m.org_validation_slug_format()),
  });
}

export type OrgFormValues = z.infer<ReturnType<typeof buildOrgSchema>>;

/** name -> slug 的 kebab-case 建议值，仅用于表单自动填充，用户可覆盖。 */
export function slugify(input: string): string {
  return input
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 100);
}
