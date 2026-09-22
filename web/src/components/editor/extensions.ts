import { mergeAttributes } from "@tiptap/core";
import { CodeBlockLowlight, type CodeBlockLowlightOptions } from "@tiptap/extension-code-block-lowlight";
import { FindAndReplace } from "@tiptap/extension-find-and-replace";
import { Image, type ImageOptions } from "@tiptap/extension-image";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import { Paragraph } from "@tiptap/extension-paragraph";
import { TableKit } from "@tiptap/extension-table";
import { Placeholder, Selection } from "@tiptap/extensions";
import { Markdown } from "@tiptap/markdown";
import { ReactNodeViewRenderer } from "@tiptap/react";
import { StarterKit } from "@tiptap/starter-kit";
import { common, createLowlight } from "lowlight";

import { CodeBlockView } from "@/components/editor/CodeBlockView";
import { Slash, type SlashItem } from "@/components/editor/SlashMenu";
import { HorizontalRule } from "@/components/tiptap-node/horizontal-rule-node/horizontal-rule-node-extension";
import { ImageUploadNode, type UploadFunction } from "@/components/tiptap-node/image-upload-node";
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
  /** upload stores a file beside the item and resolves to its link. */
  upload: UploadFunction;
  onUploadError: (error: Error) => void;
}

/** The server takes files up to 32 MiB (maxUpload in internal/api/media.go). */
const maxUpload = 32 << 20;

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

/**
 * extensions is everything the visual editor can hold, which is what markdown
 * can. The Simple Editor template's text alignment, highlight, underline and
 * super/subscript are left out: none of them has a markdown spelling, so each
 * would vanish on the next save.
 */
export function extensions(env: Env) {
  return [
    StarterKit.configure({
      underline: false,
      codeBlock: false,
      paragraph: false,
      horizontalRule: false,
      heading: { levels: [1, 2, 3, 4] },
      link: {
        openOnClick: false,
        enableClickSelection: true,
        autolink: true,
        linkOnPaste: true,
        markdownLinks: true,
        defaultProtocol: "https",
      },
      // Its color is the template's, set in paragraph-node.scss.
      dropcursor: { width: 2 },
    }),
    PlainParagraph,
    HorizontalRule,
    HighlightedCode.configure({ lowlight, defaultLanguage: null, labels: () => env.labels() }),
    RelativeImage.configure({ base: env.base }),
    // The drop zone the toolbar's image button puts in. It is never saved:
    // it has no markdown, and it becomes an image once the file is up.
    ImageUploadNode.configure({
      accept: "image/*",
      maxSize: maxUpload,
      limit: 5,
      upload: (file, onProgress, signal) => env.upload(file, onProgress, signal),
      onError: (error) => env.onUploadError(error),
    }),
    TaskList,
    TaskItem.configure({ nested: true }),
    // A dragged column width has no markdown to go to, so there is none.
    TableKit.configure({ table: { resizable: false } }),
    // Keeps the selection drawn while a toolbar field has the focus.
    Selection,
    FindAndReplace.configure({ searchDebounceMs: 300, injectCSS: false }),
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
