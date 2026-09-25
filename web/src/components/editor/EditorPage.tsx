import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChainedCommands, Editor } from "@tiptap/react";
import { ChevronLeft, Info, Minus, SlidersHorizontal, Table, Type, XCircle } from "lucide-react";
import { Link, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem, type Key } from "@/i18n";
import { useContentTypes } from "@/hooks/useContents";
import { useItem } from "@/hooks/useItem";
import { useKindLabel } from "@/hooks/useKindLabel";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { canPublish, useDelivery, usePublish } from "@/hooks/usePublish";
import { useUnsavedGuard } from "@/hooks/useUnsavedGuard";
import { isoDate } from "@/lib/dates";
import { cn } from "@/lib/utils";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { StatusLabel } from "@/components/StatusLabel";
import { AppHeader } from "@/components/layout/app-header";
import { Header } from "@/components/layout/header";
import { Main } from "@/components/layout/main";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Spinner } from "@/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

// The template's variables and animations, which its own install puts in the
// app's stylesheet; only this page needs them, so they load with it.
import "@/styles/_variables.scss";
import "@/styles/_keyframe-animations.scss";
import "@/components/editor/editor.scss";

export function EditorPage({ id, kind }: { id: string | null; kind: string }) {
  const { t, locale } = useI18n();
  const problem = useProblem();
  const navigate = useNavigate();
  const kindLabel = useKindLabel();
  const types = useContentTypes();
  const item = useItem(id, kind);
  const delivery = useDelivery();
  const publish = usePublish(
    id ? [id] : [],
    (result) => toast.success(t(result.rebased ? "publish.rebased" : "publish.done")),
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
  const guard = useUnsavedGuard(item.dirty);

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
      guard.pass();
      void navigate({ to: "/content/$kind/$id", params: { kind, id: saved }, replace: true });
    }
    return saved;
  }, [item, id, kind, guard, navigate]);

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
        if (!signal?.aborted) {
          setUploadError(
            err instanceof ApiError
              ? problem(err.code, err.message).title
              : err instanceof Error
                ? err.message
                : String(err),
          );
        }
        throw err;
      } finally {
        setUploading((n) => Math.max(0, n - 1));
      }
    },
    [id, save, item, t, problem],
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
      <>
        <Header />
        <div className="flex flex-1 items-center justify-center gap-2 py-24 text-sm text-muted-foreground">
          <Spinner />
          {t("editor.loading")}
        </div>
      </>
    );
  }
  if (!draft) {
    const said = item.error ? problem(item.error.code, item.error.detail) : null;
    return (
      <>
        <AppHeader />
        <Main className="flex flex-col gap-4">
          <Alert variant="destructive">
            <XCircle />
            <AlertTitle>{t("editor.nothingToEdit")}</AlertTitle>
            <AlertDescription>
              {said && [said.title, said.detail].filter(Boolean).join(" ")}
            </AlertDescription>
          </Alert>
          <Button variant="outline" className="self-start" asChild>
            <Link to="/content/$kind" params={{ kind }}>
              <ChevronLeft />
              {t("editor.back")}
            </Link>
          </Button>
        </Main>
      </>
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

  const words = countWords(draft.body);
  // A save from today reads as a time; an older one needs its date too.
  const stamp = (at: Date) =>
    at.toDateString() === new Date().toDateString()
      ? clock.format(at)
      : `${isoDate(at.toISOString())} ${clock.format(at)}`;
  const saved = savedAt ?? (item.base?.updated_at ? new Date(item.base.updated_at) : null);
  const lastSaved = saved ? t("editor.lastSaved", { time: stamp(saved) }) : "";

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
    // Fixed: the layout gives this page the viewport's height, and the text
    // scrolls inside it rather than the page.
    <div data-layout="fixed" className="flex min-h-0 min-w-0 flex-1 flex-col">
      <Header>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon" className="size-8 shrink-0" asChild>
              <Link to="/content/$kind" params={{ kind }} aria-label={t("editor.back")}>
                <ChevronLeft />
              </Link>
            </Button>
          </TooltipTrigger>
          <TooltipContent>{t("editor.back")}</TooltipContent>
        </Tooltip>

        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-semibold">{draft.title || t("editor.untitled")}</div>
          <div className="truncate text-xs text-muted-foreground">
            <Link to="/content/$kind" params={{ kind }} className="hover:text-foreground">
              {kindLabel.many(kind)}
            </Link>
            {" / "}
            {id ? t("editor.editing") : t("editor.creating")}
            {" · "}
            <span className={cn(!settled && "text-warning")}>{state}</span>
          </div>
        </div>

        <Badge variant="outline" className="hidden sm:inline-flex">
          <StatusLabel status={draft.status} className="gap-1.5 [&_svg]:size-3.5" />
        </Badge>

        <Button
          variant="outline"
          size="sm"
          aria-pressed={preview}
          onClick={() => setPreview(!preview)}
          className="hidden sm:inline-flex aria-pressed:bg-muted"
        >
          {t("editor.preview")}
        </Button>
        {!docked && (
          <Button
            variant="outline"
            size="icon"
            className="size-8 shrink-0"
            aria-label={t("editor.panel")}
            onClick={() => setPanel(true)}
          >
            <SlidersHorizontal />
          </Button>
        )}

        {canPublish(delivery.data) ? (
          <>
            <Button
              variant="outline"
              size="sm"
              disabled={!item.dirty || busy}
              onClick={() => void save()}
            >
              {saving && <Spinner />}
              {t("editor.save")}
            </Button>
            <Button size="sm" disabled={busy} onClick={() => void ship()}>
              {publish.pending && <Spinner />}
              {publish.needsConfirmation
                ? t("publish.anyway")
                : draft.status === "published"
                  ? t("publish.update")
                  : t("publish.action")}
            </Button>
          </>
        ) : (
          <Button size="sm" disabled={!item.dirty || busy} onClick={() => void save()}>
            {saving && <Spinner />}
            {t("editor.save")}
          </Button>
        )}
      </Header>

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
                size="sm"
                className="h-auto p-0 text-xs"
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

          <footer className="flex shrink-0 items-center justify-between gap-3 border-t px-4 py-1.5 text-xs text-muted-foreground">
            <span className="truncate">
              {t("editor.words", { count: words, n: new Intl.NumberFormat(locale).format(words) })}
              {" · Markdown"}
            </span>
            <span className="shrink-0">{lastSaved}</span>
          </footer>
        </div>

        {preview && (
          <div className="min-w-0 flex-1 md:border-s">
            <Preview draft={draft} id={id} base={item.base?.url} />
          </div>
        )}

        {docked && <aside className="w-80 shrink-0 overflow-auto border-s">{aside}</aside>}
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
        desc={t("editor.switchNote", { what: listOf(switching ?? []) })}
        confirmText={t("editor.switchAnyway")}
        handleConfirm={() => {
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
        desc={t("list.confirmDeleteNote")}
        confirmText={t("editor.delete")}
        destructive
        handleConfirm={async () => {
          setRemoving(false);
          if (await item.remove()) {
            toast.success(t("list.deleted", { count: 1 }));
            // What was not saved went with the item.
            guard.pass();
            void navigate({ to: "/content/$kind", params: { kind } });
          }
        }}
      />

      <ConfirmDialog
        open={guard.leaving}
        onOpenChange={(open) => !open && guard.cancel()}
        title={t("editor.discardTitle")}
        desc={t("editor.discardNote")}
        cancelBtnText={t("conflict.keepEditing")}
        confirmText={t("editor.discard")}
        destructive
        handleConfirm={guard.discard}
      />
    </div>
  );
}
