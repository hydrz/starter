import { QueryClient } from "@tanstack/react-query";

// 只覆盖与 TanStack Query 默认值不同的选项。
// 重复默认值会诱导后续"调优"并在上游改变默认值时悄悄过期。
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // 导航回刚加载的页面不需要重新请求，两分钟内视为新鲜。
      staleTime: 2 * 60 * 1000,
      // "always" 而非默认的 true：断网重连后，缓存的年龄并不能
      // 说明数据是否仍然正确，因此始终重新拉取。
      refetchOnReconnect: "always",
    },
    mutations: {
      // 上游默认值，在此重申是因为它是安全不变式：
      // 丢失的响应与失败的请求无法区分，重试创建操作可能执行两次。
      // 仅在确认幂等的 mutation 上按需开启重试。
      retry: false,
      onError: (error) => console.error("Mutation failed:", error),
    },
  },
});
