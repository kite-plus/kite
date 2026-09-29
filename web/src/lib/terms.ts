/**
 * termSlug writes a term as its address does, the way content.TermSlug does
 * on the server: trimmed, with A to Z in lower case and each run of spaces,
 * tabs and slashes one dash. Terms with one slug are one term however each
 * item writes it, as Go and go are both /tags/go/.
 */
export function termSlug(term: string): string {
  let slug = "";
  let dash = true;
  for (const ch of term.trim()) {
    if (ch === " " || ch === "\t" || ch === "/") {
      if (!dash) slug += "-";
      dash = true;
      continue;
    }
    slug += ch >= "A" && ch <= "Z" ? ch.toLowerCase() : ch;
    dash = false;
  }
  return slug.replace(/^-+|-+$/g, "");
}
