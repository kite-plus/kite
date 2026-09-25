import { useRef, useState, type ReactNode } from "react";
import {
  ArrowDown,
  ArrowUp,
  ChevronDown,
  ImagePlus,
  ImageUp,
  MoreHorizontal,
  Plus,
  RotateCcw,
  Trash2,
  X,
} from "lucide-react";

import type { Field } from "@/api/client";
import { locales, useI18n, type Key } from "@/i18n";
import { resolveLink } from "@/lib/links";
import { defaultOf, problemOf, sameValue, valueFields } from "@/lib/schema";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { CodeField } from "@/components/CodeField";
import { ColorField } from "@/components/ColorField";
import { DateTimePicker } from "@/components/DateTimePicker";

/** What an image field needs to take a file: somewhere to put it. */
export interface Uploads {
  /** upload stores a file and resolves to the link that reaches it. */
  upload: (file: File) => Promise<string>;
  /** base is the address links are relative to, for showing what was chosen. */
  base?: string;
  /** resolve turns a stored link into one the admin can load, when base is not enough. */
  resolve?: (link: string) => string;
}

interface Props {
  fields: Field[];
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
  uploads?: Uploads;
  /**
   * sections says how a section is drawn: a heading over its fields, or a
   * card of its own that folds away, for a form that is all sections.
   */
  sections?: "heading" | "panel";
  /** onReset puts a field back to its default; given, a field that differs offers it. */
  onReset?: (key: string) => void;
  /** idPrefix keeps ids apart when one form sits inside another. */
  idPrefix?: string;
}

/**
 * A form generated from a declared schema.
 *
 * One renderer serves an item's own fields, a theme's settings and, later, a
 * plugin's. Writing a page per content type is the reason other systems make
 * adding a type a development task; reading the schema is what keeps it a
 * configuration change.
 */
export function SchemaForm({ fields, values, onChange, uploads, sections = "heading", onReset, idPrefix = "" }: Props) {
  const set = (key: string, value: unknown) => onChange({ ...values, [key]: value });

  return (
    <div className={cn("grid [&>*]:min-w-0", sections === "panel" ? "gap-4" : "gap-6")}>
      {fields.map((field) => {
        if (!visible(field, values)) return null;
        if (field.type === "section") {
          return (
            <Section key={field.key} field={field} as={sections}>
              <SchemaForm
                fields={field.fields ?? []}
                values={values}
                onChange={onChange}
                uploads={uploads}
                sections={sections}
                onReset={onReset}
                idPrefix={idPrefix}
              />
            </Section>
          );
        }
        const value = values[field.key];
        return (
          <FieldRow
            key={field.key}
            id={idPrefix + field.key}
            field={field}
            value={value}
            onChange={(next) => set(field.key, next)}
            uploads={uploads}
            plain={sections === "panel"}
            onReset={onReset && !sameValue(value, defaultOf(field)) ? () => onReset(field.key) : undefined}
          />
        );
      })}
    </div>
  );
}

/** visible applies a field's showIf, so a form only asks what still applies. */
function visible(field: Field, values: Record<string, unknown>): boolean {
  if (!field.showIf) return true;
  return Object.entries(field.showIf).every(([key, want]) => values[key] === want);
}

function Section({ field, as, children }: { field: Field; as: "heading" | "panel"; children: ReactNode }) {
  const label = field.label || field.key;
  if (as === "heading") {
    return (
      <fieldset className="grid gap-4">
        <legend className="mb-4 w-full border-b pb-2">
          <span className="text-sm font-semibold">{label}</span>
          {field.help && <Help className="mt-0.5">{field.help}</Help>}
        </legend>
        {children}
      </fieldset>
    );
  }
  return (
    <Collapsible defaultOpen className="group/section rounded-lg border bg-card">
      <CollapsibleTrigger className="flex w-full items-center justify-between gap-2 rounded-lg px-4 py-3 text-start outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50">
        <span className="text-sm font-semibold">{label}</span>
        <ChevronDown className="size-4 text-muted-foreground transition-transform group-data-[state=closed]/section:-rotate-90" />
      </CollapsibleTrigger>
      <CollapsibleContent className="border-t px-4 pt-4 pb-5">
        {field.help && <Help className="-mt-1 mb-4">{field.help}</Help>}
        {children}
      </CollapsibleContent>
    </Collapsible>
  );
}

function Help({ children, className }: { children: ReactNode; className?: string }) {
  return <p className={cn("text-sm text-muted-foreground", className)}>{children}</p>;
}

