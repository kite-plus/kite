import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ComponentType,
  type ReactNode,
  type RefObject,
} from "react";
import { EditorContext, type ChainedCommands, type Editor } from "@tiptap/react";
import { Ellipsis, Minus, RemoveFormatting } from "lucide-react";

import { useI18n, type Key } from "@/i18n";
import { resolveLink } from "@/lib/links";
import { isMac, parseShortcutKeys } from "@/lib/tiptap-utils";
import { useIsBreakpoint } from "@/hooks/use-is-breakpoint";
import { useTiptapEditor } from "@/hooks/use-tiptap-editor";
import type { Mode } from "@/components/editor/markdown";
import { TableMenu } from "@/components/editor/TableMenu";
import { ArrowLeftIcon } from "@/components/tiptap-icons/arrow-left-icon";
import { BlockquoteIcon } from "@/components/tiptap-icons/blockquote-icon";
import { CodeBlockIcon } from "@/components/tiptap-icons/code-block-icon";
import { Code2Icon } from "@/components/tiptap-icons/code2-icon";
import { ImagePlusIcon } from "@/components/tiptap-icons/image-plus-icon";
import { LinkIcon } from "@/components/tiptap-icons/link-icon";
import { Redo2Icon } from "@/components/tiptap-icons/redo2-icon";
import { StrikeIcon } from "@/components/tiptap-icons/strike-icon";
import { Undo2Icon } from "@/components/tiptap-icons/undo2-icon";
import { BlockquoteButton } from "@/components/tiptap-ui/blockquote-button";
import { CodeBlockButton } from "@/components/tiptap-ui/code-block-button";
import { HeadingDropdownMenu } from "@/components/tiptap-ui/heading-dropdown-menu";
import { ImageUploadButton } from "@/components/tiptap-ui/image-upload-button";
import { LinkButton, LinkContent, LinkPopover } from "@/components/tiptap-ui/link-popover";
import { ListDropdownMenu } from "@/components/tiptap-ui/list-dropdown-menu";
import { MarkButton } from "@/components/tiptap-ui/mark-button";
import { SearchAndReplace, SearchAndReplaceButton } from "@/components/tiptap-ui/search-and-replace";
import { UndoRedoButton } from "@/components/tiptap-ui/undo-redo-button";
import { Button, type ButtonProps } from "@/components/tiptap-ui-primitive/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/tiptap-ui-primitive/dropdown-menu";
import { Spacer } from "@/components/tiptap-ui-primitive/spacer";
import { Toolbar, ToolbarGroup, ToolbarSeparator } from "@/components/tiptap-ui-primitive/toolbar";

interface Props {
  /** editor is null in source mode, where there is nothing to format. */
  editor: Editor | null;
  mode: Mode;
  onMode: (mode: Mode) => void;
  /** onPickImage asks for a file to put in the markdown, in source mode. */
  onPickImage: () => void;
  /** base is the item's own address, which links in the body are relative to. */
  base?: string;
}

const searchScroll: ScrollIntoViewOptions = { block: "center" };

interface Extra {
  /** tier is when it gives way to a narrow bar: the lowest first. */
  tier: number;
  label: Key;
  icon: ComponentType<{ className?: string }>;
  run: (chain: ChainedCommands) => ChainedCommands;
  can?: (editor: Editor) => boolean;
  keys?: string;
}

// The tools that make room when the bar is too narrow for all of them, in the
// order they stand on it. Undo and redo go first, having their keys; then the
// divider, which the slash menu and "---" also put in; then clearing, strike
// and inline code, the least used of the formatting; and on the narrowest bars
// quotes and code blocks, which the slash menu has too.
const extras: Extra[] = [
  {
    tier: 0,
    label: "editor.undo",
    icon: Undo2Icon,
    keys: "mod+z",
    run: (chain) => chain.undo(),
    can: (editor) => editor.can().undo(),
  },
  {
    tier: 0,
    label: "editor.redo",
    icon: Redo2Icon,
    keys: "mod+shift+z",
    run: (chain) => chain.redo(),
    can: (editor) => editor.can().redo(),
  },
  { tier: 6, label: "editor.codeBlock", icon: CodeBlockIcon, keys: "mod+alt+c", run: (chain) => chain.toggleCodeBlock() },
  { tier: 5, label: "editor.quote", icon: BlockquoteIcon, keys: "mod+shift+b", run: (chain) => chain.toggleBlockquote() },
  { tier: 3, label: "editor.strike", icon: StrikeIcon, keys: "mod+shift+s", run: (chain) => chain.toggleStrike() },
  { tier: 4, label: "editor.inlineCode", icon: Code2Icon, keys: "mod+e", run: (chain) => chain.toggleCode() },
  { tier: 1, label: "editor.divider", icon: Minus, run: (chain) => chain.setHorizontalRule() },
  { tier: 2, label: "editor.clearFormatting", icon: RemoveFormatting, run: (chain) => chain.unsetAllMarks() },
];
const tiers = 7;
const extra = (label: Key) => extras.find((each) => each.label === label)!;

