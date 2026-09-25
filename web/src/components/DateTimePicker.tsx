import { useEffect, useRef, useState } from "react";
import { CalendarIcon } from "lucide-react";
import { enUS, zhCN } from "react-day-picker/locale";

import { useI18n } from "@/i18n";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

const pad = (n: number) => String(n).padStart(2, "0");
const hours = Array.from({ length: 24 }, (_, i) => i);
const minutes = Array.from({ length: 60 }, (_, i) => i);

/**
 * DateTimePicker picks a moment: a day from the calendar and the hour and
 * minute beside it, in the browser's own time zone. The value is an ISO
 * instant, or undefined for none.
 */
export function DateTimePicker({
  id,
  value,
  onChange,
  placeholder,
}: {
  id?: string;
  value: string | undefined;
  onChange: (value: string | undefined) => void;
  placeholder: string;
}) {
  const { t, locale, date } = useI18n();
  const [open, setOpen] = useState(false);
  const at = value ? new Date(value) : undefined;

  const set = (next: Date) => {
    next.setSeconds(0, 0);
    onChange(next.toISOString());
  };
  // A day keeps the time already chosen; with none yet it takes the current one.
  const pickDay = (day: Date | undefined) => {
    if (!day) return;
    const base = at ?? new Date();
    const next = new Date(day);
    next.setHours(base.getHours(), base.getMinutes());
    set(next);
  };
  const pickTime = (hour: number, minute: number) => {
    const next = at ? new Date(at) : new Date();
    next.setHours(hour, minute);
    set(next);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          id={id}
          variant="outline"
          className={cn("w-full justify-start font-normal", !at && "text-muted-foreground")}
        >
          <CalendarIcon />
          {at ? date(value, "long") : placeholder}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="end">
        <div className="flex">
          <Calendar
            mode="single"
            selected={at}
            defaultMonth={at}
            onSelect={pickDay}
            locale={locale === "zh-CN" ? zhCN : enUS}
          />
          {/* The columns take the calendar's height and scroll within it. */}
          <div className="relative w-28 border-s">
            <div className="absolute inset-0 flex">
              <TimeColumn
                label={t("date.hour")}
                values={hours}
                selected={at?.getHours()}
                onPick={(hour) => pickTime(hour, at?.getMinutes() ?? new Date().getMinutes())}
              />
              <TimeColumn
                label={t("date.minute")}
                values={minutes}
                selected={at?.getMinutes()}
                onPick={(minute) => pickTime(at?.getHours() ?? new Date().getHours(), minute)}
              />
            </div>
          </div>
        </div>
        <div className="flex items-center justify-between border-t p-2">
          <Button type="button" size="sm" variant="ghost" onClick={() => set(new Date())}>
            {t("date.now")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={() => {
              onChange(undefined);
              setOpen(false);
            }}
          >
            {t("date.clear")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

/** TimeColumn is a scrolling list of hours or minutes, opened at the chosen one. */
function TimeColumn({
  label,
  values,
  selected,
  onPick,
}: {
  label: string;
  values: number[];
  selected: number | undefined;
  onPick: (value: number) => void;
}) {
  const list = useRef<HTMLDivElement>(null);
  useEffect(() => {
    list.current?.querySelector("[aria-selected=true]")?.scrollIntoView({ block: "center" });
    // Only as the popover opens: a pick made inside the list is already in view.
  }, []);

  return (
    <div ref={list} role="listbox" aria-label={label} className="flex-1 overflow-y-auto p-1">
      {values.map((value) => (
        <button
          key={value}
          type="button"
          role="option"
          aria-selected={value === selected}
          onClick={() => onPick(value)}
          className="flex h-8 w-full items-center justify-center rounded-md text-sm tabular-nums outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring/50 aria-selected:bg-primary aria-selected:text-primary-foreground"
        >
          {pad(value)}
        </button>
      ))}
    </div>
  );
}
