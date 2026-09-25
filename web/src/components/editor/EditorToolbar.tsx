import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { EditorContext, type Editor } from "@tiptap/react";

import { useI18n } from "@/i18n";
import { resolveLink } from "@/lib/links";
import { useIsBreakpoint } from "@/hooks/use-is-breakpoint";
import type { Mode } from "@/components/editor/markdown";
import { TableMenu } from "@/components/editor/TableMenu";
import { ArrowLeftIcon } from "@/components/tiptap-icons/arrow-left-icon";
import { ImagePlusIcon } from "@/components/tiptap-icons/image-plus-icon";
import { LinkIcon } from "@/components/tiptap-icons/link-icon";
import { BlockquoteButton } from "@/components/tiptap-ui/blockquote-button";
import { CodeBlockButton } from "@/components/tiptap-ui/code-block-button";
import { HeadingDropdownMenu } from "@/components/tiptap-ui/heading-dropdown-menu";
import { ImageUploadButton } from "@/components/tiptap-ui/image-upload-button";
import { LinkButton, LinkContent, LinkPopover } from "@/components/tiptap-ui/link-popover";
import { ListDropdownMenu } from "@/components/tiptap-ui/list-dropdown-menu";
import { MarkButton } from "@/components/tiptap-ui/mark-button";
import { SearchAndReplace, SearchAndReplaceButton } from "@/components/tiptap-ui/search-and-replace";
import { Button } from "@/components/tiptap-ui-primitive/button";
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

/**
 * The Simple Editor template's toolbar, holding only what markdown can say.
 *
 * It is laid out as the admin design draws it: left-aligned, the design's
 * buttons first and in its order, then the tools it leaves out, and the
 * markdown switch at the far end. Undo and redo are left to the keyboard, as
 * the design has no buttons for them.
 */
export function EditorToolbar({ editor, mode, onMode, onPickImage, base }: Props) {
  const { t } = useI18n();
  const isMobile = useIsBreakpoint();
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

  const closeSearch = useCallback(() => {
    setSearching(false);
    searchButton.current?.focus();
  }, []);

  // The end of the bar stays in reach when the tools scroll sideways. The
  // switch is one toggle, pressed while the markdown itself is showing.
  const end = (search?: ReactNode) => (
    <div className="kite-toolbar-end">
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
          <CodeBlockButton />
          <ListDropdownMenu modal={false} types={["bulletList", "orderedList", "taskList"]} />
        </ToolbarGroup>

        <ToolbarSeparator />

        <ToolbarGroup>
          <BlockquoteButton />
          <MarkButton type="strike" />
          <MarkButton type="code" />
          <TableMenu />
        </ToolbarGroup>

        <Spacer />

        {end(
          <SearchAndReplaceButton
            ref={searchButton}
            aria-expanded={searching}
            data-active-state={searching ? "on" : "off"}
            onClick={() => (searching ? closeSearch() : setSearching(true))}
          />,
        )}
      </>
    );
  }

  return (
    <EditorContext.Provider value={{ editor }}>
      <Toolbar className="kite-toolbar">{content}</Toolbar>
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
