import { useRef } from "react";
import { useBlocker } from "@tanstack/react-router";

/**
 * useUnsavedGuard holds a navigation back while there is work that would be
 * lost, and asks first. Leaving the page altogether gets the browser's own
 * question, which is the only one a page may ask then.
 */
export function useUnsavedGuard(dirty: boolean) {
  // One navigation the page makes itself, such as a new item moving to its
  // own address after its first save, goes through without asking.
  const passing = useRef(false);

  const blocker = useBlocker({
    shouldBlockFn: () => {
      if (passing.current) {
        passing.current = false;
        return false;
      }
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
