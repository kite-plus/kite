import { useSyncExternalStore } from "react";

/** useMediaQuery follows a CSS media query, for layout that CSS cannot express. */
export function useMediaQuery(query: string): boolean {
  return useSyncExternalStore(
    (listener) => {
      const list = window.matchMedia(query);
      list.addEventListener("change", listener);
      return () => list.removeEventListener("change", listener);
    },
    () => window.matchMedia(query).matches,
  );
}
