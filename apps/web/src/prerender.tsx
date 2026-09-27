import { renderToStaticMarkup } from "react-dom/server";
import { PrerenderedAuthPage, PrerenderedLandingPage } from "./prerender/pages";

/**
 * 构建期预渲染入口（D5）：由 `scripts/prerender.mjs` 以 Vite SSR 编译，
 * 返回公开路由的首屏 HTML 片段；注入产物模板后作为静态 HTML 嵌入二进制。
 */
export function prerender(pathname: string) {
  switch (pathname) {
    case "/":
      return renderToStaticMarkup(<PrerenderedLandingPage />);
    case "/sign-in":
      return renderToStaticMarkup(
        <PrerenderedAuthPage
          title="登录"
          subtitle="登录 Starter 账户以继续。"
          submitLabel="登录"
          footer="还没有账户？"
          footerHref="/sign-up"
          footerLabel="立即注册"
        />,
      );
    case "/sign-up":
      return renderToStaticMarkup(
        <PrerenderedAuthPage
          title="创建账户"
          subtitle="使用邮箱和密码注册。"
          submitLabel="创建账户"
          footer="已经有账户？"
          footerHref="/sign-in"
          footerLabel="去登录"
        />,
      );
    default:
      throw new Error(`Unsupported prerender route: ${pathname}`);
  }
}
