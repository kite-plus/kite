import type { Draft } from "@/api/client";

/**
 * Three versions of an item, when an edit lost a race: the base both sides
 * started from, ours as edited here, and theirs as stored meanwhile.
 */

/**
 * Side says who changed a field since the base: one side, both the same
 * way, both in different paragraphs of the text, which merge, or both
 * differently, which do not.
 */
export type Side = "ours" | "theirs" | "same" | "apart" | "both";

/** Field names a part of an item: a property of the draft, a taxonomy, or a key of its front matter. */
export type Field =
  | { kind: "property"; key: "title" | "slug" | "status" | "published_at" | "locale" | "aliases" }
  | { kind: "taxonomy"; key: string }
  | { kind: "meta"; key: string }
  | { kind: "body" };

export interface Change {
  field: Field;
  base: unknown;
  ours: unknown;
  theirs: unknown;
  side: Side;
}

/** same compares two values as JSON would write them, whatever order an object's keys are in. */
function same(a: unknown, b: unknown): boolean {
  return canonical(a) === canonical(b);
}

function canonical(value: unknown): string {
  return JSON.stringify(value ?? null, (_, v) =>
    v && typeof v === "object" && !Array.isArray(v)
      ? Object.fromEntries(Object.entries(v).sort(([x], [y]) => (x < y ? -1 : x > y ? 1 : 0)))
      : v,
  );
}

function sideOf(base: unknown, ours: unknown, theirs: unknown): Side | null {
  const oursChanged = !same(base, ours);
  const theirsChanged = !same(base, theirs);
  if (!oursChanged && !theirsChanged) return null;
  if (oursChanged && theirsChanged) return same(ours, theirs) ? "same" : "both";
  return oursChanged ? "ours" : "theirs";
}

const properties = ["title", "slug", "status", "published_at", "locale", "aliases"] as const;

/** fieldsOf lists every part the three versions have between them, in the order a form shows them. */
function fieldsOf(base: Draft, ours: Draft, theirs: Draft): Field[] {
  const taxonomies = new Set([base, ours, theirs].flatMap((d) => Object.keys(d.taxonomies ?? {})));
  const meta = new Set([base, ours, theirs].flatMap((d) => Object.keys(d.meta ?? {})));
  return [
    ...properties.map((key) => ({ kind: "property", key }) as const),
    ...[...taxonomies].sort().map((key) => ({ kind: "taxonomy", key }) as const),
    ...[...meta].sort().map((key) => ({ kind: "meta", key }) as const),
    { kind: "body" },
  ];
}

function valueOf(draft: Draft, field: Field): unknown {
  switch (field.kind) {
    case "property":
      return draft[field.key];
    case "taxonomy":
      return draft.taxonomies?.[field.key] ?? [];
    case "meta":
      return draft.meta?.[field.key];
    case "body":
      return draft.body;
  }
}

/** compare lists the parts that differ between the three versions, and who changed each. */
export function compare(base: Draft, ours: Draft, theirs: Draft): Change[] {
  const out: Change[] = [];
  for (const field of fieldsOf(base, ours, theirs)) {
    const [b, o, t] = [valueOf(base, field), valueOf(ours, field), valueOf(theirs, field)];
    let side = sideOf(b, o, t);
    // Changes to different paragraphs of the text do not collide.
    if (side === "both" && field.kind === "body" && mergeText(String(b ?? ""), String(o ?? ""), String(t ?? "")) !== null) {
      side = "apart";
    }
    if (side) out.push({ field, base: b, ours: o, theirs: t, side });
  }
  return out;
}

/**
 * merge is both sides' changes in one draft, or null when a part was changed
 * both ways: a field each side set differently, or a paragraph each side
 * rewrote. What only one side changed is taken from it.
 */
export function merge(base: Draft, ours: Draft, theirs: Draft): Draft | null {
  const out: Draft = {
    ...theirs,
    meta: { ...(theirs.meta ?? {}) },
    taxonomies: { ...(theirs.taxonomies ?? {}) },
  };
  for (const change of compare(base, ours, theirs)) {
    let value: unknown;
    if (change.field.kind === "body") {
      const text = mergeText(String(change.base ?? ""), String(change.ours ?? ""), String(change.theirs ?? ""));
      if (text === null) return null;
      value = text;
    } else if (change.side === "both") {
      return null;
    } else {
      value = change.side === "theirs" ? change.theirs : change.ours;
    }
    set(out, change.field, value);
  }
  return out;
}

