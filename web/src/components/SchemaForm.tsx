import type { components } from "@/api/schema";
import { cn } from "@/lib/cn";

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
      {fields.map((field) => (
        <FieldRow
          key={field.key}
          field={field}
          value={values[field.key]}
          onChange={(v) => set(field.key, v)}
          hidden={!visible(field, values)}
        />
      ))}
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
  hidden,
}: {
  field: Field;
  value: unknown;
  onChange: (value: unknown) => void;
  hidden: boolean;
}) {
  if (hidden) return null;

  const label = field.label || field.key;
  const input = "w-full rounded-md border border-[var(--border)] bg-[var(--background)] px-2 py-1.5 text-sm outline-none focus:border-brand focus:ring-2 focus:ring-brand/20";

  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-[var(--muted-foreground)]">
        {label}
        {field.required && <span className="ml-0.5 text-red-500">*</span>}
      </span>

      {field.type === "text" || field.type === "code" ? (
        <textarea
          rows={field.type === "code" ? 6 : 3}
          value={asString(value)}
          placeholder={field.placeholder}
          onChange={(e) => onChange(e.target.value)}
          className={cn(input, field.type === "code" && "font-mono")}
        />
      ) : field.type === "boolean" ? (
        <input
          type="checkbox"
          checked={Boolean(value)}
          onChange={(e) => onChange(e.target.checked)}
          className="size-4 accent-[var(--color-brand)]"
        />
      ) : field.type === "select" ? (
        <select value={asString(value)} onChange={(e) => onChange(e.target.value)} className={input}>
          <option value="" />
          {field.options?.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label || o.value}
            </option>
          ))}
        </select>
      ) : field.type === "number" ? (
        <input
          type="number"
          value={asString(value)}
          min={field.min}
          max={field.max}
          onChange={(e) => onChange(e.target.value === "" ? undefined : Number(e.target.value))}
          className={input}
        />
      ) : field.type === "multiselect" ? (
        <div className="flex flex-wrap gap-1.5 py-1">
          {field.options?.map((o) => {
            const chosen = Array.isArray(value) && value.includes(o.value);
            return (
              <button
                key={o.value}
                type="button"
                onClick={() =>
                  onChange(
                    chosen
                      ? (value as string[]).filter((v) => v !== o.value)
                      : [...(Array.isArray(value) ? (value as string[]) : []), o.value],
                  )
                }
                className={cn(
                  "rounded-md border px-2 py-0.5 text-xs",
                  chosen
                    ? "border-brand bg-brand/10 text-brand"
                    : "border-[var(--border)] text-[var(--muted-foreground)]",
                )}
              >
                {o.label || o.value}
              </button>
            );
          })}
        </div>
      ) : (
        <input
          type={field.type === "date" ? "datetime-local" : field.type === "color" ? "color" : "text"}
          value={asString(value)}
          placeholder={field.placeholder}
          onChange={(e) => onChange(e.target.value)}
          className={input}
        />
      )}

      {field.help && (
        <span className="mt-1 block text-xs text-[var(--muted-foreground)]">{field.help}</span>
      )}
    </label>
  );
}

function asString(value: unknown): string {
  if (value === undefined || value === null) return "";
  return String(value);
}
