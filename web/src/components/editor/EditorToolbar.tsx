import { useState, type MouseEvent, type ReactNode } from "react";
import { useEditorState, type Editor } from "@tiptap/react";
import {
  Bold,
  ChevronDown,
  Code,
  Columns3,
  FileCode2,
  Image,
  Italic,
  Link,
  List,
  ListOrdered,
  ListTodo,
  Minus,
  PenLine,
  Redo2,
  Rows3,
  SquareCode,
  Strikethrough,
  Table,
  TextQuote,
  Trash2,
  Undo2,
} from "lucide-react";

import { useI18n } from "@/i18n";
import { LinkForm } from "@/components/editor/LinkForm";
import type { Mode } from "@/components/editor/markdown";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Kbd } from "@/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { Toggle } from "@/components/ui/toggle";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

interface Props {
  /** editor is null in source mode, where there is nothing to format. */
  editor: Editor | null;
  mode: Mode;
  onMode: (mode: Mode) => void;
  onPickImage: () => void;
}

const isMac = /Mac|iPhone|iPad/.test(navigator.platform);
const mod = isMac ? "⌘" : "Ctrl+";

// Keeps the selection the command is about to act on.
const keep = (event: MouseEvent) => event.preventDefault();

/** Formatting that writes markdown, so the source stays what is stored. */
export function EditorToolbar({ editor, mode, onMode, onPickImage }: Props) {
  const { t } = useI18n();
  const [linking, setLinking] = useState(false);

  const state = useEditorState({
    editor,
    selector: ({ editor }) =>
      editor && {
        bold: editor.isActive("bold"),
        italic: editor.isActive("italic"),
        strike: editor.isActive("strike"),
        code: editor.isActive("code"),
        link: editor.isActive("link"),
        href: editor.getAttributes("link").href as string | undefined,
        heading: editor.getAttributes("heading").level as number | undefined,
        bulletList: editor.isActive("bulletList"),
        orderedList: editor.isActive("orderedList"),
        taskList: editor.isActive("taskList"),
        blockquote: editor.isActive("blockquote"),
        codeBlock: editor.isActive("codeBlock"),
        table: editor.isActive("table"),
        undo: editor.can().undo(),
        redo: editor.can().redo(),
      },
  });

  const tip = (label: string, keys?: string): ReactNode => (
    <>
      {label}
      {keys && <Kbd>{keys}</Kbd>}
    </>
  );

  const action = (
    label: string,
    icon: ReactNode,
    run: () => void,
    options: { keys?: string; disabled?: boolean } = {},
  ) => (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={label}
            disabled={options.disabled}
            onMouseDown={keep}
            onClick={run}
          />
        }
      >
        {icon}
      </TooltipTrigger>
      <TooltipContent>{tip(label, options.keys)}</TooltipContent>
    </Tooltip>
  );

  const toggle = (label: string, icon: ReactNode, pressed: boolean, run: () => void, keys?: string) => (
    <Tooltip>
      <TooltipTrigger
        render={
          <Toggle
            size="sm"
            className="size-7 min-w-7 px-0"
            aria-label={label}
            pressed={pressed}
            onPressedChange={run}
            onMouseDown={keep}
          />
        }
      >
        {icon}
      </TooltipTrigger>
      <TooltipContent>{tip(label, keys)}</TooltipContent>
    </Tooltip>
  );

  const divider = (
    <Separator orientation="vertical" className="mx-1 data-vertical:h-4 data-vertical:self-center" />
  );

  // The tools scroll when the column is narrow; the mode switch stays put,
  // and spells itself out only when the column, not the window, has room.
  const bar = (tools: ReactNode) => (
    <div className="@container flex h-11 shrink-0 items-center border-b">
      <div className="no-scrollbar flex min-w-0 flex-1 items-center gap-0.5 overflow-x-auto px-3">
        {tools}
      </div>
      <ToggleGroup
        variant="outline"
        size="sm"
        className="mx-3 shrink-0"
        value={[mode]}
        onValueChange={(value) => value[0] && onMode(value[0] as Mode)}
      >
        <ToggleGroupItem value="visual" aria-label={t("editor.visual")}>
          <PenLine />
          <span className="hidden @3xl:inline">{t("editor.visual")}</span>
        </ToggleGroupItem>
        <ToggleGroupItem value="source" aria-label={t("editor.source")}>
          <FileCode2 />
          <span className="hidden @3xl:inline">{t("editor.source")}</span>
        </ToggleGroupItem>
      </ToggleGroup>
    </div>
  );

  if (!editor || !state) {
    return bar(
      <>
        <span className="px-1 text-xs text-muted-foreground">{t("editor.sourceNote")}</span>
        {divider}
        {action(t("editor.image"), <Image />, onPickImage)}
      </>,
    );
  }

  const style = state.heading ? `h${state.heading}` : "paragraph";
  const styleLabel = state.heading
    ? t("editor.headingN", { level: state.heading })
    : t("editor.paragraph");

  const chain = () => editor.chain().focus();

  return bar(
    <>
      {action(t("editor.undo"), <Undo2 />, () => chain().undo().run(), {
        keys: `${mod}Z`,
        disabled: !state.undo,
      })}
      {action(t("editor.redo"), <Redo2 />, () => chain().redo().run(), {
        keys: `${mod}⇧Z`,
        disabled: !state.redo,
      })}
      {divider}

      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <Button
              variant="ghost"
              size="sm"
              className="w-26 justify-between px-2 font-normal"
              aria-label={t("editor.textStyle")}
              onMouseDown={keep}
            />
          }
        >
          <span className="truncate">{styleLabel}</span>
          <ChevronDown className="opacity-60" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="min-w-40">
          <DropdownMenuRadioGroup
            value={style}
            onValueChange={(value) => {
              const level = Number(String(value).replace("h", ""));
              if (level) chain().setHeading({ level: level as 1 | 2 | 3 | 4 }).run();
              else chain().setParagraph().run();
            }}
          >
            <DropdownMenuRadioItem value="paragraph">{t("editor.paragraph")}</DropdownMenuRadioItem>
            {([1, 2, 3, 4] as const).map((level) => (
              <DropdownMenuRadioItem
                key={level}
                value={`h${level}`}
                className={level === 1 ? "text-base font-bold" : level === 2 ? "font-semibold" : ""}
              >
                {t("editor.headingN", { level })}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
      {divider}

      {toggle(t("editor.bold"), <Bold />, state.bold, () => chain().toggleBold().run(), `${mod}B`)}
      {toggle(t("editor.italic"), <Italic />, state.italic, () => chain().toggleItalic().run(), `${mod}I`)}
      {toggle(
        t("editor.strike"),
        <Strikethrough />,
        state.strike,
        () => chain().toggleStrike().run(),
        `${mod}⇧X`,
      )}
      {toggle(t("editor.inlineCode"), <Code />, state.code, () => chain().toggleCode().run(), `${mod}E`)}

      <Popover open={linking} onOpenChange={setLinking}>
        <Tooltip>
          <TooltipTrigger
            render={
              <PopoverTrigger
                render={
                  <Toggle
                    size="sm"
                    className="size-7 min-w-7 px-0"
                    aria-label={t("editor.link")}
                    pressed={state.link}
                    onMouseDown={keep}
                  />
                }
              />
            }
          >
            <Link />
          </TooltipTrigger>
          <TooltipContent>{tip(t("editor.link"), `${mod}K`)}</TooltipContent>
        </Tooltip>
        <PopoverContent align="start" className="w-auto p-1.5">
          <LinkForm
            href={state.href}
            onApply={(href) => {
              const marked = chain().extendMarkRange("link");
              if (href) marked.setLink({ href }).run();
              else marked.unsetLink().run();
              setLinking(false);
            }}
            onRemove={state.link ? () => (chain().extendMarkRange("link").unsetLink().run(), setLinking(false)) : undefined}
            onCancel={() => {
              setLinking(false);
              editor.commands.focus();
            }}
          />
        </PopoverContent>
      </Popover>
      {divider}

      {toggle(
        t("editor.bulletList"),
        <List />,
        state.bulletList,
        () => chain().toggleBulletList().run(),
        `${mod}⇧8`,
      )}
      {toggle(
        t("editor.orderedList"),
        <ListOrdered />,
        state.orderedList,
        () => chain().toggleOrderedList().run(),
        `${mod}⇧7`,
      )}
      {toggle(
        t("editor.taskList"),
        <ListTodo />,
        state.taskList,
        () => chain().toggleTaskList().run(),
        `${mod}⇧9`,
      )}
      {divider}

      {toggle(
        t("editor.quote"),
        <TextQuote />,
        state.blockquote,
        () => chain().toggleBlockquote().run(),
        `${mod}⇧B`,
      )}
      {toggle(
        t("editor.codeBlock"),
        <SquareCode />,
        state.codeBlock,
        () => chain().toggleCodeBlock().run(),
        `${mod}⌥C`,
      )}
      {action(t("editor.image"), <Image />, onPickImage)}

      <DropdownMenu>
        <Tooltip>
          <TooltipTrigger
            render={
              <DropdownMenuTrigger
                render={
                  <Toggle
                    size="sm"
                    className="size-7 min-w-7 px-0"
                    aria-label={t("editor.table")}
                    pressed={state.table}
                    onMouseDown={keep}
                  />
                }
              />
            }
          >
            <Table />
          </TooltipTrigger>
          <TooltipContent>{t("editor.table")}</TooltipContent>
        </Tooltip>
        <DropdownMenuContent align="start" className="min-w-44">
          {state.table ? (
            <>
              <DropdownMenuGroup>
                <DropdownMenuItem onClick={() => chain().addRowAfter().run()}>
                  <Rows3 />
                  {t("editor.addRow")}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => chain().addColumnAfter().run()}>
                  <Columns3 />
                  {t("editor.addColumn")}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => chain().toggleHeaderRow().run()}>
                  <Table />
                  {t("editor.toggleHeader")}
                </DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem onClick={() => chain().deleteRow().run()}>
                  <Rows3 />
                  {t("editor.deleteRow")}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => chain().deleteColumn().run()}>
                  <Columns3 />
                  {t("editor.deleteColumn")}
                </DropdownMenuItem>
                <DropdownMenuItem variant="destructive" onClick={() => chain().deleteTable().run()}>
                  <Trash2 />
                  {t("editor.deleteTable")}
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </>
          ) : (
            <DropdownMenuItem
              onClick={() => chain().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()}
            >
              <Table />
              {t("editor.insertTable")}
            </DropdownMenuItem>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      {action(t("editor.divider"), <Minus />, () => chain().setHorizontalRule().run())}
    </>,
  );
}
