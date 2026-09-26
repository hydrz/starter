import { useQueryErrorResetBoundary } from "@tanstack/react-query";
import { AlertCircle } from "lucide-react";
import { ErrorBoundary } from "react-error-boundary";

import { getErrorMessage } from "../lib/errors";

interface ErrorFallbackProps {
  error: unknown;
  resetErrorBoundary: () => void;
}

function GenericErrorFallback({
  error,
  resetErrorBoundary,
}: ErrorFallbackProps) {
  return (
    <div className="flex min-h-svh flex-col items-center justify-center p-6">
      <div className="mx-auto max-w-md text-center">
        <AlertCircle className="mx-auto mb-4 h-12 w-12 text-destructive" />
        <h1 className="mb-2 text-2xl font-bold">出错了</h1>
        <p className="mb-6 text-sm text-muted-foreground">
          {getErrorMessage(error)}
        </p>
        <button
          onClick={resetErrorBoundary}
          className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:opacity-90"
        >
          重试
        </button>
      </div>
    </div>
  );
}

interface AppErrorBoundaryProps {
  children: React.ReactNode;
}

/**
 * 根级通用错误边界。
 *
 * 包裹整个应用树，捕获渲染阶段未处理的异常并显示友好的错误恢复 UI。
 * 接入 useQueryErrorResetBoundary，确保点击"重试"后 TanStack Query
 * 会重新发起失败的请求，而非直接渲染缓存的错误状态。
 *
 * 挂载位置：main.tsx，RouterProvider 外层。
 */
export function AppErrorBoundary({ children }: AppErrorBoundaryProps) {
  const { reset } = useQueryErrorResetBoundary();

  return (
    <ErrorBoundary
      FallbackComponent={GenericErrorFallback}
      onReset={reset}
      onError={(error) => console.error("Uncaught error:", error)}
    >
      {children}
    </ErrorBoundary>
  );
}
