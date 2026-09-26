import * as React from "react";
import { Check, Copy } from "lucide-react";
import { cn } from "../../lib/utils";
import { Button } from "./button";

export interface CodeProps extends React.HTMLAttributes<HTMLElement> {
  /** 是否展示一键复制按钮（默认开启） */
  copyable?: boolean;
}

/**
 * 等宽文本展示（ID / API Key / token / 时间戳），可选一键复制。
 * 消费 `--font-mono` token（DESIGN.md §2.2）。
 */
const Code = React.forwardRef<HTMLElement, CodeProps>(
  ({ className, children, copyable = true, ...props }, ref) => {
    const [copied, setCopied] = React.useState(false);
    const text = typeof children === "string" ? children : "";

    const handleCopy = async () => {
      if (!text || typeof navigator === "undefined" || !navigator.clipboard) {
        return;
      }
      try {
        await navigator.clipboard.writeText(text);
        setCopied(true);
        window.setTimeout(() => setCopied(false), 1500);
      } catch {
        // 剪贴板不可用时静默失败，不阻塞用户操作
      }
    };

    return (
      <span
        className={cn(
          "inline-flex max-w-full items-center gap-1.5 rounded-md border border-border bg-muted px-2 py-1 font-mono text-xs text-foreground",
          className,
        )}
      >
        <code ref={ref} className="truncate" {...props}>
          {children}
        </code>
        {copyable && text && (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-4 w-4 shrink-0 text-muted-foreground hover:text-foreground"
            onClick={handleCopy}
            aria-label={copied ? "已复制" : "复制到剪贴板"}
          >
            {copied ? (
              <Check className="h-3 w-3" />
            ) : (
              <Copy className="h-3 w-3" />
            )}
          </Button>
        )}
      </span>
    );
  },
);
Code.displayName = "Code";

export { Code };
