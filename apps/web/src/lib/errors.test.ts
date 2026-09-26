import { describe, expect, it } from "vitest";

import {
  getErrorMessage,
  getErrorStatus,
  isUnauthenticatedError,
} from "./errors";

describe("getErrorStatus", () => {
  it("非对象返回 undefined", () => {
    expect(getErrorStatus(null)).toBeUndefined();
    expect(getErrorStatus(undefined)).toBeUndefined();
    expect(getErrorStatus("string")).toBeUndefined();
    expect(getErrorStatus(123)).toBeUndefined();
  });

  it("直接提取 status 属性", () => {
    expect(getErrorStatus({ status: 401 })).toBe(401);
    expect(getErrorStatus({ status: 500 })).toBe(500);
  });

  it("非数字 status 返回 undefined", () => {
    expect(getErrorStatus({ status: "401" })).toBeUndefined();
    expect(getErrorStatus({ status: null })).toBeUndefined();
  });

  it("提取 axios 风格嵌套 response.status", () => {
    expect(getErrorStatus({ response: { status: 403 } })).toBe(403);
  });

  it("跟随 error.cause 链", () => {
    expect(getErrorStatus({ cause: { status: 401 } })).toBe(401);
  });

  it("支持深层 cause 链", () => {
    expect(
      getErrorStatus({ cause: { cause: { cause: { status: 500 } } } }),
    ).toBe(500);
  });

  it("循环引用不导致栈溢出", () => {
    const circular: Record<string, unknown> = { status: undefined };
    circular.cause = circular;
    expect(getErrorStatus(circular)).toBeUndefined();
  });

  it("直接 status 优先于嵌套 response.status", () => {
    expect(getErrorStatus({ status: 401, response: { status: 500 } })).toBe(
      401,
    );
  });
});

describe("getErrorMessage", () => {
  it("从 Error 实例提取 message", () => {
    expect(getErrorMessage(new Error("Something broke"))).toBe(
      "Something broke",
    );
  });

  it("字符串错误直接返回", () => {
    expect(getErrorMessage("Direct error message")).toBe(
      "Direct error message",
    );
  });

  it("从 Response-like 对象提取 statusText", () => {
    expect(getErrorMessage({ statusText: "Not Found" })).toBe("Not Found");
  });

  it("未知形状返回兜底文案", () => {
    expect(getErrorMessage(null)).toBe("An unexpected error occurred");
    expect(getErrorMessage(undefined)).toBe("An unexpected error occurred");
    expect(getErrorMessage({})).toBe("An unexpected error occurred");
    expect(getErrorMessage({ statusText: "" })).toBe(
      "An unexpected error occurred",
    );
  });
});

describe("isUnauthenticatedError", () => {
  it("401 返回 true", () => {
    expect(isUnauthenticatedError({ status: 401 })).toBe(true);
  });

  it("403 返回 false（已认证但无权限）", () => {
    expect(isUnauthenticatedError({ status: 403 })).toBe(false);
  });

  it("其他状态码返回 false", () => {
    expect(isUnauthenticatedError({ status: 500 })).toBe(false);
    expect(isUnauthenticatedError({ status: 404 })).toBe(false);
  });

  it("非错误值返回 false", () => {
    expect(isUnauthenticatedError(null)).toBe(false);
    expect(isUnauthenticatedError("error")).toBe(false);
    expect(isUnauthenticatedError({})).toBe(false);
  });

  it("从嵌套 cause 检测 401", () => {
    expect(isUnauthenticatedError({ cause: { status: 401 } })).toBe(true);
  });
});
