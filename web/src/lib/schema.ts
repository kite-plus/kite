import type { Field } from "@/api/client";
import type { Key, Values } from "@/i18n";

/**
 * What a form needs to know about a declared schema beyond drawing it: which
 * fields hold values, what each defaults to, and what the server would
 * refuse. The rules follow the server's (internal/schema), so a mistake is
 * shown where it is typed rather than after a save comes back refused.
 */

/** valueFields lists the fields that hold values, with each section's in its place. */
export function valueFields(fields: Field[] | undefined): Field[] {
  return (fields ?? []).flatMap((field) =>
    field.type === "section" ? valueFields(field.fields) : [field],
  );
}

/** defaultOf is what a field holds when nothing is stored for it. */
export function defaultOf(field: Field): unknown {
  switch (field.type) {
    case "group":
      return Object.fromEntries(
        valueFields(field.fields)
          .map((child) => [child.key, defaultOf(child)] as const)
          .filter(([, value]) => value !== undefined),
      );
    case "repeat":
      return Array.isArray(field.default) ? field.default : [];
    default:
      return field.default ?? undefined;
  }
}

/** sameValue compares two setting values the way a save would tell them apart. */
export function sameValue(a: unknown, b: unknown): boolean {
  return Object.is(a, b) || JSON.stringify(a ?? null) === JSON.stringify(b ?? null);
}

/**
 * changesOf is what a save writes, by dotted path under prefix, as
 * "theme.settings.": each setting that changed, and for one put back to its
 * default, its removal, so the default applies again and follows the theme or
 * plugin when a new version changes it.
 */
export function changesOf(
  prefix: string,
  fields: Field[],
  stored: string[],
  values: Record<string, unknown>,
  baseline: Record<string, unknown>,
): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const field of fields) {
    const value = values[field.key];
    if (sameValue(value, baseline[field.key])) continue;
    const fallback = defaultOf(field);
    const cleared = value === "" && fallback === undefined;
    if (cleared || sameValue(value, fallback)) {
      if (stored.includes(field.key)) out[prefix + field.key] = null;
    } else {
      out[prefix + field.key] = value;
    }
  }
  return out;
}

const hexColor = /^#(?:[0-9a-f]{3,4}|[0-9a-f]{6}|[0-9a-f]{8})$/i;

/** isColor reports whether a value is a color the server stores. */
export function isColor(value: unknown): value is string {
  return typeof value === "string" && hexColor.test(value);
}

type Translate = (key: Key, values?: Values) => string;

/**
 * problemOf says what the server would refuse about a value, or returns null.
 * A value left empty is only a problem when the field is required.
 */
export function problemOf(field: Field, value: unknown, t: Translate): string | null {
  const empty =
    value === undefined || value === null || (typeof value === "string" && value.trim() === "") ||
    (Array.isArray(value) && value.length === 0);
  if (empty) return field.required && field.type !== "boolean" ? t("form.required") : null;

  switch (field.type) {
    case "color":
      return isColor(value) ? null : t("form.badColor");
    case "url":
      return typeof value === "string" ? linkProblem(value, t) : t("form.badLink");
    case "number": {
      const n = Number(value);
      if (typeof value !== "number" || Number.isNaN(n)) return t("form.badNumber");
      if (field.min !== undefined && n < field.min) return t("form.tooSmall", { min: field.min });
      if (field.max !== undefined && n > field.max) return t("form.tooLarge", { max: field.max });
      return null;
    }
    case "group":
      return entryProblem(field.fields, value, t);
    case "repeat": {
      if (!Array.isArray(value)) return null;
      for (const [i, entry] of value.entries()) {
        const problem = entryProblem(field.fields, entry, t);
        if (problem) return t("form.entryProblem", { n: i + 1, problem });
      }
      return null;
    }
    default:
      return null;
  }
}

function entryProblem(fields: Field[] | undefined, value: unknown, t: Translate): string | null {
  const values = (value ?? {}) as Record<string, unknown>;
  for (const child of valueFields(fields)) {
    const problem = problemOf(child, values[child.key], t);
    if (problem) return `${child.label || child.key}: ${problem}`;
  }
  return null;
}

/** linkProblem accepts a web address, a mail or phone link and a path in the site. */
function linkProblem(text: string, t: Translate): string | null {
  if (/\s/.test(text)) return t("form.badLink");
  const scheme = /^([a-z][a-z0-9+.-]*):/i.exec(text)?.[1]?.toLowerCase();
  if (!scheme) return null;
  if (scheme === "mailto" || scheme === "tel") return null;
  if (scheme === "http" || scheme === "https") {
    try {
      return new URL(text).host ? null : t("form.badLink");
    } catch {
      return t("form.badLink");
    }
  }
  return t("form.badLink");
}

/** firstProblem finds the first field of a form the server would refuse. */
export function firstProblem(
  fields: Field[] | undefined,
  values: Record<string, unknown>,
  t: Translate,
): { field: Field; problem: string } | null {
  for (const field of valueFields(fields)) {
    const problem = problemOf(field, values[field.key], t);
    if (problem) return { field, problem };
  }
  return null;
}
