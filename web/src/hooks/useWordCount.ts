import { useEffect, useRef, useState } from "react";

import { api, type Draft } from "@/api/client";

/**
 * useWordCount is how many words a draft's page will say it has.
 *
 * The server counts, as it renders the preview: counting the markdown here
 * would be a second reading of it, bound to disagree with the site's. A new
 * count waits for a pause in typing, and the last one stands until it comes.
 */
export function useWordCount(draft: Draft | null, id: string | null): number | undefined {
  const [words, setWords] = useState<number>();
  // Read when the pause comes, so that editing anything but the body does not
  // ask again.
  const latest = useRef(draft);
  latest.current = draft;
  const counted = useRef(false);
  const body = draft?.body;

  useEffect(() => {
    if (body === undefined) return;
    const controller = new AbortController();
    const timer = setTimeout(
      async () => {
        const current = latest.current;
        if (!current) return;
        try {
          const { data } = await api.POST("/wordcount", {
            params: { query: id ? { id } : {} },
            body: current,
            signal: controller.signal,
          });
          if (data) {
            counted.current = true;
            setWords(data.words);
          }
        } catch {
          // Unreachable, or overtaken by the next count: the last one stands.
        }
      },
      // The first count is asked for at once.
      counted.current ? 300 : 0,
    );
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [body, id]);

  return words;
}
