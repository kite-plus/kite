import { useRef, useState, type DragEvent, type Ref } from "react";
import {
  Check,
  ChevronDown,
  Circle,
  CircleCheck,
  Folder,
  ImageOff,
  ImagePlus,
  LayoutTemplate,
  Link2,
  Pin,
  Plus,
  SlidersHorizontal,
  Tag,
  Tags,
  X,
} from "lucide-react";

import type { components, ContentType, Draft, Field } from "@/api/client";
import { useI18n } from "@/i18n";
import { useTerms } from "@/hooks/useContents";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { composing } from "@/lib/ime";
import { resolveLink } from "@/lib/links";
import { termSlug } from "@/lib/terms";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Label } from "@/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Spinner } from "@/components/ui/spinner";
import { fieldLabel, SchemaForm, visible, type Uploads } from "@/components/SchemaForm";

type LayoutOption = components["schemas"]["LayoutOption"];

// The default template has no name of its own, so it takes one no layout
// can have.
const DEFAULT_LAYOUT = "@default";

/**
 * fieldsOf sorts a kind's fields by where the editor shows them: the summary
 * under the title, the cover above it, and the rest among the properties.
 */
export function fieldsOf(type?: ContentType) {
  const fields = type?.fields ?? [];
  const summary = fields.find((each) => each.key === "description" && each.type === "text");
  const cover = fields.find((each) => each.key === "cover" && each.type === "image");
  const others: Field[] = fields.filter((each) => each !== summary && each !== cover);
  return { summary, cover, others };
}

/**
 * CoverField is the item's cover, above its title as a post shows it. With
 * none, quiet buttons offer one where it would go, or none at all: false,
 * which tells a theme not to show a picture from the text in its place.
 */
