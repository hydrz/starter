/* eslint-disable react-refresh/only-export-components */
import type { ComponentProps } from "react";
import { Toaster as Sonner, toast } from "sonner";

type ToasterProps = ComponentProps<typeof Sonner>;

/**
 * sonner 驱动的全局 toast 挂载点。放在应用根部一次即可，
 * 业务代码通过 `import { toast } from "@/components/ui/toast"` 触发提示。
 */
function Toaster({ ...props }: ToasterProps) {
  return (
    <Sonner
      theme="system"
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-card group-[.toaster]:text-card-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
      }}
      {...props}
    />
  );
}

export { Toaster, toast };
