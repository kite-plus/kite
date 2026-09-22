import { useEffect, useState, type MouseEvent } from "react";
import { useEditorState, type Editor } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";
import { Bold, Code, ExternalLink, Italic, Link, Pencil, Strikethrough, Trash2, Unlink } from "lucide-react";

import { useI18n } from "@/i18n";
import { LinkForm } from "@/components/editor/LinkForm";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Toggle } from "@/components/ui/toggle";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

// Keeps the selection the command is about to act on.
const keep = (event: MouseEvent) => event.preventDefault();

/**
 * The small toolbar that follows a selection.
 *
 * It has three faces: formatting for a run of text, the address of a link
 * the cursor is in, and the alt text of a picked image.
 */
export function BubbleToolbar({ editor }: { editor: Editor }) {
  const { t } = useI18n();
  const state = useEditorState({
    editor,
    selector: ({ editor }) => ({
      bold: editor.isActive("bold"),
      italic: editor.isActive("italic"),
      strike: editor.isActive("strike"),
      code: editor.isActive("code"),
      link: editor.isActive("link"),
      href: editor.getAttributes("link").href as string | undefined,
      image: editor.isActive("image"),
      alt: editor.getAttributes("image").alt as string | undefined,
      empty: editor.state.selection.empty,
      from: editor.state.selection.from,
      to: editor.state.selection.to,
    }),
  });

  const [editing, setEditing] = useState(false);
  // A new selection is a new question; the form for the old one closes.
  useEffect(() => setEditing(false), [state.from, state.to]);

  const setLink = (href: string) => {
    const chain = editor.chain().focus().extendMarkRange("link");
    if (href) chain.setLink({ href }).run();
    else chain.unsetLink().run();
    setEditing(false);
  };

  const mark = (label: string, pressed: boolean, icon: React.ReactNode, toggle: () => void) => (
    <Tooltip>
      <TooltipTrigger
        render={
          <Toggle
            size="sm"
            className="size-7 min-w-7 px-0"
            aria-label={label}
            pressed={pressed}
            onPressedChange={toggle}
            onMouseDown={keep}
          />
        }
      >
        {icon}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );

  return (
    <BubbleMenu
      editor={editor}
      updateDelay={120}
      options={{ placement: "top", offset: 8 }}
      shouldShow={({ editor, state }) => {
        if (editor.isActive("codeBlock")) return false;
        if (editor.isActive("image")) return true;
        if (state.selection.empty) return editor.isActive("link");
        return true;
      }}
      className="flex items-center gap-0.5 rounded-lg bg-popover p-1 text-popover-foreground shadow-md ring-1 ring-foreground/10"
    >
      {state.image ? (
        <div className="flex items-center gap-1.5">
          <Input
            key={`${state.from}:${state.alt ?? ""}`}
            defaultValue={state.alt ?? ""}
            placeholder={t("editor.altText")}
            aria-label={t("editor.altText")}
            className="h-7 w-52 text-xs md:text-xs"
            onChange={(event) => editor.commands.updateAttributes("image", { alt: event.target.value })}
            onKeyDown={(event) => {
              if (event.key === "Enter" || event.key === "Escape") {
                event.preventDefault();
                editor.commands.focus();
              }
            }}
          />
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t("editor.removeImage")}
                  onMouseDown={keep}
                  onClick={() => editor.chain().focus().deleteSelection().run()}
                />
              }
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>{t("editor.removeImage")}</TooltipContent>
          </Tooltip>
        </div>
      ) : editing ? (
        <LinkForm
          href={state.href}
          onApply={setLink}
          onRemove={state.link ? () => setLink("") : undefined}
          onCancel={() => {
            setEditing(false);
            editor.commands.focus();
          }}
        />
      ) : state.link && state.empty ? (
        <>
          <a
            href={state.href}
            target="_blank"
            rel="noreferrer"
            className="flex max-w-56 items-center gap-1 px-1.5 text-xs text-brand hover:underline"
          >
            <ExternalLink className="size-3 shrink-0" />
            <span className="truncate">{state.href}</span>
          </a>
          <Separator orientation="vertical" className="mx-0.5 data-vertical:h-4 data-vertical:self-center" />
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t("editor.linkEdit")}
                  onMouseDown={keep}
                  onClick={() => setEditing(true)}
                />
              }
            >
              <Pencil />
            </TooltipTrigger>
            <TooltipContent>{t("editor.linkEdit")}</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t("editor.linkRemove")}
                  onMouseDown={keep}
                  onClick={() => setLink("")}
                />
              }
            >
              <Unlink />
            </TooltipTrigger>
            <TooltipContent>{t("editor.linkRemove")}</TooltipContent>
          </Tooltip>
        </>
      ) : (
        <>
          {mark(t("editor.bold"), state.bold, <Bold />, () => editor.chain().focus().toggleBold().run())}
          {mark(t("editor.italic"), state.italic, <Italic />, () =>
            editor.chain().focus().toggleItalic().run(),
          )}
          {mark(t("editor.strike"), state.strike, <Strikethrough />, () =>
            editor.chain().focus().toggleStrike().run(),
          )}
          {mark(t("editor.inlineCode"), state.code, <Code />, () =>
            editor.chain().focus().toggleCode().run(),
          )}
          <Separator orientation="vertical" className="mx-0.5 data-vertical:h-4 data-vertical:self-center" />
          {mark(t("editor.link"), state.link, <Link />, () => setEditing(true))}
        </>
      )}
    </BubbleMenu>
  );
}
