import { useState, type ComponentType } from "react";
import type { ChainedCommands } from "@tiptap/react";
import {
  BetweenHorizontalEnd,
  BetweenVerticalEnd,
  PanelTop,
  Table,
  Trash2,
} from "lucide-react";

import { useI18n, type Key } from "@/i18n";
import { useTiptapEditor } from "@/hooks/use-tiptap-editor";
import { ChevronDownIcon } from "@/components/tiptap-icons/chevron-down-icon";
import { Button } from "@/components/tiptap-ui-primitive/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/tiptap-ui-primitive/dropdown-menu";
import { Separator } from "@/components/tiptap-ui-primitive/separator";

interface Action {
  label: Key;
  icon: ComponentType<{ className?: string }>;
  run: (chain: ChainedCommands) => ChainedCommands;
}

const insert: Action = {
  label: "editor.insertTable",
  icon: Table,
  run: (chain) => chain.insertTable({ rows: 3, cols: 3, withHeaderRow: true }),
};

// What can be done to the table the cursor is in, in two groups: adding, then removing.
const edits: Action[][] = [
  [
    { label: "editor.addRow", icon: BetweenHorizontalEnd, run: (chain) => chain.addRowAfter() },
    { label: "editor.addColumn", icon: BetweenVerticalEnd, run: (chain) => chain.addColumnAfter() },
    { label: "editor.toggleHeader", icon: PanelTop, run: (chain) => chain.toggleHeaderRow() },
  ],
  [
    { label: "editor.deleteRow", icon: Trash2, run: (chain) => chain.deleteRow() },
    { label: "editor.deleteColumn", icon: Trash2, run: (chain) => chain.deleteColumn() },
    { label: "editor.deleteTable", icon: Trash2, run: (chain) => chain.deleteTable() },
  ],
];

/**
 * The toolbar's table button. It puts a table in, and once the cursor is in
 * one it turns into that table's rows and columns, since markdown tables
 * have no handles of their own to grab.
 */
export function TableMenu() {
  const { t } = useI18n();
  const { editor } = useTiptapEditor();
  const [open, setOpen] = useState(false);

  if (!editor) return null;
  const inTable = editor.isActive("table");
  const groups = inTable ? edits : [[insert]];

  return (
    <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          role="button"
          tabIndex={-1}
          data-active-state={inTable ? "on" : "off"}
          disabled={!editor.isEditable}
          aria-label={t("editor.table")}
          tooltip={t("editor.table")}
        >
          <Table className="tiptap-button-icon" />
          <ChevronDownIcon className="tiptap-button-dropdown-small" />
        </Button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="start">
        {groups.map((actions, i) => (
          <DropdownMenuGroup key={i}>
            {i > 0 && <Separator orientation="horizontal" />}
            {actions.map((action) => (
              <DropdownMenuItem
                key={action.label}
                asChild
                onSelect={() => action.run(editor.chain().focus()).run()}
              >
                <Button type="button" variant="ghost" showTooltip={false}>
                  <action.icon className="tiptap-button-icon" />
                  <span className="tiptap-button-text">{t(action.label)}</span>
                </Button>
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
