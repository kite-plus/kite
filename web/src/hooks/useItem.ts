import { useCallback, useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { api, ApiError, heard, type Draft, type Item, type Media } from "@/api/client";
import { dropUnsaved, readUnsaved, sameDraft, unsavedKey, writeUnsaved } from "@/lib/unsaved";

/** failureOf describes a request that threw instead of being answered. */
function failureOf(err: unknown): { code?: string; detail: string } {
  return err instanceof ApiError
    ? { code: err.code, detail: err.message }
    : { detail: err instanceof Error ? err.message : String(err) };
}

/** Conflict is what the server sends when an edit lost a race. */
export interface Conflict {
  expected_revision: string;
  actual_revision: string;
  theirs?: Item;
}

type Status = "loading" | "ready" | "saving" | "conflict" | "error";

/** How long a draft waits after the last change before it saves itself. */
const idle = 2000;
/** How long a draft goes unsaved at most while the typing goes on. */
const longest = 15000;
/** How long an automatic save waits to try again when the server was out of reach. */
const retry = 10000;
/** How long a change waits before this browser keeps a copy of it. */
const keepAfter = 300;

/**
 * autosaves says whether an edit saves itself: only a draft, which no reader
 * sees, and a new item only once it has something in it. A published item
 * changes the site when it is saved, so that stays the author's decision.
 */
function autosaves(draft: Draft | null, base: Item | null): boolean {
  if (!draft || draft.status !== "draft") return false;
  if (base) return base.status === "draft";
  return draft.title.trim() !== "" || draft.body.trim() !== "";
}

/**
 * useItem holds one item being edited, together with the revision it was
 * loaded at.
 *
 * The revision travels back in If-Match on every save. That is the whole
 * conflict story: the server refuses an edit made against a version it has
 * already replaced, and this keeps hold of the base the author started from
 * so the two can be shown side by side.
 *
 * Nothing typed is lost to a reload, a crash or a lost connection: every
 * change is kept in this browser until the server has it, and put back when
 * the item opens again. A draft also saves itself shortly after the typing
 * stops. Saves run one at a time, whoever asks for them, so a new item is
 * created once however many uploads start together.
 *
 * site names the site for the copy of an item never saved; until it is known,
 * such a copy is not looked for. terms are what a new item starts with, by
 * taxonomy, as the site's default category. onCreated hears a new item's id
 * once, when its first save returns, whichever save that was.
 */
export function useItem(
  id: string | null,
  kind: string,
  {
    site,
    terms,
    onCreated,
  }: { site?: string; terms?: Record<string, string[]>; onCreated?: (id: string) => void } = {},
) {
  const queryClient = useQueryClient();

  const [draft, setDraft] = useState<Draft | null>(null);
  const [status, setStatus] = useState<Status>("loading");
  // The code travels with the message so the editor can say it in the
  // operator's language; the server's own sentence names the file or the
  // revision a general phrase cannot.
  const [error, setError] = useState<{ code?: string; detail: string } | null>(null);
  const [conflict, setConflict] = useState<Conflict | null>(null);
  const [dirty, setDirty] = useState(false);
  const [savedAt, setSavedAt] = useState<Date | null>(null);
  // Set when the item opened with work this browser had kept, and when that was.
  const [restored, setRestoredState] = useState<{ at: number } | null>(null);
  // Mirrors restored for the timers: a draft brought back does not save
  // itself until the author keeps it or goes on writing.
  const waiting = useRef(false);
  const setRestored = useCallback((next: { at: number } | null) => {
    waiting.current = next !== null;
    setRestoredState(next);
  }, []);

  // The version the editor started from, kept for the conflict view.
  const base = useRef<Item | null>(null);
  const revision = useRef<string>("");
  const editVersion = useRef(0);

  // What the saves and timers read, so a callback made before a change still
  // sends the latest text to the latest id.
  const current = useRef<Draft | null>(null);
  const itemId = useRef<string | null>(id);
  const unsaved = useRef(false);
  const state = useRef<Status>("loading");
  const running = useRef<Promise<string | null> | null>(null);
  // Why the last save failed: the server out of reach is worth another try.
  const failure = useRef<"unreachable" | "refused" | null>(null);
  const autosaveTimer = useRef<number | undefined>(undefined);
  const keepTimer = useRef<number | undefined>(undefined);
  // When the oldest change not yet saved was made.
  const pendingSince = useRef<number | null>(null);
  const siteRef = useRef(site);
  siteRef.current = site;
  const created = useRef(onCreated);
  created.current = onCreated;
  // Read when a new item opens, so a refetch of the types does not reset it.
  const startTerms = useRef(terms);
  startTerms.current = terms;
  // A save can return after the page has gone; its id then moves nothing.
  const mounted = useRef(true);

  const show = useCallback((next: Draft | null) => {
    current.current = next;
    setDraft(next);
  }, []);
  const mark = useCallback((next: boolean) => {
    unsaved.current = next;
    setDirty(next);
    if (!next) pendingSince.current = null;
  }, []);
  const enter = useCallback((next: Status) => {
    state.current = next;
    setStatus(next);
  }, []);

  /** keep writes the unsaved work to this browser now. */
  const keep = useCallback(() => {
    window.clearTimeout(keepTimer.current);
    const fresh = siteRef.current === undefined ? null : unsavedKey(null, kind, siteRef.current);
    const key = itemId.current ? unsavedKey(itemId.current, kind, "") : fresh;
    if (!key || !unsaved.current || !current.current) return;
    writeUnsaved(key, {
      v: 1,
      id: itemId.current,
      revision: revision.current,
      at: Date.now(),
      draft: current.current,
    });
    // The copy made before the first save now lives under the item's id.
    if (itemId.current && fresh) dropUnsaved(fresh);
  }, [kind]);

  /** forget removes this browser's copy, once it is stored or thrown away. */
  const forget = useCallback(() => {
    window.clearTimeout(keepTimer.current);
    if (itemId.current) dropUnsaved(unsavedKey(itemId.current, kind, ""));
    if (siteRef.current !== undefined) dropUnsaved(unsavedKey(null, kind, siteRef.current));
  }, [kind]);

  useEffect(() => {
    // A first save reopens the editor under the id it was given. That item is
    // already here, and loading it again would tear the editor down under
    // the author's cursor.
    if (id && base.current?.id === id) return;
    window.clearTimeout(autosaveTimer.current);
    if (!id) {
      // Nothing to load: a new item starts from an empty draft of its kind,
      // and gets its id from the server when it is first saved.
      base.current = null;
      revision.current = "";
      editVersion.current = 0;
      itemId.current = null;
      setConflict(null);
      setSavedAt(null);
      const key = site === undefined ? null : unsavedKey(null, kind, site);
      const kept = key ? readUnsaved(key) : null;
      if (kept) {
        show(kept.draft);
        mark(true);
        pendingSince.current = Date.now();
        setRestored({ at: kept.at });
      } else {
        // The default terms are a start, not an edit: an item left untouched
        // is not saved for them.
        const taxonomies = startTerms.current && structuredClone(startTerms.current);
        show({ kind, title: "", status: "draft", body: "", ...(taxonomies && { taxonomies }) });
        mark(false);
        setRestored(null);
      }
      enter("ready");
      return;
    }
    let cancelled = false;
    enter("loading");

    (async () => {
      let answer;
      try {
        answer = await api.GET("/contents/{id}", { params: { path: { id } } });
      } catch (err) {
        if (cancelled) return;
        setError(failureOf(err));
        enter("error");
        return;
      }
      const { data, error, response } = answer;
      if (cancelled) return;
      if (error || !data) {
        setError({
          code: error?.error.code,
          detail: error?.error.message ?? "",
        });
        enter("error");
        return;
      }
      base.current = data;
      revision.current = response.headers.get("ETag") ?? "";
      editVersion.current = 0;
      itemId.current = data.id;
      setConflict(null);
      setSavedAt(null);

      const key = unsavedKey(data.id, kind, "");
      const kept = readUnsaved(key);
      if (kept && !sameDraft(kept.draft, draftOf(data))) {
        // Put back what was typed here and not stored. It keeps the revision
        // it was typed against, so if the item has changed since, saving it
        // asks which version to keep rather than overwriting the other.
        show(kept.draft);
        if (kept.revision) revision.current = kept.revision;
        mark(true);
        pendingSince.current = Date.now();
        setRestored({ at: kept.at });
      } else {
        if (kept) dropUnsaved(key);
        show(draftOf(data));
        mark(false);
        setRestored(null);
      }
      enter("ready");
    })();

    return () => {
      cancelled = true;
    };
  }, [id, kind, site, show, mark, enter, setRestored]);

  /** run makes one save request, with whatever the draft holds by then. */
  const run = useCallback(async (): Promise<string | null> => {
    const sent = current.current;
    if (!sent) return null;
    const target = itemId.current;
    const savedVersion = editVersion.current;
    enter("saving");
    setError(null);

    let result;
    try {
      result = target
        ? await api.PUT("/contents/{id}", {
            params: { path: { id: target }, header: { "If-Match": revision.current } },
            body: sent,
          })
        : await api.POST("/contents", { body: sent });
    } catch (err) {
      // Nothing reached the server, so nothing was stored: the draft stays
      // as it is, still marked unsaved and kept here, for the next attempt.
      failure.current = "unreachable";
      setError(failureOf(err));
      enter("ready");
      return null;
    }

    if (result.error) {
      // A conflict is not a failure to report and forget: it is a decision
      // the author has to make, so it is kept until they make it.
      const body = result.error as unknown as {
        conflict?: Conflict;
        error: { code?: string; message: string };
      };
      failure.current = "refused";
      if (result.response.status === 409 && body.conflict) {
        setConflict(body.conflict);
        enter("conflict");
        return null;
      }
      setError({ code: body.error?.code, detail: body.error?.message ?? "" });
      enter("ready");
      return null;
    }

    const saved = result.data as Item;
    failure.current = null;
    base.current = saved;
    revision.current = result.response.headers.get("ETag") ?? "";
    itemId.current = saved.id;
    setSavedAt(new Date());
    setRestored(null);
    if (editVersion.current === savedVersion) {
      show(draftOf(saved));
      mark(false);
      forget();
    } else {
      // Typing went on during the save: that stays unsaved, and is kept
      // here under the item's id and its new revision.
      keep();
    }
    enter("ready");
    if (!target && mounted.current) created.current?.(saved.id);
    void queryClient.invalidateQueries({ queryKey: ["contents"] });
    void queryClient.invalidateQueries({ queryKey: ["site"] });
    return saved.id;
  }, [queryClient, show, mark, enter, keep, forget]);

  /**
   * save stores the draft and resolves to the item's id, or null on failure.
   * A save asked for while another runs waits for it, then stores what is
   * still unsaved, so two never race each other or create an item twice.
   */
  const save = useCallback(async (): Promise<string | null> => {
    window.clearTimeout(autosaveTimer.current);
    while (running.current) await running.current;
    if (!current.current) return null;
    if (itemId.current && !unsaved.current) return itemId.current;
    const request = run();
    running.current = request;
    try {
      return await request;
    } finally {
      running.current = null;
    }
  }, [run]);

  const saveRef = useRef(save);
  saveRef.current = save;

  /** plan sets the next automatic save, when this draft saves itself. */
  const plan = useCallback((delay?: number) => {
    window.clearTimeout(autosaveTimer.current);
    if (!unsaved.current || !autosaves(current.current, base.current)) return;
    if (waiting.current) return;
    if (state.current === "conflict" || state.current === "error") return;
    const since = pendingSince.current ?? Date.now();
    const wait = delay ?? Math.max(0, Math.min(idle, longest - (Date.now() - since)));
    autosaveTimer.current = window.setTimeout(async () => {
      if (state.current === "conflict" || state.current === "error") return;
      if (!unsaved.current || !autosaves(current.current, base.current)) return;
      await saveRef.current();
      if (failure.current === "unreachable") plan(retry);
      // Changes made while it saved are planned for by their own edits.
    }, wait);
  }, []);

  const edit = useCallback(
    (patch: Partial<Draft>) => {
      if (!current.current) return;
      editVersion.current += 1;
      show({ ...current.current, ...patch });
      // Writing on over what was brought back keeps it.
      if (waiting.current) setRestored(null);
      if (pendingSince.current === null) pendingSince.current = Date.now();
      mark(true);
      window.clearTimeout(keepTimer.current);
      keepTimer.current = window.setTimeout(keep, keepAfter);
      plan();
    },
    [show, mark, keep, plan, setRestored],
  );

  // Whatever the page is doing when it is hidden or closed, the unsaved work
  // is written here first: local storage finishes before the page goes.
  useEffect(() => {
    const now = () => {
      if (unsaved.current) keep();
    };
    const hidden = () => {
      if (document.visibilityState === "hidden") now();
    };
    mounted.current = true;
    window.addEventListener("pagehide", now);
    window.addEventListener("beforeunload", now);
    document.addEventListener("visibilitychange", hidden);
    return () => {
      mounted.current = false;
      window.removeEventListener("pagehide", now);
      window.removeEventListener("beforeunload", now);
      document.removeEventListener("visibilitychange", hidden);
      now();
      window.clearTimeout(autosaveTimer.current);
    };
  }, [keep]);

  /** takeTheirs abandons this edit and continues from what is stored. */
  const takeTheirs = useCallback(() => {
    if (!conflict?.theirs) return;
    base.current = conflict.theirs;
    revision.current = `"${conflict.actual_revision}"`;
    editVersion.current = 0;
    show(draftOf(conflict.theirs));
    setConflict(null);
    mark(false);
    forget();
    setRestored(null);
    enter("ready");
  }, [conflict, show, mark, forget, enter]);

  /**
   * mergeWith continues from what is stored with both sides' changes in the
   * draft, unsaved, for the author to look over before it is saved.
   */
  const mergeWith = useCallback(
    (merged: Draft) => {
      if (!conflict?.theirs) return;
      base.current = conflict.theirs;
      revision.current = `"${conflict.actual_revision}"`;
      editVersion.current += 1;
      show(merged);
      setConflict(null);
      mark(true);
      keep();
      enter("ready");
    },
    [conflict, show, mark, keep, enter],
  );

  /** keepOurs saves this edit over the stored one, deliberately. */
  const keepOurs = useCallback(async () => {
    if (!conflict) return null;
    revision.current = `"${conflict.actual_revision}"`;
    setConflict(null);
    enter("ready");
    return save();
  }, [conflict, save, enter]);

  /**
   * discard throws the unsaved work away, here and in this browser's copy,
   * and goes back to what is stored.
   */
  const discard = useCallback(() => {
    window.clearTimeout(autosaveTimer.current);
    forget();
    editVersion.current += 1;
    show(base.current ? draftOf(base.current) : { kind, title: "", status: "draft", body: "" });
    mark(false);
    setRestored(null);
  }, [kind, show, mark, forget]);

  /** forgetUnsaved drops this browser's copy when the page is left without saving. */
  const forgetUnsaved = useCallback(() => {
    window.clearTimeout(autosaveTimer.current);
    mark(false);
    forget();
  }, [mark, forget]);

  const remove = useCallback(async () => {
    const target = itemId.current;
    if (!target) return false;
    let answer;
    try {
      answer = await api.DELETE("/contents/{id}", {
        params: { path: { id: target }, header: { "If-Match": revision.current } },
      });
    } catch (err) {
      setError(failureOf(err));
      return false;
    }
    if (answer.error) {
      setError({ code: answer.error.error.code, detail: answer.error.error.message });
      return false;
    }
    window.clearTimeout(autosaveTimer.current);
    mark(false);
    forget();
    await queryClient.invalidateQueries({ queryKey: ["contents"] });
    return true;
  }, [queryClient, mark, forget]);

  /**
   * attach uploads a file into an item's bundle and returns what was stored,
   * its link among it. The item is this one unless said otherwise: a first
   * save hands out an id before this hook has been rendered with it.
   */
  const attach = useCallback(
    (
      file: File,
      into: string | null = itemId.current,
      /** onProgress hears the share of the file sent so far, from 0 to 100. */
      onProgress?: (percent: number) => void,
      signal?: AbortSignal,
    ): Promise<Media> => {
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
          heard(request.status);
          const body = request.response as (Media & { error?: { code?: string; message?: string } }) | null;
          if (request.status >= 200 && request.status < 300 && body?.link) {
            resolve(body);
            return;
          }
          reject(new ApiError(body?.error?.code ?? "internal", body?.error?.message ?? "upload failed"));
        };
        request.onerror = () => reject(new ApiError("unreachable", "upload failed"));
        request.onabort = () => reject(new DOMException("upload cancelled", "AbortError"));
        signal?.addEventListener("abort", () => request.abort(), { once: true });
        request.send(form);
      });
    },
    [],
  );

  return {
    draft,
    base: base.current,
    status,
    error,
    conflict,
    dirty,
    savedAt,
    restored,
    /** Whether this item saves itself as it is edited. */
    autosaves: autosaves(draft, base.current),
    edit,
    save,
    remove,
    attach,
    takeTheirs,
    keepOurs,
    mergeWith,
    discard,
    forgetUnsaved,
    /**
     * keepRestored accepts what was brought back: a draft then saves it as it
     * saves any change. Until then it stays only in this browser, so the
     * author sees what came back before anything is stored.
     */
    keepRestored: () => {
      setRestored(null);
      plan();
    },
  };
}

/** draftOf is an item as the editor holds it. */
export function draftOf(item: Item): Draft {
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
