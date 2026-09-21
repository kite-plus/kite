import { useSyncExternalStore } from "react";

export type Theme = "light" | "dark" | "system";

const storageKey = "kite:theme";
const system = window.matchMedia("(prefers-color-scheme: dark)");
const listeners = new Set<() => void>();

function read(): Theme {
  try {
    const saved = localStorage.getItem(storageKey);
    if (saved === "light" || saved === "dark") return saved;
  } catch {
    // A browser with storage blocked follows the system.
  }
  return "system";
}

let theme = read();

// shadcn keys dark mode off a class, so the choice is written onto the document.
function apply() {
  const dark = theme === "dark" || (theme === "system" && system.matches);
  document.documentElement.classList.toggle("dark", dark);
}

// Applied on import, before anything renders, so the page never flashes.
apply();
system.addEventListener("change", () => {
  apply();
  for (const listener of listeners) listener();
});

export function setTheme(next: Theme) {
  theme = next;
  try {
    if (next === "system") localStorage.removeItem(storageKey);
    else localStorage.setItem(storageKey, next);
  } catch {
    // Remembering the choice is a convenience, not a requirement.
  }
  apply();
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** useTheme reports the chosen theme and the one actually showing. */
export function useTheme() {
  const chosen = useSyncExternalStore(subscribe, () => theme);
  const dark = useSyncExternalStore(subscribe, () =>
    document.documentElement.classList.contains("dark"),
  );
  return { theme: chosen, resolved: dark ? ("dark" as const) : ("light" as const), setTheme };
}
