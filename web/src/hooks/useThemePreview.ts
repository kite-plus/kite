import { useEffect, useRef, useState } from "react";

import { api, ApiError, unwrap } from "@/api/client";

// How long the settings have to rest before the preview is drawn again.
const settle = 300;

function close(token: string) {
  // keepalive lets it go out even as the page itself is leaving.
  void fetch(`/api/v1/previews/${token}`, { method: "DELETE", keepalive: true, credentials: "same-origin" });
}

/**
 * useThemePreview keeps a preview of the site drawn with a theme and the
 * settings being edited. It opens one once the settings are known, draws it
 * again a moment after they stop changing, and closes it when the page goes.
 *
 * drawn counts the times it has been drawn again, which is what a frame
 * showing it reloads on.
 */
export function useThemePreview(theme: string, settings: Record<string, unknown> | undefined) {
  const [url, setUrl] = useState<string | null>(null);
  const [drawn, setDrawn] = useState(0);
  const [failed, setFailed] = useState<ApiError | null>(null);
  const token = useRef<string | null>(null);
  // The settings the preview was last drawn with, and the ones being edited.
  const shown = useRef<unknown>(undefined);
  const latest = useRef(settings);
  latest.current = settings;
  const ready = settings !== undefined;

  const redraw = useRef(async () => {});
  redraw.current = async () => {
    const current = token.current;
    const wanted = latest.current;
    if (!current || wanted === shown.current) return;
    try {
      const result = await api.PUT("/previews/{token}", {
        params: { path: { token: current } },
        body: { theme, settings: wanted },
      });
      // One left unused long enough is forgotten, so another is opened.
      if (result.response.status === 404) {
        const preview = unwrap(await api.POST("/previews", { body: { theme, settings: wanted } }));
        token.current = preview.token;
        setUrl(preview.url);
      } else {
        unwrap(result);
      }
      shown.current = wanted;
      setFailed(null);
      setDrawn((n) => n + 1);
    } catch (err) {
      setFailed(asApiError(err));
    }
  };

  useEffect(() => {
    if (!ready) return;
    let live = true;
    const wanted = latest.current;
    void api
      .POST("/previews", { body: { theme, settings: wanted } })
      .then(unwrap)
      .then((preview) => {
        if (!live) {
          close(preview.token);
          return;
        }
        token.current = preview.token;
        shown.current = wanted;
        setUrl(preview.url);
        setFailed(null);
        // Settings changed while it was being opened are drawn now.
        void redraw.current();
      })
      .catch((err: unknown) => live && setFailed(asApiError(err)));
    return () => {
      live = false;
      if (token.current) close(token.current);
      token.current = null;
    };
  }, [ready, theme]);

  useEffect(() => {
    if (!ready || settings === shown.current) return;
    const timer = setTimeout(() => void redraw.current(), settle);
    return () => clearTimeout(timer);
  }, [settings, ready]);

  return { url, drawn, failed };
}

function asApiError(err: unknown): ApiError {
  return err instanceof ApiError ? err : new ApiError("internal", String(err));
}
