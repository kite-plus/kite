import { useRef, useState, type ReactNode } from "react";
import { ImagePlus, ImageUp, X } from "lucide-react";

import type { components } from "@/api/schema";
import { locales, useI18n, type Key } from "@/i18n";
import { resolveLink } from "@/lib/links";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
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
import { DateTimePicker } from "@/components/DateTimePicker";

/** SchemaField is one declared field; Field is the control it is drawn in. */
type SchemaField = components["schemas"]["Field"];

/** What an image field needs to take a file: somewhere to put it. */
export interface Uploads {
  /** upload stores a file and resolves to the link that reaches it. */
  upload: (file: File) => Promise<string>;
  /** base is the address links are relative to, for showing what was chosen. */
  base?: string;
}

interface Props {
  fields: SchemaField[];
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
  uploads?: Uploads;
}

/**
 * A form generated from a declared schema.
 *
 * One renderer serves an item's own fields, a theme's settings and, later, a
 * plugin's. Writing a page per content type is the reason other systems make
 * adding a type a development task; reading the schema is what keeps it a
 * configuration change.
 */
export function SchemaForm({ fields, values, onChange, uploads }: Props) {
  const set = (key: string, value: unknown) => onChange({ ...values, [key]: value });

  return (
    <div className="grid gap-6">
      {fields.map((field) =>
        visible(field, values) ? (
          <FieldRow
            key={field.key}
            field={field}
            value={values[field.key]}
            onChange={(value) => set(field.key, value)}
            uploads={uploads}
          />
        ) : null,
      )}
    </div>
  );
}

/** visible applies a field's showIf, so a form only asks what still applies. */
function visible(field: SchemaField, values: Record<string, unknown>): boolean {
  if (!field.showIf) return true;
  return Object.entries(field.showIf).every(([key, want]) => values[key] === want);
}

function Help({ children, className }: { children: ReactNode; className?: string }) {
  return <p className={cn("text-sm text-muted-foreground", className)}>{children}</p>;
}

function FieldRow({
  field,
  value,
  onChange,
  uploads,
}: {
  field: SchemaField;
  value: unknown;
  onChange: (value: unknown) => void;
  uploads?: Uploads;
}) {
  const { t } = useI18n();
  const label = builtinLabel(field, t) ?? (field.label || field.key);

  // A switch reads better beside its label than under it.
  if (field.type === "boolean") {
    return (
      <div className="flex items-center justify-between gap-4 rounded-lg border p-4">
        <div className="space-y-1">
          <Label htmlFor={field.key}>{label}</Label>
          {field.help && <Help>{field.help}</Help>}
        </div>
        <Switch
          id={field.key}
          checked={Boolean(value)}
          onCheckedChange={(checked) => onChange(checked)}
        />
      </div>
    );
  }

  const chosen = Array.isArray(value) ? (value as string[]) : [];

  return (
    <div className="grid gap-2">
      <Label htmlFor={field.key}>
        {label}
        {field.required && <span className="text-destructive">*</span>}
      </Label>

      {field.type === "text" || field.type === "code" ? (
        <Textarea
          id={field.key}
          rows={field.type === "code" ? 6 : 3}
          value={asString(value)}
          placeholder={field.placeholder}
          onChange={(event) => onChange(event.target.value)}
          className={cn(field.type === "code" && "font-mono text-xs")}
        />
      ) : field.type === "select" ? (
        <Select value={asString(value)} onValueChange={(next) => next && onChange(next)}>
          <SelectTrigger id={field.key} className="w-full">
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
        <div className="flex flex-wrap gap-x-5 gap-y-2" id={field.key}>
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
        // The swatch picks and the text says, since a colour is as often
        // pasted from a brand sheet as it is chosen by eye.
        <div className="flex h-9 items-center gap-2 rounded-md border border-input px-3 shadow-xs focus-within:ring-[3px] focus-within:ring-ring/50">
          <input
            type="color"
            aria-label={label}
            value={/^#[0-9a-f]{6}$/i.test(asString(value)) ? asString(value) : "#000000"}
            onChange={(event) => onChange(event.target.value)}
            className="size-5 cursor-pointer appearance-none rounded-sm border-0 bg-transparent p-0 [&::-webkit-color-swatch]:rounded-sm [&::-webkit-color-swatch]:border [&::-webkit-color-swatch]:border-border [&::-webkit-color-swatch-wrapper]:p-0"
          />
          <input
            id={field.key}
            value={asString(value)}
            placeholder={field.placeholder ?? "#000000"}
            onChange={(event) => onChange(event.target.value)}
            className="min-w-0 flex-1 bg-transparent font-mono text-sm outline-none placeholder:text-muted-foreground"
          />
        </div>
      ) : field.type === "image" && uploads ? (
        <ImageField id={field.key} value={asString(value)} onChange={onChange} uploads={uploads} />
      ) : field.type === "date" ? (
        <DateTimePicker
          id={field.key}
          value={fromLocal(asString(value))}
          onChange={(iso) => onChange(iso ? toLocal(iso) : undefined)}
          placeholder={field.placeholder ?? t("form.pickDate")}
        />
      ) : (
        <Input
          id={field.key}
          type={inputType(field.type)}
          value={asString(value)}
          placeholder={field.placeholder}
          min={field.min}
          max={field.max}
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
    </div>
  );
}

/**
 * builtinLabel translates a field the built-in types declare. A project that
 * gave the field a label of its own keeps it: only the stock English one is
 * recognized.
 */
function builtinLabel(field: SchemaField, t: (key: Key) => string): string | undefined {
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
}: {
  id: string;
  value: string;
  onChange: (value: unknown) => void;
  uploads: Uploads;
  cover?: boolean;
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
      {value ? (
        <div className="relative overflow-hidden rounded-lg border">
          <img
            src={resolveLink(value, uploads.base)}
            alt=""
            className="aspect-video w-full bg-muted object-cover"
          />
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

function inputType(type: string): string {
  switch (type) {
    case "number":
      return "number";
    case "url":
      return "url";
    default:
      return "text";
  }
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
