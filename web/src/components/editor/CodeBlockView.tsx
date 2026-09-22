import { NodeViewContent, NodeViewWrapper, type NodeViewProps } from "@tiptap/react";

/** A code block with its language in the corner, where a fence's info string goes. */
export function CodeBlockView({ node, updateAttributes, extension }: NodeViewProps) {
  const known: string[] = extension.options.lowlight.listLanguages();
  const language = (node.attrs.language as string | null) ?? "";
  // A language the author typed is kept even when nothing here can highlight it.
  const languages = language && !known.includes(language) ? [language, ...known] : known;

  return (
    <NodeViewWrapper className="code-block">
      <select
        contentEditable={false}
        value={language}
        aria-label={extension.options.labels().language}
        onChange={(event) => updateAttributes({ language: event.target.value || null })}
      >
        <option value="">{extension.options.labels().plain}</option>
        {languages.map((name) => (
          <option key={name} value={name}>
            {name}
          </option>
        ))}
      </select>
      <pre>
        <NodeViewContent<"code"> as="code" />
      </pre>
    </NodeViewWrapper>
  );
}
