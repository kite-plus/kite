import { useMemo, useState } from "react";
import { getRouteApi, Link } from "@tanstack/react-router";
import {
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table";
import { ArrowUpDown, ExternalLink, List, Merge, MoreHorizontal, Pencil, Trash2 } from "lucide-react";

import type { components } from "@/api/client";
import { useI18n } from "@/i18n";
import { useContentTypes, useTaxonomies, useTerms } from "@/hooks/useContents";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { followRowLink } from "@/lib/row-link";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { DataTablePagination } from "@/components/data-table";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { QueryError } from "@/components/query-error";
import { TermBadge } from "@/components/TermBadge";
import { DEFAULT_PAGE_SIZE } from "@/features/contents/search";
import { TermDialog, type TermAction, type TermMode } from "./term-dialog";

type Term = components["schemas"]["TermCount"];
type Order = "count" | "name";

const route = getRouteApi("/_authenticated/taxonomies/$taxonomy");

// As many cards to a row as the column has room for, each wide enough for a
// long term beside its menu.
const grid = "grid grid-cols-[repeat(auto-fill,minmax(10rem,1fr))] gap-3";

/**
 * Terms lists one taxonomy, as a blog's categories or tags page does. Terms
 * are gathered from the items that carry them, so there is nothing to add or
 * delete here: each one leads to its items, which are where it changes.
 *
 * A term is only a name and a count, so each gets a card in a grid rather
 * than a table row stretched across the page. Renaming, merging or deleting
 * one rewrites every item that carries it.
 */
export function Terms() {
  const { taxonomy } = route.useParams();
  const { t, locale } = useI18n();
  const kindLabel = useKindLabel();
  const taxonomyLabel = useTaxonomyLabel();
  const types = useContentTypes();
  const taxonomies = useTaxonomies();
  const terms = useTerms(taxonomy);

  const url = taxonomies.data?.items.find((item) => item.name === taxonomy)?.url;
  // A term's items are listed under the first kind that carries the taxonomy.
  const kind =
    types.data?.items.find((type) => type.taxonomies?.includes(taxonomy))?.kind ?? "post";
  const name = taxonomyLabel(taxonomy);
  const many = kindLabel.many(kind);
  useDocumentTitle(name);

  const [action, setAction] = useState<TermAction | null>(null);
  const [order, setOrder] = useState<Order>("count");
  const [search, setSearch] = useState("");
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE });

  const columns = useMemo<ColumnDef<Term>[]>(
    () => [
      {
        accessorKey: "term",
        filterFn: "includesString",
        sortingFn: (a, b) => a.original.term.localeCompare(b.original.term, locale),
      },
      {
        accessorKey: "count",
        // Sorted descending, so terms used alike fall back to reverse name order.
        sortingFn: (a, b) =>
          a.original.count - b.original.count ||
          b.original.term.localeCompare(a.original.term, locale),
      },
    ],
    [locale],
  );

  const sorting = useMemo<SortingState>(
    () => [order === "count" ? { id: "count", desc: true } : { id: "term", desc: false }],
    [order],
  );
  const columnFilters = useMemo(() => (search ? [{ id: "term", value: search }] : []), [search]);

  const table = useReactTable({
    data: terms.data?.items ?? [],
    columns,
    state: { sorting, columnFilters, pagination },
    onPaginationChange: setPagination,
    getRowId: (term) => term.term,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  });

  const matched = table.getFilteredRowModel().rows.length;
  const rows = table.getRowModel().rows;

  return (
    <>
      <AppHeader />

      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <PageTitle title={name} description={t("terms.description", { name, kind: many })}>
          {url && (
            <Button variant="outline" asChild>
              <a href={url} target="_blank" rel="noreferrer">
                <ExternalLink />
                {t("terms.viewOnSite")}
              </a>
            </Button>
          )}
        </PageTitle>

        {terms.error && !terms.data ? (
          <QueryError error={terms.error} onRetry={() => void terms.refetch()} />
        ) : (
          <div className="flex flex-1 flex-col gap-4">
            <div className="flex flex-wrap items-center gap-2">
              <Input
                value={search}
                onChange={(event) => {
                  setSearch(event.target.value);
                  table.firstPage();
                }}
                placeholder={t("terms.search", { name })}
                className="h-8 w-37.5 lg:w-62.5"
              />
              <Select value={order} onValueChange={(value) => setOrder(value as Order)}>
                <SelectTrigger size="sm" aria-label={t("terms.order")}>
                  <ArrowUpDown />
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="count">{t("terms.byCount", { kind: many })}</SelectItem>
                  <SelectItem value="name">{t("terms.byName")}</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {terms.isPending ? (
              <div className={grid}>
                {Array.from({ length: 8 }, (_, i) => (
                  <Skeleton key={i} className="h-24 rounded-xl" />
                ))}
              </div>
            ) : rows.length ? (
              <div className={grid}>
                {rows.map((row) => (
                  <TermCard
                    key={row.id}
                    term={row.original}
                    kind={kind}
                    taxonomy={taxonomy}
                    unit={t("terms.unit", {
                      kind: row.original.count === 1 ? kindLabel.one(kind) : many,
                    })}
                    canMerge={(terms.data?.items.length ?? 0) > 1}
                    onAction={(mode) => setAction({ mode, term: row.original.term })}
                  />
                ))}
              </div>
            ) : (
              <div className="flex h-32 flex-col items-center justify-center gap-1 rounded-xl border border-dashed text-center">
                {terms.data?.items.length ? (
                  <span className="text-muted-foreground">{t("list.empty")}</span>
                ) : (
                  <>
                    <p className="font-medium">{t("terms.empty", { name })}</p>
                    <p className="text-sm text-muted-foreground">
                      {t("terms.emptyNote", { name, kind: kindLabel.one(kind) })}
                    </p>
                  </>
                )}
              </div>
            )}

            <DataTablePagination
              page={pagination.pageIndex + 1}
              pageSize={pagination.pageSize}
              total={matched}
              summary={t("terms.total", { count: matched, name })}
              hasPrevious={table.getCanPreviousPage()}
              hasNext={table.getCanNextPage()}
              onFirst={() => table.firstPage()}
              onPrevious={() => table.previousPage()}
              onNext={() => table.nextPage()}
              onPageSizeChange={(size) => table.setPageSize(size)}
              className="mt-auto"
            />
          </div>
        )}
      </Main>

      {action && (
        <TermDialog
          key={`${action.mode}:${action.term}`}
          taxonomy={taxonomy}
          terms={terms.data?.items ?? []}
          one={kindLabel.one(kind)}
          many={many}
          action={action}
          onClose={() => setAction(null)}
        />
      )}
    </>
  );
}