export function CoverField({
  value,
  onChange,
  upload,
  base,
  home,
}: {
  value: string | false;
  /** onChange hears null for a cover taken away, which takes it out of the file. */
  onChange: (value: string | false | null) => void;
  upload: (file: File) => Promise<string>;
  base?: string;
  home?: string;
}) {
  const { t } = useI18n();
  const input = useRef<HTMLInputElement>(null);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);

  const take = async (file?: File) => {
    if (!file?.type.startsWith("image/")) return;
    setBusy(true);
    setFailed(null);
    try {
      onChange(await upload(file));
    } catch (err) {
      setFailed(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };
  const drop = {
    onDragOver: (event: DragEvent) => event.preventDefault(),
    onDrop: (event: DragEvent) => {
      event.preventDefault();
      void take(event.dataTransfer.files[0]);
    },
  };
  const picker = (
    <input
      ref={input}
      type="file"
      accept="image/*"
      className="hidden"
      tabIndex={-1}
      onChange={(event) => {
        void take(event.target.files?.[0]);
        event.target.value = "";
      }}
    />
  );
  const problem = failed && <p className="mt-1 text-xs text-destructive">{failed}</p>;

  if (value) {
    return (
      <div className="mb-6" {...drop}>
        {picker}
        <div className="group/cover relative overflow-hidden rounded-lg border bg-muted">
          <img src={resolveLink(value, base, home)} alt="" className="aspect-[5/2] w-full object-cover" />
          {/* Always there on a screen without hover, and for the keyboard. */}
          <div className="absolute end-2 top-2 flex gap-1 opacity-0 transition-opacity group-focus-within/cover:opacity-100 group-hover/cover:opacity-100 [@media(hover:none)]:opacity-100">
            <Button
              variant="secondary"
              size="sm"
              className="h-7 shadow-sm"
              disabled={busy}
              onClick={() => input.current?.click()}
            >
              {busy ? <Spinner /> : <ImagePlus />}
              {t("editor.replaceCover")}
            </Button>
            <Button
              variant="secondary"
              size="icon"
              className="size-7 shadow-sm"
              aria-label={t("editor.removeCover")}
              onClick={() => onChange(null)}
            >
              <X />
            </Button>
          </div>
        </div>
        {problem}
      </div>
    );
  }

  if (value === false) {
    return (
      <div className="mb-1 flex items-center gap-1 text-sm text-muted-foreground">
        <ImageOff className="size-4 shrink-0" />
        <span>{t("editor.coverSkipped")}</span>
        <Button variant="ghost" size="sm" className="h-7 px-2 font-normal" onClick={() => onChange(null)}>
          {t("editor.coverSkippedUndo")}
        </Button>
      </div>
    );
  }

  const quiet = cn(
    "h-7 px-2 font-normal text-muted-foreground opacity-0 transition-opacity group-focus-within/head:opacity-100 group-hover/head:opacity-100 focus-visible:opacity-100 [@media(hover:none)]:opacity-100",
    (busy || failed) && "opacity-100",
  );
  return (
    <div className="mb-1" {...drop}>
      {picker}
      <div className="-ms-2 flex items-center gap-1">
        <Button variant="ghost" size="sm" disabled={busy} className={quiet} onClick={() => input.current?.click()}>
          {busy ? <Spinner /> : <ImagePlus />}
          {t(busy ? "editor.coverUploading" : "editor.addCover")}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          disabled={busy}
          className={quiet}
          title={t("editor.skipCoverHint")}
          onClick={() => onChange(false)}
        >
          <ImageOff />
          {t("editor.skipCover")}
        </Button>
      </div>
      {problem}
    </div>
  );
}

/**
 * SummaryField is the item's summary under its title, read like a subtitle.
 * It is one paragraph, so Enter goes on to the text, as the title's does.
 */
export function SummaryField({
  ref,
  value,
  onChange,
  onExitDown,
  onExitUp,
}: {
  ref?: Ref<HTMLTextAreaElement>;
  value: string;
  onChange: (value: string) => void;
  onExitDown: () => void;
  onExitUp: () => void;
}) {
  const { t } = useI18n();
  return (
    <textarea
      ref={ref}
      rows={1}
      value={value}
      onChange={(event) => onChange(event.target.value.replace(/\n/g, " "))}
      onKeyDown={(event) => {
        if (composing(event) || event.shiftKey) return;
        const field = event.currentTarget;
        const at = (offset: number) => field.selectionStart === offset && field.selectionEnd === offset;
        if (event.key === "Enter" || (event.key === "ArrowDown" && at(field.value.length))) {
          event.preventDefault();
          onExitDown();
        } else if (event.key === "ArrowUp" && at(0)) {
          event.preventDefault();
          onExitUp();
        }
      }}
      placeholder={t("editor.summaryPlaceholder")}
      aria-label={t("field.description")}
      className="kite-summary"
    />
  );
}

// A property reads as a small outlined button; one not set yet is dashed,
// and a switch that is on takes the accent.
const chip =
  "h-7 max-w-full min-w-0 gap-1.5 px-2.5 text-[13px] font-normal shadow-none has-[>svg]:px-2.5 [&>svg]:text-muted-foreground";
const unset = "border-dashed text-muted-foreground";
const on =
  "border-primary/30 bg-primary/10 text-primary hover:bg-primary/15 hover:text-primary dark:bg-primary/15 [&>svg]:text-primary";

/**
 * Properties are the settings of an item a reader meets on its page, set
 * where they are read: its terms, its address, the template that draws it,
 * and whatever else its kind declares, such as pinning a post.
 */
export function Properties({
  draft,
  type,
  fields,
  uploads,
  onEdit,
}: {
  draft: Draft;
  type?: ContentType;
  /** fields are the kind's own, less the summary and cover placed apart. */
  fields: Field[];
  uploads: Uploads;
  onEdit: (patch: Partial<Draft>) => void;
}) {
  // Categories lead, as they do in the listing.
  const taxonomies = [...(type?.taxonomies ?? [])].sort(
    (a, b) => Number(b === "categories") - Number(a === "categories"),
  );
  // "/posts/:slug" reads as "/posts/" in front of the slug.
  const prefix = type?.route.includes(":slug") ? type.route.split(":slug")[0] : "/";
  const meta = draft.meta ?? {};
  const layouts = type?.layouts ?? [];
  const chosen = typeof meta.layout === "string" && meta.layout ? meta.layout : undefined;

  return (
    <div className="mb-6 flex flex-wrap items-center gap-1.5 border-b pb-5">
      {taxonomies.map((taxonomy) => (
        <TermsChip
          key={taxonomy}
          taxonomy={taxonomy}
          value={draft.taxonomies?.[taxonomy] ?? []}
          onChange={(terms) => onEdit({ taxonomies: { ...draft.taxonomies, [taxonomy]: terms } })}
        />
      ))}
      <SlugChip prefix={prefix} value={draft.slug ?? ""} onChange={(slug) => onEdit({ slug })} />
      {(layouts.length > 0 || chosen) && (
        <TemplateChip
          layouts={layouts}
          chosen={chosen}
          // A cleared template goes out as null, which takes it out of the file.
          onChange={(layout) => onEdit({ meta: { ...meta, layout } })}
        />
      )}
      {fields
        .filter((field) => visible(field, meta))
        .map((field) =>
          field.type === "boolean" ? (
            <SwitchChip
              key={field.key}
              field={field}
              value={Boolean(meta[field.key] ?? field.default)}
              onChange={(value) => onEdit({ meta: { ...meta, [field.key]: value } })}
            />
          ) : (
            <FieldChip
              key={field.key}
              field={field}
              values={meta}
              onChange={(next) => onEdit({ meta: next })}
              uploads={uploads}
            />
          ),
        )}
    </div>
  );
}

/** SwitchChip turns a yes-or-no setting, such as pinning, on and off in one click. */
function SwitchChip({
  field,
  value,
  onChange,
}: {
  field: Field;
  value: boolean;
  onChange: (value: boolean) => void;
}) {
  const { t } = useI18n();
  const Icon = field.key === "pinned" ? Pin : value ? CircleCheck : Circle;
  return (
    <Button
      variant="outline"
      size="sm"
      aria-pressed={value}
      className={cn(chip, value ? on : unset)}
      onClick={() => onChange(!value)}
    >
      <Icon className={cn(value && field.key === "pinned" && "fill-current")} />
      {fieldLabel(field, t)}
    </Button>
  );
}

/** FieldChip holds any other setting the kind declares, set in a popover. */
function FieldChip({
  field,
  values,
  onChange,
  uploads,
}: {
  field: Field;
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
  uploads: Uploads;
}) {
  const { t } = useI18n();
  const label = fieldLabel(field, t);
  const value = values[field.key];
  const set = value !== undefined && value !== null && value !== "";
  // A word or a number reads in the chip; anything larger only shows it is set.
  const shown = typeof value === "string" || typeof value === "number" ? String(value) : "";

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className={cn(chip, !set && unset)}>
          <SlidersHorizontal />
          <span className="truncate">{set && shown ? t("editor.fieldValue", { label, value: shown }) : label}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80" align="start">
        <SchemaForm fields={[field]} values={values} onChange={onChange} uploads={uploads} />
      </PopoverContent>
    </Popover>
  );
}

