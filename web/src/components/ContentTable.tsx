import type { ReactNode } from "react";
import { ArrowDown, ArrowUp, ExternalLink } from "lucide-react";
import { cn } from "cn";

import type { Summary } from "@/api/client";
import { useI18n, type Key } from "@/i18n";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

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

const columns: { key: string; label: Key; className?: string }[] = [
  { key: "title", label: "list.title" },
  { key: "", label: "list.terms", className: "hidden md:table-cell" },
  { key: "", label: "list.status", className: "w-28" },
  { key: "published_at", label: "list.published", className: "w-36" },
];

/**
 * A status is one small word, so it carries its meaning in colour rather than
 * being labelled twice.
 */
const statusTone: Record<string, string> = {
  published:
    "border-emerald-600/20 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  draft: "border-amber-600/20 bg-amber-500/10 text-amber-700 dark:text-amber-400",
  scheduled: "border-sky-600/20 bg-sky-500/10 text-sky-700 dark:text-sky-400",
  archived: "border-border bg-muted text-muted-foreground",
};

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
  const { t, date } = useI18n();

  // A click on the active column reverses it; a click on another starts it
  // descending, which is what a date column almost always wants first.
  const toggle = (key: string) => {
    if (!key) return;
    onSort(sort === `-${key}` ? key : `-${key}`);
  };

  return (
    <div className="overflow-hidden rounded-md border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            {columns.map((c) => (
              <TableHead key={c.label} className={cn("h-9", c.className)}>
                {c.key ? (
                  <button
                    type="button"
                    onClick={() => toggle(c.key)}
                    className="inline-flex items-center gap-1 transition-colors hover:text-foreground"
                  >
                    {t(c.label)}
                    {sort === `-${c.key}` ? (
                      <ArrowDown className="size-3" />
                    ) : sort === c.key ? (
                      <ArrowUp className="size-3" />
                    ) : null}
                  </button>
                ) : (
                  t(c.label)
                )}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>

        <TableBody>
          {items.map((item) => (
            <TableRow key={item.id} className="group">
              <TableCell className="py-2.5">
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => onOpen(item.id, item.kind)}
                    className="truncate text-left text-sm font-medium transition-colors hover:text-primary"
                  >
                    {item.title || item.slug}
                  </button>
                  {/* Shown on hover: the address is useful, but not so often
                      that it should compete with the title. */}
                  <a
                    href={item.url}
                    target="_blank"
                    rel="noreferrer"
                    title={t("list.open")}
                    className="shrink-0 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100 hover:text-primary"
                  >
                    <ExternalLink className="size-3.5" />
                  </a>
                </div>
                <div className="truncate font-mono text-xs text-muted-foreground">
                  {item.locator}
                </div>
              </TableCell>

              <TableCell className="hidden py-2.5 md:table-cell">
                <div className="flex flex-wrap gap-1">
                  {Object.entries(item.taxonomies ?? {}).flatMap(([taxonomy, terms]) =>
                    (terms as string[]).map((term) => {
                      const key = `${taxonomy}:${term}`;
                      return (
                        <button key={key} type="button" onClick={() => onTerm(key)}>
                          <Badge
                            variant={activeTerm === key ? "default" : "secondary"}
                            className="font-normal"
                          >
                            {term}
                          </Badge>
                        </button>
                      );
                    }),
                  )}
                </div>
              </TableCell>

              <TableCell className="py-2.5">
                <Badge
                  variant="outline"
                  className={cn("font-normal", statusTone[item.status])}
                >
                  {t(`status.${item.status}` as Key)}
                </Badge>
              </TableCell>

              <TableCell className="py-2.5 text-sm tabular-nums text-muted-foreground">
                {date(item.published_at)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>

      {loading && <Empty>{t("list.loading")}</Empty>}
      {!loading && items.length === 0 && <Empty>{t("list.empty")}</Empty>}

      {items.length > 0 && (
        <div className="flex items-center justify-between border-t bg-muted/30 px-4 py-2 text-xs text-muted-foreground">
          <span>
            {total !== undefined && total > items.length
              ? t("list.of", { shown: items.length, total })
              : items.length}
          </span>
          {hasMore && (
            <Button size="sm" variant="outline" onClick={onMore} disabled={fetchingMore}>
              {fetchingMore ? t("list.loading") : t("list.more")}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

function Empty({ children }: { children: ReactNode }) {
  return (
    <div className="px-4 py-14 text-center text-sm text-muted-foreground">{children}</div>
  );
}
