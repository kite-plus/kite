import { useEffect, useMemo, useRef } from "react";
import { EditorContent, useEditor, type Editor } from "@tiptap/react";

import { extensions, type Env } from "@/components/editor/extensions";
import { ImageBubble } from "@/components/editor/ImageBubble";
import type { SlashItem } from "@/components/editor/SlashMenu";
import type { UploadFunction } from "@/components/tiptap-node/image-upload-node";

import "@/components/tiptap-node/blockquote-node/blockquote-node.scss";
import "@/components/tiptap-node/code-block-node/code-block-node.scss";
import "@/components/tiptap-node/horizontal-rule-node/horizontal-rule-node.scss";
import "@/components/tiptap-node/list-node/list-node.scss";
import "@/components/tiptap-node/image-node/image-node.scss";
import "@/components/tiptap-node/heading-node/heading-node.scss";
import "@/components/tiptap-node/paragraph-node/paragraph-node.scss";

interface Props {
  value: string;
  onChange: (markdown: string) => void;
  /** What an empty document, and an empty line in one, say. */
  placeholder: { empty: string; line: string };
  /** base is the item's own address, which its images are named relative to. */
  base?: string;
  slash: SlashItem[];
  labels: { slashEmpty: string; plain: string; language: string };
  /** upload stores a dropped, pasted or picked file and resolves to the link that reaches it. */
  upload: UploadFunction;
  /** onUploadError hears of a file refused before it was sent, too large or one too many. */
  onUploadError: (message: string) => void;
  onReady?: (editor: Editor | null) => void;
}

/** alt is a file's name without its extension, which is the best guess there is. */
const altOf = (file: File) => file.name.replace(/\.[^.]+$/, "");

/**
 * The body as formatted text, over Tiptap and its Simple Editor template.
 *
 * Markdown goes in and markdown comes out: the document model is
 * ProseMirror's, but what is stored is what a build reads. Nothing is
 * serialized until the author changes something, so opening a file and
 * closing it again leaves its bytes alone.
 */
export function RichEditor({
  value,
  onChange,
  placeholder,
  base,
  slash,
  labels,
  upload,
  onUploadError,
  onReady,
}: Props) {
  // Read through a ref so the extensions, built once, always see the latest.
  const latest = useRef({ placeholder, base, slash, labels, upload, onUploadError, onChange, onReady });
  latest.current = { placeholder, base, slash, labels, upload, onUploadError, onChange, onReady };

  const instance = useRef<Editor | null>(null);
  // The last markdown handed out, which is what a value prop is compared against.
  const emitted = useRef(value);

  const env = useMemo<Env>(
    () => ({
      base: () => latest.current.base,
      placeholder: (kind) => latest.current.placeholder[kind],
      slash: () => latest.current.slash,
      labels: () => latest.current.labels,
      upload: (file, onProgress, signal) => latest.current.upload(file, onProgress, signal),
      onUploadError: (error) => latest.current.onUploadError(error.message),
    }),
    [],
  );

  const insert = (files: File[], at?: number) => {
    for (const file of files) {
      latest.current
        .upload(file)
        .then((src) => {
          const editor = instance.current;
          if (!editor) return;
          const image = { type: "image", attrs: { src, alt: altOf(file) } };
          if (at === undefined) editor.chain().focus().insertContent(image).run();
          else editor.chain().focus().insertContentAt(at, image).run();
        })
        // The page has already said what went wrong.
        .catch(() => undefined);
    }
  };

  const editor = useEditor({
    extensions: useMemo(() => extensions(env), [env]),
    content: value,
    contentType: "markdown",
    immediatelyRender: true,
    shouldRerenderOnTransaction: false,
    editorProps: {
      attributes: { class: "simple-editor", spellcheck: "true" },
      handleDrop: (view, event, _slice, moved) => {
        const files = Array.from(event.dataTransfer?.files ?? []);
        if (moved || files.length === 0) return false;
        event.preventDefault();
        insert(files, view.posAtCoords({ left: event.clientX, top: event.clientY })?.pos);
        return true;
      },
      handlePaste: (_view, event) => {
        const files = Array.from(event.clipboardData?.files ?? []).filter((file) =>
          file.type.startsWith("image/"),
        );
        if (files.length === 0) return false;
        event.preventDefault();
        insert(files);
        return true;
      },
    },
    onUpdate: ({ editor, transaction }) => {
      // A plugin tidying the document on its own is not an edit: StarterKit
      // appends an empty paragraph to a body that does not end in one on the
      // first click, and that alone must not mark the item changed.
      if (!transaction.docChanged) return;
      emitted.current = editor.getMarkdown();
      latest.current.onChange(emitted.current);
    },
  });

  useEffect(() => {
    instance.current = editor;
    latest.current.onReady?.(editor);
    return () => {
      instance.current = null;
      latest.current.onReady?.(null);
    };
  }, [editor]);

  // Replaced only when it differs, so typing never moves the cursor. A
  // difference of surrounding blank lines is the store's own tidying of what
  // was just sent, not a new document.
  useEffect(() => {
    if (value === emitted.current) return;
    if (value.trim() === emitted.current.trim()) {
      emitted.current = value;
      return;
    }
    emitted.current = value;
    editor.commands.setContent(value, { contentType: "markdown", emitUpdate: false });
  }, [editor, value]);

  return (
    <>
      <EditorContent editor={editor} role="presentation" className="simple-editor-content" />
      <ImageBubble editor={editor} />
    </>
  );
}
