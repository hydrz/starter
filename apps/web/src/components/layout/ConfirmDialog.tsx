import * as React from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Button } from "../ui/button";
import * as m from "../../paraglide/messages";

export interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: React.ReactNode;
  description?: React.ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  /** 危险操作默认使用 destructive 按钮样式 */
  variant?: "destructive" | "default";
  onConfirm: () => void | Promise<void>;
  pending?: boolean;
}

/**
 * 全仓库唯一的二次确认弹窗（DESIGN.md §3/§8）。
 * 所有删除组织 / 撤销 API key 等危险操作都必须复用这个组件，
 * 不得各自实现确认弹窗。
 *
 * @example
 * <ConfirmDialog
 *   open={open}
 *   onOpenChange={setOpen}
 *   title={<>确认删除组织 <b>{org.name}</b>？</>}
 *   description="删除后不可恢复，组织下的所有数据将被清除。"
 *   variant="destructive"
 *   onConfirm={() => deleteOrg.mutateAsync(org.id)}
 *   pending={deleteOrg.isPending}
 * />
 */
export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  cancelLabel,
  variant = "destructive",
  onConfirm,
  pending = false,
}: ConfirmDialogProps) {
  const handleConfirm = async () => {
    await onConfirm();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description && <DialogDescription>{description}</DialogDescription>}
        </DialogHeader>
        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={pending}
            onClick={() => onOpenChange(false)}
          >
            {cancelLabel ?? m.confirm_dialog_default_cancel()}
          </Button>
          <Button
            type="button"
            variant={variant}
            disabled={pending}
            onClick={handleConfirm}
          >
            {pending
              ? m.confirm_dialog_pending()
              : (confirmLabel ?? m.confirm_dialog_default_confirm())}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
