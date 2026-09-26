import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";
import * as m from "../../paraglide/messages";

interface AuthLayoutProps {
  title: string;
  subtitle?: string;
  children?: ReactNode;
  footer?: ReactNode;
}

/**
 * 所有公开认证页面（/sign-in、/sign-up、/mfa 等）共享的外壳。
 *
 * 保持与 Landing 一致的 Cloudflare 式几何背景装饰（DESIGN.md §1），
 * 卡片居中、品牌角标 + 返回首页入口，避免每个路由各自重造布局。
 */
export function AuthLayout({
  title,
  subtitle,
  children,
  footer,
}: AuthLayoutProps) {
  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center overflow-hidden bg-background px-4 py-12">
      {/* 几何网格背景装饰，非摄影图（DESIGN.md §1） */}
      <div
        className="pointer-events-none absolute inset-0 -z-10 bg-[linear-gradient(to_right,theme(colors.border/40)_1px,transparent_1px),linear-gradient(to_bottom,theme(colors.border/40)_1px,transparent_1px)] bg-[size:64px_64px] [mask-image:radial-gradient(ellipse_60%_60%_at_50%_0%,black_10%,transparent_70%)]"
        aria-hidden="true"
      />
      <div
        className="pointer-events-none absolute -top-32 left-1/2 -z-10 h-[420px] w-[720px] -translate-x-1/2 rounded-full bg-gradient-to-tr from-primary/15 to-secondary/10 blur-3xl"
        aria-hidden="true"
      />

      <Link
        to="/"
        className="mb-8 flex items-center gap-2.5 font-semibold text-foreground transition-opacity hover:opacity-90"
      >
        <span className="flex h-7 w-7 items-center justify-center rounded-full border border-border/80 bg-primary/10 text-xs font-bold text-accent-foreground dark:text-accent">
          E
        </span>
        <span className="text-base tracking-tight font-semibold">
          {m.app_brand()}
        </span>
      </Link>

      <div className="w-full max-w-sm rounded-2xl border border-border/80 bg-card p-7 text-card-foreground shadow-md">
        <div className="mb-6 text-center">
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            {title}
          </h1>
          {subtitle ? (
            <p className="mt-1.5 text-sm text-muted-foreground">{subtitle}</p>
          ) : null}
        </div>

        {children}
      </div>

      {footer ? (
        <div className="mt-6 text-center text-sm text-muted-foreground">
          {footer}
        </div>
      ) : null}

      <Link
        to="/"
        className="mt-8 text-xs text-muted-foreground transition-colors hover:text-foreground"
      >
        {m.auth_back_to_home()}
      </Link>
    </div>
  );
}
