import { useCallback, useMemo, useRef, useState } from "react";
import { getRouteApi, Link } from "@tanstack/react-router";
import type { OnChangeFn, RowSelectionState } from "@tanstack/react-table";
import { ArrowLeft, Plus, RotateCcw, Trash2, Upload, XCircle } from "lucide-react";
import { toast } from "sonner";

import type { Summary } from "@/api/client";
import { useI18n, useProblem, type Key } from "@/i18n";
import {
  STATUSES,
  useContentPage,
  useContentTypes,
  useStatusCounts,
  useTermsOf,
} from "@/hooks/useContents";
import {
  useDeleteItems,
  useRestoreItems,
  type BatchResult,
  type Target,
} from "@/hooks/useDeleteItems";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useKindLabel } from "@/hooks/useKindLabel";
import { canPublish, useDelivery } from "@/hooks/usePublish";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { IndexProblems } from "@/components/IndexProblems";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { PublishDialog } from "@/components/publish/PublishDialog";
import { QueryError } from "@/components/query-error";
import { statuses } from "@/components/StatusLabel";
import { useContentsColumns } from "./components/contents-columns";
import { ContentsTable } from "./components/contents-table";
import { DEFAULT_PAGE_SIZE, defaultSort, termsOf, withTerms, type ContentSearch } from "./search";

const route = getRouteApi("/_authenticated/content/$kind/");

