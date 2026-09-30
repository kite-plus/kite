import { useMemo, useRef, useState } from "react";
import { Copy, FileText, Paperclip, RefreshCw, Trash2, Upload } from "lucide-react";
import { toast } from "sonner";

import { ApiError, type Draft, type Media } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useBundle, useBundleChanged, useRemoveFile, useReplaceFile } from "@/hooks/useBundle";
import { useKindLabel } from "@/hooks/useKindLabel";
import { sizeOf } from "@/lib/bytes";
import { inFields, mentions } from "@/lib/references";
import { stored } from "@/lib/uploads";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { chip, unset } from "@/components/editor/ArticleHead";

/** altOf is a file's name without its folder and extension, which is the best guess there is. */
const altOf = (name: string) => name.replace(/^.*\//, "").replace(/\.[^.]+$/, "");

/** linkOf is how the markdown names a file, in angle brackets when it holds a space. */
const linkOf = (link: string) => (/\s/.test(link) ? `<${link}>` : link);

/**
 * Attachments are the files an item keeps beside it, which its page is
 * published with: each with where the draft uses it, to copy the link to, to
 * replace in place, which keeps every link to it, or to remove. An item kept
 * as a single file has none of its own and shows no chip.
 */
export function Attachments({
  id,
  kind,
  draft,
  upload,
}: {
  id: string;
  kind: string;
  draft: Draft;
  /** upload stores a file beside the item, as dropping it on the text does. */
  upload: (file: File) => Promise<string>;
}) {
  const { t, locale } = useI18n();
  const problem = useProblem();
  const kindLabel = useKindLabel();
  const files = useBundle(id);
  const changed = useBundleChanged(id);
  const replace = useReplaceFile(id);
  const remove = useRemoveFile(id);
  const [open, setOpen] = useState(false);
  const [adding, setAdding] = useState(false);
  const [removing, setRemoving] = useState<Media | null>(null);
  // A replaced picture keeps its address, so the new one is asked for anew.
  const [renewed, setRenewed] = useState<Record<string, number>>({});
  const picker = useRef<HTMLInputElement>(null);
  const replacer = useRef<HTMLInputElement>(null);
  const replacing = useRef<Media | null>(null);

  const fields = useMemo(() => inFields(draft.meta ?? {}), [draft.meta]);
  const list = new Intl.ListFormat(locale, { type: "conjunction" });

  if (files.data && !files.data.bundle) return null;
  const items = files.data?.items ?? [];

  const uses = (media: Media) => {
    const out: string[] = [];
    if (mentions(draft.body ?? "", media.link)) out.push(t("editor.filesInBody"));
    const field = fields.get(media.link);
    if (field) out.push(field === "cover" ? t("editor.filesCover") : t("editor.filesInField"));
    return out;
  };

  const failed = (err: unknown) =>
    toast.error(
      err instanceof ApiError ? problem(err.code, err.message).title : err instanceof Error ? err.message : String(err),
    );

  const copy = async (media: Media) => {
    const link = linkOf(media.link);
    const text = media.type?.startsWith("image/") ? `![${altOf(media.name)}](${link})` : `[${media.name}](${link})`;
    try {
      await navigator.clipboard.writeText(text);
      toast.success(t("editor.filesCopied", { name: media.name }));
    } catch {
      // A browser may refuse the clipboard; the link is still there to copy.
      toast.error(t("editor.filesCopyFailed"), { description: text });
    }
  };

  const add = async (chosen: File[]) => {
    setAdding(true);
    try {
      for (const file of chosen) await upload(file);
    } catch {
      // The editor has said what went wrong.
    } finally {
      setAdding(false);
      void changed();
    }
  };

  const put = (media: Media, file: File) =>
    replace.mutate(
      { name: media.name, file },
      {
        onSuccess: (done) => {
          stored(done, t);
          toast.success(t("editor.filesReplaced", { name: media.name }));
          setRenewed((all) => ({ ...all, [media.name]: Date.now() }));
        },
        onError: failed,
      },
    );

  const busy = adding || replace.isPending;
  const used = removing ? uses(removing) : [];

  return (
    <>
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <Button variant="outline" size="sm" className={cn(chip, items.length === 0 && unset)}>
            <Paperclip />
            {t("editor.files")}
            {items.length > 0 && <span className="text-muted-foreground tabular-nums">{items.length}</span>}
          </Button>
        </SheetTrigger>
        <SheetContent className="w-full gap-0 sm:max-w-md">
          <SheetHeader className="border-b">
            <SheetTitle>{t("editor.files")}</SheetTitle>
            <SheetDescription>{t("editor.filesNote", { kind: kindLabel.one(kind) })}</SheetDescription>
          </SheetHeader>
          <div className="flex items-center justify-between gap-2 border-b px-4 py-3">
            <span className="text-sm text-muted-foreground">
              {files.data ? t("editor.filesCount", { count: items.length }) : null}
            </span>
            <Button size="sm" variant="outline" disabled={busy} onClick={() => picker.current?.click()}>
              {adding ? <Spinner /> : <Upload />}
              {t("editor.filesUpload")}
            </Button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto">
            {files.isPending ? (
              <div className="grid gap-3 p-4">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            ) : items.length === 0 ? (
              <p className="p-6 text-center text-sm text-muted-foreground">{t("editor.filesEmpty")}</p>
            ) : (
              <ul className="divide-y">
                {items.map((media) => {
                  const where = uses(media);
                  return (
                    <li key={media.name} className="flex items-center gap-3 px-4 py-2.5">
                      <Thumb media={media} renewed={renewed[media.name]} />
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium" title={media.name}>
                          {media.name}
                        </p>
                        <p className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                          <span className="tabular-nums">{sizeOf(media.size)}</span>
                          {where.length > 0 ? (
                            where.map((use) => (
                              <Badge key={use} variant="secondary" className="px-1.5 py-0 text-[11px] font-normal">
                                {use}
                              </Badge>
                            ))
                          ) : (
                            <span>{t("editor.filesUnused")}</span>
                          )}
                        </p>
                      </div>
                      <div className="flex shrink-0 items-center">
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-8"
                          aria-label={t("editor.filesCopy")}
                          title={t("editor.filesCopy")}
                          onClick={() => void copy(media)}
                        >
                          <Copy />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-8"
                          aria-label={t("editor.filesReplace")}
                          title={t("editor.filesReplace")}
                          disabled={busy}
                          onClick={() => {
                            replacing.current = media;
                            if (replacer.current) {
                              replacer.current.accept = media.type || "";
                              replacer.current.click();
                            }
                          }}
                        >
                          {replace.isPending && replace.variables?.name === media.name ? <Spinner /> : <RefreshCw />}
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-8 text-destructive hover:text-destructive"
                          aria-label={t("editor.filesDelete")}
                          title={t("editor.filesDelete")}
                          onClick={() => setRemoving(media)}
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
        </SheetContent>
      </Sheet>

      <input
        ref={picker}
        type="file"
        multiple
        className="sr-only"
        tabIndex={-1}
        onChange={(event) => {
          const chosen = Array.from(event.target.files ?? []);
          event.target.value = "";
          if (chosen.length > 0) void add(chosen);
        }}
      />
      <input
        ref={replacer}
        type="file"
        className="sr-only"
        tabIndex={-1}
        onChange={(event) => {
          const file = event.target.files?.[0];
          event.target.value = "";
          if (file && replacing.current) put(replacing.current, file);
        }}
      />

      <ConfirmDialog
        open={removing !== null}
        onOpenChange={(next) => !next && setRemoving(null)}
        title={t("editor.filesDeleteTitle", { name: removing?.name ?? "" })}
        desc={
          <span className="grid gap-2">
            <span>{t("editor.filesDeleteNote", { kind: kindLabel.one(kind) })}</span>
            {used.length > 0 && (
              <span className="text-destructive">{t("editor.filesDeleteUsed", { where: list.format(used) })}</span>
            )}
          </span>
        }
        destructive
        confirmText={t("editor.filesDelete")}
        isLoading={remove.isPending}
        handleConfirm={() => {
          const media = removing;
          if (!media) return;
          remove.mutate(media.name, {
            onSuccess: () => {
              toast.success(t("editor.filesDeleted", { name: media.name }));
              setRemoving(null);
            },
            onError: failed,
          });
        }}
      />
    </>
  );
}

/** Thumb shows a picture small and any other file by its kind. */
function Thumb({ media, renewed }: { media: Media; renewed?: number }) {
  const picture = media.type?.startsWith("image/");
  return (
    <div className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-muted">
      {picture ? (
        <img
          src={renewed ? `${media.url}?v=${renewed}` : media.url}
          alt=""
          loading="lazy"
          className="size-full object-cover"
        />
      ) : (
        <FileText className="size-4 text-muted-foreground" />
      )}
    </div>
  );
}
