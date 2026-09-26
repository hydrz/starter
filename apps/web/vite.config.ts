/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import { paraglideVitePlugin } from "@inlang/paraglide-js";
import { ensureParaglideLocalModuleServer } from "./scripts/paraglide-local-modules.mjs";

const backendPort = process.env.PORT || "8080";

export default defineConfig(async () => {
  // project.inlang/settings.json 的 `modules` 指向一个本地静态服务器
  // （见 scripts/paraglide-local-modules.mjs），必须在 paraglideVitePlugin
  // 编译消息之前启动完成。
  await ensureParaglideLocalModuleServer();

  return {
    plugins: [
      tanstackRouter({ target: "react", autoCodeSplitting: true }),
      react(),
      tailwindcss(),
      paraglideVitePlugin({
        project: "./project.inlang",
        outdir: "./src/paraglide",
        // DESIGN.md §4: 不使用 URL locale 前缀，语言选择持久化到 localStorage。
        strategy: ["localStorage", "baseLocale"],
        // 仓库 tsconfig 关闭了 allowJs，需要 .d.ts 才能被 tsc 解析类型。
        emitTsDeclarations: true,
      }),
    ],
    build: {
      outDir: "../../internal/platform/webui/dist",
      emptyOutDir: true,
    },
    server: {
      host: "127.0.0.1",
      port: 5173,
      proxy: {
        "/api": `http://127.0.0.1:${backendPort}`,
      },
    },
    test: {
      environment: "happy-dom",
      globals: true,
      setupFiles: ["./src/test/setup.ts"],
    },
  };
});
