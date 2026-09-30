import { useRef, useState, type ReactNode } from "react";
import { ArrowDown, ArrowUp, ChevronDown, GripVertical, MoreHorizontal, Plus, Trash2 } from "lucide-react";

import type { Field } from "@/api/client";
import { useI18n } from "@/i18n";
import { defaultOf, problemOf, valueFields } from "@/lib/schema";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { SchemaForm, type Uploads } from "@/components/SchemaForm";

type Entry = Record<string, unknown>;

// Fields that fit on one line, so entries made of them are edited as rows of
// a table, with the labels once above them rather than once per entry.
const inline = new Set(["string", "url", "number", "select", "date"]);

/**
 * How a list is edited, from what its entries hold: rows for a few short
 * fields; a tree for entries that are a title over a list of their own, as a
 * docs tree's groups of pages; cards for anything larger.
 */
type Shape = { kind: "rows" } | { kind: "tree"; nested: Field } | { kind: "cards" };

function shapeOf(children: Field[]): Shape {
  const short = (fields: Field[]) => fields.length > 0 && fields.length <= 3 && fields.every((f) => inline.has(f.type));
  if (short(children)) return { kind: "rows" };
  const lists = children.filter((child) => child.type === "repeat");
  const rest = children.filter((child) => child.type !== "repeat");
  if (
    lists.length === 1 &&
    short(valueFields(lists[0].fields)) &&
    rest.length <= 2 &&
    rest.every((child) => inline.has(child.type))
  ) {
    return { kind: "tree", nested: lists[0] };
  }
  return { kind: "cards" };
}

interface Props {
  id: string;
  field: Field;
  value: unknown;
  onChange: (value: unknown) => void;
  uploads?: Uploads;
}

/** RepeatField edits a list of entries, each a small form of its own. */
export function RepeatField({ id, field, value, onChange, uploads }: Props) {
  const entries = asEntries(value);
  const children = valueFields(field.fields);
  const shape = shapeOf(children);
  if (shape.kind === "tree") {
    return <TreeList id={id} fields={children} nested={shape.nested} entries={entries} onChange={onChange} />;
  }
  return (
    <FlatList
      id={id}
      fields={field.fields ?? []}
      rows={shape.kind === "rows"}
      entries={entries}
      onChange={onChange}
      uploads={uploads}
    />
  );
}

/** A list of rows or cards, in one level. */
function FlatList({
  id,
  fields,
  rows,
  entries,
  onChange,
  uploads,
}: {
  id: string;
  fields: Field[];
  rows: boolean;
  entries: Entry[];
  onChange: (value: unknown) => void;
  uploads?: Uploads;
}) {
  const { t } = useI18n();
  const children = valueFields(fields);
  const sort = useSortable(id, (from, to) => onChange(moved(entries, from.index, to.index)));
  const refs = useRef<(HTMLElement | null)[]>([]);

  return (
    <div className="grid gap-2" id={id}>
      {entries.length === 0 ? (
        <Empty />
      ) : (
        <div className="grid gap-1.5" {...sort.list()}>
          {rows && (
            <div className="flex gap-2 ps-8 pe-10 text-xs text-muted-foreground">
              {children.map((child) => (
                <span key={child.key} className="min-w-0 flex-1 truncate">
                  {child.label || child.key}
                </span>
              ))}
            </div>
          )}
          {entries.map((entry, i) => {
            const spot = { list: -1, index: i };
            const actions = (
              <EntryMenu
                first={i === 0}
                last={i === entries.length - 1}
                onMove={(step) => onChange(moved(entries, i, i + step + (step > 0 ? 1 : 0)))}
                onRemove={() => onChange(entries.filter((_, j) => j !== i))}
              />
            );
            return (
              <div
                key={i}
                ref={(element) => {
                  refs.current[i] = element;
                }}
                className="relative"
                {...sort.target(spot)}
              >
                <DropLine at={sort.line} spot={spot} />
                {rows ? (
                  <div className="grid gap-1">
                    <div className="flex items-center gap-2">
                      <Handle {...sort.handle(spot, () => refs.current[i])} />
                      <Cells id={`${id}-${i}`} fields={children} entry={entry} onChange={(next) => onChange(replaced(entries, i, next))} />
                      {actions}
                    </div>
                    <EntryProblem fields={children} entry={entry} />
                  </div>
                ) : (
                  <div className="flex items-start gap-2 rounded-lg border bg-card p-3">
                    <div className="mt-0.5 flex shrink-0 flex-col items-center gap-1">
                      <Handle {...sort.handle(spot, () => refs.current[i])} />
                      <span className="w-4 text-center text-xs text-muted-foreground tabular-nums">{i + 1}</span>
                    </div>
                    <div className="min-w-0 flex-1">
                      <SchemaForm
                        fields={fields}
                        values={entry}
                        onChange={(next) => onChange(replaced(entries, i, next))}
                        uploads={uploads}
                        idPrefix={`${id}-${i}-`}
                      />
                    </div>
                    {actions}
                  </div>
                )}
                {i === entries.length - 1 && <DropLine at={sort.line} spot={{ list: -1, index: entries.length }} end />}
              </div>
            );
          })}
        </div>
      )}
      <AddButton onClick={() => onChange([...entries, blank(children)])}>{t("form.addEntry")}</AddButton>
    </div>
  );
}

