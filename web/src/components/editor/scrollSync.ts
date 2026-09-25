import { useCallback, useEffect, useRef, useState, type RefObject } from "react";
import { syntaxTree } from "@codemirror/language";
import type { EditorView } from "@codemirror/view";

/**
 * Keeping the editor and the preview on the same part of an item.
 *
 * The preview is the theme's whole page, with a header, a title and whatever
 * else the theme puts around the body, so the two cannot be scrolled by the
 * same fraction. Instead the body's blocks are paired up, a heading with its
 * heading and a list with its list, and a scroll offset between two pairs is
 * carried across in proportion. The page is left exactly as the build writes
 * it: the pairs are found by what the blocks are and what they say.
 */

export type Kind = "text" | "heading" | "list" | "code" | "table" | "rule" | "quote" | "image" | "other";

/** Block is one block of the body, with the viewport y of its top edge. */
export interface Block {
  kind: Kind;
  /** The first letters and digits it reads, which both sides share. */
  text: string;
  top: number;
}

const letters = (text: string) => text.replace(/[^\p{L}\p{N}]+/gu, "").slice(0, 12);

const same = (a: string, b: string) => a !== "" && b !== "" && a.slice(0, 4) === b.slice(0, 4);

// Text the editor draws for its own controls, not the author's.
const controls = new Set(["LABEL", "SELECT", "OPTION", "BUTTON"]);

function readable(element: Element): string {
  let text = "";
  const walk = (node: Node) => {
    for (const child of node.childNodes) {
      if (child.nodeType === Node.TEXT_NODE) text += child.textContent;
      else if (child instanceof Element && !controls.has(child.tagName)) walk(child);
      if (text.length > 200) return;
    }
  };
  walk(element);
  return text;
}

function kindOf(element: Element, text: string): Kind {
  switch (element.tagName) {
    case "P":
      return text === "" && element.querySelector("img") ? "image" : "text";
    case "H1":
    case "H2":
    case "H3":
    case "H4":
    case "H5":
    case "H6":
      return "heading";
    case "UL":
    case "OL":
    case "DL":
      return "list";
    case "PRE":
      return "code";
    case "TABLE":
      return "table";
    case "HR":
      return "rule";
    case "BLOCKQUOTE":
      return "quote";
    case "IMG":
    case "FIGURE":
      return "image";
  }
  // The editor draws code, tables and rules inside wrappers of its own.
  if (element.querySelector("table")) return "table";
  if (element.querySelector("pre")) return "code";
  if (element.childElementCount === 1 && element.firstElementChild?.tagName === "HR") return "rule";
  return "other";
}

/** renderedBlocks reads the blocks of a rendered body, the editor's or the page's. */
export function renderedBlocks(body: Element): Block[] {
  return Array.from(body.children, (element) => {
    const text = readable(element);
    return { kind: kindOf(element, letters(text)), text: letters(text), top: element.getBoundingClientRect().top };
  });
}

const sourceKinds: Record<string, Kind> = {
  Paragraph: "text",
  ATXHeading1: "heading",
  ATXHeading2: "heading",
  ATXHeading3: "heading",
  ATXHeading4: "heading",
  ATXHeading5: "heading",
  ATXHeading6: "heading",
  SetextHeading1: "heading",
  SetextHeading2: "heading",
  BulletList: "list",
  OrderedList: "list",
  FencedCode: "code",
  CodeBlock: "code",
  Blockquote: "quote",
  HorizontalRule: "rule",
  Table: "table",
};

/**
 * sourceBlocks reads the blocks of the markdown itself, from the syntax tree
 * the source view already keeps. Only markup is stripped from their text: the
 * letters that are left are the ones the page shows.
 */
