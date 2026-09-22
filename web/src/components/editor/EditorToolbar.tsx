import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { EditorContext, type Editor } from "@tiptap/react";
import { FileCode2, PenLine } from "lucide-react";

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
import { UndoRedoButton } from "@/components/tiptap-ui/undo-redo-button";
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
 * The groups are the template's; its alignment, highlight, underline and
 * super/subscript buttons are gone, and a table menu and the switch between
 * the visual editor and the markdown take their place.
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

  // The end of the bar stays in reach when the tools scroll sideways.
  const end = (search?: ReactNode) => (
    <div className="kite-toolbar-end">
      {search && (
        <>
          <ToolbarGroup>{search}</ToolbarGroup>
          <ToolbarSeparator />
        </>
      )}
      <ToolbarGroup className="kite-toolbar-modes">
        {(["visual", "source"] as const).map((each) => (
          <Button
            key={each}
            type="button"
            variant="ghost"
            role="button"
            tabIndex={-1}
            data-active-state={mode === each ? "on" : "off"}
            aria-pressed={mode === each}
            aria-label={t(each === "visual" ? "editor.visual" : "editor.source")}
            tooltip={t(each === "visual" ? "editor.visual" : "editor.source")}
            onClick={() => onMode(each)}
          >
            {each === "visual" ? (
              <PenLine className="tiptap-button-icon" />
            ) : (
              <FileCode2 className="tiptap-button-icon" />
            )}
            {/* Hidden by editor.scss when the column is too narrow for it. */}
            <span className="tiptap-button-text">
              {t(each === "visual" ? "editor.visual" : "editor.source")}
            </span>
          </Button>
        ))}
      </ToolbarGroup>
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
        <Spacer />

        <ToolbarGroup>
          <UndoRedoButton action="undo" />
          <UndoRedoButton action="redo" />
        </ToolbarGroup>

        <ToolbarSeparator />

        <ToolbarGroup>
          <HeadingDropdownMenu modal={false} levels={[1, 2, 3, 4]} />
          <ListDropdownMenu modal={false} types={["bulletList", "orderedList", "taskList"]} />
          <BlockquoteButton />
          <CodeBlockButton />
        </ToolbarGroup>

        <ToolbarSeparator />

        <ToolbarGroup>
          <MarkButton type="bold" />
          <MarkButton type="italic" />
          <MarkButton type="strike" />
          <MarkButton type="code" />
          {isMobile ? (
            <LinkButton onClick={() => setLinking(true)} />
          ) : (
            <LinkPopover resolveUrl={resolveUrl} />
          )}
        </ToolbarGroup>

        <ToolbarSeparator />

        <ToolbarGroup>
          <ImageUploadButton />
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
