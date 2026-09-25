import { useId, useState, type ComponentType, type KeyboardEvent } from "react";
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
import { Card, CardBody } from "@/components/tiptap-ui-primitive/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/tiptap-ui-primitive/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/tiptap-ui-primitive/popover";
import { Separator } from "@/components/tiptap-ui-primitive/separator";

interface Action {
  label: Key;
  icon: ComponentType<{ className?: string }>;
  run: (chain: ChainedCommands) => ChainedCommands;
}

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

// The largest table the grid offers; a bigger one grows from the menu later.
const ROWS = 8;
const COLUMNS = 8;

/**
 * The toolbar's table button. It puts a table in, sized on a grid, and once
 * the cursor is in one it turns into that table's rows and columns, since
 * markdown tables have no handles of their own to grab.
 */
export function TableMenu() {
  const { t } = useI18n();
  const { editor } = useTiptapEditor();
  const [open, setOpen] = useState(false);

  if (!editor) return null;
  const inTable = editor.isActive("table");

  const trigger = (
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
  );

  if (!inTable) {
    return (
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>{trigger}</PopoverTrigger>
        <PopoverContent align="start" onCloseAutoFocus={(event) => event.preventDefault()}>
          <TableSize
            onPick={(rows, cols) => {
              setOpen(false);
              editor.chain().focus().insertTable({ rows, cols, withHeaderRow: true }).run();
            }}
          />
        </PopoverContent>
      </Popover>
    );
  }

  return (
    <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger asChild>{trigger}</DropdownMenuTrigger>

      <DropdownMenuContent align="start">
        {edits.map((actions, i) => (
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

const steps: Record<string, [number, number]> = {
  ArrowUp: [-1, 0],
  ArrowDown: [1, 0],
  ArrowLeft: [0, -1],
  ArrowRight: [0, 1],
};

const within = (value: number, most: number) => Math.min(most, Math.max(1, value));

/**
 * TableSize sizes a new table on a grid: the cells up to the one under the
 * pointer light up, and a click puts in a table of that many rows, the first
 * of them its header, and columns. The arrow keys and Enter do the same.
 */
function TableSize({ onPick }: { onPick: (rows: number, cols: number) => void }) {
  const { t } = useI18n();
  const [size, setSize] = useState({ rows: 3, cols: 3 });
  const said = useId();

  const keys = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onPick(size.rows, size.cols);
      return;
    }
    const step = steps[event.key];
    if (!step) return;
    event.preventDefault();
    setSize(({ rows, cols }) => ({ rows: within(rows + step[0], ROWS), cols: within(cols + step[1], COLUMNS) }));
  };

  return (
    <Card>
      <CardBody>
        <div className="kite-table-size">
          <div className="kite-table-size-head">
            <span>{t("editor.insertTable")}</span>
            <span id={said} aria-live="polite">
              {t("editor.tableRows", { count: size.rows })} × {t("editor.tableColumns", { count: size.cols })}
            </span>
          </div>
          <div
            role="group"
            tabIndex={0}
            aria-label={t("editor.insertTable")}
            aria-describedby={said}
            className="kite-table-size-grid"
            onKeyDown={keys}
            onPointerMove={(event) => {
              const cell = (event.target as HTMLElement).closest<HTMLElement>("[data-row]");
              if (cell) setSize({ rows: Number(cell.dataset.row), cols: Number(cell.dataset.col) });
            }}
            onClick={() => onPick(size.rows, size.cols)}
          >
            {Array.from({ length: ROWS * COLUMNS }, (_, i) => {
              const row = Math.floor(i / COLUMNS) + 1;
              const col = (i % COLUMNS) + 1;
              return (
                <span
                  key={i}
                  data-row={row}
                  data-col={col}
                  data-on={row <= size.rows && col <= size.cols ? "" : undefined}
                />
              );
            })}
          </div>
        </div>
      </CardBody>
    </Card>
  );
}