/**
 * TermCard is one term: a click opens its items, the menu also its page on
 * the site and the changes to the term itself.
 */
function TermCard({
  term,
  kind,
  taxonomy,
  unit,
  canMerge,
  onAction,
}: {
  term: Term;
  kind: string;
  taxonomy: string;
  unit: string;
  canMerge: boolean;
  onAction: (mode: TermMode) => void;
}) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const items = { terms: [`${taxonomy}:${term.term}`] };

  return (
    <div
      onClick={followRowLink}
      className="flex cursor-pointer flex-col gap-3 rounded-xl border bg-card p-4 text-card-foreground shadow-xs transition-colors hover:bg-accent/40"
    >
      <div className="flex items-start justify-between gap-2">
        <Link to="/content/$kind" params={{ kind }} search={items} data-row-link className="min-w-0">
          <TermBadge term={term.term} className="max-w-full truncate text-sm" />
        </Link>
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="-me-2 -mt-1.5 size-8 shrink-0 p-0 data-[state=open]:bg-muted"
            >
              <MoreHorizontal className="size-4" />
              <span className="sr-only">{t("list.actions")}</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-44">
            <DropdownMenuItem asChild>
              <Link to="/content/$kind" params={{ kind }} search={items}>
                <List />
                {t("terms.viewItems", { kind: kindLabel.many(kind) })}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <a href={term.url} target="_blank" rel="noreferrer">
                <ExternalLink />
                {t("list.open")}
              </a>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={() => onAction("rename")}>
              <Pencil />
              {t("terms.rename")}
            </DropdownMenuItem>
            <DropdownMenuItem disabled={!canMerge} onSelect={() => onAction("merge")}>
              <Merge />
              {t("terms.merge")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onSelect={() => onAction("remove")}>
              <Trash2 />
              {t("terms.remove")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <p className="text-sm text-muted-foreground">
        <span className="me-1 text-2xl font-semibold text-foreground tabular-nums">{term.count}</span>
        {unit}
      </p>
    </div>
  );
}
