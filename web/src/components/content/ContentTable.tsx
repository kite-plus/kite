import { Fragment, useEffect, useRef } from "react";
import { ExternalLink, Pencil, Trash2 } from "lucide-react";
import { cn } from "cn";

import type { Summary } from "@/api/client";
import { useI18n } from "@/i18n";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { isoDate } from "@/lib/dates";
import { linkProps, navigate } from "@/lib/router";

import { StatusDot } from "@/components/StatusDot";
import { IconDots, IconPlus } from "@/components/icons";
import { Button } from "@/components/ui/button";
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
import { Skeleton } from "@/components/ui/skeleton";

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
  const { t } = useI18n();
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

  // Comments and views are drawn where the design has them and left empty
  // until Kite counts either.
  const count = "text-right text-[12.5px] text-foreground-3 tabular-nums";

  return (
    // Columns come and go with the table's own width, since the rail and any
    // panel beside it decide that more than the window does.
    <div className="@container overflow-hidden rounded-[10px] border border-input bg-card shadow-[0_1px_2px_rgb(0_0_0/0.04)]">
      <table className="w-full table-fixed border-collapse text-left">
        <thead className="bg-surface text-xs font-medium text-muted-foreground">
          <tr className="h-[38px]">
            <th className="w-11 text-center font-medium">
              <Tick
                label={t("list.selectAll")}
                checked={all}
                indeterminate={some && !all}
                onChange={(on) => onSelected(on ? new Set(items.map((item) => item.id)) : new Set())}
              />
            </th>
            <th className="font-medium">{t("list.title")}</th>
            {column && (
              <th className="hidden w-[92px] font-medium @2xl:table-cell">{taxonomyLabel(column)}</th>
            )}
            <th className="w-[100px] font-medium">{t("list.status")}</th>
            <th className="hidden w-[68px] text-right font-medium @4xl:table-cell">
              {t("list.comments")}
            </th>
            <th className="hidden w-[84px] text-right font-medium @4xl:table-cell">
              {t("list.views")}
            </th>
            <th className="hidden w-[104px] pl-[18px] font-medium @lg:table-cell">
              {t("list.published")}
            </th>
            <th className="w-[46px]" />
          </tr>
        </thead>

        <tbody>
          {loading &&
            Array.from({ length: 6 }, (_, i) => (
              <tr key={i} className="border-t border-divider">
                <td />
                <td colSpan={7} className="py-[22px] pr-3.5">
                  <Skeleton className="h-4 w-2/3" />
                </td>
              </tr>
            ))}

          {items.map((item) => {
            const tags = inline.flatMap((taxonomy) =>
              (item.taxonomies?.[taxonomy] ?? []).map((term) => ({ taxonomy, term })),
            );
            const chosen = selected.has(item.id);
            return (
              <tr
                key={item.id}
                data-selected={chosen || undefined}
                className="border-t border-divider transition-colors hover:bg-hover data-selected:bg-brand-soft/60"
              >
                <td className="text-center">
                  <Tick
                    label={item.title || item.slug}
                    checked={chosen}
                    onChange={(on) => toggle(item.id, on)}
                  />
                </td>

                <td className="py-3 pr-3.5">
                  <div className="flex min-w-0 items-center gap-[7px]">
                    <a
                      {...linkProps({ name: "edit", kind: item.kind, id: item.id })}
                      className="truncate text-[13px] font-medium transition-colors hover:text-brand"
                    >
                      {item.title || item.slug}
                    </a>
                    {item.pinned && (
                      <span className="flex-none rounded-[5px] border border-pin-line bg-pin-soft px-1.5 py-px text-[10.5px] whitespace-nowrap text-pin">
                        {t("field.pinned")}
                      </span>
                    )}
                  </div>
                  {tags.length > 0 && (
                    <div className="mt-[3px] truncate text-[11.5px] text-subtle">
                      {tags.map((tag, i) => (
                        <Fragment key={`${tag.taxonomy}:${tag.term}`}>
                          {i > 0 && " · "}
                          <button
                            type="button"
                            onClick={() => onTerm(tag.taxonomy, tag.term)}
                            className={cn(
                              "transition-colors hover:text-brand",
                              activeTerms[tag.taxonomy] === tag.term && "text-brand",
                            )}
                          >
                            #{tag.term}
                          </button>
                        </Fragment>
                      ))}
                    </div>
                  )}
                </td>

                {column && (
                  <td className="hidden @2xl:table-cell">
                    <div className="flex flex-wrap gap-1">
                      {(item.taxonomies?.[column] ?? []).map((term) => (
                        <button
                          key={term}
                          type="button"
                          onClick={() => onTerm(column, term)}
                          className={cn(
                            "max-w-full truncate rounded-[6px] px-[9px] py-[2.5px] text-[11.5px] font-medium transition-colors",
                            activeTerms[column] === term
                              ? "bg-brand-soft text-brand"
                              : "bg-muted text-foreground-2 hover:bg-brand-soft hover:text-brand",
                          )}
                        >
                          {term}
                        </button>
                      ))}
                    </div>
                  </td>
                )}

                <td>
                  <StatusDot status={item.status} />
                </td>
                <td className={cn("hidden @4xl:table-cell", count)}>—</td>
                <td className={cn("hidden @4xl:table-cell", count)}>—</td>
                <td className="hidden pl-[18px] text-xs text-muted-foreground tabular-nums @lg:table-cell">
                  {isoDate(item.published_at)}
                </td>

                <td className="text-center">
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <button
                          type="button"
                          aria-label={t("list.actions")}
                          className="inline-flex size-[26px] items-center justify-center rounded-[6px] align-middle text-muted-foreground outline-none transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 aria-expanded:bg-muted aria-expanded:text-foreground"
                        />
                      }
                    >
                      <IconDots className="size-[15px]" />
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
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {!loading && items.length === 0 && (
        <Empty className="border-t border-divider">
          <EmptyHeader>
            <EmptyTitle>
              {onCreate ? t("list.emptyKind", { kind: kindLabel.many(kind) }) : t("list.empty")}
            </EmptyTitle>
            {onCreate && <EmptyDescription>{t("list.emptyNote")}</EmptyDescription>}
          </EmptyHeader>
          {onCreate && (
            <EmptyContent>
              <Button onClick={onCreate}>
                <IconPlus className="size-[15px]" strokeWidth={2} />
                {t("list.newKind", { kind: kindLabel.one(kind) })}
              </Button>
            </EmptyContent>
          )}
        </Empty>
      )}

      <div className="flex flex-wrap items-center justify-between gap-3 border-t px-4 py-[9px] text-[12.5px] text-muted-foreground">
        <span>{t("list.total", { count: total, kind: kindLabel.many(kind) })}</span>
        {/* Buttons rather than links: a cursor is not an address to point at. */}
        <nav className="flex items-center gap-1.5">
          <span className="mr-1.5">{t("list.page", { page, pages })}</span>
          <PageButton onClick={onPrevious}>{t("list.previous")}</PageButton>
          <PageButton onClick={onNext}>{t("list.next")}</PageButton>
        </nav>
      </div>
    </div>
  );
}

function PageButton({ onClick, children }: { onClick?: () => void; children: string }) {
  return (
    <button
      type="button"
      disabled={!onClick}
      onClick={onClick}
      className="h-7 rounded-[6px] border bg-card px-[11px] text-xs whitespace-nowrap text-foreground-2 outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/50 enabled:hover:bg-hover disabled:cursor-default disabled:text-border-strong"
    >
      {children}
    </button>
  );
}

/** A plain checkbox, as the design draws one, that can also say "some". */
function Tick({
  label,
  checked,
  indeterminate = false,
  onChange,
}: {
  label: string;
  checked: boolean;
  indeterminate?: boolean;
  onChange: (on: boolean) => void;
}) {
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (ref.current) ref.current.indeterminate = indeterminate;
  }, [indeterminate]);

  return (
    <input
      ref={ref}
      type="checkbox"
      aria-label={label}
      checked={checked}
      onChange={(event) => onChange(event.target.checked)}
      className="size-3.5 cursor-pointer align-middle accent-[#18181b] dark:accent-foreground"
    />
  );
}
