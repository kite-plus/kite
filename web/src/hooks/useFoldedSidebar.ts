import { useEffect, useRef } from "react";

import { setCookie } from "@/lib/cookies";
import { useSidebar } from "@/components/ui/sidebar";

// Whether a page has folded the sidebar. It outlives the page, so one page
// that wants the room giving way to another keeps the fold rather than
// unfolding and folding again.
let folded = false;
let unfolding: number | undefined;

/**
 * useFoldedSidebar folds the admin's sidebar to its icons while active, and
 * unfolds it again when that ends or the page goes, unless the person opened
 * it again themselves meanwhile. The fold is not remembered as their choice,
 * so leaving by closing the tab keeps the sidebar open elsewhere.
 */
export function useFoldedSidebar(active: boolean) {
  const { open, setOpen, isMobile } = useSidebar();
  const wasOpen = useRef(open);
  const latest = useRef(setOpen);
  latest.current = setOpen;

  useEffect(() => {
    window.clearTimeout(unfolding);
    return () => {
      if (!folded) return;
      unfolding = window.setTimeout(() => {
        if (!folded) return;
        folded = false;
        latest.current(true);
      });
    };
  }, []);

  useEffect(() => {
    if (isMobile) return;
    if (active && open) {
      folded = true;
      setOpen(false);
      setCookie("sidebar_state", "true", 60 * 60 * 24 * 7);
    } else if (!active && folded) {
      folded = false;
      setOpen(true);
    }
  }, [active, isMobile]);

  useEffect(() => {
    if (open && !wasOpen.current) folded = false;
    wasOpen.current = open;
  }, [open]);
}
