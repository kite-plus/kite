import { useMemo, useState } from "react";
import { getRouteApi, Link } from "@tanstack/react-router";
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table";
import { ExternalLink, List, MoreHorizontal } from "lucide-react";

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
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  DataTableColumnHeader,
  DataTablePagination,
  DataTableToolbar,
} from "@/components/data-table";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { QueryError } from "@/components/query-error";
import { TermBadge } from "@/components/TermBadge";
import { DEFAULT_PAGE_SIZE } from "@/features/contents/search";

type Term = components["schemas"]["TermCount"];

const route = getRouteApi("/_authenticated/taxonomies/$taxonomy");

/**
 * Terms lists one taxonomy, as a blog's categories or tags page does. Terms
 * are gathered from the items that carry them, so there is nothing to add or
 * delete here: each one leads to its items, which are where it changes.
 */
export function Terms() {
  const { taxonomy } = route.useParams();
  const { t } = useI18n();
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

  const [sorting, setSorting] = useState<SortingState>([]);
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: DEFAULT_PAGE_SIZE });

  const columns = useMemo<ColumnDef<Term>[]>(
    () => [
      {
        accessorKey: "term",
        header: ({ column }) => <DataTableColumnHeader column={column} title={t("terms.name")} />,
        cell: ({ row }) => (
          <Link
            to="/content/$kind"
            params={{ kind }}
            search={{ terms: [`${taxonomy}:${row.original.term}`] }}
            data-row-link
          >
            <TermBadge term={row.original.term} className="transition-opacity hover:opacity-80" />
          </Link>
        ),
        filterFn: "includesString",
        enableHiding: false,
        meta: { title: t("terms.name") },
      },
      {
        accessorKey: "count",
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t("terms.count", { kind: many })} />
        ),
        cell: ({ row }) => <span className="tabular-nums">{row.original.count}</span>,
        meta: { title: t("terms.count", { kind: many }), className: "w-32" },
      },
      {
        id: "actions",
        cell: ({ row }) => (
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="flex size-8 p-0 data-[state=open]:bg-muted">
                <MoreHorizontal className="size-4" />
                <span className="sr-only">{t("list.actions")}</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              <DropdownMenuItem asChild>
                <Link
                  to="/content/$kind"
                  params={{ kind }}
                  search={{ terms: [`${taxonomy}:${row.original.term}`] }}
                >
                  <List />
                  {t("terms.viewItems", { kind: many })}
                </Link>
              </DropdownMenuItem>
              <DropdownMenuItem asChild>
                <a href={row.original.url} target="_blank" rel="noreferrer">
                  <ExternalLink />
                  {t("list.open")}
                </a>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ),
        enableSorting: false,
        enableHiding: false,
        meta: { className: "w-10", skipRowLink: true },
      },
    ],
    [t, kind, taxonomy, many],
  );

  const table = useReactTable({
    data: terms.data?.items ?? [],
    columns,
    state: { sorting, pagination },
    onSortingChange: setSorting,
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
            <DataTableToolbar
              table={table}
              searchKey="term"
              searchPlaceholder={t("terms.search", { name })}
            />
            <div className="overflow-hidden rounded-md border">
              <Table>
                <TableHeader>
                  {table.getHeaderGroups().map((group) => (
                    <TableRow key={group.id}>
                      {group.headers.map((header) => (
                        <TableHead key={header.id} className={header.column.columnDef.meta?.className}>
                          {header.isPlaceholder
                            ? null
                            : flexRender(header.column.columnDef.header, header.getContext())}
                        </TableHead>
                      ))}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {terms.isPending ? (
                    Array.from({ length: 5 }, (_, i) => (
                      <TableRow key={i}>
                        <TableCell colSpan={columns.length} className="py-3">
                          <Skeleton className="h-5 w-1/3" />
                        </TableCell>
                      </TableRow>
                    ))
                  ) : rows.length ? (
                    rows.map((row) => (
                      <TableRow
                        key={row.id}
                        onClick={followRowLink}
                        className="has-[a[data-row-link]]:cursor-pointer"
                      >
                        {row.getVisibleCells().map((cell) => (
                          <TableCell
                            key={cell.id}
                            data-row-skip={cell.column.columnDef.meta?.skipRowLink || undefined}
                            className={cell.column.columnDef.meta?.className}
                          >
                            {flexRender(cell.column.columnDef.cell, cell.getContext())}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))
                  ) : (
                    <TableRow>
                      <TableCell colSpan={columns.length} className="h-32 text-center">
                        {terms.data?.items.length ? (
                          <span className="text-muted-foreground">{t("list.empty")}</span>
                        ) : (
                          <div>
                            <p className="font-medium">{t("terms.empty", { name })}</p>
                            <p className="text-sm text-muted-foreground">
                              {t("terms.emptyNote", { name, kind: kindLabel.one(kind) })}
                            </p>
                          </div>
                        )}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
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
    </>
  );
}
