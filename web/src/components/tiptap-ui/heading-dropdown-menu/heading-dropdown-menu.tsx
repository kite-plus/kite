import { forwardRef, useCallback, useState } from "react"
import { Type } from "lucide-react"

import { useI18n } from "@/i18n"

// --- Icons ---
import { ChevronDownIcon } from "@/components/tiptap-icons/chevron-down-icon"

// --- Hooks ---
import { useTiptapEditor } from "@/hooks/use-tiptap-editor"

// --- Tiptap UI ---
import { HeadingButton } from "@/components/tiptap-ui/heading-button"
import type { UseHeadingDropdownMenuConfig } from "@/components/tiptap-ui/heading-dropdown-menu"
import { useHeadingDropdownMenu } from "@/components/tiptap-ui/heading-dropdown-menu"

// --- UI Primitives ---
import type { ButtonProps } from "@/components/tiptap-ui-primitive/button"
import { Button } from "@/components/tiptap-ui-primitive/button"
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuGroup,
} from "@/components/tiptap-ui-primitive/dropdown-menu"

export interface HeadingDropdownMenuProps
  extends Omit<ButtonProps, "type">, UseHeadingDropdownMenuConfig {
  /**
   * Callback for when the dropdown opens or closes
   */
  onOpenChange?: (isOpen: boolean) => void
  /**
   * Whether the dropdown should use a modal
   */
  modal?: boolean
}

/**
 * Dropdown menu component for selecting heading levels in a Tiptap editor.
 *
 * For custom dropdown implementations, use the `useHeadingDropdownMenu` hook instead.
 */
export const HeadingDropdownMenu = forwardRef<
  HTMLButtonElement,
  HeadingDropdownMenuProps
>(
  (
    {
      editor: providedEditor,
      levels = [1, 2, 3, 4, 5, 6],
      hideWhenUnavailable = false,
      onOpenChange,
      children,
      modal = true,
      ...buttonProps
    },
    ref
  ) => {
    const { editor } = useTiptapEditor(providedEditor)
    const { t } = useI18n()
    const [isOpen, setIsOpen] = useState<boolean>(false)
    const { isVisible, isActive, canToggle, label, Icon } =
      useHeadingDropdownMenu({
        editor,
        levels,
        hideWhenUnavailable,
      })

    const handleOpenChange = useCallback(
      (open: boolean) => {
        if (!editor || !canToggle) return
        setIsOpen(open)
        onOpenChange?.(open)
      },
      [canToggle, editor, onOpenChange]
    )

    if (!isVisible) {
      return null
    }

    return (
      <DropdownMenu modal={modal} open={isOpen} onOpenChange={handleOpenChange}>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            data-active-state={isActive ? "on" : "off"}
            role="button"
            tabIndex={-1}
            disabled={!canToggle}
            data-disabled={!canToggle}
            aria-label={label}
            aria-pressed={isActive}
            tooltip={label}
            {...buttonProps}
            ref={ref}
          >
            {children ? (
              children
            ) : (
              <>
                <Icon className="tiptap-button-icon" />
                <ChevronDownIcon className="tiptap-button-dropdown-small" />
              </>
            )}
          </Button>
        </DropdownMenuTrigger>

        <DropdownMenuContent align="start">
          <DropdownMenuGroup>
            {/* Kite: the way back to plain text, which the slash menu offers too. */}
            <DropdownMenuItem asChild>
              <Button
                type="button"
                variant="ghost"
                data-active-state={editor?.isActive("paragraph") ? "on" : "off"}
                role="button"
                tabIndex={-1}
                onClick={() => editor?.chain().focus().setParagraph().run()}
              >
                <Type className="tiptap-button-icon" />
                <span className="tiptap-button-text">{t("editor.paragraph")}</span>
              </Button>
            </DropdownMenuItem>
            {levels.map((level) => (
              <DropdownMenuItem key={`heading-${level}`} asChild>
                <HeadingButton
                  editor={editor}
                  level={level}
                  text={t("editor.headingN", { level })}
                  showTooltip={false}
                />
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    )
  }
)

HeadingDropdownMenu.displayName = "HeadingDropdownMenu"

export default HeadingDropdownMenu