/**
 * A list whose entries each hold a list of their own, drawn as a tree: each
 * entry a row that folds, its own list indented under it. Rows are dragged
 * by their handles, and an entry of the inner lists can be dragged from one
 * entry into another.
 */
function TreeList({
  id,
  fields,
  nested,
  entries,
  onChange,
}: {
  id: string;
  fields: Field[];
  nested: Field;
  entries: Entry[];
  onChange: (value: unknown) => void;
}) {
  const { t } = useI18n();
  const own = fields.filter((field) => field.type !== "repeat");
  const inner = valueFields(nested.fields);
  const [closed, setClosed] = useState<boolean[]>([]);
  const groupRefs = useRef<(HTMLElement | null)[]>([]);
  const itemRefs = useRef<Record<string, HTMLElement | null>>({});

  const itemsOf = (entry: Entry) => asEntries(entry[nested.key]);
  const withItems = (entry: Entry, items: Entry[]) => ({ ...entry, [nested.key]: items });

  const sort = useSortable(id, (from, to) => {
    if (from.list === -1) {
      onChange(moved(entries, from.index, to.index));
      setClosed((current) => moved(padded(current, entries.length), from.index, to.index));
      return;
    }
    const item = itemsOf(entries[from.list])[from.index];
    const next = entries.map((entry, g) => {
      let items = itemsOf(entry);
      if (g === from.list) items = items.filter((_, i) => i !== from.index);
      if (g === to.list) {
        const at = from.list === to.list && to.index > from.index ? to.index - 1 : to.index;
        items = [...items.slice(0, at), item, ...items.slice(at)];
      }
      return g === from.list || g === to.list ? withItems(entry, items) : entry;
    });
    onChange(next);
    if (closed[to.list]) setClosed((current) => current.map((each, g) => (g === to.list ? false : each)));
  });

  const summary = (items: Entry[]) => {
    const label = inner[0]?.key;
    return items
      .map((item) => (label ? String(item[label] ?? "") : ""))
      .filter(Boolean)
      .join(t("form.listJoin"));
  };

  return (
    <div className="grid gap-2" id={id}>
      {entries.length === 0 ? (
        <Empty />
      ) : (
        <div className="grid gap-2" {...sort.list()}>
          {entries.map((entry, g) => {
            const items = itemsOf(entry);
            const open = !closed[g];
            const spot = { list: -1, index: g };
            const setItems = (next: Entry[]) => onChange(replaced(entries, g, withItems(entry, next)));
            return (
              <div
                key={g}
                ref={(element) => {
                  groupRefs.current[g] = element;
                }}
                className="relative"
                {...sort.target(spot, { list: g, index: items.length })}
              >
                <DropLine at={sort.line} spot={spot} />
                <div
                  className={cn(
                    "rounded-lg border bg-card",
                    sort.line?.list === g && sort.line.index === items.length && (!open || items.length === 0) && "ring-2 ring-primary/40",
                  )}
                >
                  <div className={cn("flex items-center gap-2 p-1.5 ps-2", open && items.length > 0 && "border-b")}>
                    <Handle {...sort.handle(spot, () => groupRefs.current[g])} />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="size-7 shrink-0 text-muted-foreground"
                      aria-expanded={open}
                      aria-label={nested.label || nested.key}
                      onClick={() => setClosed((current) => padded(current, entries.length).map((each, j) => (j === g ? !each : each)))}
                    >
                      <ChevronDown className={cn("transition-transform", !open && "-rotate-90")} />
                    </Button>
                    <Cells
                      id={`${id}-${g}`}
                      fields={own}
                      entry={entry}
                      onChange={(next) => onChange(replaced(entries, g, next))}
                      className="max-w-72"
                    />
                    {/* Only a folded entry sums up its list; an open one gives the room to its fields. */}
                    {!open && (
                      <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground">{summary(items)}</span>
                    )}
                    <EntryMenu
                      first={g === 0}
                      last={g === entries.length - 1}
                      onMove={(step) => {
                        const to = g + step + (step > 0 ? 1 : 0);
                        onChange(moved(entries, g, to));
                        setClosed((current) => moved(padded(current, entries.length), g, to));
                      }}
                      onRemove={() => {
                        onChange(entries.filter((_, j) => j !== g));
                        setClosed((current) => padded(current, entries.length).filter((_, j) => j !== g));
                      }}
                    />
                  </div>
                  {open && (
                    <div className="grid gap-1.5 p-2 ps-4 sm:ps-11">
                      {items.map((item, i) => {
                        const at = { list: g, index: i };
                        const key = `${g}-${i}`;
                        return (
                          <div
                            key={i}
                            ref={(element) => {
                              itemRefs.current[key] = element;
                            }}
                            className="relative grid gap-1"
                            {...sort.target(at)}
                          >
                            <DropLine at={sort.line} spot={at} />
                            <div className="flex items-center gap-2">
                              <Handle {...sort.handle(at, () => itemRefs.current[key])} />
                              <Cells
                                id={`${id}-${g}-${nested.key}-${i}`}
                                fields={inner}
                                entry={item}
                                onChange={(next) => setItems(replaced(items, i, next))}
                              />
                              <EntryMenu
                                first={i === 0}
                                last={i === items.length - 1}
                                onMove={(step) => setItems(moved(items, i, i + step + (step > 0 ? 1 : 0)))}
                                onRemove={() => setItems(items.filter((_, j) => j !== i))}
                              />
                            </div>
                            <EntryProblem fields={inner} entry={item} />
                            {i === items.length - 1 && <DropLine at={sort.line} spot={{ list: g, index: items.length }} end />}
                          </div>
                        );
                      })}
                      <AddButton onClick={() => setItems([...items, blank(inner)])}>
                        {nested.label ? t("form.addTo", { list: nested.label }) : t("form.addEntry")}
                      </AddButton>
                    </div>
                  )}
                </div>
                {g === entries.length - 1 && <DropLine at={sort.line} spot={{ list: -1, index: entries.length }} end />}
              </div>
            );
          })}
        </div>
      )}
      <AddButton onClick={() => onChange([...entries, blank(fields)])}>{t("form.addEntry")}</AddButton>
    </div>
  );
}

