import { mergeAttributes } from "@tiptap/core";
import { CodeBlockLowlight, type CodeBlockLowlightOptions } from "@tiptap/extension-code-block-lowlight";
import { Image, type ImageOptions } from "@tiptap/extension-image";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import { Paragraph } from "@tiptap/extension-paragraph";
import { TableKit } from "@tiptap/extension-table";
import { Placeholder } from "@tiptap/extensions";
import { Markdown } from "@tiptap/markdown";
import { ReactNodeViewRenderer } from "@tiptap/react";
import { StarterKit } from "@tiptap/starter-kit";
import { common, createLowlight } from "lowlight";

import { CodeBlockView } from "@/components/editor/CodeBlockView";
import { Slash, type SlashItem } from "@/components/editor/SlashMenu";
import { resolveLink } from "@/lib/links";

/**
 * What the extensions ask of the page, read late so the language, the item
 * and the commands can change under a running editor.
 */
export interface Env {
  /** base is the address links in the body are relative to. */
  base: () => string | undefined;
  placeholder: (kind: "empty" | "line") => string;
  slash: () => SlashItem[];
  labels: () => { slashEmpty: string; plain: string; language: string };
}

const lowlight = createLowlight(common);

/**
 * An image whose src is kept as written -- a file name beside the page -- and
 * resolved only for the browser, so the markdown stays portable.
 */
const RelativeImage = Image.extend<ImageOptions & { base: () => string | undefined }>({
  addOptions() {
    return { ...(this.parent?.() as ImageOptions), base: () => undefined };
  },
  renderHTML({ HTMLAttributes }) {
    return [
      "img",
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        src: resolveLink(String(HTMLAttributes.src ?? ""), this.options.base()),
      }),
    ];
  },
});

/**
 * A paragraph that writes nothing when it holds nothing. The stock one keeps
 * a run of blank paragraphs alive with "&nbsp;", which is a fair trade in an
 * app's database and a stain in a markdown file somebody else will read.
 */
const PlainParagraph = Paragraph.extend({
  renderMarkdown: (node, helpers) =>
    Array.isArray(node.content) && node.content.length > 0
      ? helpers.renderChildren(node.content)
      : "",
});

const HighlightedCode = CodeBlockLowlight.extend<
  CodeBlockLowlightOptions & { labels: () => { plain: string; language: string } }
>({
  addOptions() {
    return {
      ...(this.parent?.() as CodeBlockLowlightOptions),
      labels: () => ({ plain: "", language: "" }),
    };
  },
  addNodeView() {
    return ReactNodeViewRenderer(CodeBlockView);
  },
});

/** extensions is everything the visual editor can hold, which is what markdown can. */
export function extensions(env: Env) {
  return [
    StarterKit.configure({
      // Underline has no markdown spelling, so it is not offered; the other
      // two are replaced below.
      underline: false,
      codeBlock: false,
      paragraph: false,
      heading: { levels: [1, 2, 3, 4] },
      link: {
        openOnClick: false,
        autolink: true,
        linkOnPaste: true,
        markdownLinks: true,
        defaultProtocol: "https",
      },
      dropcursor: { color: "var(--brand)", width: 2 },
    }),
    PlainParagraph,
    HighlightedCode.configure({ lowlight, defaultLanguage: null, labels: () => env.labels() }),
    RelativeImage.configure({ base: env.base }),
    TaskList,
    TaskItem.configure({ nested: true }),
    // A dragged column width has no markdown to go to, so there is none.
    TableKit.configure({ table: { resizable: false } }),
    Placeholder.configure({
      placeholder: ({ editor, node }) => {
        if (node.type.name !== "paragraph") return "";
        return env.placeholder(editor.state.doc.childCount === 1 ? "empty" : "line");
      },
    }),
    Markdown.configure({ indentation: { style: "space", size: 2 } }),
    Slash.configure({ items: env.slash, empty: () => env.labels().slashEmpty }),
  ];
}
