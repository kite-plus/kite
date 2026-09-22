import { useEditorState, type Editor } from "@tiptap/react";
import { BubbleMenu } from "@tiptap/react/menus";

import { useI18n } from "@/i18n";
import { TrashIcon } from "@/components/tiptap-icons/trash-icon";
import { Button } from "@/components/tiptap-ui-primitive/button";
import { Input } from "@/components/tiptap-ui-primitive/input";
import { Separator } from "@/components/tiptap-ui-primitive/separator";

/**
 * What a picked image offers: its alt text, which is the part of
 * `![alt](src)` the author writes, and a way to take the image out.
 */
export function ImageBubble({ editor }: { editor: Editor }) {
  const { t } = useI18n();
  const image = useEditorState({
    editor,
    selector: ({ editor }) =>
      editor.isActive("image")
        ? {
            alt: (editor.getAttributes("image").alt as string | undefined) ?? "",
            at: editor.state.selection.from,
          }
        : null,
  });

  return (
    <BubbleMenu
      editor={editor}
      options={{ placement: "top", offset: 8 }}
      shouldShow={({ editor }) => editor.isActive("image")}
    >
      {/* The floating toolbar's look, without the Toolbar's arrow-key
          handling, which would take the caret away from the field. */}
      <div className="tiptap-toolbar kite-image-bubble" data-variant="floating">
        <Input
          key={image?.at}
          defaultValue={image?.alt}
          placeholder={t("editor.altText")}
          aria-label={t("editor.altText")}
          className="kite-alt-input"
          onChange={(event) => editor.commands.updateAttributes("image", { alt: event.target.value })}
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === "Escape") {
              event.preventDefault();
              editor.commands.focus();
            }
          }}
        />
        <Separator orientation="vertical" />
        <Button
          type="button"
          variant="ghost"
          aria-label={t("editor.removeImage")}
          tooltip={t("editor.removeImage")}
          onClick={() => editor.chain().focus().deleteSelection().run()}
        >
          <TrashIcon className="tiptap-button-icon" />
        </Button>
      </div>
    </BubbleMenu>
  );
}
