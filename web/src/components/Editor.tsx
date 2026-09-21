import { useEffect, useRef } from "react";
import { Annotation, EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, highlightActiveLine } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { markdown } from "@codemirror/lang-markdown";
import {
  syntaxHighlighting,
  defaultHighlightStyle,
  HighlightStyle,
} from "@codemirror/language";
import { tags } from "@lezer/highlight";

interface Props {
  value: string;
  onChange: (value: string) => void;
  /** onDropFiles receives anything dragged onto the editor. */
  onDropFiles?: (files: File[]) => void;
  /** insert is called with a function that puts text at the cursor. */
  onReady?: (insert: (text: string) => void) => void;
}

/**
 * loaded marks a change this component made to put a different document in
 * the editor, so it is not reported back as something the author typed. A
 * document replaced after a conflict is resolved is not an unsaved edit.
 */
const loaded = Annotation.define<boolean>();

/**
 * Markdown highlighting drawn from the app's tokens.
 *
 * CodeMirror's default style is a fixed light palette. Reusing the theme
 * variables is what keeps the editor from being the one pane that ignores
 * dark mode.
 */
const kiteHighlight = HighlightStyle.define([
  { tag: tags.heading, color: "var(--foreground)", fontWeight: "600" },
  { tag: tags.strong, color: "var(--foreground)", fontWeight: "600" },
  { tag: tags.emphasis, fontStyle: "italic" },
  { tag: tags.link, color: "var(--primary)" },
  { tag: tags.url, color: "var(--primary)", textDecoration: "underline" },
  { tag: tags.monospace, color: "var(--primary)" },
  { tag: tags.quote, color: "var(--muted-foreground)", fontStyle: "italic" },
  { tag: tags.list, color: "var(--muted-foreground)" },
  { tag: tags.contentSeparator, color: "var(--muted-foreground)" },
  { tag: tags.processingInstruction, color: "var(--muted-foreground)" },
]);

/**
 * A markdown editor over CodeMirror.
 *
 * The extensions are listed rather than pulled from basicSetup, because
 * basicSetup brings autocompletion, search and linting that a prose editor
 * does not want and that cost bundle size in a page loaded from the binary.
 */
export function Editor({ value, onChange, onDropFiles, onReady }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView>(null);

  // The callback is read through a ref so that changing it does not tear down
  // and rebuild the editor, which would lose the cursor on every keystroke.
  const emit = useRef(onChange);
  emit.current = onChange;

  useEffect(() => {
    if (!host.current) return;

    const editor = new EditorView({
      parent: host.current,
      state: EditorState.create({
        doc: value,
        extensions: [
          lineNumbers(),
          highlightActiveLine(),
          history(),
          keymap.of([...defaultKeymap, ...historyKeymap]),
          markdown(),
          // The editor reads the app's own tokens rather than carrying a
          // palette of its own, so it follows the theme instead of staying
          // light while everything around it goes dark.
          syntaxHighlighting(kiteHighlight, { fallback: true }),
          syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
          EditorView.lineWrapping,
          EditorView.updateListener.of((update) => {
            if (!update.docChanged) return;
            if (update.transactions.some((t) => t.annotation(loaded))) return;
            emit.current(update.state.doc.toString());
          }),
          EditorView.theme({
            "&": {
              height: "100%",
              fontSize: "13px",
              backgroundColor: "transparent",
              color: "var(--foreground)",
            },
            "&.cm-focused": { outline: "none" },
            ".cm-content": {
              fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
              padding: "16px 0",
              caretColor: "var(--foreground)",
            },
            ".cm-cursor, .cm-dropCursor": { borderLeftColor: "var(--foreground)" },
            ".cm-gutters": {
              border: "none",
              backgroundColor: "transparent",
              color: "var(--muted-foreground)",
              opacity: "0.6",
            },
            ".cm-activeLine": { backgroundColor: "var(--muted)" },
            ".cm-activeLineGutter": { backgroundColor: "transparent" },
            "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
              backgroundColor: "color-mix(in oklab, var(--primary) 22%, transparent)",
            },
          }),
        ],
      }),
    });
    view.current = editor;

    onReady?.((text: string) => {
      const at = editor.state.selection.main.head;
      editor.dispatch({
        changes: { from: at, insert: text },
        selection: { anchor: at + text.length },
      });
      editor.focus();
    });

    return () => {
      editor.destroy();
      view.current = null;
    };
    // Built once. Later changes to value arrive through the sync below, so
    // that typing does not recreate the editor.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Replace the document only when it differs, which is what makes loading
  // another item work without stealing the cursor while typing.
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
      className="h-full overflow-auto"
      onDragOver={(e) => {
        if (onDropFiles) e.preventDefault();
      }}
      onDrop={(e) => {
        const files = Array.from(e.dataTransfer.files);
        if (!onDropFiles || files.length === 0) return;
        e.preventDefault();
        onDropFiles(files);
      }}
    />
  );
}
