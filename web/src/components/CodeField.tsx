import { useEffect, useRef } from "react";
import { Compartment, EditorState, type Extension } from "@codemirror/state";
import { EditorView, keymap, lineNumbers, placeholder as placeholderText } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { bracketMatching, HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags } from "@lezer/highlight";

/**
 * Languages load when a field asks for one, so a form without code in it
 * does not carry five parsers.
 */
const languages: Record<string, () => Promise<Extension>> = {
  css: () => import("@codemirror/lang-css").then((m) => m.css()),
  html: () => import("@codemirror/lang-html").then((m) => m.html()),
  javascript: () => import("@codemirror/lang-javascript").then((m) => m.javascript()),
  js: () => import("@codemirror/lang-javascript").then((m) => m.javascript()),
  json: () => import("@codemirror/lang-json").then((m) => m.json()),
  yaml: () => import("@codemirror/lang-yaml").then((m) => m.yaml()),
  markdown: () => import("@codemirror/lang-markdown").then((m) => m.markdown()),
};

// Drawn from the app's tokens, so the field follows dark mode with the rest.
const highlight = HighlightStyle.define([
  { tag: [tags.keyword, tags.operatorKeyword, tags.modifier], color: "var(--primary)" },
  { tag: [tags.string, tags.special(tags.string), tags.regexp], color: "var(--success)" },
  { tag: [tags.number, tags.bool, tags.null, tags.atom], color: "var(--warning)" },
  { tag: [tags.comment, tags.meta], color: "var(--muted-foreground)", fontStyle: "italic" },
  { tag: [tags.tagName, tags.propertyName, tags.attributeName], color: "var(--info)" },
  { tag: tags.invalid, color: "var(--destructive)" },
]);

const theme = EditorView.theme({
  "&": { fontSize: "12px", backgroundColor: "transparent", color: "var(--foreground)" },
  "&.cm-focused": { outline: "none" },
  ".cm-scroller": { fontFamily: "var(--font-mono)", lineHeight: "1.6", maxHeight: "18rem" },
  ".cm-content": { minHeight: "7.5rem", padding: "6px 0", caretColor: "var(--foreground)" },
  ".cm-gutters": {
    backgroundColor: "transparent",
    color: "var(--muted-foreground)",
    border: "none",
    borderRight: "1px solid var(--border)",
  },
  ".cm-cursor": { borderLeftColor: "var(--foreground)" },
  ".cm-placeholder": { color: "var(--muted-foreground)" },
  "&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection": {
    backgroundColor: "color-mix(in oklab, var(--primary) 22%, transparent)",
  },
});

interface Props {
  id: string;
  value: string;
  onChange: (value: string) => void;
  language?: string;
  placeholder?: string;
}

/** CodeField edits a setting written in a language, such as a stylesheet. */
export function CodeField({ id, value, onChange, language, placeholder }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView>(null);
  const emit = useRef(onChange);
  emit.current = onChange;

  // Made once: rebuilding the editor on each keystroke would lose the cursor.
  useEffect(() => {
    if (!host.current) return;
    const lang = new Compartment();
    const editor = new EditorView({
      parent: host.current,
      state: EditorState.create({
        doc: value,
        extensions: [
          lineNumbers(),
          history(),
          bracketMatching(),
          keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
          syntaxHighlighting(highlight),
          theme,
          lang.of([]),
          placeholder ? placeholderText(placeholder) : [],
          EditorView.contentAttributes.of({ id, "aria-multiline": "true" }),
          EditorView.updateListener.of((update) => {
            if (update.docChanged) emit.current(update.state.doc.toString());
          }),
        ],
      }),
    });
    view.current = editor;

    const load = language ? languages[language.toLowerCase()] : undefined;
    let live = true;
    void load?.().then((extension) => {
      if (live) editor.dispatch({ effects: lang.reconfigure(extension) });
    });
    return () => {
      live = false;
      editor.destroy();
      view.current = null;
    };
    // The value is followed below, not by building the editor again.
  }, [id, language, placeholder]);

  // A value changed from outside, as when the field goes back to its default.
  useEffect(() => {
    const editor = view.current;
    if (!editor || editor.state.doc.toString() === value) return;
    editor.dispatch({ changes: { from: 0, to: editor.state.doc.length, insert: value } });
  }, [value]);

  return (
    <div
      ref={host}
      className="overflow-hidden rounded-md border border-input shadow-xs focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50"
    />
  );
}
