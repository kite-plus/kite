import type { Summary } from "@/api/client";
import { Panel, StatusBadge, Tag } from "@/components/ui";
import { cn } from "@/lib/cn";

interface Props {
  items: Summary[];
  total?: number;
  sort: string;
  onSort: (sort: string) => void;
  onTerm: (term: string) => void;
  onOpen: (id: string, kind: string) => void;
  activeTerm?: string;
  loading: boolean;
  fetchingMore: boolean;
  hasMore: boolean;
  onMore: () => void;
}

const columns: { key: string; label: string; className?: string }[] = [
  { key: "title", label: "Title" },
  { key: "", label: "Terms", className: "hidden md:table-cell" },
  { key: "", label: "Status", className: "w-28" },
  { key: "published_at", label: "Published", className: "w-36" },
];

export function ContentTable({
  items,
  total,
  sort,
  onSort,
  onTerm,
  onOpen,
  activeTerm,
  loading,
  fetchingMore,
  hasMore,
  onMore,
}: Props) {
  // A click on the active column reverses it; a click on another starts it
  // descending, which is what a date column almost always wants first.
  const toggle = (key: string) => {
    if (!key) return;
    onSort(sort === `-${key}` ? key : `-${key}`);
  };

  return (
    <Panel className="overflow-hidden">
      <table className="w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-[var(--border)] text-left">
            {columns.map((c) => (
              <th
                key={c.label}
                className={cn(
                  "px-4 py-2.5 font-medium text-[var(--muted-foreground)]",
                  c.className,
                )}
              >
                {c.key ? (
                  <button
                    type="button"
                    onClick={() => toggle(c.key)}
                    className="inline-flex items-center gap-1 hover:text-[var(--foreground)]"
                  >
                    {c.label}
                    <span className="text-xs">
                      {sort === `-${c.key}` ? "↓" : sort === c.key ? "↑" : ""}
                    </span>
                  </button>
                ) : (
                  c.label
                )}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr
              key={item.id}
              className="border-b border-[var(--border)] last:border-0 hover:bg-[var(--accent)]"
            >
              <td className="px-4 py-3">
                <button
                  type="button"
                  onClick={() => onOpen(item.id, item.kind)}
                  className="text-left font-medium hover:text-brand hover:underline"
                >
                  {item.title || item.slug}
                </button>
                <div className="mt-0.5 flex items-center gap-2">
                  <span className="truncate font-mono text-xs text-[var(--muted-foreground)]">
                    {item.locator}
                  </span>
                  <a
                    href={item.url}
                    target="_blank"
                    rel="noreferrer"
                    className="shrink-0 text-xs text-[var(--muted-foreground)] hover:text-brand"
                  >
                    view
                  </a>
                </div>
              </td>
              <td className="hidden px-4 py-3 md:table-cell">
                <div className="flex flex-wrap gap-1">
                  {Object.entries(item.taxonomies ?? {}).flatMap(
                    ([taxonomy, terms]) =>
                      (terms as string[]).map((term) => (
                        <Tag
                          key={`${taxonomy}:${term}`}
                          active={activeTerm === `${taxonomy}:${term}`}
                          onClick={() => onTerm(`${taxonomy}:${term}`)}
                        >
                          {term}
                        </Tag>
                      )),
                  )}
                </div>
              </td>
              <td className="px-4 py-3">
                <StatusBadge status={item.status} />
              </td>
              <td className="px-4 py-3 tabular-nums text-[var(--muted-foreground)]">
                {item.published_at ? item.published_at.slice(0, 10) : "—"}
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {loading && <Row>Loading</Row>}
      {!loading && items.length === 0 && <Row>Nothing matches these filters</Row>}

      {items.length > 0 && (
        <div className="flex items-center justify-between border-t border-[var(--border)] px-4 py-2.5 text-xs text-[var(--muted-foreground)]">
          <span>
            {items.length}
            {total !== undefined && total > items.length ? ` of ${total}` : ""}
          </span>
          {hasMore && (
            <button
              type="button"
              onClick={onMore}
              disabled={fetchingMore}
              className="rounded-md border border-[var(--border)] px-3 py-1 hover:bg-[var(--accent)] disabled:opacity-50"
            >
              {fetchingMore ? "Loading" : "Load more"}
            </button>
          )}
        </div>
      )}
    </Panel>
  );
}

function Row({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-4 py-10 text-center text-sm text-[var(--muted-foreground)]">
      {children}
    </div>
  );
}