function set(draft: Draft, field: Field, value: unknown) {
  switch (field.kind) {
    case "property":
      (draft as unknown as Record<string, unknown>)[field.key] = value;
      break;
    case "taxonomy":
      draft.taxonomies![field.key] = (value as string[]) ?? [];
      break;
    case "meta":
      // A key one side took out goes out of the merge too.
      if (value === undefined) delete draft.meta![field.key];
      else draft.meta![field.key] = value;
      break;
    case "body":
      draft.body = String(value ?? "");
      break;
  }
}

/**
 * paragraphs splits a text into its paragraphs, each with the blank lines
 * after it, so that joining them gives the text back byte for byte.
 */
export function paragraphs(text: string): string[] {
  const out: string[] = [];
  const pattern = /\n[ \t]*\n\s*/g;
  let from = 0;
  for (const match of text.matchAll(pattern)) {
    const end = match.index + match[0].length;
    out.push(text.slice(from, end));
    from = end;
  }
  if (from < text.length) out.push(text.slice(from));
  return out;
}

const key = (paragraph: string) => paragraph.trimEnd();

/**
 * matching pairs the paragraphs of a with those of b that read the same, in
 * order, as a longest common subsequence: at[i] is where a[i] went in b, or -1.
 */
function matching(a: string[], b: string[]): number[] {
  const n = a.length;
  const m = b.length;
  const lengths = Array.from({ length: n + 1 }, () => new Uint32Array(m + 1));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lengths[i][j] = key(a[i]) === key(b[j]) ? lengths[i + 1][j + 1] + 1 : Math.max(lengths[i + 1][j], lengths[i][j + 1]);
    }
  }
  const at = a.map(() => -1);
  for (let i = 0, j = 0; i < n && j < m; ) {
    if (key(a[i]) === key(b[j])) {
      at[i] = j;
      i++;
      j++;
    } else if (lengths[i + 1][j] >= lengths[i][j + 1]) i++;
    else j++;
  }
  return at;
}

/** A region is a stretch of the base and what each side has in its place. */
export interface Region {
  base: string[];
  ours: string[];
  theirs: string[];
}

/**
 * regions splits three texts where the base holds a paragraph neither side
 * touched, so what lies between two such paragraphs is one region each side
 * may have changed.
 */
export function regions(base: string, ours: string, theirs: string): Region[] {
  const b = paragraphs(base);
  const o = paragraphs(ours);
  const t = paragraphs(theirs);
  const inOurs = matching(b, o);
  const inTheirs = matching(b, t);
  const out: Region[] = [];
  let [bi, oi, ti] = [0, 0, 0];
  const cut = (bEnd: number, oEnd: number, tEnd: number) => {
    if (bEnd > bi || oEnd > oi || tEnd > ti) {
      out.push({ base: b.slice(bi, bEnd), ours: o.slice(oi, oEnd), theirs: t.slice(ti, tEnd) });
    }
  };
  for (let i = 0; i < b.length; i++) {
    const [oj, tj] = [inOurs[i], inTheirs[i]];
    // An anchor: kept by both sides, after where each side has got to.
    if (oj >= oi && tj >= ti) {
      cut(i, oj, tj);
      out.push({ base: [b[i]], ours: [o[oj]], theirs: [t[tj]] });
      [bi, oi, ti] = [i + 1, oj + 1, tj + 1];
    }
  }
  cut(b.length, o.length, t.length);
  return out;
}

const joined = (part: string[]) => part.map(key).join("\n");

/**
 * mergeText is a text with both sides' paragraph changes in it, or null when
 * both changed the same stretch differently.
 */
export function mergeText(base: string, ours: string, theirs: string): string | null {
  let out = "";
  for (const region of regions(base, ours, theirs)) {
    const oursChanged = joined(region.ours) !== joined(region.base);
    const theirsChanged = joined(region.theirs) !== joined(region.base);
    if (oursChanged && theirsChanged && joined(region.ours) !== joined(region.theirs)) return null;
    const next = (oursChanged ? region.ours : region.theirs).join("");
    // Paragraphs from the two sides meet with a blank line between them.
    if (out && next && !/\n[ \t]*\n\s*$/.test(out)) out = out.replace(/\n?$/, "\n\n");
    out += next;
  }
  return out;
}