/** Cells are the fields of one entry, side by side, each labelled by its column. */
function Cells({
  id,
  fields,
  entry,
  onChange,
  className,
}: {
  id: string;
  fields: Field[];
  entry: Entry;
  onChange: (entry: Entry) => void;
  className?: string;
}) {
  return (
    <div className={cn("flex min-w-0 flex-1 items-center gap-2", className)}>
      {fields.map((field) => (
        <Cell
          key={field.key}
          id={`${id}-${field.key}`}
          field={field}
          value={entry[field.key]}
          onChange={(next) => onChange({ ...entry, [field.key]: next })}
        />
      ))}
    </div>
  );
}

/** Cell is one field of an entry drawn as a row, labelled by its column. */
function Cell({ id, field, value, onChange }: { id: string; field: Field; value: unknown; onChange: (value: unknown) => void }) {
  const { t } = useI18n();
  const label = field.label || field.key;
  if (field.type === "select") {
    return (
      <Select value={asString(value)} onValueChange={(next) => next && onChange(next)}>
        <SelectTrigger id={id} aria-label={label} className="min-w-0 flex-1">
          <SelectValue placeholder={field.placeholder ?? t("form.choose")} />
        </SelectTrigger>
        <SelectContent>
          {field.options?.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label || option.value}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    );
  }
  return (
    <Input
      id={id}
      aria-label={label}
      className={cn("h-8 min-w-0 flex-1", field.type === "url" && "font-mono text-xs")}
      type={field.type === "number" ? "number" : field.type === "date" ? "date" : "text"}
      spellCheck={field.type === "url" ? false : undefined}
      value={asString(value)}
      placeholder={field.placeholder ?? label}
      aria-invalid={problemOf(field, value, t) !== null}
      onChange={(event) =>
        onChange(
          field.type === "number"
            ? event.target.value === ""
              ? undefined
              : Number(event.target.value)
            : event.target.value,
        )
      }
    />
  );
}