/**
 * The Simple Editor template's toolbar, holding only what markdown can say.
 *
 * It is laid out as the admin design draws it: left-aligned, the design's
 * buttons first and in its order, the tools it leaves out around them, and
 * the markdown switch at the far end. Where the bar is too narrow, as beside
 * the preview, the extra tools move into a More menu rather than off its edge.
 */
export function EditorToolbar({ editor, mode, onMode, onPickImage, base }: Props) {
  const { t } = useI18n();
  const isMobile = useIsBreakpoint();
  const bar = useRef<HTMLDivElement>(null);
  // A phone has no room for a popover, so the link form takes the toolbar's place.
  const [linking, setLinking] = useState(false);
  const [searching, setSearching] = useState(false);
  const searchButton = useRef<HTMLButtonElement>(null);
  const resolveUrl = useCallback((url: string) => resolveLink(url, base), [base]);

  useEffect(() => {
    if (!isMobile) setLinking(false);
  }, [isMobile]);
  // The search belongs to the visual editor, and goes when it does.
  useEffect(() => {
    if (!editor) {
      setSearching(false);
      setLinking(false);
    }
  }, [editor]);

  const hidden = useFit(bar, editor && !linking ? tiers : 0);

  const closeSearch = useCallback(() => {
    setSearching(false);
    searchButton.current?.focus();
  }, []);

  // The end of the bar stays in reach when the tools scroll sideways. The
  // switch is one toggle, pressed while the markdown itself is showing.
  const end = (search?: ReactNode, more?: ReactNode) => (
    <div className="kite-toolbar-end">
      {more}
      {search && <ToolbarGroup>{search}</ToolbarGroup>}
      <button
        type="button"
        aria-pressed={mode === "source"}
        onClick={() => onMode(mode === "source" ? "visual" : "source")}
        className="h-8 shrink-0 rounded-md bg-muted px-3 text-[13px] whitespace-nowrap text-foreground outline-none transition-colors hover:bg-border focus-visible:ring-2 focus-visible:ring-ring/50 aria-pressed:bg-input aria-pressed:font-medium"
      >
        {t("editor.source")}
      </button>
    </div>
  );

  let content;
  if (!editor) {
    content = (
      <>
        <ToolbarGroup>
          <span className="kite-toolbar-note">{t("editor.sourceNote")}</span>
        </ToolbarGroup>
        <ToolbarSeparator />
        <ToolbarGroup>
          <Button
            type="button"
            variant="ghost"
            role="button"
            tabIndex={-1}
            aria-label={t("editor.addImage")}
            tooltip={t("editor.addImage")}
            onClick={onPickImage}
          >
            <ImagePlusIcon className="tiptap-button-icon" />
          </Button>
        </ToolbarGroup>
        <Spacer />
        {end()}
      </>
    );
  } else if (linking) {
    content = (
      <>
        <ToolbarGroup>
          <Button type="button" variant="ghost" onClick={() => setLinking(false)}>
            <ArrowLeftIcon className="tiptap-button-icon" />
            <LinkIcon className="tiptap-button-icon" />
          </Button>
        </ToolbarGroup>
        <ToolbarSeparator />
        <LinkContent resolveUrl={resolveUrl} />
      </>
    );
  } else {
    content = (
      <>
        {hidden < 1 && (
          <>
            <ToolbarGroup data-tier={0}>
              <UndoRedoButton action="undo" />
              <UndoRedoButton action="redo" />
            </ToolbarGroup>
            <ToolbarSeparator data-tier={0} />
          </>
        )}

        <ToolbarGroup>
          <MarkButton type="bold" />
          <MarkButton type="italic" />
          <HeadingDropdownMenu modal={false} levels={[1, 2, 3, 4]} />
        </ToolbarGroup>

        <ToolbarSeparator />

        <ToolbarGroup>
          {isMobile ? (
            <LinkButton onClick={() => setLinking(true)} />
          ) : (
            <LinkPopover resolveUrl={resolveUrl} />
          )}
          <ImageUploadButton />
          {hidden < 7 && <CodeBlockButton data-tier={6} />}
          <ListDropdownMenu modal={false} types={["bulletList", "orderedList", "taskList"]} />
        </ToolbarGroup>

        <ToolbarSeparator />

        <ToolbarGroup>
          {hidden < 6 && <BlockquoteButton data-tier={5} />}
          {hidden < 4 && <MarkButton type="strike" data-tier={3} />}
          {hidden < 5 && <MarkButton type="code" data-tier={4} />}
          <TableMenu />
          {hidden < 2 && <ExtraButton extra={extra("editor.divider")} data-tier={1} />}
        </ToolbarGroup>

        {hidden < 3 && (
          <>
            <ToolbarSeparator data-tier={2} />
            <ToolbarGroup data-tier={2}>
              <ExtraButton extra={extra("editor.clearFormatting")} />
            </ToolbarGroup>
          </>
        )}

        <Spacer data-spacer="" />

        {end(
          <SearchAndReplaceButton
            ref={searchButton}
            aria-expanded={searching}
            data-active-state={searching ? "on" : "off"}
            onClick={() => (searching ? closeSearch() : setSearching(true))}
          />,
          // At the end, so it stays in reach however far the tools scroll.
          hidden > 0 && <MoreMenu editor={editor} extras={extras.filter((each) => each.tier < hidden)} />,
        )}
      </>
    );
  }

  return (
    <EditorContext.Provider value={{ editor }}>
      <Toolbar ref={bar} className="kite-toolbar">
        {content}
      </Toolbar>
      {editor && (
        <SearchAndReplace
          className="kite-search"
          open={searching}
          onOpen={() => {
            setLinking(false);
            setSearching(true);
          }}
          onClose={closeSearch}
          scrollIntoViewOptions={searchScroll}
        />
      )}
    </EditorContext.Provider>
  );
}

