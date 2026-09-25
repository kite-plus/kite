import { useEffect, useImperativeHandle, useRef, useState, type Ref } from "react";

import { cn } from "@/lib/utils";

/** What the page around a preview can ask of it. */
export interface FrameHandle {
  /** home goes back to the preview's home page. */
  home: () => void;
  /** href is the address of the page on show. */
  href: () => string | null;
}

export interface Page {
  path: string;
  title: string;
}

interface Props {
  /** url is the preview's home page, where it starts. */
  url: string | null;
  /** drawn changes each time the preview has been drawn again. */
  drawn: number;
  phone: boolean;
  title: string;
  onPage?: (page: Page) => void;
  ref?: Ref<FrameHandle>;
}

/**
 * PreviewFrame shows a whole-site preview, which can be followed from page
 * to page like the site itself.
 *
 * It holds two frames. When the preview is drawn again, the page on show is
 * loaded into the one behind, scrolled to where the reader was, and only then
 * brought forward, so a change of color does not flash a blank page.
 */
export function PreviewFrame({ url, drawn, phone, title, onPage, ref }: Props) {
  const first = useRef<HTMLIFrameElement>(null);
  const second = useRef<HTMLIFrameElement>(null);
  const [front, setFront] = useState(0);
  const shown = useRef(0);
  const waiting = useRef<{ frame: number; scroll: number } | null>(null);
  const report = useRef(onPage);
  report.current = onPage;

  const frame = (i: number) => (i === 0 ? first : second).current;
  const go = (i: number, href: string) => {
    frame(i)?.contentWindow?.location.replace(href);
  };

  // A preview at a new address starts on its home page.
  useEffect(() => {
    if (url) go(shown.current, url);
  }, [url]);

  useEffect(() => {
    if (!drawn || !url) return;
    const current = frame(shown.current)?.contentWindow;
    const back = 1 - shown.current;
    waiting.current = { frame: back, scroll: scrolled(current) };
    go(back, located(current) ?? url);
  }, [drawn]);

  const loaded = (i: number) => {
    const win = frame(i)?.contentWindow;
    const href = located(win);
    if (waiting.current?.frame === i) {
      win?.scrollTo(0, waiting.current.scroll);
      waiting.current = null;
      shown.current = i;
      setFront(i);
    }
    if (win && href && i === shown.current) {
      report.current?.({ path: new URL(href).pathname, title: win.document.title });
    }
  };

  useImperativeHandle(
    ref,
    () => ({
      home: () => {
        if (url) go(shown.current, url);
      },
      href: () => located(frame(shown.current)?.contentWindow) ?? url,
    }),
    [url],
  );

  return (
    <div className={cn("relative min-h-0 flex-1", phone && "bg-muted/60 p-4")}>
      <div
        className={cn(
          "relative h-full w-full",
          phone && "mx-auto max-w-[390px] overflow-hidden rounded-lg shadow-sm ring-1 ring-border",
        )}
      >
        {[first, second].map((each, i) => (
          <iframe
            key={i}
            ref={each}
            title={title}
            aria-hidden={i !== front}
            tabIndex={i === front ? undefined : -1}
            // The page is the site's own, drawn by a theme; its scripts do not
            // run here, where they would share the admin's origin.
            sandbox="allow-same-origin"
            onLoad={() => loaded(i)}
            className={cn(
              "absolute inset-0 block h-full w-full border-0 bg-white",
              i === front ? "visible" : "invisible",
            )}
          />
        ))}
      </div>
    </div>
  );
}

/**
 * located is the address of the page a frame shows, or null for none yet, and
 * for a page of another site that followed a link out of the preview, which
 * the admin may not look into.
 */
function located(win: Window | null | undefined): string | null {
  try {
    const href = win?.location.href;
    return href && href !== "about:blank" ? href : null;
  } catch {
    return null;
  }
}

function scrolled(win: Window | null | undefined): number {
  try {
    return win?.scrollY ?? 0;
  } catch {
    return 0;
  }
}
