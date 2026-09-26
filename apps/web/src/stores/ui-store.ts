import { create } from "zustand";

export type Theme = "light" | "dark" | "system";

interface UIState {
  sidebarCollapsed: boolean;
  theme: Theme;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setTheme: (theme: Theme) => void;
}

const STORAGE_KEY = "starter-theme";

function getStoredTheme(): Theme {
  if (typeof window === "undefined" || !window.localStorage) {
    return "system";
  }
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "light" || stored === "dark" || stored === "system") {
    return stored;
  }
  return "system";
}

function applyTheme(theme: Theme) {
  if (typeof document === "undefined") {
    return;
  }
  const root = document.documentElement;
  const isDark =
    theme === "dark" ||
    (theme === "system" &&
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-color-scheme: dark)").matches);

  if (isDark) {
    root.classList.add("dark");
    root.classList.remove("light");
  } else {
    root.classList.remove("dark");
    root.classList.add("light");
  }
}

// Initial theme application
if (typeof window !== "undefined") {
  const initialTheme = getStoredTheme();
  applyTheme(initialTheme);

  // Listen for system changes if system theme is preferred
  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", () => {
      const current = getStoredTheme();
      if (current === "system") {
        applyTheme("system");
      }
    });
}

export const useUIStore = create<UIState>((set) => ({
  sidebarCollapsed: false,
  theme: getStoredTheme(),
  toggleSidebar: () =>
    set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
  setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
  setTheme: (theme: Theme) => {
    if (typeof window !== "undefined" && window.localStorage) {
      localStorage.setItem(STORAGE_KEY, theme);
    }
    applyTheme(theme);
    set({ theme });
  },
}));
