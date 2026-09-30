import { useMemo } from "react";

import type { ContentType, Draft, Item } from "@/api/client";
import { useI18n, type Key } from "@/i18n";
import { draftOf, type Conflict } from "@/hooks/useItem";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { compare, merge, regions, type Change, type Region, type Side } from "@/lib/threeway";
import { cn } from "@/lib/utils";
import { fieldLabel } from "@/components/SchemaForm";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface Props {
  conflict: Conflict;
  /** base is the version this editor loaded, which both sides started from. */
  base: Item | null;
  ours: Draft;
  type?: ContentType;
  onTakeTheirs: () => void;
  onKeepOurs: () => void;
  onMerge: (merged: Draft) => void;
  onCancel: () => void;
}

const sides: Record<Side, { label: Key; tone: string }> = {
  ours: { label: "conflict.sideOurs", tone: "" },
  theirs: { label: "conflict.sideTheirs", tone: "" },
  same: { label: "conflict.sideSame", tone: "text-muted-foreground" },
  apart: { label: "conflict.sideApart", tone: "text-muted-foreground" },
  both: { label: "conflict.sideBoth", tone: "border-destructive/30 text-destructive" },
};

/**
 * The three versions, when an edit lost a race: the one this editor loaded,
 * the one written here and the one stored meanwhile, part by part, with who
 * changed what. The author chooses: keep one side, or, when no part was
 * changed both ways, take both sides' changes and look them over before
 * saving. Nothing is written until then.
 */
