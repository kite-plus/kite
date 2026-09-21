import { useSyncExternalStore, type MouseEvent } from "react";

export type Route =
  | { name: "dashboard" }
  | { name: "list"; kind: string }
  | { name: "edit"; kind: string; id: string | null }
  | { name: "taxonomies" }
  | { name: "theme" }
  | { name: "settings" };

// Vite sets BASE_URL to the mount point, "/admin/".
const base = import.meta.env.BASE_URL.replace(/\/$/, "");

export function href(route: Route): string {
  switch (route.name) {
    case "dashboard":
      return `${base}/`;
    case "list":
      return `${base}/content/${encodeURIComponent(route.kind)}`;
    case "edit":
      return `${base}/content/${encodeURIComponent(route.kind)}/${
        route.id ? encodeURIComponent(route.id) : "new"
      }`;
    default:
      return `${base}/${route.name}`;
  }
}

export function parse(pathname: string): Route {
  const rest = pathname.startsWith(base) ? pathname.slice(base.length) : pathname;
  const [head, kind, id] = rest.split("/").filter(Boolean).map(decodeURIComponent);

  if (head === "content" && kind) {
    if (!id) return { name: "list", kind };
    return { name: "edit", kind, id: id === "new" ? null : id };
  }
  if (head === "taxonomies" || head === "theme" || head === "settings") {
    return { name: head };
  }
  return { name: "dashboard" };
}

/** A guard holds a navigation back while there is work that would be lost. */
export interface Guard {
  blocked: () => boolean;
  /** ask is handed the navigation, to run once the person agrees to it. */
  ask: (proceed: () => void) => void;
}

let guard: Guard | null = null;
const listeners = new Set<() => void>();

// The address the app is showing, which a refused popstate is put back to.
let current = window.location.pathname;

function emit() {
  current = window.location.pathname;
  for (const listener of listeners) listener();
}

export function setGuard(next: Guard | null) {
  guard = next;
}

export function navigate(route: Route, options: { replace?: boolean } = {}) {
  const to = href(route);
  if (to === window.location.pathname) return;

  const go = () => {
    if (options.replace) window.history.replaceState(null, "", to);
    else window.history.pushState(null, "", to);
    emit();
  };
  if (guard?.blocked()) guard.ask(go);
  else go();
}

window.addEventListener("popstate", () => {
  const to = window.location.pathname;
  if (to === current || !guard?.blocked()) return emit();

  // The browser has already moved, so the address is put back while asking.
  window.history.pushState(null, "", current);
  guard.ask(() => {
    window.history.pushState(null, "", to);
    emit();
  });
});

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useRoute(): Route {
  const pathname = useSyncExternalStore(subscribe, () => window.location.pathname);
  return parse(pathname);
}

/** linkProps makes an anchor navigate in place while staying a real link. */
export function linkProps(route: Route) {
  return {
    href: href(route),
    onClick: (event: MouseEvent) => {
      if (event.defaultPrevented || event.button !== 0) return;
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
      event.preventDefault();
      navigate(route);
    },
  };
}
