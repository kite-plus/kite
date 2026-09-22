import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChainedCommands, Editor } from "@tiptap/react";
import {
  ChevronLeft,
  Eye,
  Info,
  Minus,
  SlidersHorizontal,
  Table,
  Type,
  Upload,
  XCircle,
} from "lucide-react";
import { toast } from "sonner";
import { cn } from "cn";

import { useI18n, useProblem, type Key } from "@/i18n";
import { useContentTypes } from "@/hooks/useContents";
import { useItem } from "@/hooks/useItem";
import { useKindLabel } from "@/hooks/useKindLabel";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { canPublish, useDelivery, usePublish } from "@/hooks/usePublish";
import { linkProps, navigate, setGuard } from "@/lib/router";

import { ConfirmDialog } from "@/components/ConfirmDialog";
import { StatusBadge } from "@/components/StatusDot";
import { ConflictDialog } from "@/components/editor/ConflictDialog";
import { EditorAside } from "@/components/editor/EditorAside";
import { EditorToolbar } from "@/components/editor/EditorToolbar";
import {
  countWords,
  losses,
  preferredMode,
  rememberMode,
  type Loss,
  type Mode,
} from "@/components/editor/markdown";
import { Preview } from "@/components/editor/Preview";
import { RichEditor } from "@/components/editor/RichEditor";
import type { SlashItem } from "@/components/editor/SlashMenu";
import { SourceEditor, type SourceHandle } from "@/components/editor/SourceEditor";
import { BlockquoteIcon } from "@/components/tiptap-icons/blockquote-icon";
import { CodeBlockIcon } from "@/components/tiptap-icons/code-block-icon";
import { HeadingFourIcon } from "@/components/tiptap-icons/heading-four-icon";
import { HeadingOneIcon } from "@/components/tiptap-icons/heading-one-icon";
import { HeadingThreeIcon } from "@/components/tiptap-icons/heading-three-icon";
import { HeadingTwoIcon } from "@/components/tiptap-icons/heading-two-icon";
import { ImagePlusIcon } from "@/components/tiptap-icons/image-plus-icon";
import { ListIcon } from "@/components/tiptap-icons/list-icon";
import { ListOrderedIcon } from "@/components/tiptap-icons/list-ordered-icon";
import { ListTodoIcon } from "@/components/tiptap-icons/list-todo-icon";
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
import { Toggle } from "@/components/ui/toggle";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

