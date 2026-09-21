import type { components } from "@/api/schema";
import { cn } from "cn";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";

type Field = components["schemas"]["Field"];

interface Props {
  fields: Field[];
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
}

/**
 * A form generated from a declared schema.
 *
 * One renderer serves an item's own fields, a theme's settings and, later, a
 * plugin's. Writing a page per content type is the reason other systems make
 * adding a type a development task; reading the schema is what keeps it a
 * configuration change.
 */
export function SchemaForm({ fields, values, onChange }: Props) {
  const set = (key: string, value: unknown) => onChange({ ...values, [key]: value });

  return (
    <div className="space-y-3">
      {fields.map((field) =>
        visible(field, values) ? (
          <FieldRow
            key={field.key}
            field={field}
            value={values[field.key]}
            onChange={(v) => set(field.key, v)}
          />
        ) : null,
      )}
    </div>
  );
}

/** visible applies a field's showIf, so a form only asks what still applies. */
function visible(field: Field, values: Record<string, unknown>): boolean {
  if (!field.showIf) return true;
  return Object.entries(field.showIf).every(([key, want]) => values[key] === want);
}

function FieldRow({
  field,
  value,
  onChange,
}: {
  field: Field;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const label = field.label || field.key;

  // A switch reads better beside its label than under it, which is the one
  // field shape that does not fit the others.
  if (field.type === "boolean") {
    return (
      <div className="flex items-center justify-between gap-3 py-0.5">
        <div className="min-w-0">
          <Label htmlFor={field.key}>{label}</Label>
          {field.help && (
            <p className="text-xs text-muted-foreground">{field.help}</p>
          )}
        </div>
        <Switch
          id={field.key}
          checked={Boolean(value)}
          onCheckedChange={(checked) => onChange(checked)}
        />
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
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
          onChange={(e) => onChange(e.target.value)}
          className={cn(field.type === "code" && "font-mono text-xs")}
        />
      ) : field.type === "select" ? (
        <Select value={asString(value)} onValueChange={(v) => v && onChange(v)}>
          <SelectTrigger id={field.key} className="w-full">
            <SelectValue placeholder={field.placeholder ?? "Choose"} />
          </SelectTrigger>
          <SelectContent>
            {field.options?.map((o) => (
              <SelectItem key={o.value} value={o.value}>
                {o.label || o.value}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : field.type === "multiselect" ? (
        <div className="flex flex-wrap gap-1.5">
          {field.options?.map((o) => {
            const chosen = Array.isArray(value) && value.includes(o.value);
            return (
              <Button
                key={o.value}
                type="button"
                size="sm"
                variant={chosen ? "default" : "outline"}
                className="h-7 px-2 text-xs font-normal"
                onClick={() =>
                  onChange(
                    chosen
                      ? (value as string[]).filter((v) => v !== o.value)
                      : [...(Array.isArray(value) ? (value as string[]) : []), o.value],
                  )
                }
              >
                {o.label || o.value}
              </Button>
            );
          })}
        </div>
      ) : (
        <Input
          id={field.key}
          type={inputType(field.type)}
          value={asString(value)}
          placeholder={field.placeholder}
          min={field.min}
          max={field.max}
          onChange={(e) =>
            onChange(
              field.type === "number"
                ? e.target.value === ""
                  ? undefined
                  : Number(e.target.value)
                : e.target.value,
            )
          }
          className={cn(field.type === "color" && "h-9 w-16 p-1")}
        />
      )}

      {field.help && field.type !== "boolean" && (
        <p className="text-xs text-muted-foreground">{field.help}</p>
      )}
    </div>
  );
}

function inputType(type: string): string {
  switch (type) {
    case "date":
      return "datetime-local";
    case "color":
      return "color";
    case "number":
      return "number";
    case "url":
      return "url";
    default:
      return "text";
  }
}

function asString(value: unknown): string {
  if (value === undefined || value === null) return "";
  return String(value);
}
