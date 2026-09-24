import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { Tabs as TabsPrimitive } from "@base-ui/react/tabs";
import { RotateCcw, Trash2, Upload, X, XCircle } from "lucide-react";
import { toast } from "sonner";

import type { Summary } from "@/api/client";
import { useI18n, useProblem, type Key } from "@/i18n";
import {
  PAGE_SIZE,
  STATUSES,
  useContentPage,
  useContentTypes,
  useStatusCounts,
  useTermsOf,
} from "@/hooks/useContents";
import { useDeleteItems, useRestoreItems, type BatchResult, type Target } from "@/hooks/useDeleteItems";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { canPublish, useDelivery } from "@/hooks/usePublish";
import { useListState } from "@/lib/listState";
import { navigate } from "@/lib/router";

import { ConfirmDialog } from "@/components/ConfirmDialog";
import { IndexProblems } from "@/components/IndexProblems";
import { ContentTable } from "@/components/content/ContentTable";
import { IconPlus, IconSearch } from "@/components/icons";
import { PublishDialog } from "@/components/publish/PublishDialog";
import { Page } from "@/components/shell/Page";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

// The design's toolbar controls: 32px, content-wide, the chevron beside the words.
const control =
  "h-8 gap-1.5 bg-background px-[11px] py-0 text-[12.5px] text-foreground-2 hover:bg-hover [&_svg]:size-[13px]";

/** The value a Select uses for "no filter", since it cannot hold an empty one. */
const ANY = "__any__";

