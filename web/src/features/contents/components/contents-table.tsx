import { useEffect, useState, type ReactNode } from "react";
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type ColumnDef,
  type ColumnFiltersState,
  type OnChangeFn,
  type RowSelectionState,
  type SortingState,
  type Updater,
  type VisibilityState,
} from "@tanstack/react-table";

import type { Summary } from "@/api/client";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { DataTableBulkActions, DataTablePagination, DataTableToolbar } from "@/components/data-table";
import { termsOf, withTerms, type ContentSearch } from "../search";

function apply<T>(updater: Updater<T>, current: T): T {
  return typeof updater === "function" ? (updater as (old: T) => T)(current) : updater;
}

interface Props {
  items: Summary[];
  loading: boolean;
  columns: ColumnDef<Summary>[];
  search: ContentSearch;
  onSearch: (patch: Partial<ContentSearch>) => void;
  /** sort is the order in force, the kind's default when the address has none. */
  sort: string;
  taxonomies: string[];
  termOptions: Record<string, { label: string; value: string; count: number }[]>;
  rowSelection: RowSelectionState;
  onRowSelectionChange: OnChangeFn<RowSelectionState>;
  entityName: string;
  searchPlaceholder: string;
  /** bulkActions draws the buttons for what is selected. */
  bulkActions: (selected: Summary[]) => ReactNode;
  empty: ReactNode;
  pagination: Omit<Parameters<typeof DataTablePagination>[0], "className">;
}

/**
 * ContentsTable is shadcn-admin's data table over a listing the server pages,
 * sorts and filters. The table only shows; the address holds every choice.
 */
export function ContentsTable({
  items,
  loading,
  columns,
  search,
  onSearch,
  sort,
  taxonomies,
  termOptions,
  rowSelection,
  onRowSelectionChange,
  entityName,
  searchPlaceholder,
  bulkActions,
  empty,
  pagination,
}: Props) {
  const taxonomyLabel = useTaxonomyLabel();
  const [columnVisibility, setColumnVisibility] = useState<VisibilityState>({});

  // What is typed is shown at once and asked of the server once typing stops,
  // so a slow answer for an old prefix cannot replace a fresh one.
  const [query, setQuery] = useState(search.q ?? "");
  useEffect(() => setQuery(search.q ?? ""), [search.q]);
  useEffect(() => {
    if (query === (search.q ?? "")) return;
    const timer = setTimeout(() => onSearch({ q: query || undefined }), 300);
    return () => clearTimeout(timer);
  }, [query, search.q, onSearch]);

  const sorting: SortingState = sort ? [{ id: sort.replace(/^-/, ""), desc: sort.startsWith("-") }] : [];
  const columnFilters: ColumnFiltersState = taxonomies
    .map((taxonomy) => ({ id: taxonomy, value: termsOf(search.terms, taxonomy) }))
    .filter((filter) => filter.value.length > 0);

  const table = useReactTable({
    data: items,
    columns,
    getRowId: (item) => item.id,
    state: { sorting, columnFilters, globalFilter: query, rowSelection, columnVisibility },
    manualSorting: true,
    manualFiltering: true,
    manualPagination: true,
    enableRowSelection: true,
    enableSortingRemoval: false,
    onSortingChange: (updater) => {
      const [first] = apply(updater, sorting);
      if (first) onSearch({ sort: `${first.desc ? "-" : ""}${first.id}` });
    },
    onColumnFiltersChange: (updater) => {
      const next = apply(updater, columnFilters);
      let terms = search.terms;
      for (const taxonomy of taxonomies) {
        const chosen = next.find((filter) => filter.id === taxonomy)?.value;
        terms = withTerms(terms, taxonomy, Array.isArray(chosen) ? (chosen as string[]) : []);
      }
      onSearch({ terms });
    },
    onGlobalFilterChange: (updater) => setQuery(String(apply(updater, query) ?? "")),
    onRowSelectionChange,
    onColumnVisibilityChange: setColumnVisibility,
    getCoreRowModel: getCoreRowModel(),
  });

  const selected = table.getSelectedRowModel().rows.map((row) => row.original);

  return (
    <div
      className={cn(
        'max-sm:has-[div[role="toolbar"]]:mb-16', // Room for the bulk bar on a phone.
        "flex flex-1 flex-col gap-4",
      )}
    >
      <DataTableToolbar
        table={table}
        searchPlaceholder={searchPlaceholder}
        filters={taxonomies
          .filter((taxonomy) => termOptions[taxonomy]?.length)
          .map((taxonomy) => ({
            columnId: taxonomy,
            title: taxonomyLabel(taxonomy),
            options: termOptions[taxonomy],
          }))}
      />
      <div className="overflow-hidden rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    colSpan={header.colSpan}
                    className={cn(
                      header.column.columnDef.meta?.className,
                      header.column.columnDef.meta?.thClassName,
                    )}
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {loading ? (
              Array.from({ length: 6 }, (_, i) => (
                <TableRow key={i}>
                  <TableCell colSpan={columns.length} className="py-3">
                    <Skeleton className="h-5 w-2/3" />
                  </TableCell>
                </TableRow>
              ))
            ) : table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} data-state={row.getIsSelected() && "selected"}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell
                      key={cell.id}
                      className={cn(
                        cell.column.columnDef.meta?.className,
                        cell.column.columnDef.meta?.tdClassName,
                      )}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-32 text-center">
                  {empty}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <DataTablePagination {...pagination} className="mt-auto" />
      <DataTableBulkActions table={table} entityName={entityName}>
        {bulkActions(selected)}
      </DataTableBulkActions>
    </div>
  );
}
