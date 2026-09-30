/**
 * The files a draft names, as its own bundle holds them: by their path
 * within it, the way a link beside the page writes them.
 */

/** normal is a link's target as a path within the bundle: undecorated, unescaped, without ./ in front. */
function normal(target: string): string {
  let path = target.trim().replace(/^<|>$/g, "").split(/[?#]/)[0];
  try {
    path = decodeURI(path);
  } catch {
    // An escape that does not read is kept as written.
  }
  return path.replace(/^\.\//, "");
}

const special = /[.*+?^${}()|[\]\\]/g;

/**
 * mentions reports whether the text names a file, however it does: a link or
 * a picture, a reference definition, HTML, or a shortcode's parameter. The
 * name has to stand whole, so data.svg is not a.svg, and may be written
 * escaped, as my%20photo.jpg.
 */
export function mentions(body: string, name: string): boolean {
  for (const form of new Set([name, encodeURI(name)])) {
    const quoted = form.replace(special, "\\$&");
    if (new RegExp(`(?:^|[\\s"'(=<\\[]|\\./)${quoted}(?=$|[\\s"')>\\]#?])`, "m").test(body)) return true;
  }
  return false;
}

/** inFields maps every value the front matter holds to the field that holds it. */
export function inFields(meta: Record<string, unknown>): Map<string, string> {
  const found = new Map<string, string>();
  const walk = (key: string, value: unknown) => {
    if (typeof value === "string") found.set(normal(value), key);
    else if (Array.isArray(value)) value.forEach((each) => walk(key, each));
    else if (value && typeof value === "object") Object.values(value).forEach((each) => walk(key, each));
  };
  for (const [key, value] of Object.entries(meta)) walk(key, value);
  return found;
}
