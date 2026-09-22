import { useEffect, useRef } from "react";
import { Annotation, EditorState } from "@codemirror/state";
import { EditorView, keymap, placeholder as placeholderText } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { markdown } from "@codemirror/lang-markdown";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags } from "@lezer/highlight";

/** What the page can ask of the source view. */
export interface SourceHandle {
  insert: (text: string) => void;
  focus: () => void;
}

interface Props {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  onDropFiles?: (files: File[]) => void;
  onReady?: (handle: SourceHandle | null) => void;
}

// Marks a document swapped in by this component, which is not an author's edit.
const loaded = Annotation.define<boolean>();

// Drawn from the app's tokens, so the editor follows dark mode with the rest.
const highlight = HighlightStyle.define([
  { tag: tags.heading1, fontSize: "1.6em", fontWeight: "700", lineHeight: "1.4" },
  { tag: tags.heading2, fontSize: "1.3em", fontWeight: "600", lineHeight: "1.5" },
  { tag: tags.heading3, fontSize: "1.12em", fontWeight: "600" },
  { tag: tags.heading, fontWeight: "600" },
  { tag: tags.strong, fontWeight: "600" },
  { tag: tags.emphasis, fontStyle: "italic" },
  { tag: tags.strikethrough, textDecoration: "line-through" },
  { tag: tags.link, color: "var(--brand)" },
  { tag: tags.url, color: "var(--brand)" },
  { tag: tags.monospace, fontFamily: "var(--font-mono)", fontSize: "0.9em" },
  { tag: tags.quote, color: "var(--muted-foreground)" },
  { tag: tags.contentSeparator, color: "var(--muted-foreground)" },
  { tag: tags.processingInstruction, color: "var(--muted-foreground)", opacity: "0.6" },
]);

const theme = EditorView.theme({
  "&": { fontSize: "1rem", backgroundColor: "transparent", color: "var(--foreground)" },
  "&.cm-focused": { outline: "none" },
  // The page scrolls, not the editor, so the title travels with the text.
  ".cm-scroller": { overflow: "visible", fontFamily: "inherit", lineHeight: "1.8" },
  ".cm-content": { padding: "0", minHeight: "40vh", caretColor: "var(--foreground)" },
  ".cm-line": { padding: "0" },
  ".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--foreground)" },
  ".cm-placeholder": { color: "var(--muted-foreground)" },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
    backgroundColor: "color-mix(in oklab, var(--brand) 22%, transparent)",
  },
});

/**
 * The markdown as it is stored, in a CodeMirror view.
 *
 * It formats nothing on the author's behalf: this is where footnotes, raw
 * html and anything else the visual editor cannot hold are edited by hand.
 */
export function SourceEditor({ value, onChange, placeholder, onDropFiles, onReady }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView>(null);

  // Read through refs so a new callback does not rebuild the editor.
  const emit = useRef(onChange);
  emit.current = onChange;
  const ready = useRef(onReady);
  ready.current = onReady;
  // The last text handed out, which is what a value prop is compared against.
  const emitted = useRef(value);

  useEffect(() => {
    if (!host.current) return;

    const editor = new EditorView({
      parent: host.current,
      state: EditorState.create({
        doc: value,
        extensions: [
          history(),
          keymap.of([...defaultKeymap, ...historyKeymap]),
          markdown(),
          syntaxHighlighting(highlight),
          placeholderText(placeholder ?? ""),
          EditorView.lineWrapping,
          EditorView.updateListener.of((update) => {
            if (!update.docChanged) return;
            if (update.transactions.some((tr) => tr.annotation(loaded))) return;
            emitted.current = update.state.doc.toString();
            emit.current(emitted.current);
          }),
          theme,
        ],
      }),
    });
    view.current = editor;
    ready.current?.({
      insert: (text) => {
        const at = editor.state.selection.main.head;
        editor.dispatch({ changes: { from: at, insert: text }, selection: { anchor: at + text.length } });
        editor.focus();
      },
      focus: () => editor.focus(),
    });

    return () => {
      ready.current?.(null);
      editor.destroy();
      view.current = null;
    };
    // Built once; later values arrive through the sync below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Replaced only when it differs, so typing never moves the cursor. A
  // difference of surrounding blank lines is the store's own tidying of what
  // was just sent, not a new document.
  useEffect(() => {
    const editor = view.current;
    if (!editor || value === emitted.current) return;
    if (value.trim() === emitted.current.trim()) {
      emitted.current = value;
      return;
    }
    emitted.current = value;
    editor.dispatch({
      changes: { from: 0, to: editor.state.doc.length, insert: value },
      annotations: loaded.of(true),
    });
  }, [value]);

  return (
    <div
      ref={host}
      onDragOver={(event) => {
        if (onDropFiles) event.preventDefault();
      }}
      onDrop={(event) => {
        const files = Array.from(event.dataTransfer.files);
        if (!onDropFiles || files.length === 0) return;
        event.preventDefault();
        onDropFiles(files);
      }}
    />
  );
}
