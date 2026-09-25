import type { ReactNode } from "react";
import { Clock, Trash2 } from "lucide-react";

import type { components, ContentType, Draft } from "@/api/client";
import { locales, useI18n, type Key } from "@/i18n";
import { STATUSES } from "@/hooks/useContents";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { canPublish, type DeliveryState, type usePublish } from "@/hooks/usePublish";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";
import { ImageField, SchemaForm, type Uploads } from "@/components/SchemaForm";
import { DateTimePicker } from "@/components/DateTimePicker";
import { TermsInput } from "@/components/editor/TermsInput";
import { DeliveryStages, PublishProblems } from "@/components/publish/Delivery";

type LayoutOption = components["schemas"]["LayoutOption"];

// Radix gives an empty value no item, so the default template takes a name
// no layout can have.
const DEFAULT_LAYOUT = "@default";

/**
 * layoutText is a layout's label and note as the operator reads them. The
 * built-in theme's are translated when they are its stock English words; a
 * theme's own are shown as it wrote them.
 */
function layoutText(layout: LayoutOption, t: (key: Key) => string) {
  const stock = locales.en.catalog as Record<string, string>;
  const translated = (key: string, text: string | undefined) =>
    text !== undefined && stock[key] === text ? t(key as Key) : text;
  return {
    label: translated(`layout.${layout.name}`, layout.label) ?? layout.name,
    note: translated(`layoutNote.${layout.name}`, layout.description),
  };
}

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

/** Everything about an item that is not its text. */
export function EditorAside({ draft, type, onEdit, delivery, publish, uploads, onDelete }: Props) {
  const { t, date } = useI18n();
  const taxonomyLabel = useTaxonomyLabel();

  // Categories lead, as they do in the listing.
  const taxonomies = [...(type?.taxonomies ?? [])].sort(
    (a, b) => Number(b === "categories") - Number(a === "categories"),
  );

  // "/posts/:slug" reads as "/posts/" in front of the field.
  const prefix = type?.route.includes(":slug") ? type.route.split(":slug")[0] : "/";

  // The built-in summary and cover have places of their own; whatever else
  // the type declares follows them.
  const fields = type?.fields ?? [];
  const summary = fields.find((each) => each.key === "description" && each.type === "text");
  const cover = fields.find((each) => each.key === "cover" && each.type === "image");
  const others = fields.filter((each) => each !== summary && each !== cover);
  const meta = draft.meta ?? {};
  const setMeta = (key: string, value: unknown) => onEdit({ meta: { ...meta, [key]: value } });

  // The templates the theme offers this kind. One the item names that the
  // theme does not offer is still listed, so it can be seen and undone.
  const layouts = type?.layouts ?? [];
  const chosen = typeof meta.layout === "string" && meta.layout ? meta.layout : undefined;
  const offered = layouts.find((each) => each.name === chosen);

  return (
    <div className="flex flex-col gap-5 p-4">
      <Section title={t("publish.title")}>
        <div className="grid gap-2">
          <Label htmlFor="status">{t("list.status")}</Label>
          <Select
            value={draft.status}
            onValueChange={(status) =>
              // Publishing is when the date is taken, unless one is set already.
              onEdit(
                status === "published" && !draft.published_at
                  ? { status, published_at: new Date().toISOString() }
                  : { status },
              )
            }
          >
            <SelectTrigger id="status" className="w-full">
              <SelectValue />
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
      </Section>

      {(layouts.length > 0 || chosen) && (
        <Section title={t("editor.template")}>
          <div className="grid gap-2">
            <Select
              value={chosen ?? DEFAULT_LAYOUT}
              // A cleared template goes out as null, which takes it out of the file.
              onValueChange={(value) => setMeta("layout", value === DEFAULT_LAYOUT ? null : value)}
            >
              <SelectTrigger className="w-full" aria-label={t("editor.template")}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  <SelectItem value={DEFAULT_LAYOUT}>{t("editor.templateDefault")}</SelectItem>
                  {layouts.map((layout) => (
                    <SelectItem key={layout.name} value={layout.name}>
                      {layoutText(layout, t).label}
                    </SelectItem>
                  ))}
                  {chosen && !offered && <SelectItem value={chosen}>{chosen}</SelectItem>}
                </SelectGroup>
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {chosen && !offered
                ? t("editor.templateMissing", { name: chosen })
                : offered
                  ? layoutText(offered, t).note
                  : t("editor.templateDefaultNote")}
            </p>
          </div>
        </Section>
      )}

      {taxonomies.length > 0 && (
        <Section title={t("editor.terms")}>
          {taxonomies.map((taxonomy) => (
            <div key={taxonomy} className="grid gap-2">
              <Label htmlFor={`terms-${taxonomy}`}>{taxonomyLabel(taxonomy)}</Label>
              <TermsInput
                id={`terms-${taxonomy}`}
                label={taxonomyLabel(taxonomy)}
                taxonomy={taxonomy}
                placeholder={taxonomy === "tags" ? t("editor.addTag") : t("editor.addTerm")}
                value={draft.taxonomies?.[taxonomy] ?? []}
                onChange={(terms) => onEdit({ taxonomies: { ...draft.taxonomies, [taxonomy]: terms } })}
              />
            </div>
          ))}
        </Section>
      )}

      <Section title={t("editor.details")}>
        <div className="grid gap-2">
          <Label htmlFor="slug">{t("editor.slug")}</Label>
          <div className="flex h-9 items-center overflow-hidden rounded-md border border-input shadow-xs focus-within:ring-[3px] focus-within:ring-ring/50">
            <span className="shrink-0 border-e bg-muted px-2 text-xs leading-9 text-muted-foreground">
              {prefix}
            </span>
            <input
              id="slug"
              className="min-w-0 flex-1 bg-transparent px-2 font-mono text-sm outline-none placeholder:font-sans placeholder:text-muted-foreground"
              value={draft.slug ?? ""}
              onChange={(event) => onEdit({ slug: event.target.value })}
              placeholder={t("editor.slugPlaceholder")}
            />
          </div>
        </div>
        {summary && (
          <div className="grid gap-2">
            <Label htmlFor={summary.key}>{t("field.description")}</Label>
            <Textarea
              id={summary.key}
              rows={3}
              placeholder={summary.placeholder}
              value={typeof meta[summary.key] === "string" ? (meta[summary.key] as string) : ""}
              onChange={(event) => setMeta(summary.key, event.target.value)}
            />
          </div>
        )}
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
        <>
          <Separator />
          <Button variant="destructive" onClick={onDelete}>
            <Trash2 />
            {t("editor.delete")}
          </Button>
        </>
      )}
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="grid gap-3">
      <h3 className="text-sm font-semibold">{title}</h3>
      {children}
    </section>
  );
}

/** waitsForDate reports whether the site holds the item back until its date. */
function waitsForDate(draft: Draft): boolean {
  if (draft.status !== "published" && draft.status !== "scheduled") return false;
  return !!draft.published_at && new Date(draft.published_at).getTime() > Date.now();
}

