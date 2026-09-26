import { Link } from "@tanstack/react-router";
import { ArrowRight, Check, Copy, Terminal } from "lucide-react";
import { useState } from "react";

export function CtaSection() {
  const [copied, setCopied] = useState(false);

  const command =
    "git clone https://github.com/hydrz/starter.git my-app && cd my-app && cp .env.example .env && pnpm install && pnpm db:up && pnpm db:migrate && pnpm dev";

  function handleCopy() {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <section
      id="quick-start"
      className="py-20 border-t border-border/40 bg-muted/30"
    >
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-3xl text-center">
          <span className="text-xs font-semibold uppercase tracking-wider text-accent-foreground dark:text-accent">
            QUICK START IN SECONDS
          </span>
          <h2 className="mt-2 text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            准备好构建下一个优秀的全栈应用了吗？
          </h2>
          <p className="mt-3 text-base text-muted-foreground text-balance">
            一条命令克隆启动完整开发环境，包含本地数据库容器、契约监听、前后端热重载与实时
            API。
          </p>

          {/* Terminal Window Box */}
          <div className="mt-8 overflow-hidden rounded-xl border border-border/80 bg-neutral-950 text-left shadow-lg">
            <div className="flex items-center justify-between border-b border-neutral-800 bg-neutral-900/90 px-4 py-2.5">
              <div className="flex items-center gap-2">
                <span className="h-3 w-3 rounded-full bg-red-500/80" />
                <span className="h-3 w-3 rounded-full bg-yellow-500/80" />
                <span className="h-3 w-3 rounded-full bg-green-500/80" />
                <div className="ml-2 flex items-center gap-1.5 font-mono text-[11px] text-neutral-400">
                  <Terminal size={12} />
                  <span>bash - starter-quickstart</span>
                </div>
              </div>
              <button
                type="button"
                onClick={handleCopy}
                className="flex items-center gap-1.5 rounded bg-neutral-800 px-2 py-1 text-[11px] font-medium text-neutral-300 transition-colors hover:bg-neutral-700 hover:text-white"
                aria-label="复制代码命令"
              >
                {copied ? (
                  <>
                    <Check size={12} className="text-emerald-400" />
                    <span className="text-emerald-400">已复制</span>
                  </>
                ) : (
                  <>
                    <Copy size={12} />
                    <span>复制命令</span>
                  </>
                )}
              </button>
            </div>

            <div className="p-4 font-mono text-xs text-neutral-200 overflow-x-auto whitespace-pre leading-relaxed">
              <div className="text-neutral-500"># 1. 克隆模板并进入目录</div>
              <div>git clone https://github.com/hydrz/starter.git my-app</div>
              <div>cd my-app && cp .env.example .env</div>
              <div className="mt-2 text-neutral-500">
                # 2. 安装依赖并启动本地数据库
              </div>
              <div>pnpm install</div>
              <div>pnpm db:up && pnpm db:migrate</div>
              <div className="mt-2 text-neutral-500">
                # 3. 启动前后端热重载全栈开发服务
              </div>
              <div className="text-emerald-400">pnpm dev</div>
            </div>
          </div>

          <div className="mt-8 flex justify-center gap-4">
            <Link
              to="/app"
              className="inline-flex h-11 items-center gap-2 rounded-lg bg-primary px-6 text-sm font-medium text-primary-foreground shadow-xs transition-colors hover:opacity-95"
            >
              <span>立即体验 Starter 控制台</span>
              <ArrowRight size={15} />
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}