/** ExtraButton is one of the extra tools, on the bar. */
function ExtraButton({ extra, ...props }: { extra: Extra } & Omit<ButtonProps, "type">) {
  const { t } = useI18n();
  const { editor } = useTiptapEditor();
  return (
    <Button
      type="button"
      variant="ghost"
      role="button"
      tabIndex={-1}
      aria-label={t(extra.label)}
      tooltip={t(extra.label)}
      disabled={!editor?.isEditable}
      onClick={() => editor && extra.run(editor.chain().focus()).run()}
      {...props}
    >
      <extra.icon className="tiptap-button-icon" />
    </Button>
  );
}

/** MoreMenu holds the extra tools that have given way. */
function MoreMenu({ editor, extras }: { editor: Editor; extras: Extra[] }) {
  const { t } = useI18n();
  return (
    <ToolbarGroup data-more="">
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            role="button"
            tabIndex={-1}
            aria-label={t("editor.more")}
            tooltip={t("editor.more")}
          >
            <Ellipsis className="tiptap-button-icon" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuGroup>
            {extras.map((extra) => {
              const off = extra.can ? !extra.can(editor) : false;
              return (
                <DropdownMenuItem
                  key={extra.label}
                  asChild
                  disabled={off}
                  onSelect={() => extra.run(editor.chain().focus()).run()}
                >
                  <Button type="button" variant="ghost" showTooltip={false} disabled={off}>
                    <extra.icon className="tiptap-button-icon" />
                    <span className="tiptap-button-text">{t(extra.label)}</span>
                    {extra.keys && (
                      <kbd className="kite-menu-keys">
                        {parseShortcutKeys({ shortcutKeys: extra.keys }).join(isMac() ? "" : "+")}
                      </kbd>
                    )}
                  </Button>
                </DropdownMenuItem>
              );
            })}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </ToolbarGroup>
  );
}

/**
 * useFit counts the tiers of extra tools that have to give way for the bar
 * to fit its width. A tier is measured as it leaves and comes back once the
 * room left over is that wide again; the More button, which holds the tiers,
 * frees its own room with the last of them.
 */
function useFit(bar: RefObject<HTMLElement | null>, count: number): number {
  const [hidden, setHidden] = useState(0);
  const current = useRef(hidden);
  current.current = hidden;
  const widths = useRef<number[]>([]);

  const fit = useCallback(() => {
    const element = bar.current;
    if (!element) return;
    const gap = parseFloat(getComputedStyle(element).columnGap) || 0;
    const width = (selector: string) =>
      Array.from(element.querySelectorAll(selector)).reduce(
        (sum, part) => sum + part.getBoundingClientRect().width + gap,
        0,
      );

    const now = Math.min(current.current, count);
    let next = now;
    if (element.scrollWidth > element.clientWidth + 1) {
      if (now < count) {
        widths.current[now] = width(`[data-tier="${now}"]`);
        next = now + 1;
      }
    } else if (now > 0) {
      const room = width("[data-spacer]") - gap + (now === 1 ? width("[data-more]") : 0);
      if (room >= widths.current[now - 1]) next = now - 1;
    }
    if (next !== current.current) setHidden(next);
  }, [bar, count]);

  // After each change, for the next tier may have to go, or may fit again.
  useLayoutEffect(fit, [fit, hidden]);
  useEffect(() => {
    const element = bar.current;
    if (!element) return;
    const observer = new ResizeObserver(() => fit());
    observer.observe(element);
    return () => observer.disconnect();
  }, [bar, fit]);

  return Math.min(hidden, count);
}
