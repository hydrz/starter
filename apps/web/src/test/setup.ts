import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach } from "vitest";
import { localStorageKey } from "../paraglide/runtime";

// Paraglide 的 baseLocale 是 "en"（DESIGN.md §4），但仓库现有页面文案仍以中文
// 为主。测试环境里把 locale 固定为 "zh"，保持既有断言（中文文案）稳定，
// 与 Stage 1 之前的行为一致。
beforeEach(() => {
  window.localStorage.setItem(localStorageKey, "zh");
});

afterEach(() => {
  cleanup();
});