function EntryProblem({ fields, entry }: { fields: Field[]; entry: Entry }) {
  const { t } = useI18n();
  const problem = fields.map((field) => problemOf(field, entry[field.key], t)).find((each) => each !== null);
  return problem ? (
    <p role="alert" className="ps-8 text-sm text-destructive">
      {problem}
    </p>
  ) : null;
}

function EntryMenu({
  first,
  last,
  onMove,
  onRemove,
}: {
  first: boolean;
  last: boolean;
  onMove: (step: -1 | 1) => void;
  onRemove: () => void;
}) {
  const { t } = useI18n();
  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8 shrink-0 text-muted-foreground" aria-label={t("form.entryMenu")}>
          <MoreHorizontal />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem disabled={first} onSelect={() => onMove(-1)}>
          <ArrowUp />
          {t("form.moveUp")}
        </DropdownMenuItem>
        <DropdownMenuItem disabled={last} onSelect={() => onMove(1)}>
          <ArrowDown />
          {t("form.moveDown")}
        </DropdownMenuItem>
        <DropdownMenuItem variant="destructive" onSelect={onRemove}>
          <Trash2 />
          {t("form.removeEntry")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function Handle(props: React.HTMLAttributes<HTMLSpanElement> & { draggable: boolean }) {
  const { t } = useI18n();
  return (
    <span
      {...props}
      role="img"
      aria-label={t("form.dragToSort")}
      title={t("form.dragToSort")}
      className="flex size-6 shrink-0 cursor-grab items-center justify-center rounded text-muted-foreground/60 hover:bg-muted hover:text-muted-foreground active:cursor-grabbing"
    >
      <GripVertical className="size-4" />
    </span>
  );
}

function AddButton({ onClick, children }: { onClick: () => void; children: ReactNode }) {
  return (
    <Button type="button" variant="outline" size="sm" className="justify-self-start" onClick={onClick}>
      <Plus />
      {children}
    </Button>
  );
}

function Empty() {
  const { t } = useI18n();
  return (
    <p className="rounded-md border border-dashed px-3 py-3 text-center text-sm text-muted-foreground">
      {t("form.noEntries")}
    </p>
  );
}

/** A place in a list: the outer list is -1, an entry's own list its index. */
interface Spot {
  list: number;
  index: number;
}

function DropLine({ at, spot, end = false }: { at: Spot | null; spot: Spot; end?: boolean }) {
  if (!at || at.list !== spot.list || at.index !== spot.index) return null;
  return (
    <div
      aria-hidden
      className={cn("pointer-events-none absolute inset-x-0 z-10 h-0.5 rounded-full bg-primary", end ? "-bottom-1" : "-top-1")}
    />
  );
}

// The row being dragged. Only one can be, and the lists of the same field
// read it to decide whether they take it.
let dragging: ({ scope: string } & Spot) | null = null;

/**
 * useSortable lets the rows of a field's lists be dragged by a handle and
 * dropped between other rows. A row of the outer list lands among the outer
 * rows, a row of an inner list in any inner list. Where it would land is
 * shown as a line, and onMove is told the move once it is dropped.
 */
function useSortable(scope: string, onMove: (from: Spot, to: Spot) => void) {
  const [line, setLine] = useState<Spot | null>(null);
  const current = useRef<Spot | null>(null);
  const show = (spot: Spot | null) => {
    current.current = spot;
    setLine(spot);
  };

  const handle = (spot: Spot, row: () => HTMLElement | null) => ({
    draggable: true,
    onDragStart: (event: React.DragEvent) => {
      dragging = { scope, ...spot };
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("text/plain", "");
      const element = row();
      if (element) {
        const rect = element.getBoundingClientRect();
        event.dataTransfer.setDragImage(element, event.clientX - rect.left, event.clientY - rect.top);
      }
    },
    onDragEnd: () => {
      dragging = null;
      show(null);
    },
  });

  /**
   * target makes a row one to drop on, before or after it by which half the
   * pointer is over. inside, for an outer row, is where a dragged inner row
   * lands on it: at the end of its own list.
   */
  const target = (spot: Spot, inside?: Spot) => ({
    onDragOver: (event: React.DragEvent<HTMLElement>) => {
      if (!dragging || dragging.scope !== scope) return;
      const outer = dragging.list === -1;
      if (outer !== (spot.list === -1)) {
        if (outer || !inside) return;
        event.preventDefault();
        event.stopPropagation();
        show(inside);
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      event.dataTransfer.dropEffect = "move";
      const rect = event.currentTarget.getBoundingClientRect();
      const after = event.clientY > rect.top + rect.height / 2;
      show({ list: spot.list, index: spot.index + (after ? 1 : 0) });
    },
    onDrop: (event: React.DragEvent) => {
      const to = current.current;
      if (!dragging || dragging.scope !== scope || !to) return;
      event.preventDefault();
      event.stopPropagation();
      const from = { list: dragging.list, index: dragging.index };
      dragging = null;
      show(null);
      const same = from.list === to.list && (to.index === from.index || to.index === from.index + 1);
      if (!same) onMove(from, to);
    },
  });

  // Leaving the lists altogether takes the line away.
  const list = () => ({
    onDragLeave: (event: React.DragEvent<HTMLElement>) => {
      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) show(null);
    },
  });

  return { line, handle, target, list };
}

/** moved takes the entry at from and puts it before the one at to. */
function moved<T>(list: T[], from: number, to: number): T[] {
  if (from < 0 || from >= list.length) return list;
  const next = [...list];
  const [item] = next.splice(from, 1);
  next.splice(to > from ? to - 1 : to, 0, item);
  return next;
}

function replaced<T>(list: T[], at: number, item: T): T[] {
  return list.map((each, i) => (i === at ? item : each));
}

function padded(flags: boolean[], length: number): boolean[] {
  return Array.from({ length }, (_, i) => flags[i] ?? false);
}

function blank(fields: Field[]): Entry {
  return Object.fromEntries(
    valueFields(fields)
      .map((field) => [field.key, defaultOf(field)] as const)
      .filter(([, value]) => value !== undefined),
  );
}

function asEntries(value: unknown): Entry[] {
  return Array.isArray(value) ? (value as Entry[]) : [];
}

function asString(value: unknown): string {
  if (value === undefined || value === null) return "";
  return String(value);
}