/**
 * TermsChip holds the terms an item carries in one taxonomy. Terms the
 * project already uses are offered, since a tag typed slightly differently
 * is a second tag; anything else typed becomes a new one. A term is matched
 * by its slug, as the site does: one written another way is the term the
 * list already has, not a new one.
 */
function TermsChip({
  taxonomy,
  value,
  onChange,
}: {
  taxonomy: string;
  value: string[];
  onChange: (value: string[]) => void;
}) {
  const { t } = useI18n();
  const label = useTaxonomyLabel()(taxonomy);
  const terms = useTerms(taxonomy);
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  // The terms held when the list opened lead it. They stay where they are
  // while it is open, so a click never moves another term under the pointer.
  const [held, setHeld] = useState<string[]>([]);

  const known = terms.data?.items ?? [];
  const count = new Map(known.map((item) => [termSlug(item.term), item.count]));
  const heldSlugs = new Set(held.map(termSlug));
  const listed = [
    ...held,
    ...value.filter((term) => !held.includes(term) && !count.has(termSlug(term))),
    ...known.map((item) => item.term).filter((term) => !heldSlugs.has(termSlug(term))),
  ];
  const typed = query.trim();
  // The listed term the typed text writes another way, if any. Its row takes
  // the typed text as its value, so the filter keeps it in sight.
  const same = typed === "" ? undefined : listed.find((term) => termSlug(term) === termSlug(typed));
  const fresh = typed !== "" && same === undefined && termSlug(typed) !== "";
  const Icon = taxonomy === "tags" ? Tags : taxonomy === "categories" ? Folder : Tag;

  const toggle = (term: string) => {
    onChange(value.includes(term) ? value.filter((each) => each !== term) : [...value, term]);
    setQuery("");
  };

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (next) setHeld(value);
        else setQuery("");
      }}
    >
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className={cn(chip, value.length === 0 && unset)}>
          <Icon />
          <span className="sr-only">{label}</span>
          <span className="truncate">
            {value.length > 0
              ? value.join(t("form.listJoin"))
              : taxonomy === "tags"
                ? t("editor.addTags")
                : taxonomy === "categories"
                  ? t("editor.addCategories")
                  : t("editor.addTerms", { what: label })}
          </span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-64 p-0" align="start">
        <Command>
          <CommandInput value={query} onValueChange={setQuery} placeholder={t("editor.findTerm")} />
          <CommandList>
            <CommandEmpty>{t("palette.empty")}</CommandEmpty>
            <CommandGroup>
              {fresh && (
                <CommandItem value={`create:${typed}`} onSelect={() => toggle(typed)}>
                  <Plus />
                  {t("editor.createTerm", { term: typed })}
                </CommandItem>
              )}
              {listed.map((term) => (
                <CommandItem
                  key={term}
                  value={term === same ? `create:${typed}` : term}
                  onSelect={() => toggle(term)}
                >
                  <Check className={cn("text-primary", !value.includes(term) && "invisible")} />
                  <span className="flex-1 truncate">{term}</span>
                  <span className="text-xs text-muted-foreground tabular-nums">{count.get(termSlug(term))}</span>
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

/** SlugChip is the item's address; the slug is the part after the prefix. */
function SlugChip({
  prefix,
  value,
  onChange,
}: {
  prefix: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const { t } = useI18n();
  const [open, setOpen] = useState(false);
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className={chip}>
          <Link2 />
          <span className="sr-only">{t("editor.slug")}</span>
          <span className="truncate font-mono text-xs">
            {prefix}
            {value || <span className="font-sans text-muted-foreground">{t("editor.slugFromTitle")}</span>}
          </span>
        </Button>
      </PopoverTrigger>
      <PopoverContent className="grid w-80 gap-2" align="start">
        <Label htmlFor="slug">{t("editor.slug")}</Label>
        <div className="flex h-9 items-center overflow-hidden rounded-md border border-input shadow-xs focus-within:ring-[3px] focus-within:ring-ring/50">
          <span className="shrink-0 border-e bg-muted px-2 text-xs leading-9 text-muted-foreground">{prefix}</span>
          <input
            id="slug"
            className="min-w-0 flex-1 bg-transparent px-2 font-mono text-sm outline-none placeholder:font-sans placeholder:text-muted-foreground"
            value={value}
            onChange={(event) => onChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter" && !composing(event)) setOpen(false);
            }}
            placeholder={t("editor.slugPlaceholder")}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
}

/**
 * TemplateChip picks the template the theme draws the item with. One the item
 * names that the theme does not offer is still listed, so it can be seen and
 * undone.
 */
function TemplateChip({
  layouts,
  chosen,
  onChange,
}: {
  layouts: LayoutOption[];
  chosen?: string;
  onChange: (layout: string | null) => void;
}) {
  const { t } = useI18n();
  const offered = layouts.find((each) => each.name === chosen);
  const missing = chosen !== undefined && !offered;
  const name = offered ? offered.label || offered.name : chosen;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="outline" size="sm" className={cn(chip, missing && "text-warning")}>
          <LayoutTemplate />
          <span className="truncate">
            {name ? t("editor.templateNamed", { name }) : t("editor.templateDefault")}
          </span>
          <ChevronDown />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-72">
        <DropdownMenuRadioGroup
          value={chosen ?? DEFAULT_LAYOUT}
          onValueChange={(value) => onChange(value === DEFAULT_LAYOUT ? null : value)}
        >
          <Choice value={DEFAULT_LAYOUT} label={t("editor.templateDefault")} note={t("editor.templateDefaultNote")} />
          {layouts.map((layout) => (
            <Choice
              key={layout.name}
              value={layout.name}
              label={layout.label || layout.name}
              note={layout.description}
            />
          ))}
          {missing && (
            <Choice value={chosen} label={chosen} note={t("editor.templateMissing", { name: chosen })} />
          )}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function Choice({ value, label, note }: { value: string; label: string; note?: string }) {
  return (
    <DropdownMenuRadioItem value={value} className="items-start [&>span:first-child]:mt-[3px]">
      <span className="grid gap-0.5">
        <span>{label}</span>
        {note && <span className="text-xs text-muted-foreground">{note}</span>}
      </span>
    </DropdownMenuRadioItem>
  );
}
