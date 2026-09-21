import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { Plus, Search, Trash2, Upload, X, XCircle } from "lucide-react";
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
import { useDeleteItems } from "@/hooks/useDeleteItems";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { canPublish, useDelivery } from "@/hooks/usePublish";
import { useListState } from "@/lib/listState";
import { navigate } from "@/lib/router";

import { ConfirmDialog } from "@/components/ConfirmDialog";
import { IndexProblems } from "@/components/IndexProblems";
import { ContentTable } from "@/components/content/ContentTable";
import { PublishDialog } from "@/components/publish/PublishDialog";
import { Page } from "@/components/shell/Page";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

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
      status: state.status === "all" ? undefined : state.status,
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
  const [publishing, setPublishing] = useState(false);

  useEffect(() => setSelected(new Set()), [filterKey]);

  const page = useContentPage(filters, cursors.at(-1));
  const items = page.data?.items ?? [];
  const total = page.data?.total ?? 0;
  const pages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const remove = useDeleteItems();
  const chosen = items.filter((item) => selected.has(item.id));

  const turn = (next: string[]) => {
    setPaging({ key: filterKey, cursors: next });
    setSelected(new Set());
  };

  const confirmDelete = () => {
    if (!confirming) return;
    remove.mutate(
      confirming.map((item) => ({ id: item.id, revision: item.revision })),
      {
        onSuccess: (count) => toast.success(t("list.deleted", { count })),
        onError: (error) => {
          const said = problem(
            error instanceof Error && "code" in error ? String(error.code) : undefined,
            error.message,
          );
          toast.error(said.title, { description: said.detail });
        },
        onSettled: () => {
          setConfirming(null);
          setSelected(new Set());
        },
      },
    );
  };

  return (
    <Page
      title={kindLabel.many(kind)}
      description={t("list.description", { kind: kindLabel.many(kind) })}
      actions={
        <Button onClick={() => navigate({ name: "edit", kind, id: null })}>
          <Plus data-icon="inline-start" />
          {t("list.newKind", { kind: kindLabel.one(kind) })}
        </Button>
      }
      toolbar={
        <div className="flex flex-wrap items-center justify-between gap-3">
          <Tabs value={state.status} onValueChange={(status) => set({ status: String(status) })}>
            <TabsList>
              {(["all", ...STATUSES] as const).map((status) => (
                <TabsTrigger key={status} value={status} className="px-2.5">
                  {status === "all" ? t("list.filterAny") : t(`status.${status}` as Key)}
                  <span className="text-xs tabular-nums opacity-60">{counts[status] ?? ""}</span>
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>

          {chosen.length > 0 ? (
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm text-muted-foreground">
                {t("list.selected", { count: chosen.length })}
              </span>
              {canPublish(delivery.data) && (
                <Button variant="outline" onClick={() => setPublishing(true)}>
                  <Upload data-icon="inline-start" />
                  {t("publish.action")}
                </Button>
              )}
              <Button variant="destructive" onClick={() => setConfirming(chosen)}>
                <Trash2 data-icon="inline-start" />
                {t("editor.delete")}
              </Button>
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
              {taxonomies.map((taxonomy) => (
                <Filter
                  key={taxonomy}
                  any={t("list.allOf", { name: taxonomyLabel(taxonomy) })}
                  value={state.terms[taxonomy]}
                  options={terms[taxonomy].map((item) => item.term)}
                  onChange={(term) => {
                    const next = { ...state.terms };
                    if (term) next[taxonomy] = term;
                    else delete next[taxonomy];
                    set({ terms: next });
                  }}
                />
              ))}

              {sortable.length > 1 && (
                <Select value={sort} onValueChange={(value) => set({ sort: String(value) })}>
                  <SelectTrigger className="w-auto min-w-28">
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

              <InputGroup className="w-52">
                <InputGroupAddon>
                  <Search />
                </InputGroupAddon>
                <InputGroupInput
                  value={state.q}
                  onChange={(event) => set({ q: event.target.value })}
                  placeholder={t("list.searchKind", { kind: kindLabel.many(kind) })}
                />
              </InputGroup>
            </div>
          )}
        </div>
      }
    >
      <div className="flex flex-col gap-3.5">
        <IndexProblems />

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
            onTerm={(taxonomy, term) => set({ terms: { ...state.terms, [taxonomy]: term } })}
            onDelete={(item) => setConfirming([item])}
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
      <SelectTrigger className="w-auto min-w-28">
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
