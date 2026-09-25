import { useMemo } from "react";
import { Link } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { Pin } from "lucide-react";

import type { Summary } from "@/api/client";
import { useI18n } from "@/i18n";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { isoDate } from "@/lib/dates";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { DataTableColumnHeader } from "@/components/data-table";
import { StatusLabel } from "@/components/StatusLabel";
import { RowActions } from "./row-actions";

interface Options {
  /** The fields the kind can be sorted by. */
  sortable: string[];
  taxonomies: string[];
  trashed: boolean;
  chosenTerms: (taxonomy: string) => string[];
  onTerm: (taxonomy: string, term: string) => void;
  onDelete: (item: Summary) => void;
  onRestore: (item: Summary) => void;
}

/** useContentsColumns lays out a listing: one column per taxonomy the kind has. */
export function useContentsColumns({
  sortable,
  taxonomies,
  trashed,
  chosenTerms,
  onTerm,
  onDelete,
  onRestore,
}: Options): ColumnDef<Summary>[] {
  const { t } = useI18n();
  const taxonomyLabel = useTaxonomyLabel();

  return useMemo<ColumnDef<Summary>[]>(
    () => [
      {
        id: "select",
        header: ({ table }) => (
          <Checkbox
            checked={
              table.getIsAllPageRowsSelected() ||
              (table.getIsSomePageRowsSelected() && "indeterminate")
            }
            onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
            aria-label={t("list.selectAll")}
            className="translate-y-[2px]"
          />
        ),
        cell: ({ row }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label={row.original.title || row.original.slug}
            className="translate-y-[2px]"
          />
        ),
        enableSorting: false,
        enableHiding: false,
        meta: { className: "w-10", skipRowLink: true },
      },
      {
        id: "title",
        accessorKey: "title",
        header: ({ column }) => <DataTableColumnHeader column={column} title={t("list.title")} />,
        cell: ({ row }) => {
          const item = row.original;
          const name = item.title || item.slug;
          return (
            // A clamp rather than truncate lets a long title give way on a narrow screen.
            <div className="flex max-w-[28rem] min-w-0 items-center gap-2">
              {trashed ? (
                <span className="line-clamp-1 font-medium whitespace-normal wrap-anywhere">{name}</span>
              ) : (
                <Link
                  to="/content/$kind/$id"
                  params={{ kind: item.kind, id: item.id }}
                  data-row-link
                  className="line-clamp-1 font-medium whitespace-normal wrap-anywhere hover:underline"
                >
                  {name}
                </Link>
              )}
              {item.pinned && (
                <Badge variant="outline" className="shrink-0 gap-1 text-amber-700 dark:text-amber-400">
                  <Pin className="size-3" />
                  {t("field.pinned")}
                </Badge>
              )}
            </div>
          );
        },
        enableSorting: sortable.includes("title"),
        enableHiding: false,
        meta: { title: t("list.title") },
      },
      ...taxonomies.map(
        (taxonomy): ColumnDef<Summary> => ({
          id: taxonomy,
          accessorFn: (item) => item.taxonomies?.[taxonomy] ?? [],
          header: ({ column }) => (
            <DataTableColumnHeader column={column} title={taxonomyLabel(taxonomy)} />
          ),
          cell: ({ row }) => {
            const terms = row.original.taxonomies?.[taxonomy] ?? [];
            const chosen = chosenTerms(taxonomy);
            return (
              <div className="flex max-w-56 flex-wrap gap-1">
                {terms.map((term) => (
                  <button key={term} type="button" onClick={() => onTerm(taxonomy, term)}>
                    <Badge
                      variant={chosen.includes(term) ? "default" : "secondary"}
                      className="max-w-40 truncate font-normal"
                    >
                      {term}
                    </Badge>
                  </button>
                ))}
              </div>
            );
          },
          enableSorting: false,
          meta: {
            title: taxonomyLabel(taxonomy),
            className: taxonomy === "tags" ? "hidden xl:table-cell" : "hidden md:table-cell",
          },
        }),
      ),
      {
        id: "status",
        accessorKey: "status",
        header: ({ column }) => <DataTableColumnHeader column={column} title={t("list.status")} />,
        cell: ({ row }) => <StatusLabel status={row.original.status} />,
        enableSorting: false,
        meta: { title: t("list.status") },
      },
      ...(["published_at", "updated_at"] as const).map(
        (field): ColumnDef<Summary> => ({
          id: field,
          accessorKey: field,
          header: ({ column }) => (
            <DataTableColumnHeader
              column={column}
              title={t(field === "published_at" ? "list.publishedAt" : "list.updatedAt")}
            />
          ),
          cell: ({ row }) => {
            const item = row.original;
            if (field === "updated_at") {
              // An item that records no time shows when its file last changed.
              return (
                <span className="text-muted-foreground tabular-nums">
                  {isoDate(item.updated_at ?? item.modified_at)}
                </span>
              );
            }
            // Published without a date is missing something; a draft is not.
            if (!item.published_at && item.status === "published") {
              return <span className="text-muted-foreground">{t("list.noDate")}</span>;
            }
            return (
              <span className="text-muted-foreground tabular-nums">{isoDate(item.published_at)}</span>
            );
          },
          enableSorting: sortable.includes(field),
          meta: {
            title: t(field === "published_at" ? "list.publishedAt" : "list.updatedAt"),
            className: cn(field === "updated_at" ? "hidden lg:table-cell" : "hidden sm:table-cell"),
          },
        }),
      ),
      {
        id: "actions",
        cell: ({ row }) => (
          <RowActions item={row.original} trashed={trashed} onDelete={onDelete} onRestore={onRestore} />
        ),
        enableSorting: false,
        enableHiding: false,
        meta: { className: "w-10", skipRowLink: true },
      },
    ],
    [t, taxonomyLabel, sortable, taxonomies, trashed, chosenTerms, onTerm, onDelete, onRestore],
  );
}