function Problem({ children }: { children: ReactNode }) {
  return (
    <p role="alert" className="text-sm text-destructive">
      {children}
    </p>
  );
}

function ResetButton({ onReset }: { onReset: () => void }) {
  const { t } = useI18n();
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="-my-1 size-6 text-muted-foreground"
          aria-label={t("form.resetDefault")}
          onClick={onReset}
        >
          <RotateCcw className="size-3.5" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>{t("form.resetDefault")}</TooltipContent>
    </Tooltip>
  );
}

function FieldRow({
  id,
  field,
  value,
  onChange,
  uploads,
  plain,
  onReset,
}: {
  id: string;
  field: Field;
  value: unknown;
  onChange: (value: unknown) => void;
  uploads?: Uploads;
  /** plain drops the border a switch otherwise sits in, inside a card that has one. */
  plain?: boolean;
  onReset?: () => void;
}) {
  const { t } = useI18n();
  const label = builtinLabel(field, t) ?? (field.label || field.key);
  const problem = problemOf(field, value, t);

  // A switch reads better beside its label than under it.
  if (field.type === "boolean") {
    return (
      <div className={cn("flex items-center justify-between gap-4", !plain && "rounded-lg border p-4")}>
        <div className="space-y-1">
          <Label htmlFor={id}>{label}</Label>
          {field.help && <Help>{field.help}</Help>}
        </div>
        <div className="flex items-center gap-1">
          {onReset && <ResetButton onReset={onReset} />}
          <Switch id={id} checked={Boolean(value)} onCheckedChange={(checked) => onChange(checked)} />
        </div>
      </div>
    );
  }

  const chosen = Array.isArray(value) ? (value as string[]) : [];

  return (
    <div className="grid gap-2">
      <div className="flex min-h-5 items-center justify-between gap-2">
        <Label htmlFor={id}>
          {label}
          {field.required && <span className="text-destructive">*</span>}
        </Label>
        {onReset && <ResetButton onReset={onReset} />}
      </div>

      {field.type === "text" ? (
        <Textarea
          id={id}
          rows={3}
          value={asString(value)}
          placeholder={field.placeholder}
          aria-invalid={problem !== null}
          onChange={(event) => onChange(event.target.value)}
        />
      ) : field.type === "code" ? (
        <CodeField
          id={id}
          value={asString(value)}
          language={field.language}
          placeholder={field.placeholder}
          onChange={onChange}
        />
      ) : field.type === "select" ? (
        <Select value={asString(value)} onValueChange={(next) => next && onChange(next)}>
          <SelectTrigger id={id} className="w-full">
            <SelectValue placeholder={field.placeholder ?? t("form.choose")} />
          </SelectTrigger>
          <SelectContent>
            <SelectGroup>
              {field.options?.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {option.label || option.value}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      ) : field.type === "multiselect" ? (
        <div className="flex flex-wrap gap-x-5 gap-y-2" id={id}>
          {field.options?.map((option) => (
            <label key={option.value} className="flex items-center gap-2 text-sm">
              <Checkbox
                checked={chosen.includes(option.value)}
                onCheckedChange={(on) =>
                  onChange(
                    on ? [...chosen, option.value] : chosen.filter((each) => each !== option.value),
                  )
                }
              />
              {option.label || option.value}
            </label>
          ))}
        </div>
      ) : field.type === "color" ? (
        <ColorField
          id={id}
          value={asString(value)}
          onChange={onChange}
          presets={field.options}
          placeholder={field.placeholder}
        />
      ) : field.type === "image" && uploads ? (
        <ImageField id={id} value={asString(value)} onChange={onChange} uploads={uploads} compact />
      ) : field.type === "date" ? (
        <DateTimePicker
          id={id}
          value={fromLocal(asString(value))}
          onChange={(iso) => onChange(iso ? toLocal(iso) : undefined)}
          placeholder={field.placeholder ?? t("form.pickDate")}
        />
      ) : field.type === "group" ? (
        <div className="rounded-lg border p-4">
          <SchemaForm
            fields={field.fields ?? []}
            values={asRecord(value)}
            onChange={onChange}
            uploads={uploads}
            idPrefix={`${id}-`}
          />
        </div>
      ) : field.type === "repeat" ? (
        <RepeatField id={id} field={field} value={value} onChange={onChange} uploads={uploads} />
      ) : (
        <Input
          id={id}
          type={field.type === "number" ? "number" : "text"}
          inputMode={field.type === "url" ? "url" : undefined}
          spellCheck={field.type === "url" ? false : undefined}
          value={asString(value)}
          placeholder={field.placeholder}
          min={field.min}
          max={field.max}
          step={field.step}
          aria-invalid={problem !== null}
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
      )}

      {field.help && <Help>{field.help}</Help>}
      {problem && field.type !== "repeat" && field.type !== "group" && <Problem>{problem}</Problem>}
    </div>
  );
}

// Entries whose fields fit on one line are edited as rows of a table, with
// the labels once above them rather than once per entry.
const inline = new Set(["string", "url", "number", "select", "date"]);

/** RepeatField edits a list of entries, each a small form of its own. */
function RepeatField({
  id,
  field,
  value,
  onChange,
  uploads,
}: {
  id: string;
  field: Field;
  value: unknown;
  onChange: (value: unknown) => void;
  uploads?: Uploads;
}) {
  const { t } = useI18n();
  const entries = Array.isArray(value) ? (value as Record<string, unknown>[]) : [];
  const children = valueFields(field.fields);
  const rows = children.length > 0 && children.length <= 3 && children.every((child) => inline.has(child.type));

  const blank = () =>
    Object.fromEntries(
      children.map((child) => [child.key, defaultOf(child)] as const).filter(([, v]) => v !== undefined),
    );
  const move = (from: number, to: number) => {
    const next = [...entries];
    const [entry] = next.splice(from, 1);
    next.splice(to, 0, entry);
    onChange(next);
  };

  const actions = (i: number) => (
    <div className="flex shrink-0 items-center">
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon" className="size-8 text-muted-foreground" aria-label={t("form.entryMenu")}>
            <MoreHorizontal />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem disabled={i === 0} onSelect={() => move(i, i - 1)}>
            <ArrowUp />
            {t("form.moveUp")}
          </DropdownMenuItem>
          <DropdownMenuItem disabled={i === entries.length - 1} onSelect={() => move(i, i + 1)}>
            <ArrowDown />
            {t("form.moveDown")}
          </DropdownMenuItem>
          <DropdownMenuItem variant="destructive" onSelect={() => onChange(entries.filter((_, j) => j !== i))}>
            <Trash2 />
            {t("form.removeEntry")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );

  return (
    <div className="grid gap-2" id={id}>
      {entries.length === 0 ? (
        <p className="rounded-md border border-dashed px-3 py-3 text-center text-sm text-muted-foreground">
          {t("form.noEntries")}
        </p>
      ) : rows ? (
        <div className="grid gap-1.5">
          <div className="flex gap-2 pe-10 text-xs text-muted-foreground">
            {children.map((child) => (
              <span key={child.key} className="min-w-0 flex-1 truncate">
                {child.label || child.key}
              </span>
            ))}
          </div>
          {entries.map((entry, i) => {
            const problem = children
              .map((child) => problemOf(child, entry[child.key], t))
              .find((each) => each !== null);
            return (
              <div key={i} className="grid gap-1">
                <div className="flex items-center gap-2">
                  {children.map((child) => (
                    <Cell
                      key={child.key}
                      id={`${id}-${i}-${child.key}`}
                      field={child}
                      value={entry[child.key]}
                      onChange={(next) => onChange(replace(entries, i, { ...entry, [child.key]: next }))}
                    />
                  ))}
                  {actions(i)}
                </div>
                {problem && <Problem>{problem}</Problem>}
              </div>
            );
          })}
        </div>
      ) : (
        entries.map((entry, i) => (
          <div key={i} className="flex items-start gap-2 rounded-lg border p-3">
            <span className="mt-1 w-4 shrink-0 text-center text-xs text-muted-foreground tabular-nums">{i + 1}</span>
            <div className="min-w-0 flex-1">
              <SchemaForm
                fields={field.fields ?? []}
                values={entry}
                onChange={(next) => onChange(replace(entries, i, next))}
                uploads={uploads}
                idPrefix={`${id}-${i}-`}
              />
            </div>
            {actions(i)}
          </div>
        ))
      )}
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="justify-self-start"
        onClick={() => onChange([...entries, blank()])}
      >
        <Plus />
        {t("form.addEntry")}
      </Button>
    </div>
  );
}

/** Cell is one field of an entry drawn as a row, labelled by its column. */
function Cell({
  id,
  field,
  value,
  onChange,
}: {
  id: string;
  field: Field;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
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
      className="min-w-0 flex-1"
      type={field.type === "number" ? "number" : field.type === "date" ? "date" : "text"}
      spellCheck={field.type === "url" ? false : undefined}
      value={asString(value)}
      placeholder={field.placeholder}
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

/**
 * builtinLabel translates a field the built-in types declare. A project that
 * gave the field a label of its own keeps it: only the stock English one is
 * recognized.
 */
function builtinLabel(field: Field, t: (key: Key) => string): string | undefined {
  const key = `field.${field.key}`;
  const stock = (locales.en.catalog as Record<string, string>)[key];
  return stock !== undefined && stock === field.label ? t(key as Key) : undefined;
}

/** An image is chosen by handing over a file, not by typing where one is. */
export function ImageField({
  id,
  value,
  onChange,
  uploads,
  cover = false,
  compact = false,
}: {
  id: string;
  value: string;
  onChange: (value: unknown) => void;
  uploads: Uploads;
  cover?: boolean;
  /** compact shows a small picture beside its name, for an icon or a logo. */
  compact?: boolean;
}) {
  const { t } = useI18n();
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  const take = async (file?: File) => {
    if (!file) return;
    setBusy(true);
    setFailed(null);
    try {
      onChange(await uploads.upload(file));
    } catch (err) {
      setFailed(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const drop = {
    onDragOver: (event: React.DragEvent) => event.preventDefault(),
    onDrop: (event: React.DragEvent) => {
      event.preventDefault();
      void take(event.dataTransfer.files[0]);
    },
  };
  const shown = value ? (uploads.resolve?.(value) ?? resolveLink(value, uploads.base)) : "";

  return (
    <>
      <input
        ref={input}
        id={id}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={(event) => {
          void take(event.target.files?.[0]);
          event.target.value = "";
        }}
      />
      {compact ? (
        <div className="flex flex-wrap items-center gap-3 rounded-lg border p-2" {...drop}>
          <div className="kite-checker flex size-14 shrink-0 items-center justify-center overflow-hidden rounded-md border">
            {value ? (
              <img src={shown} alt="" className="size-full object-contain" />
            ) : (
              <ImageUp className="size-5 text-muted-foreground" />
            )}
          </div>
          <div className="min-w-0 flex-1">
            <p className="truncate font-mono text-xs text-muted-foreground">
              {value || t("form.noImage")}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => input.current?.click()}>
              {busy && <Spinner />}
              {value ? t("form.replaceImage") : t("form.pickImage")}
            </Button>
            {value && (
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="size-8"
                aria-label={t("form.removeImage")}
                onClick={() => onChange(undefined)}
              >
                <X />
              </Button>
            )}
          </div>
        </div>
      ) : value ? (
        <div className="relative overflow-hidden rounded-lg border">
          <img src={shown} alt="" className="aspect-video w-full bg-muted object-cover" />
          <div className="flex items-center gap-1 border-t px-2.5 py-1">
            <span className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">
              {value}
            </span>
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              aria-label={t("form.removeImage")}
              onClick={() => onChange(undefined)}
            >
              <X />
            </Button>
          </div>
        </div>
      ) : (
        <Button
          variant="outline"
          disabled={busy}
          className={cn(
            "w-full flex-col gap-1.5 border-dashed font-normal text-muted-foreground",
            cover ? "h-28 bg-muted/40" : "h-28",
          )}
          onClick={() => input.current?.click()}
          {...drop}
        >
          {busy ? <Spinner /> : cover ? <ImagePlus /> : <ImageUp />}
          <span className="text-xs">{t(cover ? "editor.coverPrompt" : "form.chooseImage")}</span>
        </Button>
      )}
      {failed && <Help className="text-destructive">{failed}</Help>}
    </>
  );
}

// A date field keeps the local "2026-09-01T10:00" it has always been written
// as, so a theme that reads one sees no change of format.
function toLocal(iso: string): string {
  const at = new Date(iso);
  return new Date(at.getTime() - at.getTimezoneOffset() * 60_000).toISOString().slice(0, 16);
}

function fromLocal(value: string): string | undefined {
  if (!value) return undefined;
  const at = new Date(value);
  return Number.isNaN(at.getTime()) ? undefined : at.toISOString();
}

function asString(value: unknown): string {
  if (value === undefined || value === null) return "";
  return String(value);
}

function replace<T>(list: T[], at: number, item: T): T[] {
  return list.map((each, i) => (i === at ? item : each));
}

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
}
