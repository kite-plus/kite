import { useEffect, useRef } from "react";
import { Annotation, EditorState } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, highlightActiveLine } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { markdown } from "@codemirror/lang-markdown";
import { syntaxHighlighting, defaultHighlightStyle } from "@codemirror/language";

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
          syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
          EditorView.lineWrapping,
          EditorView.updateListener.of((update) => {
            if (!update.docChanged) return;
            if (update.transactions.some((t) => t.annotation(loaded))) return;
            emit.current(update.state.doc.toString());
          }),
          EditorView.theme({
            "&": { height: "100%", fontSize: "14px" },
            "&.cm-focused": { outline: "none" },
            ".cm-content": {
              fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
              padding: "12px 0",
            },
            ".cm-gutters": { border: "none", background: "transparent", opacity: "0.5" },
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
