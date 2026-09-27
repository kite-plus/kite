import { useRef, useState, type ReactNode } from "react";
import { ImageUp, RotateCcw, X } from "lucide-react";

import type { Field } from "@/api/client";
import { locales, useI18n, type Key } from "@/i18n";
import { resolveLink } from "@/lib/links";
import { defaultOf, problemOf, sameValue } from "@/lib/schema";
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
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { CodeField } from "@/components/CodeField";
import { ColorField } from "@/components/ColorField";
import { DateTimePicker } from "@/components/DateTimePicker";
import { RepeatField } from "@/components/RepeatField";

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
export function SchemaForm({ fields, values, onChange, uploads, onReset, idPrefix = "" }: Props) {
  const set = (key: string, value: unknown) => onChange({ ...values, [key]: value });

  return (
    <div className="grid gap-6 [&>*]:min-w-0">
      {fields.map((field) => {
        if (!visible(field, values)) return null;
        if (field.type === "section") {
          return (
            <Section key={field.key} field={field}>
              <SchemaForm
                fields={field.fields ?? []}
                values={values}
                onChange={onChange}
                uploads={uploads}
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

function Section({ field, children }: { field: Field; children: ReactNode }) {
  return (
    <fieldset className="grid gap-4">
      <legend className="mb-4 w-full border-b pb-2">
        <span className="text-sm font-semibold">{field.label || field.key}</span>
        {field.help && <Help className="mt-0.5">{field.help}</Help>}
      </legend>
      {children}
    </fieldset>
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
  onReset,
}: {
  id: string;
  field: Field;
  value: unknown;
  onChange: (value: unknown) => void;
  uploads?: Uploads;
  onReset?: () => void;
}) {
  const { t } = useI18n();
  const label = fieldLabel(field, t);
  const problem = problemOf(field, value, t);

  // A switch reads better beside its label than under it.
  if (field.type === "boolean") {
    return (
      <div data-field={field.key} className="flex items-center justify-between gap-4 rounded-lg border p-4">
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
    <div data-field={field.key} className="grid gap-2">
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
        <ImageField id={id} value={asString(value)} onChange={onChange} uploads={uploads} />
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

/**
 * fieldLabel names a field. One the built-in types declare is translated; a
 * project that gave the field a label of its own keeps it, since only the
 * stock English one is recognized.
 */
export function fieldLabel(field: Field, t: (key: Key) => string): string {
  const key = `field.${field.key}`;
  const stock = (locales.en.catalog as Record<string, string>)[key];
  return stock !== undefined && stock === field.label ? t(key as Key) : field.label || field.key;
}

/**
 * An image is chosen by handing over a file, not by typing where one is, and
 * shows as a small picture beside its name.
 */
function ImageField({
  id,
  value,
  onChange,
  uploads,
}: {
  id: string;
  value: string;
  onChange: (value: unknown) => void;
  uploads: Uploads;
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
      <div
        className="flex flex-wrap items-center gap-3 rounded-lg border p-2"
        onDragOver={(event) => event.preventDefault()}
        onDrop={(event) => {
          event.preventDefault();
          void take(event.dataTransfer.files[0]);
        }}
      >
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

function asRecord(value: unknown): Record<string, unknown> {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {};
}
