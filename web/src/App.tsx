import { useDeferredValue, useState } from "react";

import {
  useContentTypes,
  useContents,
  useSite,
  useTaxonomies,
  useTerms,
  type Filters,
} from "@/hooks/useContents";
import { ContentTable } from "@/components/ContentTable";
import { EditorPage } from "@/components/EditorPage";
import { Failure, Panel, Select } from "@/components/ui";

/** open is which item the editor holds: nothing, a new one, or an existing id. */
type Open = { id: string | null; kind: string } | null;

export default function App() {
  const site = useSite();
  const types = useContentTypes();
  const taxonomies = useTaxonomies();

  const [open, setOpen] = useState<Open>(null);
  const [filters, setFilters] = useState<Filters>({});
  const [search, setSearch] = useState("");
  const [taxonomy, setTaxonomy] = useState("");

  // The query follows the typing rather than the keystroke, so a slow answer
  // for an old prefix cannot replace a fresh one.
  const deferred = useDeferredValue(search);
  const terms = useTerms(taxonomy || undefined);

  const contents = useContents({ ...filters, q: deferred });
  const total = contents.data?.pages[0]?.total;
  const items = contents.data?.pages.flatMap((p) => p.items) ?? [];

  const set = (patch: Partial<Filters>) =>
    setFilters((current) => ({ ...current, ...patch }));

  if (open) {
    return (
      <EditorPage
        id={open.id}
        kind={open.kind}
        onClose={() => setOpen(null)}
        onCreated={(id) => setOpen({ id, kind: open.kind })}
      />
    );
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <header className="mb-6 flex flex-wrap items-baseline justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">
            {site.data?.title ?? "Kite Studio"}
          </h1>
          <p className="mt-1 text-sm text-[var(--muted-foreground)]">
            {site.data
              ? `${site.data.store} store, ${site.data.runtime} runtime`
              : "connecting"}
          </p>
        </div>
        <div className="flex items-center gap-4 text-sm text-[var(--muted-foreground)]">
          {Object.entries(site.data?.counts ?? {}).map(([kind, n]) => (
            <span key={kind}>
              <strong className="text-[var(--foreground)]">{n}</strong> {kind}
              {n === 1 ? "" : "s"}
            </span>
          ))}
          <button
            type="button"
            onClick={() => setOpen({ id: null, kind: filters.kind ?? "post" })}
            className="rounded-md bg-brand px-3 py-1.5 text-sm text-white hover:opacity-90"
          >
            New
          </button>
        </div>
      </header>

      {site.data?.problems?.length ? (
        <Panel className="mb-6 border-amber-500/40 bg-amber-500/5 p-4">
          <p className="text-sm font-medium text-amber-800 dark:text-amber-400">
            {site.data.problems.length} file
            {site.data.problems.length === 1 ? "" : "s"} could not be indexed
          </p>
          <ul className="mt-2 space-y-1 font-mono text-xs text-[var(--muted-foreground)]">
            {site.data.problems.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
        </Panel>
      ) : null}

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <input
          type="search"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search title, excerpt and body"
          className="min-w-56 flex-1 rounded-md border border-[var(--border)] bg-[var(--background)] px-3 py-1.5 text-sm outline-none focus:border-brand focus:ring-2 focus:ring-brand/20"
        />

        <Select
          label="Kind"
          value={filters.kind ?? ""}
          onChange={(e) => set({ kind: e.target.value || undefined })}
        >
          <option value="">any</option>
          {types.data?.items.map((t) => (
            <option key={t.kind} value={t.kind}>
              {t.label}
            </option>
          ))}
        </Select>

        <Select
          label="Status"
          value={filters.status ?? ""}
          onChange={(e) => set({ status: e.target.value || undefined })}
        >
          <option value="">any</option>
          {["published", "draft", "scheduled", "archived"].map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </Select>

        <Select
          label="In"
          value={taxonomy}
          onChange={(e) => {
            setTaxonomy(e.target.value);
            set({ term: undefined });
          }}
        >
          <option value="">any</option>
          {taxonomies.data?.items.map((t) => (
            <option key={t.name} value={t.name}>
              {t.name}
            </option>
          ))}
        </Select>

        {taxonomy && (
          <Select
            label="Term"
            value={filters.term?.split(":")[1] ?? ""}
            onChange={(e) =>
              set({
                term: e.target.value ? `${taxonomy}:${e.target.value}` : undefined,
              })
            }
          >
            <option value="">any</option>
            {terms.data?.items.map((t) => (
              <option key={t.term} value={t.term}>
                {t.term} ({t.count})
              </option>
            ))}
          </Select>
        )}
      </div>

      {contents.error ? (
        <Failure error={contents.error} />
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
    </div>
  );
}
