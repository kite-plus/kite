import { useEffect, useRef, useState } from "react";
import { XCircle } from "lucide-react";

import { Alert, AlertTitle } from "@/components/ui/alert";
import type { Draft } from "@/api/client";
import { useI18n } from "@/i18n";

interface Props {
  draft: Draft | null;
  id: string | null;
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
export function Preview({ draft, id }: Props) {
  const { t } = useI18n();
  const [html, setHtml] = useState("");
  const [failed, setFailed] = useState<string | null>(null);
  const frame = useRef<HTMLIFrameElement>(null);

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
        if ((err as Error).name !== "AbortError") setFailed(String(err));
      }
    }, 250);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [draft, id, t]);

  // The document is written into the frame rather than assigned to srcdoc, so
  // that the scroll position survives a re-render and an author does not lose
  // their place on every keystroke.
  useEffect(() => {
    const doc = frame.current?.contentDocument;
    if (!doc || !html) return;

    const scroll = doc.documentElement.scrollTop;
    doc.open();
    doc.write(html);
    doc.close();
    doc.documentElement.scrollTop = scroll;
  }, [html]);

  return (
    <div className="relative h-full">
      {failed && (
        <Alert variant="destructive" className="absolute inset-x-0 top-0 z-10 rounded-none">
          <XCircle />
          <AlertTitle>{failed}</AlertTitle>
        </Alert>
      )}
      <iframe
        ref={frame}
        title="Preview"
        className="h-full w-full border-0 bg-white"
        sandbox="allow-same-origin"
      />
    </div>
  );
}
