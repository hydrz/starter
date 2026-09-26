import type { FieldValues, UseFormReturn } from "react-hook-form";
import { HttpError } from "../../api/client";
import { getErrorMessage } from "../../lib/errors";
import * as m from "../../paraglide/messages";

/**
 * 把任意后端错误（优先 `HttpError`，统一 ApiError `{code, message}` 形状）
 * 映射为表单级错误（`root`），复用 `ui/form.tsx` 的 FormMessage 渲染。
 *
 * DESIGN.md §8：服务端错误必须映射到对应字段，未知 code 落到表单级提示，
 * 不允许被吞掉。本轮认证表单的服务端校验错误（如邮箱已注册）不区分
 * 字段，统一走表单级错误展示。
 */
export function applyAuthApiError<T extends FieldValues>(
  form: UseFormReturn<T>,
  error: unknown,
): void {
  const message =
    error instanceof HttpError && error.data?.message
      ? error.data.message
      : getErrorMessage(error) || m.auth_error_generic();

  form.setError("root", { type: "server", message });
}
