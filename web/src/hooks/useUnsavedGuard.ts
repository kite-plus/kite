import { useEffect, useRef, useState } from "react";

import { setGuard } from "@/lib/router";

export function useUnsavedGuard(dirty: boolean) {
  const dirtyRef = useRef(dirty);
  dirtyRef.current = dirty;
  const [leaving, setLeaving] = useState<(() => void) | null>(null);

  useEffect(() => {
    setGuard({
      blocked: () => dirtyRef.current,
      ask: (proceed) => setLeaving(() => proceed),
    });
    const warn = (event: BeforeUnloadEvent) => {
      if (dirtyRef.current) event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => {
      setGuard(null);
      window.removeEventListener("beforeunload", warn);
    };
  }, []);

  return {
    leaving: leaving !== null,
    cancel: () => setLeaving(null),
    discard: () => {
      dirtyRef.current = false;
      leaving?.();
      setLeaving(null);
    },
  };
}
