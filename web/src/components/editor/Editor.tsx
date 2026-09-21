import { useEffect, useRef } from "react";
import { Annotation, EditorSelection, EditorState } from "@codemirror/state";
import { EditorView, keymap, placeholder as placeholderText } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { markdown } from "@codemirror/lang-markdown";
import { HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags } from "@lezer/highlight";

/** What the toolbar can ask of the editor. */
export interface EditorHandle {
  insert: (text: string) => void;
  /** wrap puts markers around the selection, or takes them off again. */
  wrap: (before: string, after: string, fallback: string) => void;
  /** prefix starts each selected line with a marker, or removes it. */
  prefix: (marker: string) => void;
  fence: () => void;
  link: (fallback: string) => void;
}

interface Props {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  onDropFiles?: (files: File[]) => void;
  onReady?: (handle: EditorHandle) => void;
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
  "&": { fontSize: "15px", backgroundColor: "transparent", color: "var(--foreground)" },
  "&.cm-focused": { outline: "none" },
  // The page scrolls, not the editor, so the title travels with the text.
  ".cm-scroller": { overflow: "visible", fontFamily: "inherit", lineHeight: "1.85" },
  ".cm-content": { padding: "0 0 40vh", minHeight: "40vh", caretColor: "var(--foreground)" },
  ".cm-line": { padding: "0" },
  ".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--foreground)" },
  ".cm-placeholder": { color: "var(--muted-foreground)" },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
    backgroundColor: "color-mix(in oklab, var(--brand) 22%, transparent)",
  },
});

/** A markdown editor over CodeMirror, set as prose rather than as code. */
export function Editor({ value, onChange, placeholder, onDropFiles, onReady }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView>(null);

  // Read through a ref so a new callback does not rebuild the editor.
  const emit = useRef(onChange);
  emit.current = onChange;

  useEffect(() => {
    if (!host.current) return;

    const handle = commands(() => view.current);
    const editor = new EditorView({
      parent: host.current,
      state: EditorState.create({
        doc: value,
        extensions: [
          history(),
          keymap.of([
            { key: "Mod-b", run: () => (handle.wrap("**", "**", ""), true) },
            { key: "Mod-i", run: () => (handle.wrap("*", "*", ""), true) },
            ...defaultKeymap,
            ...historyKeymap,
          ]),
          markdown(),
          syntaxHighlighting(highlight),
          placeholderText(placeholder ?? ""),
          EditorView.lineWrapping,
          EditorView.updateListener.of((update) => {
            if (!update.docChanged) return;
            if (update.transactions.some((tr) => tr.annotation(loaded))) return;
            emit.current(update.state.doc.toString());
          }),
          theme,
        ],
      }),
    });
    view.current = editor;
    onReady?.(handle);

    return () => {
      editor.destroy();
      view.current = null;
    };
    // Built once; later values arrive through the sync below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Replaced only when it differs, so typing never moves the cursor.
  useEffect(() => {
    const editor = view.current;
    if (!editor || editor.state.doc.toString() === value) return;
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

function commands(current: () => EditorView | null): EditorHandle {
  const run = (fn: (editor: EditorView) => void) => {
    const editor = current();
    if (!editor) return;
    fn(editor);
    editor.focus();
  };

  return {
    insert: (text) =>
      run((editor) => {
        const at = editor.state.selection.main.head;
        editor.dispatch({
          changes: { from: at, insert: text },
          selection: { anchor: at + text.length },
        });
      }),

    wrap: (before, after, fallback) =>
      run((editor) => {
        const { from, to } = editor.state.selection.main;
        const doc = editor.state.doc;
        const wrapped =
          from >= before.length &&
          doc.sliceString(from - before.length, from) === before &&
          doc.sliceString(to, Math.min(doc.length, to + after.length)) === after;

        if (wrapped) {
          editor.dispatch({
            changes: [
              { from: from - before.length, to: from },
              { from: to, to: to + after.length },
            ],
          });
          return;
        }
        const inner = doc.sliceString(from, to) || fallback;
        editor.dispatch({
          changes: { from, to, insert: before + inner + after },
          selection: EditorSelection.range(
            from + before.length,
            from + before.length + inner.length,
          ),
        });
      }),

    prefix: (marker) =>
      run((editor) => {
        const { from, to } = editor.state.selection.main;
        const doc = editor.state.doc;
        const lines = [];
        for (let n = doc.lineAt(from).number; n <= doc.lineAt(to).number; n++) {
          lines.push(doc.line(n));
        }
        const remove = lines.every((line) => line.text.startsWith(marker));
        // A heading replaces whatever level the line already had.
        const existing = marker.startsWith("#") ? /^#{1,6}\s+/ : null;

        editor.dispatch({
          changes: lines.map((line) => {
            if (remove) return { from: line.from, to: line.from + marker.length };
            const old = existing?.exec(line.text)?.[0].length ?? 0;
            return { from: line.from, to: line.from + old, insert: marker };
          }),
        });
      }),

    fence: () =>
      run((editor) => {
        const { from, to } = editor.state.selection.main;
        const inner = editor.state.doc.sliceString(from, to);
        const open = "```\n";
        editor.dispatch({
          changes: { from, to, insert: `${open}${inner}\n\`\`\`\n` },
          selection: { anchor: from + open.length + inner.length },
        });
      }),

    link: (fallback) =>
      run((editor) => {
        const { from, to } = editor.state.selection.main;
        const text = editor.state.doc.sliceString(from, to) || fallback;
        const url = "https://";
        const start = from + text.length + 3;
        editor.dispatch({
          changes: { from, to, insert: `[${text}](${url})` },
          selection: EditorSelection.range(start, start + url.length),
        });
      }),
  };
}