export function ContentList() {
  const { kind } = route.useParams();
  const search = route.useSearch();
  const navigate = route.useNavigate();
  const { t } = useI18n();
  const problem = useProblem();
  const kindLabel = useKindLabel();

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
  const sortable = useMemo(() => type?.sortable ?? [], [type]);
  const terms = useTermsOf(taxonomies);
  const counts = useStatusCounts(kind);
  const delivery = useDelivery();

  const onSearch = useCallback(
    (patch: Partial<ContentSearch>) =>
      void navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true }),
    [navigate],
  );

  const trashed = search.trash === true;
  useDocumentTitle(trashed ? t("status.trash") : kindLabel.many(kind));
  const sort = search.sort ?? (sortable[0] ? defaultSort(sortable[0]) : "");
  const pageSize = search.size ?? DEFAULT_PAGE_SIZE;
  const filters = useMemo(
    () => ({
      kind,
      status: search.status,
      deletedOnly: trashed,
      terms: search.terms,
      q: search.q,
      sort,
      limit: pageSize,
    }),
    [kind, search.status, trashed, search.terms, search.q, sort, pageSize],
  );

  // The cursors handed out so far; the last one is the page on screen. They
  // belong to one set of filters, and are dropped the moment those change.
  const filterKey = JSON.stringify(filters);
  const [paging, setPaging] = useState({ key: filterKey, cursors: [] as string[] });
  const cursors = paging.key === filterKey ? paging.cursors : [];
  const [selection, setSelection] = useState({ key: filterKey, rows: {} as RowSelectionState });
  const rowSelection = selection.key === filterKey ? selection.rows : {};
  // Worked out from the latest selection, since two clicks can land before
  // the table has drawn the first.
  const setRowSelection: OnChangeFn<RowSelectionState> = (updater) =>
    setSelection((prev) => {
      const current = prev.key === filterKey ? prev.rows : {};
      return { key: filterKey, rows: typeof updater === "function" ? updater(current) : updater };
    });

  const page = useContentPage(filters, cursors.at(-1));
  const items = page.data?.items ?? [];

  const turn = (next: string[]) => {
    setPaging({ key: filterKey, cursors: next });
    setRowSelection({});
  };

  const remove = useDeleteItems();
  const restore = useRestoreItems();
  const [confirming, setConfirming] = useState<Summary[] | null>(null);
  const [publishing, setPublishing] = useState<Summary[] | null>(null);
  const [batchResult, setBatchResult] = useState<{
    action: "delete" | "restore";
    result: BatchResult;
  } | null>(null);

  const targetOf = (item: Summary): Target => ({
    id: item.id,
    revision: item.revision,
    title: item.title,
  });
  const runBatch = (action: "delete" | "restore", targets: Target[]) => {
    const mutation = action === "delete" ? remove : restore;
    mutation.mutate(targets, {
      onSuccess: (result) => {
        if (result.succeeded.length > 0) {
          toast.success(
            t(action === "delete" ? "list.deleted" : "list.restored", {
              count: result.succeeded.length,
            }),
          );
        }
        setBatchResult(result.failed.length > 0 ? { action, result } : null);
        // What failed stays selected, to be looked at and tried again.
        setRowSelection(Object.fromEntries(result.failed.map(({ target }) => [target.id, true])));
        setConfirming(null);
      },
    });
  };
  const reviewFailed = async () => {
    await page.refetch();
    setRowSelection({});
    setBatchResult(null);
  };

  const chosenTerms = useCallback(
    (taxonomy: string) => termsOf(search.terms, taxonomy),
    [search.terms],
  );
  const onTerm = useCallback(
    (taxonomy: string, term: string) => {
      const chosen = termsOf(search.terms, taxonomy);
      const next = chosen.includes(term)
        ? chosen.filter((each) => each !== term)
        : [...chosen, term];
      onSearch({ terms: withTerms(search.terms, taxonomy, next) });
    },
    [search.terms, onSearch],
  );
  const onDelete = useCallback((item: Summary) => setConfirming([item]), []);
  // The columns hold on to this, so it stays the same function; a new one
  // would redraw every row and close any menu open in it.
  const latestBatch = useRef(runBatch);
  latestBatch.current = runBatch;
  const onRestore = useCallback(
    (item: Summary) =>
      latestBatch.current("restore", [{ id: item.id, revision: item.revision, title: item.title }]),
    [],
  );
  const columns = useContentsColumns({
    sortable,
    taxonomies,
    trashed,
    chosenTerms,
    onTerm,
    onDelete,
    onRestore,
  });

  const termOptions = useMemo(
    () =>
      Object.fromEntries(
        taxonomies.map((taxonomy) => [
          taxonomy,
          (terms[taxonomy] ?? []).map((item) => ({
            label: item.term,
            value: item.term,
            count: item.count,
          })),
        ]),
      ),
    [taxonomies, terms],
  );

  const unfiltered = !search.status?.length && !search.q && !search.terms?.length;
  // The counts are of what is not deleted, so the trash's facet goes without.
  const statusOptions = STATUSES.map((status) => ({
    label: t(`status.${status}` as Key),
    value: status,
    icon: statuses[status].icon,
    className: statuses[status].className,
    count: trashed ? undefined : counts[status],
  }));
  const total = page.data?.total;

  return (
    <>
      <AppHeader />

      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <PageTitle
          title={trashed ? t("status.trash") : kindLabel.many(kind)}
          description={t(trashed ? "list.trashNote" : "list.description", {
            kind: kindLabel.many(kind),
          })}
        >
          {trashed ? (
            <Button variant="outline" asChild>
              <Link to="/content/$kind" params={{ kind }}>
                <ArrowLeft />
                {t("list.backTo", { kind: kindLabel.many(kind) })}
              </Link>
            </Button>
          ) : (
            <>
              <Button variant="outline" asChild>
                <Link to="/content/$kind" params={{ kind }} search={{ trash: true }}>
                  <Trash2 />
                  {t("status.trash")}
                  {counts.trash ? (
                    <span className="text-muted-foreground tabular-nums">{counts.trash}</span>
                  ) : null}
                </Link>
              </Button>
              <Button asChild>
                <Link to="/content/$kind/$id" params={{ kind, id: "new" }}>
                  <Plus />
                  {t("list.newKind", { kind: kindLabel.one(kind) })}
                </Link>
              </Button>
            </>
          )}
        </PageTitle>

        <IndexProblems />

        {batchResult && (
          <Alert variant="destructive">
            <XCircle />
            <AlertTitle>
              {t("list.partial", {
                done: batchResult.result.succeeded.length,
                failed: batchResult.result.failed.length,
              })}
            </AlertTitle>
            <AlertDescription>
              <ul className="list-inside list-disc">
                {batchResult.result.failed.map(({ target, error }) => {
                  const said = problem(
                    error instanceof Error && "code" in error ? String(error.code) : undefined,
                    error.message,
                  );
                  return (
                    <li key={target.id}>
                      {target.title}: {said.title}
                    </li>
                  );
                })}
              </ul>
              <Button
                variant="outline"
                size="sm"
                className="mt-2"
                disabled={remove.isPending || restore.isPending}
                onClick={reviewFailed}
              >
                {t("list.reviewFailed")}
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {page.error && !page.data ? (
          <QueryError error={page.error} onRetry={() => void page.refetch()} />
        ) : (
          <ContentsTable
            items={items}
            loading={page.isPending}
            columns={columns}
            search={search}
            onSearch={onSearch}
            sort={sort}
            taxonomies={taxonomies}
            statusOptions={statusOptions}
            termOptions={termOptions}
            rowSelection={rowSelection}
            onRowSelectionChange={setRowSelection}
            entityName={kindLabel.many(kind)}
            searchPlaceholder={t("list.searchKind", { kind: kindLabel.many(kind) })}
            empty={
              trashed && unfiltered ? (
                <span className="text-muted-foreground">{t("list.trashEmpty")}</span>
              ) : unfiltered ? (
                <div className="flex flex-col items-center gap-3">
                  <div>
                    <p className="font-medium">
                      {t("list.emptyKind", { kind: kindLabel.many(kind) })}
                    </p>
                    <p className="text-sm text-muted-foreground">{t("list.emptyNote")}</p>
                  </div>
                  <Button asChild size="sm">
                    <Link to="/content/$kind/$id" params={{ kind, id: "new" }}>
                      <Plus />
                      {t("list.newKind", { kind: kindLabel.one(kind) })}
                    </Link>
                  </Button>
                </div>
              ) : (
                <span className="text-muted-foreground">{t("list.empty")}</span>
              )
            }
            bulkActions={(selected) =>
              trashed ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => runBatch("restore", selected.map(targetOf))}
                >
                  <RotateCcw />
                  {t("list.restore")}
                </Button>
              ) : (
                <>
                  {canPublish(delivery.data) && (
                    <Button variant="outline" size="sm" onClick={() => setPublishing(selected)}>
                      <Upload />
                      {t("publish.action")}
                    </Button>
                  )}
                  <Button variant="destructive" size="sm" onClick={() => setConfirming(selected)}>
                    <Trash2 />
                    {t("editor.delete")}
                  </Button>
                </>
              )
            }
            pagination={{
              page: cursors.length + 1,
              pageSize,
              total,
              summary:
                total === undefined ? undefined : t("list.total", { count: total, kind: kindLabel.many(kind) }),
              hasPrevious: cursors.length > 0,
              hasNext: Boolean(page.data?.has_more && page.data.next_cursor),
              onFirst: () => turn([]),
              onPrevious: () => turn(cursors.slice(0, -1)),
              onNext: () => {
                if (page.data?.next_cursor) turn([...cursors, page.data.next_cursor]);
              },
              onPageSizeChange: (size) =>
                onSearch({ size: size === DEFAULT_PAGE_SIZE ? undefined : size }),
            }}
          />
        )}
      </Main>

      <ConfirmDialog
        open={confirming !== null}
        onOpenChange={(open) => !open && setConfirming(null)}
        title={t("list.confirmDelete", { count: confirming?.length ?? 0 })}
        desc={t("list.confirmDeleteNote")}
        confirmText={t("editor.delete")}
        destructive
        isLoading={remove.isPending}
        handleConfirm={() => confirming && runBatch("delete", confirming.map(targetOf))}
      />

      <PublishDialog
        ids={publishing?.map((item) => item.id) ?? []}
        open={publishing !== null}
        onOpenChange={(open) => !open && setPublishing(null)}
        onDone={() => setRowSelection({})}
      />
    </>
  );
}
