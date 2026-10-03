import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { ChainedCommands, Editor } from "@tiptap/react";
import { ChevronLeft, Eye, History, Info, Minus, Table, Type, X, XCircle } from "lucide-react";
import { Link, useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem, type Key } from "@/i18n";
import { useContentTypes, useSite, useWritable } from "@/hooks/useContents";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useFoldedSidebar } from "@/hooks/useFoldedSidebar";
import { useItem } from "@/hooks/useItem";
import { useKindLabel } from "@/hooks/useKindLabel";
import { canPublish, useDelivery, usePublish } from "@/hooks/usePublish";
import { useUnsavedGuard } from "@/hooks/useUnsavedGuard";
import { useWordCount } from "@/hooks/useWordCount";
import { siteHome } from "@/lib/links";
import { isoDate } from "@/lib/dates";
import { composing } from "@/lib/ime";
import { stored } from "@/lib/uploads";
import { cn } from "@/lib/utils";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { ReadOnlyNote } from "@/components/read-only-note";
import { AppHeader } from "@/components/layout/app-header";
import { Header } from "@/components/layout/header";
import { Main } from "@/components/layout/main";
import { CoverField, fieldsOf, Properties, SummaryField } from "@/components/editor/ArticleHead";
import { Attachments } from "@/components/editor/Attachments";
import { ConflictDialog } from "@/components/editor/ConflictDialog";
import { EditorToolbar } from "@/components/editor/EditorToolbar";
import {
  losses,
  preferredMode,
  rememberMode,
  type Loss,
  type Mode,
} from "@/components/editor/markdown";
import { Preview } from "@/components/editor/Preview";
import { PublishMenu } from "@/components/editor/PublishMenu";
import { RichEditor } from "@/components/editor/RichEditor";
import { renderedBlocks, useScrollSync } from "@/components/editor/scrollSync";
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
import { Button } from "@/components/ui/button";
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
  // Images named from the site's root are shown under its path, so the body
  // waits for the site as it does for the item.
  const site = useSite();
  const writable = useWritable();
  const home = siteHome(site.data);
  // A new item moves to its own address once its first save gives it an id,
  // whichever save that was: the button, a shortcut, an upload or its own.
  const moved = useRef<(saved: string) => void>(() => {});
  const item = useItem(id, kind, {
    site: site.data?.base_url,
    terms: types.data?.items.find((entry) => entry.kind === kind)?.new_terms,
    onCreated: (saved) => moved.current(saved),
  });
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

  // Writing wants the room; the sidebar folds to its icons meanwhile.
  useFoldedSidebar(true);

  const [preview, setPreview] = useState(false);
  const [uploading, setUploading] = useState(0);
  const [uploadError, setUploadError] = useState<string | null>(null);
  const [removing, setRemoving] = useState(false);
  // The conflict can be put aside to keep writing, and brought back.
  const [resolving, setResolving] = useState(false);
  // Leaving a draft saves it first; the question is asked only if that fails.
  const [leaveSaveFailed, setLeaveSaveFailed] = useState(false);

  const [mode, setMode] = useState<Mode>(preferredMode);
  // What the source view is protecting, when a document opened in it for a reason.
  const [lost, setLost] = useState<Loss[]>([]);
  const [switching, setSwitching] = useState<Loss[] | null>(null);

  // State rather than a ref: the toolbar draws from it, and must redraw when
  // one editor is torn down and the other comes up.
  const [rich, setRich] = useState<Editor | null>(null);
  const source = useRef<SourceHandle | null>(null);
  const title = useRef<HTMLTextAreaElement>(null);
  const summaryInput = useRef<HTMLTextAreaElement>(null);
  const scroller = useRef<HTMLDivElement>(null);
  const picker = useRef<HTMLInputElement>(null);
  const dirty = useRef(false);
  dirty.current = item.dirty;

  // Work that is not saved is asked about before anything navigates away.
  // It is kept in this browser too, so a session that ends can go to the
  // sign-in form without losing it.
  const guard = useUnsavedGuard(item.dirty, { kept: true });
  // Leaving saves a new draft on the way out; it is not reopened then.
  const leavingNow = useRef(false);
  leavingNow.current = guard.leaving;
  moved.current = (saved) => {
    if (id || leavingNow.current) return;
    guard.pass();
    void navigate({ to: "/content/$kind/$id", params: { kind, id: saved }, replace: true });
  };

  useEffect(() => {
    if (item.conflict) setResolving(true);
  }, [item.conflict]);

  // A draft saves itself, so leaving one saves what the last seconds held
  // rather than asking about it.
  const leaving = guard.leaving;
  useEffect(() => {
    if (!leaving) {
      setLeaveSaveFailed(false);
      return;
    }
    if (!item.autosaves) return;
    let cancelled = false;
    void item.save().then((saved) => {
      if (cancelled) return;
      if (saved) guard.discard();
      else setLeaveSaveFailed(true);
    });
    return () => {
      cancelled = true;
    };
    // Only a new attempt to leave starts this over.
  }, [leaving]);

  const draft = item.draft;
  useDocumentTitle(draft ? draft.title || t("editor.untitled") : undefined);
  const words = useWordCount(draft, id);
  // The preview follows the text as it scrolls, and the text the preview.
  const followPreview = useScrollSync(preview, scroller, () =>
    rich ? renderedBlocks(rich.view.dom) : (source.current?.blocks() ?? []),
  );
  const type = types.data?.items.find((entry) => entry.kind === (draft?.kind ?? kind));
  const { summary, cover, others } = fieldsOf(type);

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

  const save = item.save;

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
        return stored(await item.attach(file, target, (progress) => onProgress?.({ progress }), signal), t);
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
      // After what is selected rather than over it: a picture just put in is
      // selected, and the next one would replace it.
      if (rich) {
        const at = rich.state.selection.to;
        rich.chain().focus().insertContentAt(at, { type: "image", attrs: { src: link, alt } }).run();
      } else source.current?.insert(`\n![${alt}](${link})\n`);
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

  if (item.status === "loading" || site.isPending) {
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
  // A draft that saves itself reads as saving while it waits to; anything
  // else unsaved is said to be kept in this browser meanwhile.
  const state =
    uploading > 0
      ? t("editor.uploading", { count: uploading })
      : item.status === "conflict"
        ? t("editor.conflicted")
        : saving || (item.dirty && item.autosaves && !item.restored)
          ? t("editor.saving")
          : item.dirty
            ? t("editor.keptHere")
            : item.savedAt
              ? t("editor.savedAt", { time: clock.format(item.savedAt) })
              : id
                ? t("editor.saved")
                : t("editor.notSaved");
  const troubled =
    item.status === "conflict" || (item.dirty && !saving && (!item.autosaves || item.restored !== null));

  // A save from today reads as a time; an older one needs its date too.
  const stamp = (at: Date) =>
    at.toDateString() === new Date().toDateString()
      ? clock.format(at)
      : `${isoDate(at.toISOString())} ${clock.format(at)}`;
  const saved = item.savedAt ?? (item.base?.updated_at ? new Date(item.base.updated_at) : null);
  const lastSaved = saved ? t("editor.lastSaved", { time: stamp(saved) }) : "";

  const focusBody = () => {
    if (rich) rich.commands.focus("start");
    else source.current?.focus();
  };
  const focusTitle = () => {
    const field = title.current;
    field?.focus();
    field?.setSelectionRange(field.value.length, field.value.length);
  };
  const focusSummary = (at: "start" | "end") => {
    const field = summaryInput.current;
    const offset = at === "start" ? 0 : (field?.value.length ?? 0);
    field?.focus();
    field?.setSelectionRange(offset, offset);
  };
  // Moving between the title and the text stops at the summary on the way.
  const belowTitle = () => (summary ? focusSummary("start") : focusBody());
  const aboveText = () => (summary ? focusSummary("end") : focusTitle());

  const meta = draft.meta ?? {};
  const text = (key: string) => (typeof meta[key] === "string" ? (meta[key] as string) : "");
  const setMeta = (key: string, value: unknown) => item.edit({ meta: { ...meta, [key]: value } });

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
            {" · "}
            <span className={cn(troubled && "text-warning")}>{state}</span>
            {item.status === "conflict" && !resolving && (
              <>
                {" · "}
                <button type="button" className="underline-offset-2 hover:underline" onClick={() => setResolving(true)}>
                  {t("editor.resolve")}
                </button>
              </>
            )}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-1.5 sm:gap-2">
          <PublishMenu
            draft={draft}
            onEdit={item.edit}
            delivery={delivery.data}
            publish={publish}
            onDelete={id ? () => setRemoving(true) : undefined}
            disabled={!writable}
          />
          {/* On a phone the preview takes the editor's place, and shows by its icon alone. */}
          <Button
            variant="outline"
            size="sm"
            aria-pressed={preview}
            onClick={() => setPreview(!preview)}
            className="aria-pressed:bg-muted max-sm:px-2"
          >
            <Eye className="sm:hidden" />
            <span className="max-sm:sr-only">{t("editor.preview")}</span>
          </Button>

          {!writable ? null : canPublish(delivery.data) ? (
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
        </div>
      </Header>

      {!writable && (
        <div className="border-b px-4 py-2">
          <ReadOnlyNote />
        </div>
      )}

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

      {item.restored && (
        <div className="flex items-center gap-2 border-b bg-muted/40 px-4 py-1.5 text-xs text-muted-foreground">
          <History className="size-3.5 shrink-0" />
          <span className="min-w-0 flex-1 truncate">
            {t("editor.restored", { time: stamp(new Date(item.restored.at)) })}
          </span>
          <Button variant="link" size="sm" className="h-auto p-0 text-xs" onClick={item.discard}>
            {t("editor.restoredDiscard")}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="size-6"
            aria-label={t("editor.restoredKeep")}
            onClick={item.keepRestored}
          >
            <X className="size-3.5" />
          </Button>
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
            home={home}
            readOnly={!writable}
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

          <div ref={scroller} className="min-h-0 flex-1 overflow-auto">
            <div className="kite-editor-page">
              {/* The head reads as the item's page opens: cover, title, summary. */}
              <fieldset disabled={!writable} className="group/head">
                {cover && (
                  <CoverField
                    value={meta[cover.key] === false ? false : text(cover.key)}
                    onChange={(value) => setMeta(cover.key, value)}
                    upload={upload}
                    base={item.base?.url}
                    home={home}
                  />
                )}
                {/* A textarea so a long title wraps; it still holds one line of text. */}
                <textarea
                  ref={title}
                  rows={1}
                  autoFocus={!id}
                  value={draft.title}
                  onChange={(event) => item.edit({ title: event.target.value.replace(/\n/g, " ") })}
                  onKeyDown={(event) => {
                    // While an input method composes, Enter and the arrows pick
                    // its candidates.
                    if (composing(event)) return;
                    const field = event.currentTarget;
                    // Down leaves from the end of the title, as up comes back
                    // into it from the line below.
                    const atEnd =
                      field.selectionStart === field.value.length &&
                      field.selectionEnd === field.value.length;
                    if (event.key === "Enter" || (event.key === "ArrowDown" && atEnd && !event.shiftKey)) {
                      event.preventDefault();
                      belowTitle();
                    }
                  }}
                  placeholder={t("editor.titlePlaceholder")}
                  aria-label={t("editor.titlePlaceholder")}
                  className="kite-title"
                />
                {summary && (
                  <SummaryField
                    ref={summaryInput}
                    value={text(summary.key)}
                    // A summary cleared goes out as null, which takes it out of the file.
                    onChange={(value) => setMeta(summary.key, value || null)}
                    onExitDown={focusBody}
                    onExitUp={focusTitle}
                  />
                )}
                <Properties
                  draft={draft}
                  type={type}
                  fields={others}
                  uploads={uploads}
                  onEdit={item.edit}
                  files={id ? <Attachments id={id} kind={kind} draft={draft} upload={(file) => upload(file)} /> : null}
                />
              </fieldset>
              {mode === "visual" ? (
                <RichEditor
                  value={draft.body}
                  onChange={(body) => item.edit({ body })}
                  placeholder={{ empty: t("editor.bodyPlaceholder"), line: t("editor.slashHint") }}
                  base={item.base?.url}
                  home={home}
                  slash={slash}
                  labels={{
                    slashEmpty: t("editor.slashEmpty"),
                    plain: t("editor.plainText"),
                    language: t("editor.language"),
                  }}
                  upload={upload}
                  onUploadError={setUploadError}
                  onExitTop={aboveText}
                  onReady={setRich}
                  editable={writable}
                />
              ) : (
                <SourceEditor
                  value={draft.body}
                  onChange={(body) => item.edit({ body })}
                  placeholder={t("editor.bodyPlaceholder")}
                  onDropFiles={writable ? attach : undefined}
                  onExitTop={aboveText}
                  readOnly={!writable}
                  onReady={(handle) => {
                    source.current = handle;
                  }}
                />
              )}
            </div>
          </div>

          <footer className="flex shrink-0 items-center justify-between gap-3 border-t px-4 py-1.5 text-xs text-muted-foreground">
            <span className="truncate">
              {words !== undefined &&
                `${t("editor.words", { count: words, n: new Intl.NumberFormat(locale).format(words) })} · `}
              Markdown
            </span>
            <span className="shrink-0">{lastSaved}</span>
          </footer>
        </div>

        {preview && (
          <div className="min-w-0 flex-1 md:border-s">
            <Preview
              draft={draft}
              id={id}
              base={item.base?.url}
              live={item.base?.status === "published" ? item.base.url : undefined}
              onClose={() => setPreview(false)}
              onFrame={followPreview}
            />
          </div>
        )}
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

      {item.conflict && resolving && (
        <ConflictDialog
          conflict={item.conflict}
          base={item.base}
          ours={draft}
          type={type}
          onTakeTheirs={item.takeTheirs}
          onKeepOurs={item.keepOurs}
          onMerge={(merged) => {
            item.mergeWith(merged);
            setResolving(false);
          }}
          onCancel={() => setResolving(false)}
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
        open={guard.leaving && (!item.autosaves || leaveSaveFailed)}
        onOpenChange={(open) => !open && guard.cancel()}
        title={t("editor.discardTitle")}
        desc={t("editor.discardNote")}
        cancelBtnText={t("conflict.keepEditing")}
        confirmText={t("editor.discard")}
        destructive
        handleConfirm={() => {
          // Thrown away on purpose: the copy in this browser goes too.
          item.forgetUnsaved();
          guard.discard();
        }}
      />
    </div>
  );
}
