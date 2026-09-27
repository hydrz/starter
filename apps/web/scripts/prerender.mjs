#!/usr/bin/env node

import { execFileSync } from "node:child_process";
import { mkdir, readFile, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const webRoot = path.resolve(fileURLToPath(import.meta.url), "../..");
const outputDirectory = path.resolve(
  webRoot,
  "../../internal/platform/webui/dist",
);
const serverOutputDirectory = path.join(webRoot, ".prerender");

const publicRoutes = [
  {
    pathname: "/",
    title: "Starter — 端到端全栈极速交付",
    description:
      "生产级全栈开发模板：TypeSpec 契约驱动、Go 与 React、单二进制嵌入交付。",
  },
  {
    pathname: "/sign-in",
    title: "登录 Starter",
    description: "登录 Starter 账户并继续使用全栈开发平台。",
  },
  {
    pathname: "/sign-up",
    title: "注册 Starter",
    description: "创建 Starter 账户，开始端到端全栈开发。",
  },
];

function routeOutputPath(pathname) {
  if (pathname === "/") {
    return path.join(outputDirectory, "index.html");
  }
  return path.join(outputDirectory, pathname.slice(1), "index.html");
}

function renderDocument(template, route, content) {
  const escapedTitle = route.title.replaceAll("&", "&amp;");
  const escapedDescription = route.description.replaceAll("&", "&amp;");
  const head = [
    `<title>${escapedTitle}</title>`,
    `<meta name="description" content="${escapedDescription}" />`,
    `<meta property="og:type" content="website" />`,
    `<meta property="og:title" content="${escapedTitle}" />`,
    `<meta property="og:description" content="${escapedDescription}" />`,
  ].join("\n    ");

  return template
    .replace(/<title>[\s\S]*?<\/title>/, head)
    .replace('<div id="root"></div>', `<div id="root">${content}</div>`);
}

async function buildServerRenderer() {
  await rm(serverOutputDirectory, { force: true, recursive: true });
  execFileSync(
    process.execPath,
    [
      path.join(webRoot, "node_modules", "vite", "bin", "vite.js"),
      "build",
      "--ssr",
      "src/prerender.tsx",
      "--outDir",
      ".prerender",
    ],
    { cwd: webRoot, stdio: "inherit" },
  );
  return import(
    pathToFileURL(path.join(serverOutputDirectory, "prerender.js")).href
  );
}

async function writePrerenderedPages(prerender) {
  const template = await readFile(
    path.join(outputDirectory, "index.html"),
    "utf8",
  );
  for (const route of publicRoutes) {
    const outputPath = routeOutputPath(route.pathname);
    await mkdir(path.dirname(outputPath), { recursive: true });
    await writeFile(
      outputPath,
      renderDocument(template, route, prerender(route.pathname)),
    );
  }
}

async function writeSearchFiles() {
  const paths = publicRoutes.map((route) => route.pathname);
  const sitemap = [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    ...paths.map((pathname) => `  <url><loc>${pathname}</loc></url>`),
    "</urlset>",
    "",
  ].join("\n");
  const robots = ["User-agent: *", "Allow: /", "Sitemap: /sitemap.xml", ""];

  await writeFile(path.join(outputDirectory, "sitemap.xml"), sitemap);
  await writeFile(path.join(outputDirectory, "robots.txt"), robots.join("\n"));
}

try {
  const { prerender } = await buildServerRenderer();
  await writePrerenderedPages(prerender);
  await writeSearchFiles();
  await rm(serverOutputDirectory, { force: true, recursive: true });
  console.log("[prerender] 已生成公开路由、sitemap.xml 和 robots.txt");
} catch (error) {
  await rm(serverOutputDirectory, { force: true, recursive: true });
  console.error("[prerender] 生成失败", error);
  process.exitCode = 1;
}
