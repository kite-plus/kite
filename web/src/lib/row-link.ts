import type { MouseEvent } from "react";

/**
 * followRowLink lets a click anywhere on a table row open the row's link, the
 * anchor in it marked data-row-link. The anchor stays the real link, so the
 * keyboard, middle-click and the browser's menu keep working on it.
 *
 * A click is left alone when it lands on a control or in a cell marked
 * data-row-skip, comes from a portal such as the row's menu (React bubbles
 * those through the row), or ends a text selection.
 */
export function followRowLink(event: MouseEvent<HTMLElement>) {
  const target = event.target as Element;
  if (!event.currentTarget.contains(target)) return;
  if (target.closest('a, button, input, label, [role="checkbox"], [data-row-skip]')) return;
  if (window.getSelection()?.toString()) return;

  const link = event.currentTarget.querySelector<HTMLAnchorElement>("a[data-row-link]");
  if (!link) return;
  if (event.metaKey || event.ctrlKey) {
    window.open(link.href, "_blank", "noopener");
    return;
  }
  link.click();
}
