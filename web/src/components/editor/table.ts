import type { JSONContent, MarkdownRendererHelpers } from "@tiptap/core";
import { Table } from "@tiptap/extension-table";

/** The delimiter cell for each alignment a cell can carry; none is "---". */
const rules = new Map([
  ["left", ":---"],
  ["center", ":---:"],
  ["right", "---:"],
]);

/**
 * row writes cells as "| a | b |". Padding them into columns would rewrite
 * every row whenever one cell grew, and lines up nothing once a cell holds
 * CJK text.
 */
const row = (cells: string[]) => `|${cells.map((cell) => (cell ? ` ${cell} ` : " ")).join("|")}|`;

/** cellsOf splits a line of a table's markdown into trimmed cells, as marked does. */
function cellsOf(line: string): string[] {
  const cells = [""];
  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    if (char === "|") cells.push("");
    // An escaped character stays in its cell, an escaped pipe too.
    else if (char === "\\") cells[cells.length - 1] += char + (line[++i] ?? "");
    else cells[cells.length - 1] += char;
  }
  if (!cells[0].trim()) cells.shift();
  if (cells.length > 0 && !cells[cells.length - 1].trim()) cells.pop();
  return cells.map((cell) => cell.trim());
}

/**
 * cellText is a cell on one line. The lines in it, from a break or a second
 * paragraph, are joined by <br>, which reads back as a break, and a bare pipe
 * is escaped, in code too, so it cannot end the cell.
 */
function cellText(cell: JSONContent, h: MarkdownRendererHelpers): string {
  return (cell.content ?? [])
    .map((block) => h.renderChildren(block))
    .join("\n")
    .replace(/[ \t]*\r?\n[ \t]*/g, "<br>")
    .trim()
    .replace(/\\*\|/g, (run) => (run.length % 2 === 1 ? `\\${run}` : run));
}

function renderTable(node: JSONContent, h: MarkdownRendererHelpers): string {
  const rows = (node.content ?? []).map((tableRow) =>
    (tableRow.content ?? []).map((cell) => ({
      text: cellText(cell, h),
      header: cell.type === "tableHeader",
      rule: rules.get(cell.attrs?.align),
    })),
  );
  const width = rows.reduce((most, cells) => Math.max(most, cells.length), 0);
  if (width === 0) return "";
  const columns = Array.from({ length: width }, (_, i) => i);
  const texts = (cells: (typeof rows)[number]) => columns.map((i) => cells[i]?.text ?? "");

  const hasHeader = rows[0].some((cell) => cell.header);
  // A markdown table cannot go without a header row, so one is left empty.
  const header = row(hasHeader ? texts(rows[0]) : columns.map(() => ""));
  const body = (hasHeader ? rows.slice(1) : rows).map((cells) => row(texts(cells)));
  const delimiter = row(columns.map((i) => rows.find((cells) => cells[i]?.rule)?.[i].rule ?? "---"));

  // A row that still reads as a line of the source is written as that line,
  // whatever its padding. Line 1 of the source is its delimiter row.
  const source: string[] = typeof node.attrs?.source === "string" ? node.attrs.source.split("\n") : [];
  const spellings = new Map<string, string[]>();
  source.forEach((text, i) => {
    if (i === 1) return;
    const key = row(cellsOf(text));
    spellings.set(key, [...(spellings.get(key) ?? []), text]);
  });
  const spell = (canonical: string) => spellings.get(canonical)?.shift() ?? canonical;
  const sameRule =
    source.length > 1 && row(cellsOf(source[1]).map((cell) => cell.replace(/-+/, "---"))) === delimiter;

  return [spell(header), sameRule ? source[1] : delimiter, ...body.map(spell)].join("\n");
}

/**
 * StableTable is the stock table with markdown that holds still. The stock
 * one sets a table off with extra blank lines and pads every cell to its
 * column, so any save rewrites every table in the body; this one writes back
 * each row that still reads the same exactly as it was read, and a new or
 * changed row compact.
 */
export const StableTable = Table.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      // The table's markdown as it was read. It is kept out of the page and
      // the clipboard, so a pasted table is written fresh.
      source: { default: null, rendered: false, parseHTML: () => null },
    };
  },

  parseMarkdown: (token, helpers) => {
    const table = Table.config.parseMarkdown?.(token, helpers) as JSONContent;
    return { ...table, attrs: { ...table.attrs, source: token.raw?.replace(/\n+$/, "") ?? null } };
  },

  renderMarkdown: (node, helpers) => renderTable(node, helpers),
});