export function ConflictDialog({ conflict, base, ours, type, onTakeTheirs, onKeepOurs, onMerge, onCancel }: Props) {
  const { t } = useI18n();
  const stored = useMemo(() => (conflict.theirs ? draftOf(conflict.theirs) : null), [conflict.theirs]);
  const started = useMemo(() => (base ? draftOf(base) : null), [base]);
  const changes = useMemo(
    () => (started && stored ? compare(started, ours, stored) : []),
    [started, ours, stored],
  );
  const merged = useMemo(
    () => (started && stored ? merge(started, ours, stored) : null),
    [started, ours, stored],
  );

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="grid max-h-[90svh] grid-rows-[auto_minmax(0,1fr)_auto] sm:max-w-5xl">
        <DialogHeader>
          <DialogTitle>{t("conflict.title")}</DialogTitle>
          <DialogDescription>{t("conflict.description")}</DialogDescription>
        </DialogHeader>

        <div className="min-h-0 overflow-y-auto">
          {!started || !stored ? (
            <Sides ours={ours} stored={stored} />
          ) : changes.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("conflict.noChanges")}</p>
          ) : (
            <Changes changes={changes} type={type} />
          )}
        </div>

        <DialogFooter className="items-center gap-2 sm:justify-between">
          <p className="text-xs text-muted-foreground">
            {stored && started ? t(merged ? "conflict.mergeNote" : "conflict.noMergeNote") : null}
          </p>
          <div className="flex flex-wrap justify-end gap-2">
            <Button variant="ghost" onClick={onCancel}>
              {t("conflict.keepEditing")}
            </Button>
            {stored && (
              <Button variant="outline" onClick={onTakeTheirs}>
                {t("conflict.takeTheirs")}
              </Button>
            )}
            <Button variant={merged ? "outline" : "default"} onClick={onKeepOurs}>
              {t("conflict.keepOurs")}
            </Button>
            {merged && <Button onClick={() => onMerge(merged)}>{t("conflict.merge")}</Button>}
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** Changes lays the parts that differ out side by side, the text's changed paragraphs below. */
function Changes({ changes, type }: { changes: Change[]; type?: ContentType }) {
  const { t } = useI18n();
  const taxonomyLabel = useTaxonomyLabel();
  const body = changes.find((change) => change.field.kind === "body");
  const labelOf = (change: Change): string => {
    const { field } = change;
    if (field.kind === "property") return t(`conflict.field.${field.key}` as Key);
    if (field.kind === "taxonomy") return taxonomyLabel(field.key);
    if (field.kind === "body") return t("conflict.field.body");
    const declared = type?.fields?.find((each) => each.key === field.key);
    return declared ? fieldLabel(declared, t) : field.key;
  };

  return (
    <div className="grid gap-5">
      <table className="w-full table-fixed text-sm">
        <thead className="text-xs text-muted-foreground">
          <tr className="border-b">
            <th className="w-28 py-2 pe-3 text-start font-medium">{t("conflict.part")}</th>
            <th className="py-2 pe-3 text-start font-medium">{t("conflict.base")}</th>
            <th className="py-2 pe-3 text-start font-medium">{t("conflict.ours")}</th>
            <th className="py-2 pe-3 text-start font-medium">{t("conflict.theirs")}</th>
            <th className="w-32 py-2 text-start font-medium" />
          </tr>
        </thead>
        <tbody>
          {changes.map((change) => {
            const key = `${change.field.kind}:${"key" in change.field ? change.field.key : ""}`;
            const oursChanged = change.side !== "theirs";
            const theirsChanged = change.side !== "ours";
            return (
              <tr key={key} className="border-b align-top last:border-0">
                <td className="py-2 pe-3 font-medium">{labelOf(change)}</td>
                <td className="py-2 pe-3 text-muted-foreground">
                  <Value change={change} value={change.base} />
                </td>
                <td className={cn("py-2 pe-3", oursChanged && "font-medium")}>
                  <Value change={change} value={change.ours} />
                </td>
                <td className={cn("py-2 pe-3", theirsChanged && "font-medium")}>
                  <Value change={change} value={change.theirs} />
                </td>
                <td className="py-2">
                  <Badge variant="outline" className={cn("font-normal", sides[change.side].tone)}>
                    {t(sides[change.side].label)}
                  </Badge>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {body && <Paragraphs change={body} />}
    </div>
  );
}

/** Value shows one version of a part as a form would, the text by what changed in it. */
function Value({ change, value }: { change: Change; value: unknown }) {
  const { t, date } = useI18n();
  const none = <span className="text-muted-foreground">—</span>;
  if (change.field.kind === "body") return <span>{t("conflict.textBelow")}</span>;
  if (value === undefined || value === null || value === "" || (Array.isArray(value) && value.length === 0)) {
    return none;
  }
  if (change.field.kind === "property" && change.field.key === "status") {
    return <span>{t(`status.${String(value)}` as Key)}</span>;
  }
  if (change.field.kind === "property" && change.field.key === "published_at") {
    return <span>{date(String(value), "long")}</span>;
  }
  if (typeof value === "boolean") return <span>{t(value ? "conflict.yes" : "conflict.no")}</span>;
  if (Array.isArray(value)) return <span className="break-words">{value.map(String).join(", ")}</span>;
  if (typeof value === "object") return <code className="text-xs break-all">{JSON.stringify(value)}</code>;
  return <span className="break-words">{String(value)}</span>;
}

/** Paragraphs are the stretches of the text a side changed, each side's version beside the other. */
function Paragraphs({ change }: { change: Change }) {
  const { t } = useI18n();
  const changed = useMemo(
    () =>
      regions(String(change.base ?? ""), String(change.ours ?? ""), String(change.theirs ?? "")).filter(
        (region) => text(region.ours) !== text(region.base) || text(region.theirs) !== text(region.base),
      ),
    [change],
  );

  return (
    <section className="grid gap-2">
      <h3 className="text-sm font-medium">{t("conflict.textChanges", { count: changed.length })}</h3>
      <div className="grid gap-2">
        {changed.map((region, i) => (
          <div key={i} className="grid gap-2 sm:grid-cols-3">
            <Stretch paragraphs={region.base} changed={false} />
            <Stretch paragraphs={region.ours} changed={text(region.ours) !== text(region.base)} />
            <Stretch paragraphs={region.theirs} changed={text(region.theirs) !== text(region.base)} />
          </div>
        ))}
      </div>
    </section>
  );
}

const text = (part: Region["base"]) => part.map((paragraph) => paragraph.trimEnd()).join("\n\n");

function Stretch({ paragraphs, changed }: { paragraphs: string[]; changed: boolean }) {
  const { t } = useI18n();
  const shown = text(paragraphs);
  return (
    <pre
      className={cn(
        "max-h-40 overflow-auto rounded-md border p-2 text-xs whitespace-pre-wrap",
        changed ? "border-primary/30 bg-primary/5" : "bg-muted/40 text-muted-foreground",
      )}
    >
      {shown || <span className="italic">{t("conflict.nothing")}</span>}
    </pre>
  );
}

/** Sides is the two versions whole, for when the one this editor started from cannot be compared. */
function Sides({ ours, stored }: { ours: Draft; stored: Draft | null }) {
  const { t } = useI18n();
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      {[
        { title: t("conflict.ours"), note: t("conflict.oursNote"), draft: ours },
        { title: t("conflict.theirs"), note: t("conflict.theirsNote"), draft: stored },
      ].map((side) => (
        <div key={side.title} className="min-w-0">
          <div className="mb-1 flex items-baseline gap-2">
            <span className="text-sm font-medium">{side.title}</span>
            <span className="text-xs text-muted-foreground">{side.note}</span>
          </div>
          <div className="mb-1 truncate text-sm">{side.draft?.title ?? ""}</div>
          <pre className="max-h-72 overflow-auto rounded-md border bg-muted/50 p-3 text-xs whitespace-pre-wrap">
            {side.draft?.body ?? t("conflict.unreadable")}
          </pre>
        </div>
      ))}
    </div>
  );
}