export function ContentListPage({ kind }: { kind: string }) {
  const { t } = useI18n();
  const problem = useProblem();
  const kindLabel = useKindLabel();
  const taxonomyLabel = useTaxonomyLabel();

  const types = useContentTypes();
  const type = types.data?.items.find((item) => item.kind === kind);
  // Categories lead, as the broader of the two a reader filters by.
  const taxonomies = useMemo(
    () =>
      [...(type?.taxonomies ?? [])].sort(
        (a, b) => Number(b === "categories") - Number(a === "categories"),
      ),
    [type],
  );
  const terms = useTermsOf(taxonomies);
  const counts = useStatusCounts(kind);
  const delivery = useDelivery();

  const [state, set] = useListState(kind);
  // The query follows the typing rather than the keystroke, so a slow answer
  // for an old prefix cannot replace a fresh one.
  const q = useDeferredValue(state.q);

  const sortable = type?.sortable ?? [];
  const sort = state.sort || (sortable[0] ? defaultSort(sortable[0]) : "");

  const filters = useMemo(
    () => ({
      kind,
      status: state.status === "all" || state.status === "trash" ? undefined : state.status,
      deletedOnly: state.status === "trash",
      terms: Object.entries(state.terms).map(([taxonomy, term]) => `${taxonomy}:${term}`),
      q,
      sort,
    }),
    [kind, state.status, state.terms, q, sort],
  );

  // The cursors handed out so far; the last one is the page on screen. They
  // belong to one set of filters, and are dropped the moment those change.
  const filterKey = JSON.stringify(filters);
  const [paging, setPaging] = useState({ key: filterKey, cursors: [] as string[] });
  const cursors = paging.key === filterKey ? paging.cursors : [];

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirming, setConfirming] = useState<Summary[] | null>(null);
  const [batchResult, setBatchResult] = useState<{ action: "delete" | "restore"; result: BatchResult } | null>(null);
  const [publishing, setPublishing] = useState(false);

  useEffect(() => {
    setSelected(new Set());
    setBatchResult(null);
  }, [filterKey]);

  const page = useContentPage(filters, cursors.at(-1));
  const items = page.data?.items ?? [];
  const total = page.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const remove = useDeleteItems();
  const restore = useRestoreItems();
  const chosen = items.filter((item) => selected.has(item.id));
  const unfiltered =
    state.status === "all" && !state.q && Object.keys(state.terms).length === 0;
  const create = () => navigate({ name: "edit", kind, id: null });

  // The design's tabs are all, published and draft; the other two statuses
  // take a tab only while something is in them.
  const tabs = (["all", ...STATUSES, "trash"] as const).filter(
    (status) =>
      status === "all" ||
      status === "trash" ||
      status === "published" ||
      status === "draft" ||
      Boolean(counts[status]) ||
      state.status === status,
  );
  // The first taxonomy has a filter of its own, as the column it fills; the
  // rest are narrowed by clicking a term under a title, and show here once set.
  const [column, ...rest] = taxonomies;
  const setTerm = (taxonomy: string, term: string | undefined) => {
    const next = { ...state.terms };
    if (term) next[taxonomy] = term;
    else delete next[taxonomy];
    set({ terms: next });
  };

  const turn = (next: string[]) => {
    setPaging({ key: filterKey, cursors: next });
    setSelected(new Set());
  };

  const targetOf = (item: Summary): Target => ({ id: item.id, revision: item.revision, title: item.title });
  const runBatch = (action: "delete" | "restore", targets: Target[]) => {
    const mutation = action === "delete" ? remove : restore;
    mutation.mutate(targets, {
      onSuccess: (result) => {
        if (result.succeeded.length > 0) {
          toast.success(t(action === "delete" ? "list.deleted" : "list.restored", {
            count: result.succeeded.length,
          }));
        }
        setBatchResult(result.failed.length > 0 ? { action, result } : null);
        setSelected(new Set(result.failed.map(({ target }) => target.id)));
        setConfirming(null);
      },
    });
  };
  const confirmDelete = () => {
    if (confirming) runBatch("delete", confirming.map(targetOf));
  };
  const reviewFailed = async () => {
    if (!batchResult) return;
    await page.refetch();
    setSelected(new Set());
    setBatchResult(null);
  };

  return (
    <Page
      flush
      title={kindLabel.many(kind)}
      description={t("list.description", { kind: kindLabel.many(kind) })}
      actions={
        <Button onClick={create}>
          <IconPlus className="size-[15px]" strokeWidth={2} />
          {t("list.newKind", { kind: kindLabel.one(kind) })}
        </Button>
      }
      toolbar={
        <div className="flex flex-wrap items-center justify-between gap-3">
          <TabsPrimitive.Root
            className="max-w-full"
            value={state.status}
            onValueChange={(status) => set({ status: String(status) })}
          >
            <TabsPrimitive.List className="flex max-w-full gap-0.5 overflow-x-auto rounded-[9px] bg-muted p-[3px]">
              {tabs.map((status) => (
                <TabsPrimitive.Tab
                  key={status}
                  value={status}
                  className="flex shrink-0 items-center gap-1.5 rounded-[7px] px-3 py-[5px] text-[12.5px] whitespace-nowrap text-muted-foreground outline-none transition-colors hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 data-active:bg-card data-active:font-medium data-active:text-foreground data-active:shadow-[0_1px_3px_rgb(0_0_0/0.08)]"
                >
                  <span>{status === "all" ? t("list.filterAny") : t(`status.${status}` as Key)}</span>
                  <span className="text-[11px] tabular-nums opacity-65">{counts[status] ?? ""}</span>
                </TabsPrimitive.Tab>
              ))}
            </TabsPrimitive.List>
          </TabsPrimitive.Root>

          {chosen.length > 0 ? (
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm text-muted-foreground">
                {t("list.selected", { count: chosen.length })}
              </span>
              {state.status !== "trash" && canPublish(delivery.data) && (
                <Button variant="outline" onClick={() => setPublishing(true)}>
                  <Upload data-icon="inline-start" />
                  {t("publish.action")}
                </Button>
              )}
              {state.status === "trash" ? (
                <Button variant="outline" onClick={() => runBatch("restore", chosen.map(targetOf))}>
                  <RotateCcw data-icon="inline-start" />
                  {t("list.restore")}
                </Button>
              ) : (
                <Button variant="destructive" onClick={() => setConfirming(chosen)}>
                  <Trash2 data-icon="inline-start" />
                  {t("editor.delete")}
                </Button>
              )}
              <Button
                variant="ghost"
                size="icon"
                aria-label={t("list.clearSelection")}
                onClick={() => setSelected(new Set())}
              >
                <X />
              </Button>
            </div>
          ) : (
            <div className="flex flex-wrap items-center gap-2">
              {rest.map(
                (taxonomy) =>
                  state.terms[taxonomy] && (
                    <span
                      key={taxonomy}
                      className="flex h-8 items-center gap-1 rounded-[7px] border border-input bg-background pr-1 pl-[11px] text-[12.5px] text-foreground-2"
                    >
                      {taxonomyLabel(taxonomy)}：{state.terms[taxonomy]}
                      <button
                        type="button"
                        aria-label={t("list.clearFilter")}
                        onClick={() => setTerm(taxonomy, undefined)}
                        className="flex size-6 items-center justify-center rounded-[5px] text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                      >
                        <X className="size-3.5" />
                      </button>
                    </span>
                  ),
              )}

              {column && (
                <Filter
                  any={t("list.allOf", { name: taxonomyLabel(column) })}
                  value={state.terms[column]}
                  options={terms[column].map((item) => item.term)}
                  onChange={(term) => setTerm(column, term)}
                />
              )}

              {sortable.length > 1 && (
                <Select value={sort} onValueChange={(value) => set({ sort: String(value) })}>
                  <SelectTrigger className={control}>
                    <SelectValue>
                      {(value: string) => t(`sort.${value.replace(/^-/, "")}` as Key)}
                    </SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {sortable.map((field) => (
                        <SelectItem key={field} value={defaultSort(field)}>
                          {t(`sort.${field}` as Key)}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              )}

              <label className="flex h-8 w-full items-center gap-[7px] rounded-[7px] border border-input bg-background px-2.5 text-subtle transition-colors focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 sm:w-[210px] dark:bg-input/30">
                <IconSearch className="size-3.5" />
                <input
                  aria-label={t("list.searchKind", { kind: kindLabel.many(kind) })}
                  value={state.q}
                  onChange={(event) => set({ q: event.target.value })}
                  placeholder={t("list.searchKind", { kind: kindLabel.many(kind) })}
                  className="min-w-0 flex-1 bg-transparent text-[12.5px] text-foreground outline-none placeholder:text-subtle"
                />
              </label>
            </div>
          )}
        </div>
      }
    >
      <div className="flex flex-col gap-3.5">
        <IndexProblems />

        {batchResult && (
          <Alert variant="destructive">
            <XCircle />
            <AlertTitle>{t("list.partial", {
              done: batchResult.result.succeeded.length,
              failed: batchResult.result.failed.length,
            })}</AlertTitle>
            <AlertDescription>
              <ul className="list-inside list-disc">
                {batchResult.result.failed.map(({ target, error }) => {
                  const said = problem(error instanceof Error && "code" in error ? String(error.code) : undefined, error.message);
                  return <li key={target.id}>{target.title}: {said.title}</li>;
                })}
              </ul>
              <Button variant="outline" size="sm" disabled={remove.isPending || restore.isPending} onClick={reviewFailed}>
                {t("list.reviewFailed")}
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {page.error ? (
          <Alert variant="destructive">
            <XCircle />
            <AlertTitle>{t("list.failed")}</AlertTitle>
            <AlertDescription>{String(page.error)}</AlertDescription>
          </Alert>
        ) : (
          <ContentTable
            kind={kind}
            items={items}
            taxonomies={taxonomies}
            loading={page.isPending}
            selected={selected}
            onSelected={setSelected}
            activeTerms={state.terms}
            onTerm={(taxonomy, term) =>
              setTerm(taxonomy, state.terms[taxonomy] === term ? undefined : term)
            }
            onDelete={(item) => setConfirming([item])}
            onRestore={(item) => runBatch("restore", [targetOf(item)])}
            trashed={state.status === "trash"}
            onCreate={unfiltered ? create : undefined}
            total={total}
            page={cursors.length + 1}
            pages={pages}
            onPrevious={cursors.length > 0 ? () => turn(cursors.slice(0, -1)) : undefined}
            onNext={
              page.data?.has_more && page.data.next_cursor
                ? () => turn([...cursors, page.data.next_cursor as string])
                : undefined
            }
          />
        )}
      </div>

      <ConfirmDialog
        open={confirming !== null}
        onOpenChange={(open) => !open && setConfirming(null)}
        title={t("list.confirmDelete", { count: confirming?.length ?? 0 })}
        description={t("list.confirmDeleteNote")}
        confirmLabel={t("editor.delete")}
        destructive
        pending={remove.isPending}
        onConfirm={confirmDelete}
      />

      <PublishDialog
        ids={chosen.map((item) => item.id)}
        open={publishing}
        onOpenChange={setPublishing}
        onDone={() => setSelected(new Set())}
      />
    </Page>
  );
}

/** A date column wants newest first; a title wants A to Z. */
function defaultSort(field: string): string {
  return field === "title" ? field : `-${field}`;
}

function Filter({
  any,
  value,
  options,
  onChange,
}: {
  any: string;
  value?: string;
  options: string[];
  onChange: (value: string | undefined) => void;
}) {
  if (options.length === 0) return null;
  return (
    <Select
      value={value ?? ANY}
      onValueChange={(next) => onChange(!next || next === ANY ? undefined : String(next))}
    >
      <SelectTrigger className={control}>
        {/* base-ui would show the sentinel itself, which is not for reading. */}
        <SelectValue>{(next: string) => (!next || next === ANY ? any : next)}</SelectValue>
      </SelectTrigger>
      <SelectContent>
        <SelectGroup>
          <SelectItem value={ANY}>{any}</SelectItem>
          {options.map((option) => (
            <SelectItem key={option} value={option}>
              {option}
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  );
}
