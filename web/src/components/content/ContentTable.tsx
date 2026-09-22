import { ExternalLink, MoreHorizontal, Pencil, Plus, Trash2 } from "lucide-react";

import type { Summary } from "@/api/client";
import { useI18n } from "@/i18n";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { linkProps, navigate } from "@/lib/router";

import { StatusDot } from "@/components/StatusDot";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from "@/components/ui/empty";
import { Pagination, PaginationContent, PaginationItem } from "@/components/ui/pagination";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

interface Props {
  kind: string;
  items: Summary[];
  taxonomies: string[];
  loading: boolean;
  selected: Set<string>;
  onSelected: (selected: Set<string>) => void;
  activeTerms: Record<string, string>;
  onTerm: (taxonomy: string, term: string) => void;
  onDelete: (item: Summary) => void;
  /** onCreate is offered on an empty listing that no filter is narrowing. */
  onCreate?: () => void;
  total: number;
  page: number;
  pages: number;
  onPrevious?: () => void;
  onNext?: () => void;
}

export function ContentTable({
  kind,
  items,
  taxonomies,
  loading,
  selected,
  onSelected,
  activeTerms,
  onTerm,
  onDelete,
  onCreate,
  total,
  page,
  pages,
  onPrevious,
  onNext,
}: Props) {
  const { t, date, relative } = useI18n();
  const kindLabel = useKindLabel();
  const taxonomyLabel = useTaxonomyLabel();

  // One taxonomy gets a column of its own; the rest sit under the title.
  const column = taxonomies.includes("categories") ? "categories" : taxonomies[0];
  const inline = taxonomies.filter((taxonomy) => taxonomy !== column);

  const all = items.length > 0 && items.every((item) => selected.has(item.id));
  const some = items.some((item) => selected.has(item.id));

  const toggle = (id: string, on: boolean) => {
    const next = new Set(selected);
    if (on) next.add(id);
    else next.delete(id);
    onSelected(next);
  };

  return (
    // Columns come and go with the table's own width, since the rail and any
    // panel beside it decide that more than the window does.
    <div className="@container overflow-hidden rounded-lg border bg-card shadow-xs">
      <Table>
        <TableHeader className="bg-muted/40">
          <TableRow className="hover:bg-transparent">
            <TableHead className="w-11 text-center">
              <Checkbox
                aria-label={t("list.selectAll")}
                checked={all}
                indeterminate={some && !all}
                onCheckedChange={(on) =>
                  onSelected(on ? new Set(items.map((item) => item.id)) : new Set())
                }
              />
            </TableHead>
            <TableHead>{t("list.title")}</TableHead>
            {column && (
              <TableHead className="hidden w-36 @2xl:table-cell">{taxonomyLabel(column)}</TableHead>
            )}
            <TableHead className="w-24">{t("list.status")}</TableHead>
            <TableHead className="hidden w-28 @4xl:table-cell">{t("list.updated")}</TableHead>
            <TableHead className="hidden w-32 @lg:table-cell">{t("list.published")}</TableHead>
            <TableHead className="w-11" />
          </TableRow>
        </TableHeader>

        <TableBody>
          {loading &&
            Array.from({ length: 6 }, (_, i) => (
              <TableRow key={i} className="hover:bg-transparent">
                <TableCell />
                <TableCell colSpan={6} className="py-3">
                  <Skeleton className="h-5 w-2/3" />
                </TableCell>
              </TableRow>
            ))}

          {items.map((item) => {
            const tags = inline.flatMap((taxonomy) => item.taxonomies?.[taxonomy] ?? []);
            return (
              <TableRow key={item.id} data-state={selected.has(item.id) ? "selected" : undefined}>
                <TableCell className="text-center">
                  <Checkbox
                    aria-label={item.title || item.slug}
                    checked={selected.has(item.id)}
                    onCheckedChange={(on) => toggle(item.id, on)}
                  />
                </TableCell>

                <TableCell className="max-w-0 py-3 pr-4">
                  <a
                    {...linkProps({ name: "edit", kind: item.kind, id: item.id })}
                    className="block truncate font-medium transition-colors hover:text-brand"
                  >
                    {item.title || item.slug}
                  </a>
                  {tags.length > 0 && (
                    <div className="mt-0.5 truncate text-xs text-muted-foreground">
                      {tags.map((tag) => `#${tag}`).join(" · ")}
                    </div>
                  )}
                </TableCell>

                {column && (
                  <TableCell className="hidden @2xl:table-cell">
                    <div className="flex flex-wrap gap-1">
                      {(item.taxonomies?.[column] ?? []).map((term) => (
                        <Badge
                          key={term}
                          variant={activeTerms[column] === term ? "default" : "secondary"}
                          render={<button type="button" onClick={() => onTerm(column, term)} />}
                        >
                          {term}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                )}

                <TableCell>
                  <StatusDot status={item.status} />
                </TableCell>
                <TableCell className="hidden text-muted-foreground @4xl:table-cell">
                  {relative(item.updated_at)}
                </TableCell>
                <TableCell className="hidden text-muted-foreground tabular-nums @lg:table-cell">
                  {date(item.published_at)}
                </TableCell>

                <TableCell className="text-center">
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button variant="ghost" size="icon-sm" aria-label={t("list.actions")} />
                      }
                    >
                      <MoreHorizontal />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end" className="min-w-40">
                      <DropdownMenuGroup>
                        <DropdownMenuItem
                          onClick={() => navigate({ name: "edit", kind: item.kind, id: item.id })}
                        >
                          <Pencil />
                          {t("list.edit")}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          render={<a href={item.url} target="_blank" rel="noreferrer" />}
                        >
                          <ExternalLink />
                          {t("list.open")}
                        </DropdownMenuItem>
                      </DropdownMenuGroup>
                      <DropdownMenuSeparator />
                      <DropdownMenuGroup>
                        <DropdownMenuItem variant="destructive" onClick={() => onDelete(item)}>
                          <Trash2 />
                          {t("editor.delete")}
                        </DropdownMenuItem>
                      </DropdownMenuGroup>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>

      {!loading && items.length === 0 && (
        <Empty>
          <EmptyHeader>
            <EmptyTitle>
              {onCreate ? t("list.emptyKind", { kind: kindLabel.many(kind) }) : t("list.empty")}
            </EmptyTitle>
            {onCreate && <EmptyDescription>{t("list.emptyNote")}</EmptyDescription>}
          </EmptyHeader>
          {onCreate && (
            <EmptyContent>
              <Button onClick={onCreate}>
                <Plus data-icon="inline-start" />
                {t("list.newKind", { kind: kindLabel.one(kind) })}
              </Button>
            </EmptyContent>
          )}
        </Empty>
      )}

      <div className="flex items-center justify-between gap-3 border-t px-4 py-2 text-sm text-muted-foreground">
        <span>{t("list.total", { count: total, kind: kindLabel.many(kind) })}</span>
        {/* Buttons rather than links: a cursor is not an address to point at. */}
        <Pagination className="mx-0 w-auto">
          <PaginationContent className="gap-1.5">
            <PaginationItem className="mr-1.5">{t("list.page", { page, pages })}</PaginationItem>
            <PaginationItem>
              <Button variant="outline" size="sm" disabled={!onPrevious} onClick={onPrevious}>
                {t("list.previous")}
              </Button>
            </PaginationItem>
            <PaginationItem>
              <Button variant="outline" size="sm" disabled={!onNext} onClick={onNext}>
                {t("list.next")}
              </Button>
            </PaginationItem>
          </PaginationContent>
        </Pagination>
      </div>
    </div>
  );
}
