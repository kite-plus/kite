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

/**
 * resolveLink turns a link written relative to a page into one the admin can
 * load. A link from the site's root, as "/uploads/a.jpg", is under home, the
 * path the site is previewed at, as the build puts it under the path the
 * site is published at.
 */
export function resolveLink(link: string, base?: string, home = "/"): string {
  const root = home.replace(/\/$/, "");
  if (root && link.startsWith("/") && !link.startsWith("//") && link !== root && !link.startsWith(root + "/")) {
    link = root + link;
  }
  try {
    return new URL(link, new URL(base ?? "/", window.location.origin)).toString();
  } catch {
    return link;
  }
}
