import { useRef } from "react";
import {
  Bold,
  Code,
  Heading2,
  Image,
  Italic,
  Link,
  List,
  TextQuote,
  type LucideIcon,
} from "lucide-react";

import { useI18n, type Key } from "@/i18n";
import type { EditorHandle } from "@/components/editor/Editor";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Kbd } from "@/components/ui/kbd";
import { Separator } from "@/components/ui/separator";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface Props {
  editor: () => EditorHandle | null;
  format: string;
  /** onFiles is absent until the item is saved and has somewhere to keep them. */
  onFiles?: (files: File[]) => void;
}

interface Tool {
  label: Key;
  icon: LucideIcon;
  keys?: string;
  run: (editor: EditorHandle) => void;
}

/** Formatting that writes markdown, so the source stays what is stored. */
export function EditorToolbar({ editor, format, onFiles }: Props) {
  const { t } = useI18n();
  const picker = useRef<HTMLInputElement>(null);

  const text: Tool[] = [
    { label: "editor.bold", icon: Bold, keys: "⌘B", run: (e) => e.wrap("**", "**", t("editor.bold")) },
    { label: "editor.italic", icon: Italic, keys: "⌘I", run: (e) => e.wrap("*", "*", t("editor.italic")) },
    { label: "editor.heading", icon: Heading2, run: (e) => e.prefix("## ") },
  ];
  const blocks: Tool[] = [
    { label: "editor.link", icon: Link, run: (e) => e.link(t("editor.linkText")) },
    { label: "editor.image", icon: Image, run: () => picker.current?.click() },
    { label: "editor.codeBlock", icon: Code, run: (e) => e.fence() },
    { label: "editor.list", icon: List, run: (e) => e.prefix("- ") },
    { label: "editor.quote", icon: TextQuote, run: (e) => e.prefix("> ") },
  ];

  const button = (tool: Tool) => (
    <Tooltip key={tool.label}>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t(tool.label)}
            disabled={tool.label === "editor.image" && !onFiles}
            // Keeps the selection the command is about to act on.
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => {
              const handle = editor();
              if (handle) tool.run(handle);
            }}
          />
        }
      >
        <tool.icon />
      </TooltipTrigger>
      <TooltipContent>
        {tool.label === "editor.image" && !onFiles ? t("editor.saveFirst") : t(tool.label)}
        {tool.keys && <Kbd>{tool.keys}</Kbd>}
      </TooltipContent>
    </Tooltip>
  );

  return (
    <div className="flex items-center gap-0.5 border-b px-3.5 py-1.5">
      {text.map(button)}
      <Separator orientation="vertical" className="mx-1.5 data-vertical:h-4 data-vertical:self-center" />
      {blocks.map(button)}
      <Badge variant="secondary" className="ml-auto capitalize">
        {format}
      </Badge>

      <input
        ref={picker}
        type="file"
        accept="image/*"
        multiple
        className="sr-only"
        tabIndex={-1}
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          if (files.length > 0) onFiles?.(files);
          event.target.value = "";
        }}
      />
    </div>
  );
}
