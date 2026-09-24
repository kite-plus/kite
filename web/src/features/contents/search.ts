import { STATUSES } from "@/hooks/useContents";

/**
 * ContentSearch is a listing's state as its address says it, so a filtered
 * list survives a reload and can be shared as a link.
 */
export interface ContentSearch {
  /** The statuses an item has one of, or absent for any. */
  status?: string[];
  /** trash lists what was deleted instead. */
  trash?: boolean;
  q?: string;
  /** A sortable field, with a leading "-" for descending. */
  sort?: string;
  /** taxonomy:term pairs an item must carry all of. */
  terms?: string[];
  size?: number;
}

export const PAGE_SIZES = [10, 20, 30, 50];
export const DEFAULT_PAGE_SIZE = 20;

const statuses = new Set<string>(STATUSES);

/**
 * validateContentSearch keeps what it understands and drops the rest, so a
 * mangled link opens the plain listing rather than an error.
 */
export function validateContentSearch(search: Record<string, unknown>): ContentSearch {
  const out: ContentSearch = {};
  const status = (Array.isArray(search.status) ? search.status : [search.status]).filter(
    (each): each is string => typeof each === "string" && statuses.has(each),
  );
  if (status.length > 0) out.status = status;
  if (search.trash === true) out.trash = true;
  if (typeof search.q === "string" && search.q.trim()) out.q = search.q;
  if (typeof search.sort === "string" && /^-?[a-z_]+$/.test(search.sort)) out.sort = search.sort;

  const terms = (Array.isArray(search.terms) ? search.terms : [search.terms]).filter(
    (term): term is string => typeof term === "string" && term.includes(":"),
  );
  if (terms.length > 0) out.terms = terms;

  const size = Number(search.size);
  if (PAGE_SIZES.includes(size)) out.size = size;
  return out;
}

/** A date column wants newest first; a title wants A to Z. */
export function defaultSort(field: string): string {
  return field === "title" ? field : `-${field}`;
}

/** termsOf reads the chosen terms of one taxonomy out of the pairs. */
export function termsOf(terms: string[] | undefined, taxonomy: string): string[] {
  return (terms ?? [])
    .filter((pair) => pair.startsWith(`${taxonomy}:`))
    .map((pair) => pair.slice(taxonomy.length + 1));
}

/** withTerms replaces the chosen terms of one taxonomy. */
export function withTerms(
  terms: string[] | undefined,
  taxonomy: string,
  chosen: string[],
): string[] | undefined {
  const others = (terms ?? []).filter((pair) => !pair.startsWith(`${taxonomy}:`));
  const next = [...others, ...chosen.map((term) => `${taxonomy}:${term}`)];
  return next.length > 0 ? next : undefined;
}
