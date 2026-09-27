import { useRef } from "react";
import { useBlocker } from "@tanstack/react-router";

/**
 * useUnsavedGuard holds a navigation back while there is work that would be
 * lost, and asks first. Leaving the page altogether gets the browser's own
 * question, which is the only one a page may ask then.
 *
 * kept says the work also lives outside the page, as the editor keeps it in
 * this browser: then a session that ends goes to the sign-in form at once,
 * and the work is there again after signing back in.
 */
export function useUnsavedGuard(dirty: boolean, { kept = false }: { kept?: boolean } = {}) {
  // One navigation the page makes itself, such as a new item moving to its
  // own address after its first save, goes through without asking.
  const passing = useRef(false);

  const blocker = useBlocker({
    shouldBlockFn: ({ next }) => {
      if (passing.current) {
        passing.current = false;
        return false;
      }
      if (kept && next.fullPath === "/sign-in") return false;
      return dirty;
    },
    enableBeforeUnload: () => dirty,
    withResolver: true,
  });

  return {
    leaving: blocker.status === "blocked",
    cancel: () => blocker.reset?.(),
    discard: () => blocker.proceed?.(),
    pass: () => {
      passing.current = true;
    },
  };
}
