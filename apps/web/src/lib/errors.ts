/**
 * 从各种错误形状中提取 HTTP 状态码（含循环引用防护）。
 *
 * 支持：
 *  - 直接 status 属性（标准 HttpError、fetch Response）
 *  - 嵌套 response.status（axios 风格）
 *  - Error cause 链
 */
export function getErrorStatus(
  error: unknown,
  seen = new WeakSet<object>(),
): number | undefined {
  if (!error || typeof error !== "object") {
    return undefined;
  }
  if (seen.has(error)) {
    return undefined;
  }
  seen.add(error);

  const err = error as Record<string, unknown>;

  // 直接 status
  if (typeof err.status === "number") {
    return err.status;
  }

  // axios 风格嵌套 response.status
  if (
    err.response &&
    typeof (err.response as Record<string, unknown>).status === "number"
  ) {
    return (err.response as Record<string, unknown>).status as number;
  }

  // Error cause 链
  if (err.cause) {
    return getErrorStatus(err.cause, seen);
  }

  return undefined;
}

/**
 * 判断错误是否表示"未认证"（HTTP 401）。
 *
 * 不匹配 403（已认证但无权限）。
 * 401 才是"请先登录"，403 是"登录了但没权限"。
 */
export function isUnauthenticatedError(error: unknown): boolean {
  return getErrorStatus(error) === 401;
}

/**
 * 安全地从任意 thrown 值中提取人类可读的错误消息。
 */
export function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === "string") {
    return error;
  }
  // fetch Response 风格
  if (error && typeof error === "object" && "statusText" in error) {
    const statusText = (error as { statusText?: string }).statusText;
    if (statusText) {
      return statusText;
    }
  }
  return "An unexpected error occurred";
}
