/**
 * What a redraw of the preview changed on the page on show, found by
 * comparing the page before and after it and marked there for a moment: an
 * edited headline is outlined where it stands, and scrolled to when it is
 * out of sight. A change the page does not show, or one that repaints it all
 * like a new color, marks nothing.
 */

// Marks are put on the page itself; this attribute keeps them out of the
// next comparison.
const mark = "data-kite-mark";

// More changed places than this, as when the preview was opened anew, mean
// the page was not edited in one spot and nothing is marked.
const tooMany = 40;

// Siblings changed together, as reordered links, are marked as one.
const together = 6;

export function outlineChanges(before: Document, win: Window) {
  const after = win.document;
  if (!before.body || !after.body || before.URL !== after.URL) return;

  let found = differences(before.body, after.body, []);
  if (found.length === 0 || found.length > tooMany) return;
  found = found.filter(shown);
  if (found.length > together) {
    const common = ancestor(found);
    found = common && common !== after.body ? [common] : [];
  }
  if (found.length === 0) return;

  const boxes = found.map((element) => {
    const rect = boxOf(element);
    return { top: rect.top + win.scrollY, left: rect.left + win.scrollX, width: rect.width, height: rect.height };
  });
  for (const box of boxes) draw(after, box);

  const first = boxes.reduce((a, b) => (b.top < a.top ? b : a));
  const inSight = first.top >= win.scrollY && first.top + Math.min(first.height, 80) <= win.scrollY + win.innerHeight;
  if (!inSight) win.scrollTo({ top: Math.max(0, first.top - win.innerHeight / 3), behavior: "smooth" });
}

/**
 * differences walks two copies of a page side by side and collects the
 * elements of the second that differ from their twin in the first. Where an
 * element itself differs it is taken whole, and its children are not visited.
 */
function differences(a: Element, b: Element, out: Element[]): Element[] {
  const left = children(a);
  const right = children(b);
  if (a.tagName !== b.tagName || left.length !== right.length || !sameAttributes(a, b) || ownText(a) !== ownText(b)) {
    out.push(b);
    return out;
  }
  for (let i = 0; i < left.length && out.length <= tooMany; i++) differences(left[i], right[i], out);
  return out;
}

function children(element: Element): Element[] {
  return [...element.children].filter((child) => !child.hasAttribute(mark) && child.tagName !== "SCRIPT");
}

function sameAttributes(a: Element, b: Element): boolean {
  if (a.attributes.length !== b.attributes.length) return false;
  for (const attribute of a.attributes) {
    if (b.getAttribute(attribute.name) !== attribute.value) return false;
  }
  return true;
}

function ownText(element: Element): string {
  let text = "";
  for (const node of element.childNodes) {
    if (node.nodeType === Node.TEXT_NODE) text += node.textContent;
  }
  return text.replace(/\s+/g, " ").trim();
}

function shown(element: Element): boolean {
  const rect = element.getBoundingClientRect();
  return rect.width > 0 && rect.height > 0;
}

/**
 * boxOf is where an element's content is drawn, which for a headline is its
 * words rather than the whole width of the column, or where the element is
 * when it has no content of its own to measure, as an image.
 */
function boxOf(element: Element): DOMRect {
  const range = element.ownerDocument.createRange();
  range.selectNodeContents(element);
  const content = range.getBoundingClientRect();
  return content.width > 0 && content.height > 0 ? content : element.getBoundingClientRect();
}

function ancestor(elements: Element[]): Element | null {
  let common: Element | null = elements[0];
  while (common && !elements.every((element) => common!.contains(element))) common = common.parentElement;
  return common;
}

function draw(doc: Document, box: { top: number; left: number; width: number; height: number }) {
  const pad = 4;
  const outline = doc.createElement("div");
  outline.setAttribute(mark, "");
  Object.assign(outline.style, {
    position: "absolute",
    top: `${box.top - pad}px`,
    left: `${box.left - pad}px`,
    width: `${box.width + pad * 2}px`,
    height: `${box.height + pad * 2}px`,
    boxSizing: "border-box",
    border: "2px solid #4a77d6",
    borderRadius: "6px",
    background: "rgba(74, 119, 214, 0.08)",
    pointerEvents: "none",
    zIndex: "2147483647",
    transition: "opacity 0.6s ease",
  });
  // Put on the root rather than the body, so a body with a position of its
  // own does not move it.
  doc.documentElement.appendChild(outline);
  window.setTimeout(() => {
    outline.style.opacity = "0";
    window.setTimeout(() => outline.remove(), 700);
  }, 1400);
}
