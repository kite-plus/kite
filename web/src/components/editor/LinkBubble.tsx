import { useEffect, useState } from "react";
import { getMarkRange, posToDOMRect, type Editor } from "@tiptap/core";
import { useEditorState } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";

import { useI18n } from "@/i18n";
import { composing } from "@/lib/ime";
import { sanitizeUrl } from "@/lib/tiptap-utils";
import { CornerDownLeftIcon } from "@/components/tiptap-icons/corner-down-left-icon";
import { ExternalLinkIcon } from "@/components/tiptap-icons/external-link-icon";
import { TrashIcon } from "@/components/tiptap-icons/trash-icon";
import { Button } from "@/components/tiptap-ui-primitive/button";
import { Input } from "@/components/tiptap-ui-primitive/input";
import { Separator } from "@/components/tiptap-ui-primitive/separator";

/** linkRange is the whole link the caret or selection is in, if any. */
function linkRange(editor: Editor) {
  const { selection, schema } = editor.state;
  const type = schema.marks.link;
  if (!type) return undefined;
  return getMarkRange(selection.$from, type) ?? getMarkRange(selection.$to, type);
}

/**
 * LinkBubble sits under a link while the caret is in it, with its address
 * to change, open or take away. It leaves the caret where it is, so the
 * link's words can still be typed over; only a click into the address
 * edits the address.
 */
export function LinkBubble({ editor, resolveUrl }: { editor: Editor; resolveUrl: (url: string) => string }) {
  const { t } = useI18n();
  const link = useEditorState({
    editor,
    selector: ({ editor }) => {
      if (!editor.isActive("link")) return null;
      const range = linkRange(editor);
      return { href: String(editor.getAttributes("link").href ?? ""), from: range?.from };
    },
  });
  const [url, setUrl] = useState("");
  // Another link, or this one changed elsewhere, starts the field over.
  useEffect(() => setUrl(link?.href ?? ""), [link?.href, link?.from]);

  const apply = () => {
    const href = url.trim();
    // An address cleared away takes the link with it.
    if (!href) return remove();
    editor.chain().focus().extendMarkRange("link").setLink({ href }).run();
  };
  const remove = () => {
    editor.chain().focus().extendMarkRange("link").unsetLink().setMeta("preventAutolink", true).run();
  };
  const open = () => {
    const safe = sanitizeUrl(resolveUrl(link?.href ?? ""), window.location.href);
    if (safe !== "#") window.open(safe, "_blank", "noopener,noreferrer");
  };

  return (
    <BubbleMenu
      editor={editor}
      pluginKey="linkBubble"
      options={{ placement: "bottom-start", offset: 6, shift: { padding: 8 }, flip: { padding: 8 } }}
      shouldShow={({ editor, view, element }) =>
        editor.isEditable && (view.hasFocus() || element.contains(document.activeElement)) && editor.isActive("link")
      }
      // Under the whole link rather than the caret, wherever in it the caret is.
      getReferencedVirtualElement={() => {
        const range = linkRange(editor);
        if (!range) return null;
        const rect = posToDOMRect(editor.view, range.from, range.to);
        return { getBoundingClientRect: () => rect, getClientRects: () => [rect] };
      }}
    >
      {/* The floating toolbar's look, as the image's bubble has it. */}
      <div className="tiptap-toolbar kite-link-bubble" data-variant="floating">
        <Input
          type="url"
          value={url}
          placeholder={t("editor.linkPlaceholder")}
          aria-label={t("editor.link")}
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="off"
          className="kite-link-input"
          onChange={(event) => setUrl(event.target.value)}
          onKeyDown={(event) => {
            if (composing(event)) return;
            if (event.key === "Enter") {
              event.preventDefault();
              apply();
            } else if (event.key === "Escape") {
              event.preventDefault();
              setUrl(link?.href ?? "");
              editor.commands.focus();
            }
          }}
        />
        <Button
          type="button"
          variant="ghost"
          aria-label={t("editor.linkApply")}
          tooltip={t("editor.linkApply")}
          disabled={url.trim() === (link?.href ?? "")}
          onClick={apply}
        >
          <CornerDownLeftIcon className="tiptap-button-icon" />
        </Button>
        <Separator orientation="vertical" />
        <Button
          type="button"
          variant="ghost"
          aria-label={t("editor.linkOpen")}
          tooltip={t("editor.linkOpen")}
          disabled={!link?.href}
          onClick={open}
        >
          <ExternalLinkIcon className="tiptap-button-icon" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          aria-label={t("editor.linkRemove")}
          tooltip={t("editor.linkRemove")}
          onClick={remove}
        >
          <TrashIcon className="tiptap-button-icon" />
        </Button>
      </div>
    </BubbleMenu>
  );
}
