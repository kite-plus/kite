import type { ReactNode } from "react";
import { Clock, Trash2 } from "lucide-react";
import { cn } from "cn";

import type { ContentType, Draft } from "@/api/client";
import { useI18n, type Key } from "@/i18n";
import { STATUSES } from "@/hooks/useContents";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { canPublish, type DeliveryState, type usePublish } from "@/hooks/usePublish";

import { ImageField, SchemaForm, type Uploads } from "@/components/SchemaForm";
import { Soon } from "@/components/Soon";
import { TermsInput } from "@/components/editor/TermsInput";
import { IconChevronDown } from "@/components/icons";
import { DeliveryStages, PublishProblems } from "@/components/publish/Delivery";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface Props {
  draft: Draft;
  type?: ContentType;
  onEdit: (patch: Partial<Draft>) => void;
  delivery?: DeliveryState;
  publish: ReturnType<typeof usePublish>;
  /** uploads is absent until the item is saved and has a folder of its own. */
  uploads?: Uploads;
  onDelete?: () => void;
}

// The design's controls in this panel: 30px high, 12px text, 170px beside a label.
const field =
  "h-[30px] rounded-[7px] border border-input bg-background px-2.5 text-xs outline-none transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30";

/** Everything about an item that is not its text. */
export function EditorAside({ draft, type, onEdit, delivery, publish, uploads, onDelete }: Props) {
  const { t, date } = useI18n();
  const taxonomyLabel = useTaxonomyLabel();

  // Categories lead, as they do in the listing, and read as a dropdown.
  const taxonomies = [...(type?.taxonomies ?? [])].sort(
    (a, b) => Number(b === "categories") - Number(a === "categories"),
  );

  // "/posts/:slug" reads as "/posts/" in front of the field.
  const prefix = type?.route.includes(":slug") ? type.route.split(":slug")[0] : "/";

  // The built-in summary and cover have places of their own in the design;
  // whatever else the type declares follows them.
  const fields = type?.fields ?? [];
  const summary = fields.find((each) => each.key === "description" && each.type === "text");
  const cover = fields.find((each) => each.key === "cover" && each.type === "image");
  const others = fields.filter((each) => each !== summary && each !== cover);
  const meta = draft.meta ?? {};
  const setMeta = (key: string, value: unknown) => onEdit({ meta: { ...meta, [key]: value } });

  return (
    <div className="flex flex-col gap-5 px-4 pt-[18px] pb-[26px]">
      <Section title={t("publish.title")}>
        <div className="flex flex-col gap-2">
          <Row label={t("list.status")} htmlFor="status">
            <Select value={draft.status} onValueChange={(status) => status && onEdit({ status })}>
              <SelectTrigger
                id="status"
                className={cn(field, "w-[170px] gap-1.5 py-0 data-[size=default]:h-[30px] [&_svg]:size-3")}
              >
                {/* The stored value is English; a person reads their own language. */}
                <SelectValue>{(status: string) => t(`status.${status}` as Key)}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {STATUSES.map((status) => (
                    <SelectItem key={status} value={status}>
                      {t(`status.${status}` as Key)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Row>

          <Row label={t("editor.visibility")}>
            {/* Everything Kite publishes is public; there is nothing to choose yet. */}
            <Soon side="left">
              <div
                tabIndex={0}
                aria-disabled
                className={cn(field, "flex w-[170px] cursor-default items-center justify-between text-subtle")}
              >
                {t("editor.public")}
                <IconChevronDown className="size-3" />
              </div>
            </Soon>
          </Row>

          <Row label={t("editor.publishedAt")} htmlFor="published_at">
            <input
              id="published_at"
              type="datetime-local"
              className={cn(field, "w-[170px]")}
              value={toLocalInput(draft.published_at)}
              onChange={(event) =>
                onEdit({
                  published_at: event.target.value
                    ? new Date(event.target.value).toISOString()
                    : undefined,
                })
              }
            />
          </Row>
          {waitsForDate(draft) && (
            <p className="flex items-center justify-end gap-1.5 text-[11.5px] text-subtle">
              <Clock className="size-3 shrink-0" />
              {t("editor.waitsForDate", { date: date(draft.published_at, "long") })}
            </p>
          )}
        </div>
        <div className="mt-3 empty:hidden">
          <PublishProblems publish={publish} />
        </div>
      </Section>

      {taxonomies.length > 0 && (
        <Section title={t("editor.terms")}>
          <div className="flex flex-col gap-2">
            {taxonomies.map((taxonomy, i) => (
              <TermsInput
                key={taxonomy}
                id={`terms-${taxonomy}`}
                label={taxonomyLabel(taxonomy)}
                taxonomy={taxonomy}
                variant={i === 0 && taxonomies.length > 1 ? "select" : "chips"}
                placeholder={
                  i === 0 && taxonomies.length > 1
                    ? t("editor.chooseTerm")
                    : taxonomy === "tags"
                      ? t("editor.addTag")
                      : t("editor.addTerm")
                }
                value={draft.taxonomies?.[taxonomy] ?? []}
                onChange={(terms) =>
                  onEdit({ taxonomies: { ...draft.taxonomies, [taxonomy]: terms } })
                }
              />
            ))}
          </div>
        </Section>
      )}

      <Section title={t("editor.details")}>
        <div className="flex flex-col gap-2">
          <div className="flex h-[30px] items-center overflow-hidden rounded-[7px] border border-input bg-background transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 dark:bg-input/30">
            <span className="shrink-0 border-r bg-surface px-2 text-[11px] leading-7 whitespace-nowrap text-subtle">
              {prefix}
            </span>
            <input
              id="slug"
              aria-label={t("editor.slug")}
              className="min-w-0 flex-1 bg-transparent px-2 font-mono text-xs outline-none placeholder:font-sans placeholder:text-subtle"
              value={draft.slug ?? ""}
              onChange={(event) => onEdit({ slug: event.target.value })}
              placeholder={t("editor.slugPlaceholder")}
            />
          </div>
          {summary && (
            <textarea
              id={summary.key}
              rows={3}
              aria-label={t("field.description")}
              placeholder={summary.placeholder ?? t("field.description")}
              value={typeof meta[summary.key] === "string" ? (meta[summary.key] as string) : ""}
              onChange={(event) => setMeta(summary.key, event.target.value)}
              className="w-full resize-y rounded-[7px] border border-input bg-background px-2.5 py-2 text-xs leading-[1.6] outline-none transition-colors placeholder:text-subtle focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
            />
          )}
        </div>
      </Section>

      {cover && uploads && (
        <Section title={t("field.cover")}>
          <ImageField
            id={cover.key}
            cover
            value={typeof meta[cover.key] === "string" ? (meta[cover.key] as string) : ""}
            onChange={(value) => setMeta(cover.key, value)}
            uploads={uploads}
          />
        </Section>
      )}

      {canPublish(delivery) && (
        <Section title={t("publish.delivery")}>
          <DeliveryStages delivery={delivery} publish={publish} />
        </Section>
      )}

      {others.length > 0 && (
        <Section title={t("editor.more")}>
          <SchemaForm
            fields={others}
            values={meta}
            onChange={(next) => onEdit({ meta: next })}
            uploads={uploads}
          />
        </Section>
      )}

      {onDelete && (
        <Button variant="destructive" onClick={onDelete}>
          <Trash2 data-icon="inline-start" />
          {t("editor.delete")}
        </Button>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h3 className="mb-2.5 text-[12.5px] font-semibold">{title}</h3>
      {children}
    </section>
  );
}

function Row({ label, htmlFor, children }: { label: string; htmlFor?: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-2.5">
      <label htmlFor={htmlFor} className="shrink-0 text-xs text-muted-foreground">
        {label}
      </label>
      {children}
    </div>
  );
}

/** waitsForDate reports whether the site holds the item back until its date. */
function waitsForDate(draft: Draft): boolean {
  if (draft.status !== "published" && draft.status !== "scheduled") return false;
  return !!draft.published_at && new Date(draft.published_at).getTime() > Date.now();
}

/** toLocalInput formats an instant for a datetime-local control. */
function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const at = new Date(iso);
  const local = new Date(at.getTime() - at.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
