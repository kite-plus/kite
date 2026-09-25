import { useEffect, useRef, useState, type ReactNode } from "react";
import { ExternalLink, Monitor, Smartphone, X, XCircle } from "lucide-react";

import { Alert, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { Draft } from "@/api/client";
import { useI18n } from "@/i18n";
import { cn } from "@/lib/utils";

interface Props {
  draft: Draft | null;
  id: string | null;
  /** base is the item's own address, which its images are linked relative to. */
  base?: string;
  /** live is the page the site serves for the saved item, when it serves one. */
  live?: string;
  onClose: () => void;
  /** onFrame hears of the frame each time a page has been written into it. */
  onFrame?: (frame: HTMLIFrameElement) => void;
}

type Width = "desktop" | "phone";

const widthKey = "kite:preview-width";

function preferredWidth(): Width {
  try {
    return localStorage.getItem(widthKey) === "phone" ? "phone" : "desktop";
  } catch {
    return "desktop";
  }
}

/**
 * A live preview, rendered by the server.
 *
 * The admin never renders markdown itself. A front end running its own
 * library would disagree with the build about footnotes, highlighting, raw
 * html and every extension either side adds later, and "the preview does not
 * match the site" would be a complaint with no end. What is shown here is the
 * page the build would write.
 */
export function Preview({ draft, id, base, live, onClose, onFrame }: Props) {
  const { t } = useI18n();
  const [html, setHtml] = useState("");
  const [failed, setFailed] = useState<string | null>(null);
  const [width, setWidth] = useState<Width>(preferredWidth);
  const frame = useRef<HTMLIFrameElement>(null);
  const written = useRef(onFrame);
  written.current = onFrame;

  const chooseWidth = (next: Width) => {
    setWidth(next);
    try {
      localStorage.setItem(widthKey, next);
    } catch {
      // Remembering the choice is a convenience, not a requirement.
    }
  };

  useEffect(() => {
    if (!draft) return;
    const controller = new AbortController();

    // A render per keystroke would queue requests the author has already
    // typed past. Rendering the pause instead keeps the preview one step
    // behind the cursor and no further.
    const timer = setTimeout(async () => {
      try {
        const response = await fetch(
          `/api/v1/preview${id ? `?id=${encodeURIComponent(id)}` : ""}`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(draft),
            signal: controller.signal,
          },
        );
        if (!response.ok) {
          const body = await response.json().catch(() => null);
          setFailed(
            body?.error?.message ?? t("editor.previewFailed", { status: response.status }),
          );
          return;
        }
        setFailed(null);
        setHtml(await response.text());
      } catch (err) {
        if ((err as Error).name === "AbortError") return;
        setFailed(err instanceof TypeError ? t("problem.unreachable") : String(err));
      }
    }, 250);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [draft, id, t]);

  // The document is written into the frame rather than assigned to srcdoc, so
  // that the scroll position survives a re-render and an author does not lose
  // their place on every keystroke. The page links its images by name, as
  // the build will publish them beside it, so it is told where it lives.
  useEffect(() => {
    const doc = frame.current?.contentDocument;
    if (!doc || !html) return;

    const located = base
      ? html.replace(/<head([^>]*)>/i, `<head$1><base href="${new URL(base, window.location.origin)}">`)
      : html;
    const scroll = doc.documentElement.scrollTop;
    doc.open();
    doc.write(located);
    doc.close();
    doc.documentElement.scrollTop = scroll;
    if (frame.current) written.current?.(frame.current);
  }, [html, base]);

  const phone = width === "phone";

  return (
    <div className="flex h-full flex-col">
      {/* The toolbar's height, so the rule under both runs straight across. */}
      <div className="flex h-[49px] shrink-0 items-center gap-1 border-b px-3.5">
        <span className="me-auto text-[13px] font-medium">{t("editor.preview")}</span>
        <div
          role="group"
          aria-label={t("editor.previewWidth")}
          className="flex items-center gap-0.5 rounded-md bg-muted p-0.5"
        >
          <WidthButton
            label={t("editor.previewDesktop")}
            pressed={!phone}
            onClick={() => chooseWidth("desktop")}
          >
            <Monitor />
          </WidthButton>
          <WidthButton
            label={t("editor.previewPhone")}
            pressed={phone}
            onClick={() => chooseWidth("phone")}
          >
            <Smartphone />
          </WidthButton>
        </div>
        {live && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" className="size-8" asChild>
                <a href={live} target="_blank" rel="noreferrer" aria-label={t("list.open")}>
                  <ExternalLink />
                </a>
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("list.open")}</TooltipContent>
          </Tooltip>
        )}
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className="size-8"
              aria-label={t("editor.previewClose")}
              onClick={onClose}
            >
              <X />
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t("editor.previewClose")}</TooltipContent>
        </Tooltip>
      </div>

      {/* One frame in both widths: a new one would lose the page written into it. */}
      <div className={cn("relative min-h-0 flex-1", phone && "bg-muted/60 p-4")}>
        {failed && (
          <Alert variant="destructive" className="absolute inset-x-0 top-0 z-10 rounded-none">
            <XCircle />
            <AlertTitle>{failed}</AlertTitle>
          </Alert>
        )}
        <iframe
          ref={frame}
          title={t("editor.preview")}
          className={cn(
            "block h-full w-full border-0 bg-white",
            phone && "mx-auto max-w-[390px] rounded-lg shadow-sm ring-1 ring-border",
          )}
          sandbox="allow-same-origin"
        />
      </div>
    </div>
  );
}

function WidthButton({
  label,
  pressed,
  onClick,
  children,
}: {
  label: string;
  pressed: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          aria-pressed={pressed}
          onClick={onClick}
          className="flex size-7 items-center justify-center rounded-[5px] text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 aria-pressed:bg-background aria-pressed:text-foreground aria-pressed:shadow-xs [&_svg]:size-4"
        >
          {children}
        </button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
