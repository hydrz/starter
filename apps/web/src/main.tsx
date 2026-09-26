import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { AppErrorBoundary } from "./components/AppErrorBoundary";
import { Toaster } from "./components/ui/toast";
import { queryClient } from "./lib/query";
import App from "./App";
import "./styles.css";

const root = document.getElementById("root");

if (!root) {
  throw new Error("Root element was not found");
}

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <AppErrorBoundary>
        <App />
        <Toaster />
      </AppErrorBoundary>
    </QueryClientProvider>
  </StrictMode>,
);
