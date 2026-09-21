import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ChevronLeft, Eye, SlidersHorizontal, Upload, XCircle } from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";

import { useI18n, useProblem } from "@/i18n";
import { useContentTypes } from "@/hooks/useContents";
import { useItem } from "@/hooks/useItem";
import { useKindLabel } from "@/hooks/useKindLabel";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { canPublish, useDelivery, usePublish } from "@/hooks/usePublish";
import { linkProps, navigate, setGuard } from "@/lib/router";

import { ConfirmDialog } from "@/components/ConfirmDialog";
import { StatusBadge } from "@/components/StatusDot";
import { ConflictDialog } from "@/components/editor/ConflictDialog";
import { Editor, type EditorHandle } from "@/components/editor/Editor";
import { EditorAside } from "@/components/editor/EditorAside";
import { EditorToolbar } from "@/components/editor/EditorToolbar";
import { Preview } from "@/components/editor/Preview";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import { Button } from "@/components/ui/button";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { Toggle } from "@/components/ui/toggle";

export function EditorPage({ id, kind }: { id: string | null; kind: string }) {
  const { t, locale } = useI18n();
  const problem = useProblem();
  const kindLabel = useKindLabel();
  const types = useContentTypes();
  const item = useItem(id, kind);
  const delivery = useDelivery();
  const publish = usePublish(
    id ? [id] : [],
    () => toast.success(t("publish.done")),
    // The panel that lists the reasons may be closed, so the headline is said here.
    (failure, confirmable) => {
      const said = problem(failure.code, failure.detail);
      (confirmable ? toast.warning : toast.error)(said.title, { description: said.detail });
    },
  );

  const [preview, setPreview] = useState(false);
  const [panel, setPanel] = useState(false);
  // The details sit beside the text where there is room, and in a sheet
  // where there is not; a preview takes the room they would have had.
  const docked = useMediaQuery(preview ? "(min-width: 1536px)" : "(min-width: 1024px)");
  const [uploading, setUploading] = useState(0);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<Date | null>(null);
  const [removing, setRemoving] = useState(false);
  const [leaving, setLeaving] = useState<(() => void) | null>(null);

  const editor = useRef<EditorHandle | null>(null);
  const dirty = useRef(false);
  dirty.current = item.dirty;

  // Work that is not saved is asked about before anything navigates away.
  useEffect(() => {
    setGuard({
      blocked: () => dirty.current,
      ask: (proceed) => setLeaving(() => proceed),
    });
    const warn = (event: BeforeUnloadEvent) => {
      if (dirty.current) event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => {
      setGuard(null);
      window.removeEventListener("beforeunload", warn);
    };
  }, []);

  const draft = item.draft;
  const type = types.data?.items.find((entry) => entry.kind === (draft?.kind ?? kind));

  const save = useCallback(async () => {
    const saved = await item.save();
    if (!saved) return null;
    setSavedAt(new Date());
    // A new item has no id until the server gives it one.
    if (!id) {
      dirty.current = false;
      navigate({ name: "edit", kind, id: saved }, { replace: true });
    }
    return saved;
  }, [item, id, kind]);

  // Read through a ref so the listener is not rebound on every keystroke.
  const saveRef = useRef(save);
  saveRef.current = save;
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() === "s" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        if (dirty.current) void saveRef.current();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const ship = async () => {
    const target = item.dirty || !id ? await save() : id;
    if (target) await publish.run([target]);
  };

  const attach = useCallback(
    async (files: File[]) => {
      setUploadError(null);
      setUploading((n) => n + files.length);
      try {
        for (const file of files) {
          const link = await item.attach(file);
          const alt = file.name.replace(/\.[^.]+$/, "");
          editor.current?.insert(`\n![${alt}](${link})\n`);
        }
      } catch (err) {
        setUploadError(err instanceof Error ? err.message : String(err));
      } finally {
        setUploading(0);
      }
    },
    [item],
  );

  const uploads = useMemo(
    () => ({
      upload: async (file: File) => {
        if (!id) throw new Error(t("editor.saveFirst"));
        return item.attach(file);
      },
      base: item.base?.url,
    }),
    [id, item, t],
  );

  if (item.status === "loading") {
    return (
      <Empty className="h-svh">
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <Spinner />
          </EmptyMedia>
          <EmptyTitle>{t("editor.loading")}</EmptyTitle>
        </EmptyHeader>
      </Empty>
    );
  }
  if (!draft) {
    return (
      <div className="flex flex-col gap-4 p-6">
        <Alert variant="destructive">
          <XCircle />
          <AlertTitle>{t("editor.nothingToEdit")}</AlertTitle>
          <AlertDescription>{item.error?.detail}</AlertDescription>
        </Alert>
        <Button variant="outline" className="self-start" onClick={() => navigate({ name: "list", kind })}>
          <ChevronLeft data-icon="inline-start" />
          {t("editor.back")}
        </Button>
      </div>
    );
  }

  const saving = item.status === "saving";
  const busy = saving || publish.pending;
  const clock = new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit" });
  const state =
    uploading > 0
      ? t("editor.uploading", { count: uploading })
      : saving
        ? t("editor.saving")
        : item.dirty
          ? t("editor.unsaved")
          : savedAt
            ? t("editor.savedAt", { time: clock.format(savedAt) })
            : t("editor.saved");

  const aside = (
    <EditorAside
      draft={draft}
      type={type}
      onEdit={item.edit}
      delivery={delivery.data}
      publish={publish}
      uploads={uploads}
      onDelete={id ? () => setRemoving(true) : undefined}
    />
  );

  return (
    <div className="flex h-svh min-w-0 flex-col">
      <header className="flex items-center gap-2.5 border-b px-3 py-2">
        <SidebarTrigger className="md:hidden" />
        <Button
          variant="ghost"
          size="icon"
          aria-label={t("editor.back")}
          onClick={() => navigate({ name: "list", kind })}
        >
          <ChevronLeft />
        </Button>

        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-semibold">
            {draft.title || t("editor.untitled")}
          </div>
          <div className="hidden items-center gap-1.5 text-xs text-muted-foreground sm:flex">
            <Breadcrumb>
              <BreadcrumbList className="gap-1 text-xs sm:gap-1">
                <BreadcrumbItem>
                  <BreadcrumbLink render={<a {...linkProps({ name: "list", kind })} />}>
                    {kindLabel.many(kind)}
                  </BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator>/</BreadcrumbSeparator>
                <BreadcrumbItem>
                  <BreadcrumbPage className="text-muted-foreground">
                    {id ? t("editor.editing") : t("editor.creating")}
                  </BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>
            <span aria-hidden>·</span>
            <span className="truncate">{state}</span>
          </div>
        </div>

        <div className="hidden items-center gap-2.5 sm:flex">
          <StatusBadge status={draft.status} />
          <Separator orientation="vertical" className="data-vertical:h-4 data-vertical:self-center" />
        </div>

        <Toggle
          variant="outline"
          size="sm"
          pressed={preview}
          onPressedChange={setPreview}
          className="hidden sm:inline-flex"
        >
          <Eye />
          {t("editor.preview")}
        </Toggle>
        {!docked && (
          <Button
            variant="outline"
            size="icon-sm"
            aria-label={t("editor.panel")}
            onClick={() => setPanel(true)}
          >
            <SlidersHorizontal />
          </Button>
        )}

        {canPublish(delivery.data) ? (
          <>
            <Button variant="outline" size="sm" disabled={!item.dirty || busy} onClick={() => void save()}>
              {saving && <Spinner data-icon="inline-start" />}
              {t("editor.save")}
            </Button>
            <Button size="sm" disabled={busy} onClick={() => void ship()}>
              {publish.pending ? (
                <Spinner data-icon="inline-start" />
              ) : (
                <Upload data-icon="inline-start" />
              )}
              {publish.needsConfirmation ? t("publish.anyway") : t("publish.action")}
            </Button>
          </>
        ) : (
          <Button size="sm" disabled={!item.dirty || busy} onClick={() => void save()}>
            {saving && <Spinner data-icon="inline-start" />}
            {t("editor.save")}
          </Button>
        )}
      </header>

      {(item.error || uploadError) && (
        <div className="border-b px-4 py-2">
          <Alert variant="destructive">
            <XCircle />
            <AlertTitle>
              {item.error ? problem(item.error.code, item.error.detail).title : uploadError}
            </AlertTitle>
            {item.error && (
              <AlertDescription>{problem(item.error.code, item.error.detail).detail}</AlertDescription>
            )}
          </Alert>
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        <div className={cn("flex min-w-0 flex-1 flex-col", preview && "hidden md:flex")}>
          <EditorToolbar
            editor={() => editor.current}
            format={item.base?.body_format ?? "markdown"}
            onFiles={id ? attach : undefined}
          />
          <div className="min-h-0 flex-1 overflow-auto">
            <div className="mx-auto max-w-[780px] px-6 pt-8 sm:px-10">
              {/* A textarea so a long title wraps; it still holds one line of text. */}
              <Textarea
                rows={1}
                value={draft.title}
                onChange={(event) => item.edit({ title: event.target.value.replace(/\n/g, " ") })}
                onKeyDown={(event) => {
                  if (event.key === "Enter") event.preventDefault();
                }}
                placeholder={t("editor.titlePlaceholder")}
                aria-label={t("editor.titlePlaceholder")}
                className="mb-4 min-h-0 resize-none rounded-none border-0 bg-transparent p-0 text-2xl font-bold tracking-tight shadow-none focus-visible:ring-0 md:text-2xl dark:bg-transparent"
              />
              <Editor
                value={draft.body}
                onChange={(body) => item.edit({ body })}
                placeholder={t("editor.bodyPlaceholder")}
                onDropFiles={id ? attach : undefined}
                onReady={(handle) => {
                  editor.current = handle;
                }}
              />
            </div>
          </div>
          <footer className="flex items-center justify-between border-t px-4 py-1.5 text-xs text-muted-foreground">
            <span>{t("editor.words", { count: countWords(draft.body) })}</span>
            <span>{state}</span>
          </footer>
        </div>

        {preview && (
          <div className="min-w-0 flex-1 md:border-l">
            <Preview draft={draft} id={id} />
          </div>
        )}

        {docked && <aside className="w-73 shrink-0 overflow-auto border-l">{aside}</aside>}
      </div>

      <Sheet open={panel && !docked} onOpenChange={setPanel}>
        <SheetContent className="overflow-auto">
          <SheetHeader>
            <SheetTitle>{t("editor.panel")}</SheetTitle>
            <SheetDescription className="sr-only">{t("editor.panel")}</SheetDescription>
          </SheetHeader>
          {aside}
        </SheetContent>
      </Sheet>

      {item.conflict && (
        <ConflictDialog
          conflict={item.conflict}
          ours={draft}
          onTakeTheirs={item.takeTheirs}
          onKeepOurs={item.keepOurs}
          onCancel={() => item.edit({})}
        />
      )}

      <ConfirmDialog
        open={removing}
        onOpenChange={setRemoving}
        title={t("list.confirmDelete", { count: 1 })}
        description={t("list.confirmDeleteNote")}
        confirmLabel={t("editor.delete")}
        destructive
        onConfirm={async () => {
          setRemoving(false);
          if (await item.remove()) {
            dirty.current = false;
            toast.success(t("list.deleted", { count: 1 }));
            navigate({ name: "list", kind });
          }
        }}
      />

      <ConfirmDialog
        open={leaving !== null}
        onOpenChange={(open) => !open && setLeaving(null)}
        title={t("editor.discardTitle")}
        description={t("editor.discardNote")}
        confirmLabel={t("editor.discard")}
        cancelLabel={t("conflict.keepEditing")}
        destructive
        onConfirm={() => {
          dirty.current = false;
          leaving?.();
          setLeaving(null);
        }}
      />
    </div>
  );
}

/** countWords counts a CJK character as a word, and a run of letters as one. */
function countWords(text: string): number {
  const cjk = /[぀-ヿ㐀-鿿가-힯]/g;
  const characters = text.match(cjk)?.length ?? 0;
  const words = text.replace(cjk, " ").match(/[\p{L}\p{N}]+/gu)?.length ?? 0;
  return characters + words;
}
