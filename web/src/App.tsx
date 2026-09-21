import { Suspense, lazy, useDeferredValue, useState } from "react";
import { Plus, Search } from "lucide-react";

import { useI18n } from "@/i18n";

import {
  useContents,
  useSite,
  useTaxonomies,
  useTerms,
  type Filters,
} from "@/hooks/useContents";
import { ContentTable } from "@/components/ContentTable";

import { SettingsPage } from "@/components/SettingsPage";
import { PublishPanel } from "@/components/PublishPanel";
import { Page, Shell, type Section } from "@/components/Shell";
import { Alert } from "@/components/Alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// The editor carries CodeMirror, which is the heaviest thing in the admin and
// is needed on one screen out of four. Loading it when that screen opens keeps
// the list -- the first thing anybody sees -- light.
const EditorPage = lazy(() =>
  import("@/components/EditorPage").then((m) => ({ default: m.EditorPage })),
);

/** open is what the editor holds: nothing, a new item, or an existing id. */
type Open = { id: string | null; kind: string } | null;

/** The value a Select uses for "no filter", since it cannot hold an empty one. */
const ANY = "__any__";

export default function App() {
  const { t } = useI18n();
  const site = useSite();
  const taxonomies = useTaxonomies();

  const [section, setSection] = useState<Section>("post");
  const [open, setOpen] = useState<Open>(null);
  const [filters, setFilters] = useState<Filters>({});
  const [search, setSearch] = useState("");
  const [taxonomy, setTaxonomy] = useState("");

  // The query follows the typing rather than the keystroke, so a slow answer
  // for an old prefix cannot replace a fresh one.
  const deferred = useDeferredValue(search);
  const terms = useTerms(taxonomy || undefined);

  const kind = section === "post" || section === "page" ? section : undefined;
  const contents = useContents({ ...filters, kind, q: deferred });
  const total = contents.data?.pages[0]?.total;
  const items = contents.data?.pages.flatMap((p) => p.items) ?? [];

  const set = (patch: Partial<Filters>) =>
    setFilters((current) => ({ ...current, ...patch }));

  if (open) {
    return (
      <Suspense
        fallback={
          <div className="p-10 text-center text-sm text-muted-foreground">
            {t("list.loading")}
          </div>
        }
      >
        <EditorPage
          id={open.id}
          kind={open.kind}
          onClose={() => setOpen(null)}
          onCreated={(id) => setOpen({ id, kind: open.kind })}
        />
      </Suspense>
    );
  }

  const shell = (children: React.ReactNode) => (
    <Shell
      site={site.data}
      counts={site.data?.counts}
      section={section}
      onSection={setSection}
      footer={<PublishPanel compact />}
    >
      {children}
    </Shell>
  );

  if (section === "settings") {
    return shell(<SettingsPage />);
  }
  if (section === "taxonomies") {
    return shell(
      <Page title={t("taxonomies.title")} description={t("taxonomies.description")}>
        <div className="grid gap-4 sm:grid-cols-2">
          {taxonomies.data?.items.map((item) => (
            <div key={item.name} className="rounded-md border p-4">
              <div className="flex items-baseline justify-between">
                <span className="text-sm font-medium">{item.name}</span>
                <span className="text-xs text-muted-foreground">
                  {t("taxonomies.terms", { count: item.terms })}
                </span>
              </div>
              <a
                href={item.url}
                target="_blank"
                rel="noreferrer"
                className="mt-1 block truncate font-mono text-xs text-muted-foreground hover:text-primary"
              >
                {item.url}
              </a>
            </div>
          ))}
        </div>
      </Page>,
    );
  }

  return shell(
    <Page
      title={t(section === "post" ? "nav.posts" : "nav.pages")}
      description={
        total !== undefined ? t("list.inSection", { count: total }) : undefined
      }
      actions={
        <Button size="sm" onClick={() => setOpen({ id: null, kind: section })}>
          <Plus className="size-4" />
          {t("list.new")}
        </Button>
      }
    >
      {site.data?.problems?.length ? (
        <Alert
          tone="warn"
          className="mb-4"
          title={t("problems.notIndexed", { count: site.data.problems.length })}
        >
          <ul className="mt-1 space-y-0.5 font-mono text-xs">
            {site.data.problems.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
        </Alert>
      ) : null}

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="relative min-w-52 flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t("list.search")}
            className="h-8 pl-8"
          />
        </div>

        <Filter
          label={t("list.status")}
          value={filters.status}
          onChange={(v) => set({ status: v ?? undefined })}
          options={["published", "draft", "scheduled", "archived"]}
        />

        <Filter
          label={t("list.filterIn")}
          value={taxonomy}
          onChange={(v) => {
            setTaxonomy(v ?? "");
            set({ term: undefined });
          }}
          options={taxonomies.data?.items.map((t) => t.name) ?? []}
        />

        {taxonomy && (
          <Filter
            label={t("list.filterTerm")}
            value={filters.term?.split(":")[1]}
            onChange={(v) => set({ term: v ? `${taxonomy}:${v}` : undefined })}
            options={terms.data?.items.map((t) => t.term) ?? []}
          />
        )}
      </div>

      {contents.error ? (
        <Alert tone="stop" title={t("list.failed")}>
          {String(contents.error)}
        </Alert>
      ) : (
        <ContentTable
          items={items}
          total={total ?? undefined}
          sort={filters.sort ?? "-published_at"}
          onSort={(sort) => set({ sort })}
          onOpen={(id, kind) => setOpen({ id, kind })}
          onTerm={(term) => {
            setTaxonomy(term.split(":")[0]);
            set({ term });
          }}
          activeTerm={filters.term}
          loading={contents.isPending}
          fetchingMore={contents.isFetchingNextPage}
          hasMore={contents.hasNextPage}
          onMore={() => contents.fetchNextPage()}
        />
      )}
    </Page>,
  );
}

function Filter({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value?: string;
  onChange: (value: string | undefined) => void;
  options: string[];
}) {
  const { t } = useI18n();
  if (options.length === 0) return null;
  return (
    <Select
      value={value ?? ANY}
      onValueChange={(v) => onChange(!v || v === ANY ? undefined : v)}
    >
      <SelectTrigger size="sm" className="w-auto min-w-28">
        {/* base-ui renders the raw value unless told otherwise, and the
            sentinel is not something to show a person. */}
        <SelectValue>
          {(v: string) => (
            <span className="truncate">
              <span className="text-muted-foreground">{label}: </span>
              {!v || v === ANY ? t("list.filterAny") : v}
            </span>
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        <SelectItem value={ANY}>{t("list.filterAny")}</SelectItem>
        {options.map((o) => (
          <SelectItem key={o} value={o}>
            {o}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
