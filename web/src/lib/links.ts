/**
 * siteHome is where the server this studio runs on shows the site: the path
 * of the base URL, on this origin, since a site published under a path is
 * previewed under it too.
 */
export function siteHome(site?: { base_url: string }): string {
  try {
    return new URL(site?.base_url ?? "/", window.location.origin).pathname;
  } catch {
    return "/";
  }
}

/** resolveLink turns a link written relative to a page into one the admin can load. */
export function resolveLink(link: string, base?: string): string {
  try {
    return new URL(link, new URL(base ?? "/", window.location.origin)).toString();
  } catch {
    return link;
  }
}
