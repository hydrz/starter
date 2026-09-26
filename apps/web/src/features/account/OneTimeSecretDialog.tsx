import { AlertTriangle } from "lucide-react";
import type { ReactNode } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../components/ui/dialog";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { Button } from "../../components/ui/button";
import * as m from "../../paraglide/messages";

export interface OneTimeSecretDialogProps {
  open: boolean;
  /** 关闭即视为"已确认保存"：调用方必须在这里清空持有的明文密钥/恢复码状态，不得保留。 */
  onAcknowledge: () => void;
  title: ReactNode;
  description: ReactNode;
  /** 一个或多个 `ui/code.tsx` 渲染的明文值（API key 明文、恢复码列表等）。 */
  children: ReactNode;
}

/**
 * API key 明文 / TOTP 恢复码的"仅展示一次"通用弹窗（DESIGN.md §8）。
 *
 * 两个调用方（`SessionsPage` 创建 API key、`SecurityPage` 确认 TOTP 注册）
 * 共用同一套一次性展示 + 强提示 + 关闭后再也不显示的状态机，不各自实现。
 * 关闭对话框（无论点"我已保存"还是右上角 X/Esc/点击遮罩）都必须把明文从
 * 组件状态里清空——本组件本身不持有明文，只负责展示 `children` 传入的
 * 内容并在关闭时通知调用方清空；调用方必须保证 `onAcknowledge` 之后
 * 不会再把同一个明文传回 `children`。
 */
export function OneTimeSecretDialog({
  open,
  onAcknowledge,
  title,
  description,
  children,
}: OneTimeSecretDialogProps) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onAcknowledge();
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <Alert variant="warning">
          <AlertTriangle />
          <AlertDescription>{m.one_time_secret_warning()}</AlertDescription>
        </Alert>

        <div className="space-y-2">{children}</div>

        <DialogFooter>
          <Button type="button" onClick={onAcknowledge}>
            {m.one_time_secret_acknowledge()}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
