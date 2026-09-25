import { useEffect, useState } from "react";
import { HexColorPicker } from "react-colorful";
import { Check, ChevronDown } from "lucide-react";

import { useI18n } from "@/i18n";
import { isColor } from "@/lib/schema";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

interface Props {
  id: string;
  value: string;
  onChange: (value: string) => void;
  /** presets are colors the theme suggests, each with a name. */
  presets?: { value: string; label: string }[];
  placeholder?: string;
}

/**
 * ColorField picks a color: one of the theme's suggestions with a click, or
 * any other by eye or by its hex code, since a color is as often pasted from
 * a brand sheet as it is chosen.
 */
export function ColorField({ id, value, onChange, presets = [], placeholder }: Props) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  const [typed, setTyped] = useState(value);
  const valid = isColor(value);
  const custom = valid && !presets.some((preset) => same(preset.value, value));

  // The text follows the picker and the swatches, not the other way round
  // while someone is typing a half-finished code.
  useEffect(() => setTyped(value), [value]);

  return (
    <div className="flex flex-wrap items-center gap-2">
      {presets.map((preset) => {
        const on = same(preset.value, value);
        return (
          <button
            key={preset.value}
            type="button"
            aria-pressed={on}
            onClick={() => onChange(preset.value)}
            className={cn(
              "flex h-8 items-center gap-2 rounded-md border px-2.5 text-sm outline-none transition-colors hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50",
              on && "border-primary bg-primary/5",
            )}
          >
            <span className="size-4 rounded-sm ring-1 ring-black/10" style={{ background: preset.value }} />
            {preset.label || preset.value}
            {on && <Check className="-me-0.5 size-3.5 text-primary" />}
          </button>
        );
      })}

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            id={id}
            type="button"
            className={cn(
              "flex h-8 items-center gap-2 rounded-md border px-2.5 text-sm outline-none transition-colors hover:bg-accent focus-visible:ring-[3px] focus-visible:ring-ring/50",
              (custom || presets.length === 0) && "border-primary bg-primary/5",
              presets.length === 0 && "w-full",
            )}
          >
            <span
              className="kite-checker size-4 rounded-sm ring-1 ring-black/10"
              style={valid ? { background: value } : undefined}
            />
            <span className={cn("font-mono text-xs", !valid && "text-muted-foreground")}>
              {valid ? value : (placeholder ?? t("form.customColor"))}
            </span>
            <ChevronDown className="ms-auto size-3.5 opacity-50" />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" className="kite-color-picker w-60 p-3">
          <HexColorPicker color={valid ? value : "#000000"} onChange={onChange} />
          <Input
            aria-label={t("form.hexCode")}
            value={typed}
            spellCheck={false}
            className="mt-3 h-8 font-mono text-xs"
            onChange={(event) => {
              const next = event.target.value.trim();
              setTyped(next);
              const code = next.startsWith("#") ? next : `#${next}`;
              if (isColor(code)) onChange(code.toLowerCase());
              else if (next === "") onChange("");
            }}
          />
        </PopoverContent>
      </Popover>
    </div>
  );
}

function same(a: string, b: string) {
  return a.toLowerCase() === b.toLowerCase();
}
