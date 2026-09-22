/** resolveLink turns a link written relative to a page into one the admin can load. */
export function resolveLink(link: string, base?: string): string {
  try {
    return new URL(link, new URL(base ?? "/", window.location.origin)).toString();
  } catch {
    return link;
  }
}