export function sourceBlocks(view: EditorView): Block[] {
  const blocks: Block[] = [];
  for (let node = syntaxTree(view.state).topNode.firstChild; node; node = node.nextSibling) {
    const raw = view.state.doc.sliceString(node.from, Math.min(node.to, node.from + 300));
    let kind = sourceKinds[node.name] ?? "other";
    // The source view reads plain CommonMark, which has no tables.
    if (kind === "text" && raw.startsWith("|")) kind = "table";
    const text = letters(
      raw
        .replace(/^\s*(`{3,}|~{3,}).*\n?/, "")
        .replace(/!\[[^\]]*\]\([^)]*\)/g, "")
        .replace(/\]\([^)]*\)/g, "]")
        .replace(/^\s*(?:[-*+]|\d+[.)])\s+(?:\[[ xX]\]\s+)?/gm, "")
        .replace(/^\s*>\s?/gm, "")
        .replace(/^#{1,6}\s+/gm, ""),
    );
    if (kind === "text" && text === "" && /^!\[/.test(raw)) kind = "image";
    blocks.push({ kind, text, top: view.documentTop + view.lineBlockAt(node.from).top });
  }
  return blocks;
}

const selectors: Partial<Record<Kind, string>> = {
  text: "p",
  heading: "h1, h2, h3, h4, h5, h6",
  list: "ul, ol, dl",
  quote: "blockquote",
  table: "table",
};

/**
 * pageBody finds the element of a theme's page that holds the rendered body:
 * the parent that the first few blocks of the editor are found in, which is
 * not fooled by the same words repeated elsewhere, as in a summary.
 */
function pageBody(doc: Document, blocks: Block[]): Element | null {
  const found = new Map<Element, number>();
  for (const block of blocks.filter((each) => each.text !== "" && selectors[each.kind]).slice(0, 4)) {
    for (const element of doc.body.querySelectorAll(selectors[block.kind]!)) {
      if (!element.parentElement || !same(letters(readable(element)), block.text)) continue;
      found.set(element.parentElement, (found.get(element.parentElement) ?? 0) + 1);
    }
  }
  let body: Element | null = null;
  let most = 0;
  for (const [element, count] of found) {
    if (count > most) [body, most] = [element, count];
  }
  return body;
}

/**
 * pairBlocks pairs the editor's blocks with the page's, in order. The next
 * block of the same kind is taken as the pair; a page block is skipped only
 * when a later one says the same thing, since a theme may put something of
 * its own among the body's blocks.
 */
export function pairBlocks(editor: Block[], page: Block[]): [number, number][] {
  const pairs: [number, number][] = [];
  let next = 0;
  for (const block of editor) {
    for (let at = next; at < Math.min(page.length, next + 4); at++) {
      const candidate = page[at];
      if (candidate.kind !== block.kind) continue;
      if (at > next && !same(block.text, candidate.text)) continue;
      pairs.push([block.top, candidate.top]);
      next = at + 1;
      break;
    }
  }
  return pairs;
}

/** carry maps an offset on one side of the anchors to the other side. */
function carry(offset: number, anchors: [number, number][], from: 0 | 1): number {
  const to = from === 0 ? 1 : 0;
  let at = 0;
  while (at < anchors.length - 2 && anchors[at + 1][from] <= offset) at++;
  const [a, b] = [anchors[at], anchors[at + 1]];
  const span = b[from] - a[from];
  const share = span > 0 ? Math.min(1, Math.max(0, (offset - a[from]) / span)) : 0;
  return a[to] + share * (b[to] - a[to]);
}

type Side = "editor" | "page";

/**
 * useScrollSync keeps the preview on the part of the item the editor shows,
 * and the editor on the part of the preview being read. Whichever side the
 * pointer or the keyboard is on leads; the other follows, and its own scroll
 * events are not taken as a lead in return.
 *
 * It returns what the preview calls with its frame each time it has written
 * a page into it.
 */
export function useScrollSync(
  on: boolean,
  scroller: RefObject<HTMLElement | null>,
  blocks: () => Block[],
): (frame: HTMLIFrameElement) => void {
  const read = useRef(blocks);
  read.current = blocks;

  // Made once: a listener has to be the same function to be taken off again,
  // or to be added again without being doubled.
  const [sync] = useState(() => {
    let frame: HTMLIFrameElement | null = null;
    let lead: Side = "editor";
    let pending = 0;

    // Scroll offsets, editor first, that show the same part of the item.
    const anchors = (): [number, number][] | null => {
      const editor = scroller.current;
      const win = frame?.contentWindow;
      const doc = frame?.contentDocument;
      if (!editor || !win || !doc?.body) return null;

      const base = editor.getBoundingClientRect().top - editor.scrollTop;
      // An empty paragraph writes nothing, so the page has none to pair with.
      const mine = read
        .current()
        .filter((block) => block.kind !== "other" && !(block.kind === "text" && block.text === ""))
        .map((block) => ({ ...block, top: block.top - base }));
      const body = pageBody(doc, mine);
      const theirs = body
        ? renderedBlocks(body).map((block) => ({ ...block, top: block.top + win.scrollY }))
        : [];

      const editorEnd = editor.scrollHeight - editor.clientHeight;
      const pageEnd = (doc.scrollingElement?.scrollHeight ?? 0) - win.innerHeight;
      if (editorEnd <= 0 || pageEnd <= 0) return null;

      const anchors: [number, number][] = [[0, 0]];
      for (const [mineTop, theirTop] of pairBlocks(mine, theirs)) {
        const [lastMine, lastTheirs] = anchors[anchors.length - 1];
        if (mineTop > lastMine && theirTop >= lastTheirs && mineTop < editorEnd && theirTop < pageEnd) {
          anchors.push([mineTop, theirTop]);
        }
      }
      // Both reach their ends together.
      anchors.push([editorEnd, pageEnd]);
      return anchors;
    };

    const move = (from: Side) => {
      const editor = scroller.current;
      const win = frame?.contentWindow;
      const found = anchors();
      if (!editor || !win || !found) return;
      if (from === "editor") {
        const top = carry(editor.scrollTop, found, 0);
        if (Math.abs(win.scrollY - top) > 1) win.scrollTo({ top, behavior: "instant" });
      } else {
        const top = carry(win.scrollY, found, 1);
        if (Math.abs(editor.scrollTop - top) > 1) editor.scrollTo({ top, behavior: "instant" });
      }
    };

    // One pass a frame, however many scroll events come in it.
    const follow = (from: Side) => {
      cancelAnimationFrame(pending);
      pending = requestAnimationFrame(() => move(from));
    };

    return {
      follow,
      cancel: () => cancelAnimationFrame(pending),
      show: (next: HTMLIFrameElement) => {
        frame = next;
      },
      editorScrolled: () => {
        if (lead === "editor") follow("editor");
      },
      pageScrolled: () => {
        if (lead === "page") follow("page");
      },
      // A new width reflows the page, which moves everything in it.
      pageResized: () => follow("editor"),
      editorLeads: () => {
        lead = "editor";
      },
      pageLeads: () => {
        lead = "page";
      },
    };
  });

  useEffect(() => {
    const editor = scroller.current;
    if (!on || !editor) return;
    editor.addEventListener("scroll", sync.editorScrolled, { passive: true });
    editor.addEventListener("pointerenter", sync.editorLeads);
    editor.addEventListener("keydown", sync.editorLeads);
    return () => {
      editor.removeEventListener("scroll", sync.editorScrolled);
      editor.removeEventListener("pointerenter", sync.editorLeads);
      editor.removeEventListener("keydown", sync.editorLeads);
      sync.cancel();
    };
  }, [on, scroller, sync]);

  // Writing a page into the frame drops the listeners its window had, so they
  // are added again each time; adding one that is still there does nothing.
  return useCallback(
    (frame: HTMLIFrameElement) => {
      sync.show(frame);
      frame.addEventListener("pointerenter", sync.pageLeads);
      frame.contentWindow?.addEventListener("scroll", sync.pageScrolled, { passive: true });
      frame.contentWindow?.addEventListener("resize", sync.pageResized);
      sync.follow("editor");
    },
    [sync],
  );
}
