import { useEffect, useMemo, useRef, useState, type FormEvent } from "react";
import { Check } from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useTerm, useTermChanges } from "@/hooks/useTerm";
import { ConfirmDialog } from "@/components/confirm-dialog";
import {
  Command,
  CommandEmpty,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { StatusLabel } from "@/components/StatusLabel";

export type TermMode = "rename" | "merge" | "remove";

export interface TermAction {
  mode: TermMode;
  term: string;
}

/**
 * TermDialog renames, merges or deletes a term. Each is a change to every
 * item carrying the term, so the dialog lists those items first, and the
 * change is refused if any of them changed after the list was read.
 */
export function TermDialog({
  taxonomy,
  terms,
  one,
  many,
  action,
  onClose,
}: {
  taxonomy: string;
  /** Every term the taxonomy holds, to merge into and to spot a merge. */
  terms: { term: string; count: number }[];
  /** The kind of item that carries the terms, as one and as many. */
  one: string;
  many: string;
  action: TermAction;
  onClose: () => void;
}) {
  const { t, locale } = useI18n();
  const problem = useProblem();
  const detail = useTerm(taxonomy, action.term);
  const { rename, remove } = useTermChanges(taxonomy);
  const [value, setValue] = useState(action.mode === "rename" ? action.term : "");
  const input = useRef<HTMLInputElement>(null);

  // The dialog focuses its cancel button as it opens; a rename wants the name.
  useEffect(() => {
    if (action.mode !== "rename") return;
    const timer = setTimeout(() => {
      input.current?.focus();
      input.current?.select();
    });
    return () => clearTimeout(timer);
  }, [action.mode]);

  const { mode, term } = action;
  const target = value.trim();
  const merging = mode === "merge" || (target !== term && terms.some((other) => other.term === target));
  // The terms to merge into, by name, as a list to search rather than scroll.
  const others = useMemo(
    () =>
      terms
        .filter((other) => other.term !== term)
        .sort((a, b) => a.term.localeCompare(b.term, locale)),
    [terms, term, locale],
  );
  const valid = mode === "remove" || (target !== "" && target !== term);
  const pending = rename.isPending || remove.isPending;

  const confirm = async () => {
    const revision = detail.data?.revision;
    if (!revision || !valid || pending) return;
    try {
      if (mode === "remove") {
        await remove.mutateAsync({ term, revision });
        toast.success(t("terms.removed", { term }));
      } else {
        await rename.mutateAsync({ term, name: target, revision });
        toast.success(t(merging ? "terms.merged" : "terms.renamed", { from: term, to: target }));
      }
      onClose();
    } catch (error) {
      // The list refetches as the change settles, so the dialog stays open
      // on what is there now.
      if (error instanceof ApiError && error.code === "conflict") {
        toast.error(t("terms.changedMeanwhile"));
        return;
      }
      const said = problem(
        error instanceof ApiError ? error.code : undefined,
        error instanceof Error ? error.message : String(error),
      );
      toast.error(said.title, { description: said.detail });
    }
  };

  const submit = (event: FormEvent) => {
    event.preventDefault();
    void confirm();
  };

  const count = detail.data?.items.length ?? 0;
  const titles = {
    rename: t("terms.renameTitle", { term }),
    merge: t("terms.mergeTitle", { term }),
    remove: t("terms.removeTitle", { term }),
  };
  const notes = {
    rename: t("terms.renameNote", { kind: many }),
    merge: t("terms.mergeNote", { kind: one }),
    remove: t("terms.removeNote", { kind: many }),
  };

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => {
        if (!open && !pending) onClose();
      }}
      title={titles[mode]}
      desc={notes[mode]}
      destructive={mode === "remove"}
      confirmText={
        mode === "remove" ? t("terms.remove") : merging ? t("terms.mergeAction") : t("terms.rename")
      }
      isLoading={pending}
      disabled={!detail.data || !valid}
      handleConfirm={() => void confirm()}
    >
      {mode === "rename" && (
        <form onSubmit={submit} className="grid gap-2">
          <Label htmlFor="term-name">{t("terms.newName")}</Label>
          <Input
            ref={input}
            id="term-name"
            value={value}
            onChange={(event) => setValue(event.target.value)}
            autoComplete="off"
          />
          {merging && target !== "" && (
            <p className="text-sm text-warning">{t("terms.mergesInto", { term: target })}</p>
          )}
        </form>
      )}

      {mode === "merge" && (
        <div className="grid gap-2">
          <Label>{t("terms.mergeTarget")}</Label>
          <Command className="rounded-md border bg-background">
            <CommandInput placeholder={t("terms.mergePick")} />
            <CommandList className="max-h-40">
              <CommandEmpty>{t("list.empty")}</CommandEmpty>
              {others.map((other) => (
                <CommandItem
                  key={other.term}
                  value={other.term}
                  onSelect={() => setValue(other.term)}
                >
                  <span className="min-w-0 truncate">{other.term}</span>
                  <span className="ms-auto text-xs text-muted-foreground tabular-nums">{other.count}</span>
                  <Check className={value === other.term ? "text-primary" : "invisible"} />
                </CommandItem>
              ))}
            </CommandList>
          </Command>
        </div>
      )}

      {detail.isPending ? (
        <Skeleton className="h-24 w-full" />
      ) : detail.data ? (
        <div className="grid gap-1.5">
          <p className="text-sm font-medium">
            {t("terms.affected", { count, kind: count === 1 ? one : many })}
          </p>
          <ul className="max-h-48 divide-y overflow-auto rounded-md border text-sm">
            {detail.data.items.map((item) => (
              <li key={item.id} className="flex items-center justify-between gap-3 px-3 py-2">
                <span className="min-w-0 truncate">{item.title || item.slug}</span>
                {item.trashed ? (
                  <span className="shrink-0 text-xs text-muted-foreground">{t("status.trash")}</span>
                ) : (
                  <StatusLabel status={item.status} className="shrink-0 gap-1.5 text-xs [&_svg]:size-3.5" />
                )}
              </li>
            ))}
          </ul>
        </div>
      ) : detail.error ? (
        <p className="text-sm text-destructive">
          {
            problem(
              detail.error instanceof ApiError ? detail.error.code : undefined,
              detail.error.message,
            ).title
          }
        </p>
      ) : null}
    </ConfirmDialog>
  );
}