// The template's variables and animations, which its own install puts in the
// app's stylesheet; only this page needs them, so they load with it.
import "@/styles/_variables.scss";
import "@/styles/_keyframe-animations.scss";
import "@/components/editor/editor.scss";

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

  const [mode, setMode] = useState<Mode>(preferredMode);
  // What the source view is protecting, when a document opened in it for a reason.
  const [lost, setLost] = useState<Loss[]>([]);
  const [switching, setSwitching] = useState<Loss[] | null>(null);

  // State rather than a ref: the toolbar draws from it, and must redraw when
  // one editor is torn down and the other comes up.
  const [rich, setRich] = useState<Editor | null>(null);
  const source = useRef<SourceHandle | null>(null);
  const picker = useRef<HTMLInputElement>(null);
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

  // A document that uses what the visual editor would damage opens as
  // source, whatever this browser prefers. Decided once per document.
  const opened = useRef<string | null>(null);
  const body = useRef("");
  body.current = draft?.body ?? "";
  useEffect(() => {
    if (item.status === "loading" || item.status === "error") return;
    const key = id ?? "new";
    if (opened.current === key) return;
    // A first save gives a new item its id without reopening anything.
    if (opened.current === "new" && id) {
      opened.current = key;
      return;
    }
    opened.current = key;
    const found = losses(body.current);
    setLost(found);
    setMode(found.length > 0 ? "source" : preferredMode());
  }, [id, item.status]);

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

  /**
   * upload stores one file beside the item and resolves to its link. A new
   * item is saved first, since until then it has no folder to keep it in.
   */
  const upload = useCallback(
    async (
      file: File,
      onProgress?: (event: { progress: number }) => void,
      signal?: AbortSignal,
    ): Promise<string> => {
      setUploadError(null);
      setUploading((n) => n + 1);
      try {
        const target = id ?? (await save());
        if (!target) throw new Error(t("editor.saveFirst"));
        return await item.attach(file, target, (progress) => onProgress?.({ progress }), signal);
      } catch (err) {
        // A cancelled upload was the author's doing, not a failure to report.
        if (!signal?.aborted) setUploadError(err instanceof Error ? err.message : String(err));
        throw err;
      } finally {
        setUploading((n) => Math.max(0, n - 1));
      }
    },
    [id, save, item, t],
  );

  // Files chosen from the toolbar or dropped on the source view.
  const attach = async (files: File[]) => {
    for (const file of files) {
      let link: string;
      try {
        link = await upload(file);
      } catch {
        return;
      }
      const alt = file.name.replace(/\.[^.]+$/, "");
      if (rich) rich.chain().focus().setImage({ src: link, alt }).run();
      else source.current?.insert(`\n![${alt}](${link})\n`);
    }
  };
  const pick = () => picker.current?.click();

  const uploads = useMemo(
    () => ({ upload, base: item.base?.url }),
    [upload, item.base?.url],
  );

  const slash = useMemo<SlashItem[]>(() => {
    const item = (
      group: string,
      id: string,
      label: string,
      keywords: string,
      icon: SlashItem["icon"],
      run: (chain: ChainedCommands) => ChainedCommands,
    ): SlashItem => ({
      id,
      label,
      group,
      keywords,
      icon,
      run: (editor, range) => run(editor.chain().focus().deleteRange(range)).run(),
    });
    const style = t("editor.slash.style");
    const insert = t("editor.slash.insert");
    const headings = [HeadingOneIcon, HeadingTwoIcon, HeadingThreeIcon, HeadingFourIcon];
    return [
      item(style, "paragraph", t("editor.paragraph"), "text p 正文 段落", Type, (chain) => chain.setParagraph()),
      ...headings.map((icon, i) => {
        const level = (i + 1) as 1 | 2 | 3 | 4;
        return item(style, `h${level}`, t("editor.headingN", { level }), `h${level} heading title 标题`, icon, (chain) =>
          chain.setHeading({ level }),
        );
      }),
      item(style, "bulletList", t("editor.bulletList"), "ul bullet 列表 无序", ListIcon, (chain) => chain.toggleBulletList()),
      item(style, "orderedList", t("editor.orderedList"), "ol number 编号 有序", ListOrderedIcon, (chain) =>
        chain.toggleOrderedList(),
      ),
      item(style, "taskList", t("editor.taskList"), "todo task checkbox 任务 待办", ListTodoIcon, (chain) =>
        chain.toggleTaskList(),
      ),
      item(style, "quote", t("editor.quote"), "blockquote 引用", BlockquoteIcon, (chain) => chain.toggleBlockquote()),
      item(style, "codeBlock", t("editor.codeBlock"), "code pre 代码", CodeBlockIcon, (chain) => chain.toggleCodeBlock()),
      // The same drop zone the toolbar's image button puts in.
      item(insert, "image", t("editor.image"), "img picture photo 图片 上传", ImagePlusIcon, (chain) =>
        chain.insertContent({ type: "imageUpload" }),
      ),
      item(insert, "table", t("editor.table"), "grid 表格", Table, (chain) =>
        chain.insertTable({ rows: 3, cols: 3, withHeaderRow: true }),
      ),
      item(insert, "divider", t("editor.divider"), "hr rule line 分割线 分隔", Minus, (chain) => chain.setHorizontalRule()),
    ];
  }, [t]);

  const listOf = (found: Loss[]) =>
    new Intl.ListFormat(locale, { type: "conjunction" }).format(
      found.map((loss) => t(`editor.loss.${loss}` as Key)),
    );

  const switchMode = (next: Mode) => {
    if (next === mode) return;
    if (next === "visual") {
      const found = losses(body.current);
      if (found.length > 0) {
        setSwitching(found);
        return;
      }
    }
    rememberMode(next);
    setMode(next);
    setLost([]);
  };

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
  const settled = uploading === 0 && !saving && !item.dirty;

  const focusBody = () => {
    if (rich) rich.commands.focus("start");
    else source.current?.focus();
  };

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
      <header className="flex h-12 shrink-0 items-center gap-1.5 border-b px-2 sm:px-3">
        <SidebarTrigger className="md:hidden" />
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("editor.back")}
                onClick={() => navigate({ name: "list", kind })}
              />
            }
          >
            <ChevronLeft />
          </TooltipTrigger>
          <TooltipContent>{t("editor.back")}</TooltipContent>
        </Tooltip>

        <Breadcrumb className="min-w-0 flex-1">
          <BreadcrumbList className="flex-nowrap gap-1 sm:gap-1.5">
            <BreadcrumbItem className="shrink-0">
              <BreadcrumbLink render={<a {...linkProps({ name: "list", kind })} />}>
                {kindLabel.many(kind)}
              </BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
            <BreadcrumbItem className="min-w-0">
              <BreadcrumbPage className="truncate font-medium">
                {draft.title || t("editor.untitled")}
              </BreadcrumbPage>
            </BreadcrumbItem>
          </BreadcrumbList>
        </Breadcrumb>

        <div className="hidden min-w-0 items-center gap-1.5 text-xs text-muted-foreground md:flex">
          <span
            aria-hidden
            className={cn("size-1.5 shrink-0 rounded-full", settled ? "bg-success" : "bg-warning")}
          />
          <span className="truncate">{state}</span>
        </div>
        <StatusBadge status={draft.status} className="hidden sm:inline-flex" />
        <Separator
          orientation="vertical"
          className="mx-1 hidden data-vertical:h-4 data-vertical:self-center sm:block"
        />

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
        <div className={cn("kite-editor flex min-w-0 flex-1 flex-col", preview && "hidden md:flex")}>
          <EditorToolbar
            editor={mode === "visual" ? rich : null}
            mode={mode}
            onMode={switchMode}
            onPickImage={pick}
            base={item.base?.url}
          />

          {mode === "source" && lost.length > 0 && (
            <div className="flex items-center gap-2 border-b bg-muted/40 px-4 py-1.5 text-xs text-muted-foreground">
              <Info className="size-3.5 shrink-0" />
              <span className="min-w-0 flex-1 truncate">
                {t("editor.lossNote", { what: listOf(lost) })}
              </span>
              <Button
                variant="link"
                size="xs"
                className="h-auto p-0 text-brand"
                onClick={() => switchMode("visual")}
              >
                {t("editor.switchAnyway")}
              </Button>
            </div>
          )}

          <div className="min-h-0 flex-1 overflow-auto">
            <div className="kite-editor-page">
              {/* A textarea so a long title wraps; it still holds one line of text. */}
              <textarea
                rows={1}
                autoFocus={!id}
                value={draft.title}
                onChange={(event) => item.edit({ title: event.target.value.replace(/\n/g, " ") })}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    event.preventDefault();
                    focusBody();
                  }
                }}
                placeholder={t("editor.titlePlaceholder")}
                aria-label={t("editor.titlePlaceholder")}
                className="kite-title"
              />
              {mode === "visual" ? (
                <RichEditor
                  value={draft.body}
                  onChange={(body) => item.edit({ body })}
                  placeholder={{ empty: t("editor.bodyPlaceholder"), line: t("editor.slashHint") }}
                  base={item.base?.url}
                  slash={slash}
                  labels={{
                    slashEmpty: t("editor.slashEmpty"),
                    plain: t("editor.plainText"),
                    language: t("editor.language"),
                  }}
                  upload={upload}
                  onUploadError={setUploadError}
                  onReady={setRich}
                />
              ) : (
                <SourceEditor
                  value={draft.body}
                  onChange={(body) => item.edit({ body })}
                  placeholder={t("editor.bodyPlaceholder")}
                  onDropFiles={attach}
                  onReady={(handle) => {
                    source.current = handle;
                  }}
                />
              )}
            </div>
          </div>

          <footer className="flex h-8 shrink-0 items-center justify-between border-t px-4 text-xs text-muted-foreground">
            <span>{t("editor.words", { count: countWords(draft.body) })}</span>
            <span className="md:hidden">{state}</span>
            <span className="hidden md:inline">Markdown</span>
          </footer>
        </div>

        {preview && (
          <div className="min-w-0 flex-1 md:border-l">
            <Preview draft={draft} id={id} base={item.base?.url} />
          </div>
        )}

        {docked && <aside className="w-73 shrink-0 overflow-auto border-l">{aside}</aside>}
      </div>

      <input
        ref={picker}
        type="file"
        accept="image/*"
        multiple
        className="sr-only"
        tabIndex={-1}
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          if (files.length > 0) void attach(files);
          event.target.value = "";
        }}
      />

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
        open={switching !== null}
        onOpenChange={(open) => !open && setSwitching(null)}
        title={t("editor.switchTitle")}
        description={t("editor.switchNote", { what: listOf(switching ?? []) })}
        confirmLabel={t("editor.switchAnyway")}
        onConfirm={() => {
          setSwitching(null);
          rememberMode("visual");
          setMode("visual");
          setLost([]);
        }}
      />

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
