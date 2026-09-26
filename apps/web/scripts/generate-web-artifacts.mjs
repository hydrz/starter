#!/usr/bin/env node
// @ts-check
/**
 * 生成物防漂移：一次性生成 TanStack Router 的 routeTree.gen.ts 与 Paraglide
 * 的消息运行时（两者都被 .gitignore 忽略，视为构建产物）。
 *
 * `vite dev` / `vite build` / `vitest run` 会通过各自的 Vite 插件自动生成，
 * 本脚本用于 `pnpm --filter @starter/web check` 里独立运行的 `tsc -b`
 * ——tsc 不会执行 Vite 插件，需要这些文件已经落盘才能类型检查通过。
 */
import path from "node:path";
import { fileURLToPath } from "node:url";
import { getConfig, Generator } from "@tanstack/router-generator";
import { compile } from "@inlang/paraglide-js";
import { ensureParaglideLocalModuleServer } from "./paraglide-local-modules.mjs";

const root = path.resolve(fileURLToPath(import.meta.url), "../..");

async function generateRoutes() {
  const config = getConfig(
    {
      target: "react",
      autoCodeSplitting: true,
    },
    root,
  );
  const generator = new Generator({ config, root });
  await generator.run();
  console.log("[generate-web-artifacts] routeTree.gen.ts 已生成");
}

async function generateParaglide() {
  await ensureParaglideLocalModuleServer();
  await compile({
    project: path.join(root, "project.inlang"),
    outdir: path.join(root, "src/paraglide"),
    // DESIGN.md §4: 不使用 URL locale 前缀，语言选择持久化到 localStorage。
    strategy: ["localStorage", "baseLocale"],
    // 仓库 tsconfig 关闭了 allowJs，需要 .d.ts 才能被 tsc 解析类型。
    emitTsDeclarations: true,
  });
  console.log("[generate-web-artifacts] paraglide 消息运行时已生成");
}

try {
  await generateRoutes();
  await generateParaglide();
  process.exit(0);
} catch (error) {
  console.error("[generate-web-artifacts] 生成失败", error);
  process.exit(1);
}
