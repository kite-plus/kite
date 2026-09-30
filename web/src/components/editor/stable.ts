/** Lex splits markdown into its top-level tokens, as the editor's own parser does. */
export type Lex = (markdown: string) => { type: string; raw: string }[];

interface Block {
  type: string;
  /** raw is the block as written, with the blank lines after it. */
  raw: string;
  /** key is the block without those blank lines, which two versions are compared by. */
  key: string;
}

const trailing = /\s+$/;

function blocksOf(lex: Lex, markdown: string): Block[] {
  const blocks: Block[] = [];
  for (const token of lex(markdown)) {
    const last = blocks.at(-1);
    if (token.type === "space") {
      if (last) last.raw += token.raw;
      continue;
    }
    blocks.push({ type: token.type, raw: token.raw, key: token.raw.replace(trailing, "") });
  }
  return blocks;
}

/** Past this many comparisons the middle of two versions is not matched. */
const most = 1_000_000;

/**
 * matching pairs the items of a and b that same finds alike, in order, as a
 * longest common subsequence: at[i] is the index in b that a[i] pairs with, or
 * -1. An edit touches one place, so what comes before and after it pairs up
 * without the table.
 */
function matching<T>(a: T[], b: T[], same: (x: T, y: T) => boolean): number[] {
  const at = a.map(() => -1);
  let start = 0;
  while (start < a.length && start < b.length && same(a[start], b[start])) {
    at[start] = start;
    start++;
  }
  let endA = a.length;
  let endB = b.length;
  while (endA > start && endB > start && same(a[endA - 1], b[endB - 1])) {
    endA--;
    endB--;
    at[endA] = endB;
  }
  const n = endA - start;
  const m = endB - start;
  if (n === 0 || m === 0 || n * m > most) return at;

  // lengths[i][j] is how long the longest match of a[start+i:] and b[start+j:] is.
  const lengths = Array.from({ length: n + 1 }, () => new Uint32Array(m + 1));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lengths[i][j] = same(a[start + i], b[start + j])
        ? lengths[i + 1][j + 1] + 1
        : Math.max(lengths[i + 1][j], lengths[i][j + 1]);
    }
  }
  for (let i = 0, j = 0; i < n && j < m; ) {
    if (same(a[start + i], b[start + j])) {
      at[start + i] = start + j;
      i++;
      j++;
    } else if (lengths[i + 1][j] >= lengths[i][j + 1]) i++;
    else j++;
  }
  return at;
}

/**
 * keeper returns what writes the editor's markdown of a body back as the body
 * was written wherever a block did not change. The editor writes a whole body
 * again on every edit, in its own way: `snake_case` as `snake\_case`, `_a_` as
 * `*a*`, a wrapped paragraph on one line. Only the blocks an author changed
 * should differ.
 *
 * baseline is the editor's writing of original before any edit. A block of it
 * stands for the block of original it lines up with only when write, the
 * editor's own reading and writing of some markdown, makes that block into
 * it, so a block the editor split or joined is never written as another.
 */
export function keeper(
  lex: Lex,
  write: (markdown: string) => string,
  original: string,
  baseline: string,
): (fresh: string) => string {
  const was = blocksOf(lex, original);
  const unedited = blocksOf(lex, baseline);
  const source = matching(unedited, was, (x, y) => x.type === y.type).map((i, k) =>
    i >= 0 && write(was[i].raw).replace(trailing, "") === unedited[k].key ? i : -1,
  );

  return (fresh) => {
    if (fresh === baseline) return original;
    const now = blocksOf(lex, fresh);
    const kept = matching(now, unedited, (x, y) => x.key === y.key);

    let out = "";
    let previous = -2; // the block of original written last, when one was
    for (const [j, block] of now.entries()) {
      const i = kept[j] >= 0 ? source[kept[j]] : -1;
      // Blocks that did not follow each other in original are set apart by a
      // blank line, which one written as a continuation of the other would
      // not survive.
      if (out && !(i >= 0 && i === previous + 1) && !out.endsWith("\n\n")) {
        out = out.replace(/\n?$/, "\n\n");
      }
      out += i >= 0 ? was[i].raw : block.raw;
      previous = i >= 0 ? i : -2;
    }
    return out;
  };
}
