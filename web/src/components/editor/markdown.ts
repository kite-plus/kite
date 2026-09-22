/** The two ways of editing a body: as formatted text, or as the markdown itself. */
export type Mode = "visual" | "source";

/** What the visual editor cannot carry through a save, named so a notice can say which. */
export type Loss = "footnotes" | "html" | "entities";

const checks: [Loss, RegExp][] = [
  ["footnotes", /\[\^[^\]\s]+\]/],
  ["html", /<\/?[a-zA-Z][\w-]*(?:\s[^>]*)?>|<!--/],
  ["entities", /&(?:#\d+|#x[0-9a-f]+|[a-z][a-z0-9]*);/i],
];

/**
 * losses lists the syntax in a body that the visual editor would turn into
 * plain text. Code spans and fences are taken out first: a tag inside them
 * is text already, and survives.
 */
export function losses(body: string): Loss[] {
  const prose = body.replace(/```[\s\S]*?(?:```|$)|~~~[\s\S]*?(?:~~~|$)|`[^`\n]*`/g, "");
  return checks.filter(([, pattern]) => pattern.test(prose)).map(([name]) => name);
}

/** countWords counts a CJK character as a word, and a run of letters as one. */
export function countWords(text: string): number {
  const cjk = /[぀-ヿ㐀-鿿가-힯]/g;
  const characters = text.match(cjk)?.length ?? 0;
  const words = text.replace(cjk, " ").match(/[\p{L}\p{N}]+/gu)?.length ?? 0;
  return characters + words;
}

const modeKey = "kite:editor-mode";

/** preferredMode is the mode last chosen in this browser. */
export function preferredMode(): Mode {
  try {
    return localStorage.getItem(modeKey) === "source" ? "source" : "visual";
  } catch {
    return "visual";
  }
}

export function rememberMode(mode: Mode) {
  try {
    localStorage.setItem(modeKey, mode);
  } catch {
    // Remembering the choice is a convenience, not a requirement.
  }
}
