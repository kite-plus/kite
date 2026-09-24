import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { api, ApiError, type Draft, type Item } from "@/api/client";

/** Conflict is what the server sends when an edit lost a race. */
export interface Conflict {
  expected_revision: string;
  actual_revision: string;
  theirs?: Item;
}

type Status = "loading" | "ready" | "saving" | "conflict" | "error";

/**
 * useItem holds one item being edited, together with the revision it was
 * loaded at.
 *
 * The revision travels back in If-Match on every save. That is the whole
 * conflict story: the server refuses an edit made against a version it has
 * already replaced, and this keeps hold of the base the author started from
 * so the two can be shown side by side.
 */
export function useItem(id: string | null, kind: string) {
  const queryClient = useQueryClient();

  const [draft, setDraft] = useState<Draft | null>(null);
  const [status, setStatus] = useState<Status>("loading");
  // The code travels with the message so the editor can say it in the
  // operator's language; the server's own sentence names the file or the
  // revision a general phrase cannot.
  const [error, setError] = useState<{ code?: string; detail: string } | null>(null);
  const [conflict, setConflict] = useState<Conflict | null>(null);
  const [dirty, setDirty] = useState(false);

  // The version the editor started from, kept for the conflict view.
  const base = useRef<Item | null>(null);
  const revision = useRef<string>("");
  const editVersion = useRef(0);

  useEffect(() => {
    // A first save reopens the editor under the id it was given. That item is
    // already here, and loading it again would tear the editor down under
    // the author's cursor.
    if (id && base.current?.id === id) return;
    if (!id) {
      // Nothing to load: a new item starts from an empty draft of its kind,
      // and gets its id from the server when it is first saved.
      base.current = null;
      revision.current = "";
      editVersion.current = 0;
      setDraft({ kind, title: "", status: "draft", body: "" });
      setConflict(null);
      setDirty(false);
      setStatus("ready");
      return;
    }
    let cancelled = false;
    setStatus("loading");

    (async () => {
      const { data, error, response } = await api.GET("/contents/{id}", {
        params: { path: { id } },
      });
      if (cancelled) return;
      if (error || !data) {
        setError({
          code: error?.error.code,
          detail: error?.error.message ?? "",
        });
        setStatus("error");
        return;
      }
      base.current = data;
      revision.current = response.headers.get("ETag") ?? "";
      editVersion.current = 0;
      setDraft(draftOf(data));
      setConflict(null);
      setDirty(false);
      setStatus("ready");
    })();

    return () => {
      cancelled = true;
    };
  }, [id, kind]);

  const edit = useCallback((patch: Partial<Draft>) => {
    editVersion.current += 1;
    setDraft((current: Draft | null) => (current ? { ...current, ...patch } : current));
    setDirty(true);
  }, []);

  /** save stores the draft and returns the item's id, or null on failure. */
  const save = useCallback(async (): Promise<string | null> => {
    if (!draft) return null;
    const savedVersion = editVersion.current;
    setStatus("saving");
    setError(null);

    const result = id
      ? await api.PUT("/contents/{id}", {
          params: { path: { id }, header: { "If-Match": revision.current } },
          body: draft,
        })
      : await api.POST("/contents", { body: draft });

    if (result.error) {
      // A conflict is not a failure to report and forget: it is a decision
      // the author has to make, so it is kept until they make it.
      const body = result.error as unknown as {
        conflict?: Conflict;
        error: { code?: string; message: string };
      };
      if (result.response.status === 409 && body.conflict) {
        setConflict(body.conflict);
        setStatus("conflict");
        return null;
      }
      setError({ code: body.error?.code, detail: body.error?.message ?? "" });
      setStatus("ready");
      return null;
    }

    base.current = result.data as Item;
    revision.current = result.response.headers.get("ETag") ?? "";
    if (editVersion.current === savedVersion) {
      setDraft(draftOf(result.data as Item));
      setDirty(false);
    }
    setStatus("ready");
    await queryClient.invalidateQueries({ queryKey: ["contents"] });
    await queryClient.invalidateQueries({ queryKey: ["site"] });
    return (result.data as Item).id;
  }, [draft, id, queryClient]);

  /** takeTheirs abandons this edit and continues from what is stored. */
  const takeTheirs = useCallback(() => {
    if (!conflict?.theirs) return;
    base.current = conflict.theirs;
    revision.current = `"${conflict.actual_revision}"`;
    editVersion.current = 0;
    setDraft(draftOf(conflict.theirs));
    setConflict(null);
    setDirty(false);
    setStatus("ready");
  }, [conflict]);

  /** keepOurs saves this edit over the stored one, deliberately. */
  const keepOurs = useCallback(async () => {
    if (!conflict) return null;
    revision.current = `"${conflict.actual_revision}"`;
    setConflict(null);
    return save();
  }, [conflict, save]);

  const remove = useCallback(async () => {
    if (!id) return false;
    const { error } = await api.DELETE("/contents/{id}", {
      params: { path: { id }, header: { "If-Match": revision.current } },
    });
    if (error) {
      setError({ code: error.error.code, detail: error.error.message });
      return false;
    }
    await queryClient.invalidateQueries({ queryKey: ["contents"] });
    return true;
  }, [id, queryClient]);

  /**
   * attach uploads a file into an item's bundle and returns its link. The
   * item is this one unless said otherwise: a first save hands out an id
   * before this hook has been rendered with it.
   */
  const attach = useCallback(
    (
      file: File,
      into: string | null = id,
      /** onProgress hears the share of the file sent so far, from 0 to 100. */
      onProgress?: (percent: number) => void,
      signal?: AbortSignal,
    ): Promise<string> => {
      if (!into) {
        return Promise.reject(new ApiError("invalid_request", "save this item before adding files"));
      }
      // Cancelled while a new item was being saved to make room for the file.
      if (signal?.aborted) return Promise.reject(new DOMException("upload cancelled", "AbortError"));

      const form = new FormData();
      form.append("file", file);

      // XMLHttpRequest rather than fetch: only it reports upload progress.
      return new Promise((resolve, reject) => {
        const request = new XMLHttpRequest();
        request.open("POST", `/api/v1/contents/${into}/media`);
        request.responseType = "json";
        request.upload.onprogress = (event) => {
          if (event.lengthComputable) onProgress?.(Math.round((event.loaded / event.total) * 100));
        };
        request.onload = () => {
          const body = request.response as { link?: string; error?: { code?: string; message?: string } } | null;
          if (request.status >= 200 && request.status < 300 && body?.link) {
            resolve(body.link);
            return;
          }
          reject(new ApiError(body?.error?.code ?? "internal", body?.error?.message ?? "upload failed"));
        };
        request.onerror = () => reject(new ApiError("internal", "upload failed"));
        request.onabort = () => reject(new DOMException("upload cancelled", "AbortError"));
        signal?.addEventListener("abort", () => request.abort(), { once: true });
        request.send(form);
      });
    },
    [id],
  );

  return {
    draft,
    base: base.current,
    status,
    error,
    conflict,
    dirty,
    edit,
    save,
    remove,
    attach,
    takeTheirs,
    keepOurs,
  };
}

function draftOf(item: Item): Draft {
  return {
    kind: item.kind,
    title: item.title,
    slug: item.slug,
    status: item.status,
    body: item.body,
    meta: item.meta,
    taxonomies: item.taxonomies,
    aliases: item.aliases,
    locale: item.locale,
    published_at: item.published_at,
  };
}
