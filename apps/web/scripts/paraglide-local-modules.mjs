// @ts-check
/**
 * inlang 的 modules 字段只接受可 fetch 的 URL（含 http(s)/data URL），
 * 官方默认模板固定指向 cdn.jsdelivr.net。本仓库的沙箱/CI 出站网络策略
 * 不允许访问 jsdelivr，因此改为在本机 127.0.0.1 上起一个只读静态服务器，
 * 把已通过 npm registry 安装好的 `@inlang/plugin-message-format` 包内容
 * 原样吐出来，project.inlang/settings.json 的 `modules` 指向这个本地地址。
 *
 * 由 `vite.config.ts`（dev/build/vitest）与
 * `scripts/generate-web-artifacts.mjs`（独立生成，供 `tsc -b` 使用）共用。
 */
import http from "node:http";
import fs from "node:fs";
import { fileURLToPath } from "node:url";

export const LOCAL_MODULE_SERVER_PORT = 47821;
export const LOCAL_MESSAGE_FORMAT_PLUGIN_URL = `http://127.0.0.1:${LOCAL_MODULE_SERVER_PORT}/plugin-message-format.js`;

/** @type {Promise<import("node:http").Server> | undefined} */
let serverPromise;

export function ensureParaglideLocalModuleServer() {
  if (serverPromise) {
    return serverPromise;
  }

  serverPromise = new Promise((resolve, reject) => {
    let code;
    try {
      const pluginEntry = fileURLToPath(
        import.meta.resolve("@inlang/plugin-message-format"),
      );
      code = fs.readFileSync(pluginEntry, "utf-8");
    } catch (error) {
      reject(error);
      return;
    }

    const server = http.createServer((_req, res) => {
      res.setHeader("Content-Type", "text/javascript; charset=utf-8");
      res.end(code);
    });
    server.on("error", reject);
    server.listen(LOCAL_MODULE_SERVER_PORT, "127.0.0.1", () => {
      // 只在编译期被 fetch 一次；`unref` 让它不阻塞 `vite build` /
      // `vitest run` 等一次性命令的进程退出（`vite dev` 常驻不受影响）。
      server.unref();
      resolve(server);
    });
  });

  return serverPromise;
}
