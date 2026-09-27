import type { Draft } from "@/api/client";

/**
 * Work the editor has not stored yet, kept in this browser so that a reload,
 * a crash or a dead battery does not take it with them. There is one entry
 * per item: under its id, or, for an item never saved, under its site and
 * kind. Local storage rather than IndexedDB, because it writes synchronously
 * and so can still be written while the page is being closed.
 */

const prefix = "kite:unsaved:";

/** How many entries are kept when the browser runs out of room for more. */
const kept = 20;

export interface Unsaved {
  v: 1;
  /** The item's id, or null for an item not saved yet. */
  id: string | null;
  /** The revision the edit started from, empty for a new item. */
  revision: string;
  /** When the entry was written, in milliseconds since the epoch. */
  at: number;
  draft: Draft;
}

/** unsavedKey names the entry of an item, or of a new item of a kind on a site. */
export function unsavedKey(id: string | null, kind: string, site: string): string {
  return prefix + (id ?? `new:${site}:${kind}`);
}

export function readUnsaved(key: string): Unsaved | null {
  try {
    const text = localStorage.getItem(key);
    if (!text) return null;
    const entry = JSON.parse(text) as Partial<Unsaved>;
    if (entry.v !== 1 || !entry.draft || typeof entry.draft.body !== "string") return null;
    return entry as Unsaved;
  } catch {
    return null;
  }
}

/** writeUnsaved keeps an entry, and says whether the browser took it. */
export function writeUnsaved(key: string, entry: Unsaved): boolean {
  const text = JSON.stringify(entry);
  try {
    localStorage.setItem(key, text);
    return true;
  } catch {
    // Storage is full: the oldest entries of other items make room.
    prune(key);
    try {
      localStorage.setItem(key, text);
      return true;
    } catch {
      return false;
    }
  }
}

export function dropUnsaved(key: string) {
  try {
    localStorage.removeItem(key);
  } catch {
    // Storage blocked: nothing was kept, so nothing is left behind.
  }
}

function prune(except: string) {
  try {
    const entries: { key: string; at: number }[] = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (!key?.startsWith(prefix) || key === except) continue;
      entries.push({ key, at: readUnsaved(key)?.at ?? 0 });
    }
    entries.sort((a, b) => b.at - a.at);
    for (const { key } of entries.slice(kept - 1)) localStorage.removeItem(key);
  } catch {
    // Nothing more can be done about a full or blocked storage.
  }
}

/**
 * sameDraft says whether two drafts would store the same item. A value left
 * out, null or empty counts as absent, and the body's surrounding whitespace
 * does not count, as neither editor keeps it apart.
 */
export function sameDraft(a: Draft, b: Draft): boolean {
  return canonical({ ...a, body: a.body.trim() }) === canonical({ ...b, body: b.body.trim() });
}

function canonical(value: unknown): string {
  return JSON.stringify(normalize(value)) ?? "";
}

function normalize(value: unknown): unknown {
  if (Array.isArray(value)) return value.length > 0 ? value.map(normalize) : undefined;
  if (value && typeof value === "object") {
    const out: Record<string, unknown> = {};
    for (const key of Object.keys(value).sort()) {
      const inner = normalize((value as Record<string, unknown>)[key]);
      if (inner !== undefined) out[key] = inner;
    }
    return Object.keys(out).length > 0 ? out : undefined;
  }
  if (value === null || value === "") return undefined;
  return value;
}
