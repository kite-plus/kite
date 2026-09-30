import { useEffect, useState } from "react";
import { ChevronDown, Clock, Trash2 } from "lucide-react";

import type { Draft } from "@/api/client";
import { useI18n, type Key } from "@/i18n";
import { canPublish, type DeliveryState, type usePublish } from "@/hooks/usePublish";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { DateTimePicker } from "@/components/DateTimePicker";
import { statuses } from "@/components/StatusLabel";
import { DeliveryStages, PublishProblems } from "@/components/publish/Delivery";

// In the order an item usually goes through them.
const ORDER = ["draft", "published", "scheduled", "archived"];

/**
 * PublishMenu is the item's standing, beside the button that publishes it:
 * its status and date, how far the site has got with publishing, and last,
 * the way to the trash. A refused publish opens it, since the reasons and
 * their fixes are here.
 */
export function PublishMenu({
  draft,
  onEdit,
  delivery,
  publish,
  onDelete,
  disabled,
}: {
  draft: Draft;
  onEdit: (patch: Partial<Draft>) => void;
  delivery?: DeliveryState;
  publish: ReturnType<typeof usePublish>;
  /** onDelete is absent until the item is saved. */
  onDelete?: () => void;
  /** disabled shows the status without offering to change it. */
  disabled?: boolean;
}) {
  const { t, date } = useI18n();
  const [open, setOpen] = useState(false);
  useEffect(() => {
    if (publish.failure) setOpen(true);
  }, [publish.failure]);
  const troubled = Boolean(publish.failure || publish.plan?.problems?.length);
  const { icon: Icon, className: tone } = statuses[draft.status] ?? statuses.draft;

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="relative h-8 gap-1.5 px-2.5" disabled={disabled}>
          <span className="sr-only">{t("editor.publishing")}</span>
          {/* On a phone the status shows by its icon and color alone. */}
          <Icon className={cn("size-3.5", tone)} />
          <span className={cn("text-[13px] max-sm:sr-only", tone)}>{t(`status.${draft.status}` as Key)}</span>
          <ChevronDown className="-ms-0.5 size-3.5 text-muted-foreground" />
          {troubled && <span className="absolute -end-1 -top-1 size-2.5 rounded-full border-2 border-background bg-destructive" />}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="grid w-80 gap-4">
        <div className="grid gap-2">
          <Label id="status-label">{t("list.status")}</Label>
          <div role="group" aria-labelledby="status-label" className="grid grid-cols-4 gap-0.5 rounded-lg bg-muted p-[3px]">
            {ORDER.map((status) => {
              const on = status === draft.status;
              return (
                <button
                  key={status}
                  type="button"
                  aria-pressed={on}
                  className={cn(
                    "h-7 truncate rounded-md px-1 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground",
                    on && ["bg-background shadow-sm", statuses[status].className],
                  )}
                  onClick={() =>
                    // Publishing is when the date is taken, unless one is set already.
                    onEdit(
                      status === "published" && !draft.published_at
                        ? { status, published_at: new Date().toISOString() }
                        : { status },
                    )
                  }
                >
                  {t(`status.${status}` as Key)}
                </button>
              );
            })}
          </div>
        </div>

        <div className="grid gap-2">
          <Label htmlFor="published_at">{t("editor.publishedAt")}</Label>
          <DateTimePicker
            id="published_at"
            value={draft.published_at}
            onChange={(published_at) => onEdit({ published_at })}
            placeholder={t("editor.pickPublishDate")}
          />
          {waitsForDate(draft) && (
            <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Clock className="size-3 shrink-0" />
              {t("editor.waitsForDate", { date: date(draft.published_at, "long") })}
            </p>
          )}
          {draft.status === "published" && !draft.published_at && (
            <p className="flex flex-wrap items-center gap-x-1.5 text-xs text-muted-foreground">
              {t("editor.noPublishDate")}
              <button
                type="button"
                className="font-medium text-primary hover:underline"
                onClick={() => onEdit({ published_at: new Date().toISOString() })}
              >
                {t("editor.useNow")}
              </button>
            </p>
          )}
        </div>

        <div className="empty:hidden">
          <PublishProblems publish={publish} />
        </div>

        {canPublish(delivery) && (
          <>
            <Separator />
            <div className="grid gap-3">
              <Label>{t("publish.delivery")}</Label>
              <DeliveryStages delivery={delivery} publish={publish} />
            </div>
          </>
        )}

        {onDelete && (
          <>
            <Separator />
            <Button
              variant="ghost"
              size="sm"
              className="-mx-2 -my-1 justify-start text-destructive hover:bg-destructive/10 hover:text-destructive"
              onClick={() => {
                setOpen(false);
                onDelete();
              }}
            >
              <Trash2 />
              {t("editor.delete")}
            </Button>
          </>
        )}
      </PopoverContent>
    </Popover>
  );
}

/** waitsForDate reports whether the site holds the item back until its date. */
function waitsForDate(draft: Draft): boolean {
  if (draft.status !== "published" && draft.status !== "scheduled") return false;
  return !!draft.published_at && new Date(draft.published_at).getTime() > Date.now();
}
