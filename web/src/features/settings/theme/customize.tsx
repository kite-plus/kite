import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";
import { ChevronLeft, ExternalLink, House, MoreHorizontal, PanelRight, RotateCcw, XCircle } from "lucide-react";
import { toast } from "sonner";

import { ApiError, type Field } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useContentTypes, useLatest, useSite } from "@/hooks/useContents";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useFoldedSidebar } from "@/hooks/useFoldedSidebar";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { useSettingsDraft } from "@/hooks/useSettingsDraft";
import { useThemePreview } from "@/hooks/useThemePreview";
import { uploadSiteMedia, useSaveTheme, useTheme } from "@/hooks/useThemes";
import { useUnsavedGuard } from "@/hooks/useUnsavedGuard";
import { resolveLink, siteHome } from "@/lib/links";
import { changesOf, defaultOf, problemOf, sameValue, valueFields } from "@/lib/schema";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Header } from "@/components/layout/header";
import { SchemaForm, type Uploads } from "@/components/SchemaForm";
import { usePreviewWidth, WidthToggle } from "@/components/editor/Preview";
import { PreviewFrame, desktopWidth, type FrameHandle, type Page } from "./preview-frame";
import { search, SectionNav, type Flags, type Match } from "./section-nav";

/**
 * A theme's settings, a page of their own: the theme's sections listed on
 * one side, the one chosen as a form, and beside it, when wanted, the whole
 * site drawn with the settings as they change. Nothing reaches the site until
 * it is saved, so a theme not in use yet can be tried the same way, and put
 * in use with the settings chosen for it.
 */
export function ThemeCustomizer({ name, section }: { name: string; section?: string }) {
  const { t } = useI18n();
  const problem = useProblem();
  const navigate = useNavigate();
  const theme = useTheme(name);
  const detail = theme.data;
  useDocumentTitle(t("customize.title", { theme: detail?.title ?? name }));

  const form = useSettingsDraft(detail?.values, detail?.revision);
  const values = form.values;
  const save = useSaveTheme();
  const guard = useUnsavedGuard(form.dirty);
  const preview = useThemePreview(name, detail?.problem ? undefined : values);
  const uploads = useSiteUploads();
  const pages = usePreviewPages();

  const wide = useMediaQuery("(min-width: 768px)");
  const [open, setOpen] = usePreviewOpen();
  const showing = wide && open && !detail?.problem;
  useFoldedSidebar(showing);

  const [width, setWidth] = usePreviewWidth();
  const [page, setPage] = useState<Page | null>(null);
  const [scale, setScale] = useState(1);
  const [tab, setTab] = useState<"settings" | "preview">("settings");
  const [resetting, setResetting] = useState(false);
  const [query, setQuery] = useState("");
  const frame = useRef<FrameHandle>(null);
  const body = useRef<HTMLDivElement>(null);
  const formBox = useRef<HTMLDivElement>(null);
  const [formWidth, setFormWidth] = useFormWidth();

  const sections = useMemo(() => arrange(detail?.schema ?? [], t("customize.general")), [detail?.schema, t]);
  const fields = useMemo(() => valueFields(detail?.schema), [detail?.schema]);
  const current = sections.find((each) => each.key === section) ?? sections[0];
  const problems = values ? fields.filter((field) => problemOf(field, values[field.key], t)).length : 0;
  const trying = detail !== undefined && !detail.active;

  const flags = useMemo(() => {
    const out: Record<string, Flags> = {};
    for (const each of sections) {
      const own = valueFields(each.fields);
      out[each.key] = {
        dirty: !!values && !!form.baseline && own.some((field) => !sameValue(values[field.key], form.baseline![field.key])),
        problem: !!values && own.some((field) => problemOf(field, values[field.key], t) !== null),
      };
    }
    return out;
  }, [sections, values, form.baseline, t]);

  const matches = useMemo(() => search(sections, (each) => valueFields(each.fields), query), [sections, query]);

  const choose = (key: string) => {
    void navigate({ to: "/settings/theme/$name/{-$section}", params: { name, section: key }, replace: true });
    formBox.current?.scrollTo({ top: 0 });
  };

  // A setting found by a search is brought into view once its section is on
  // show, and marked for a moment.
  const [found, setFound] = useState<string | null>(null);
  const pick = (match: Match) => {
    setQuery("");
    if (match.section.key !== current?.key) choose(match.section.key);
    setFound(match.field.key);
    if (!wide) setTab("settings");
  };
  useEffect(() => {
    if (!found) return;
    const row = formBox.current?.querySelector<HTMLElement>(`[data-section-form] > div > [data-field="${CSS.escape(found)}"]`);
    if (!row) return;
    row.scrollIntoView({ block: "center" });
    // The field's own control, rather than the button that resets it.
    const control =
      row.querySelector<HTMLElement>(`#${CSS.escape(found)}:not([type=file])`) ??
      row.querySelector<HTMLElement>("input:not([type=file]), textarea, [role=switch], [role=combobox], button:not([aria-label])");
    control?.focus({ preventScroll: true });
    row.classList.remove("kite-found");
    void row.offsetWidth;
    row.classList.add("kite-found");
    window.setTimeout(() => row.classList.remove("kite-found"), 1600);
    setFound(null);
  }, [found, current?.key]);

  // The preview goes to the page the section is about, when it names one:
  // the page last looked at while in it, or else the one the theme names.
  const visited = useRef<Record<string, string>>({});
  const onPage = (next: Page) => {
    setPage(next);
    if (current?.preview && preview.url) visited.current[current.key] = pathInside(next.path, preview.url);
  };
  const wanted = current?.preview ? (visited.current[current.key] ?? pages(current.preview)) : null;
  useEffect(() => {
    if (!wanted || !preview.url) return;
    if (page && pathInside(page.path, preview.url) === wanted) return;
    frame.current?.open(inPreview(preview.url, wanted));
  }, [current?.key, preview.url, wanted]);

  const submit = () => {
    if (!detail || !values) return;
    const changes = changesOf("theme.settings.", fields, detail.stored, values, form.baseline ?? {});
    if (trying) changes["theme.name"] = name;
    save.mutate(
      { changes, revision: form.revision ?? detail.revision },
      {
        onSuccess: (result) => {
          form.saved(values, result.revision);
          toast.success(trying ? t("customize.usedSaved", { theme: detail.title }) : t("customize.saved"));
        },
      },
    );
  };

  const reset = (key: string) => {
    const field = fields.find((each) => each.key === key);
    if (field && values) form.change({ ...values, [key]: defaultOf(field) });
  };

  const failure = save.error ?? (theme.error && detail ? theme.error : null);
  const said = failure
    ? failure instanceof ApiError && failure.code === "conflict"
      ? { title: t("settings.conflict"), detail: undefined }
      : problem(failure instanceof ApiError ? failure.code : undefined, failure.message)
    : null;
  const conflict = save.error instanceof ApiError && save.error.code === "conflict";

  const state = save.isPending
    ? t("editor.saving")
    : form.dirty
      ? t("customize.unsaved")
      : trying
        ? t("customize.trying")
        : t("customize.inUse");

  if (!detail) {
    return (
      <div data-layout="fixed" className="flex min-h-0 min-w-0 flex-1 flex-col">
        <Header>
          <BackButton />
          <span className="text-sm font-semibold">{name}</span>
        </Header>
        {theme.error ? (
          <div className="p-6">
            <Alert variant="destructive">
              <XCircle />
              <AlertTitle>
                {theme.error instanceof ApiError && theme.error.code === "not_found"
                  ? t("customize.missing")
                  : problem(theme.error instanceof ApiError ? theme.error.code : undefined, theme.error.message).title}
              </AlertTitle>
            </Alert>
          </div>
        ) : (
          <div className="flex flex-1 items-center justify-center py-24">
            <Spinner />
          </div>
        )}
      </div>
    );
  }

  const alerts = (
    <>
      {said && (
        <Alert variant="destructive">
          <XCircle />
          <AlertTitle>{said.title}</AlertTitle>
          <AlertDescription>
            {said.detail && <p>{said.detail}</p>}
            {conflict && (
              <Button
                variant="outline"
                size="sm"
                className="mt-2"
                onClick={async () => {
                  const result = await theme.refetch();
                  if (result.data) form.reset(result.data.values, result.data.revision);
                  save.reset();
                }}
              >
                {t("settings.reload")}
              </Button>
            )}
          </AlertDescription>
        </Alert>
      )}
      {detail.problem && (
        <Alert variant="destructive">
          <XCircle />
          <AlertTitle>{t("customize.unusable")}</AlertTitle>
          <AlertDescription>{detail.problem}</AlertDescription>
        </Alert>
      )}
    </>
  );

  const sectionForm = !values ? (
    <Skeleton className="h-72 w-full" />
  ) : !current ? (
    <div className="rounded-lg border border-dashed p-6 text-center">
      <p className="font-medium">{t("customize.noSettings")}</p>
      <p className="text-sm text-muted-foreground">{t("customize.noSettingsNote")}</p>
    </div>
  ) : (
    <div className="grid gap-6">
      <div>
        <h2 className="text-lg font-semibold">{current.label || current.key}</h2>
        {current.help && <p className="mt-1 text-sm text-muted-foreground">{current.help}</p>}
      </div>
      <div data-section-form>
        <SchemaForm
          key={current.key}
          fields={current.fields ?? []}
          values={values}
          onChange={form.change}
          uploads={uploads}
          onReset={reset}
        />
      </div>
    </div>
  );

  const nav = (
    <SectionNav
      sections={sections}
      current={current?.key}
      flags={flags}
      query={query}
      matches={matches}
      onQuery={setQuery}
      onChoose={(key) => {
        choose(key);
        if (!wide) setTab("settings");
      }}
      onPick={pick}
      className="w-48 shrink-0 border-e"
    />
  );

  const previewPanel = (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      {/* The toolbar's height in the editor, so the two screens line up. */}
      <div className="flex h-[49px] shrink-0 items-center gap-1 border-b px-3.5">
        <IconButton label={t("customize.home")} onClick={() => frame.current?.home()}>
          <House />
        </IconButton>
        <span className="me-auto min-w-0 truncate text-[13px]">
          <span className="font-medium">{page?.title || t("customize.previewTitle")}</span>
          {page && (
            <span className="ms-2 font-mono text-xs text-muted-foreground">{pathInside(page.path, preview.url)}</span>
          )}
        </span>
        {width === "desktop" && scale < 1 && (
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="px-1 font-mono text-xs text-muted-foreground tabular-nums">{Math.round(scale * 100)}%</span>
            </TooltipTrigger>
            <TooltipContent>{t("customize.scaled", { width: desktopWidth })}</TooltipContent>
          </Tooltip>
        )}
        <WidthToggle width={width} onChange={setWidth} />
        <IconButton
          label={t("customize.openTab")}
          onClick={() => {
            const href = frame.current?.href();
            if (href) window.open(href, "_blank", "noopener");
          }}
        >
          <ExternalLink />
        </IconButton>
      </div>
      {preview.failed && (
        <Alert variant="destructive" className="rounded-none border-x-0 border-t-0">
          <XCircle />
          <AlertTitle>{t("customize.previewFailed")}</AlertTitle>
          <AlertDescription>{preview.failed.message}</AlertDescription>
        </Alert>
      )}
      <PreviewFrame
        ref={frame}
        url={preview.url}
        drawn={preview.drawn}
        phone={width === "phone"}
        title={t("customize.previewTitle")}
        onPage={onPage}
        onScale={setScale}
      />
    </div>
  );

  return (
    <div data-layout="fixed" className="flex min-h-0 min-w-0 flex-1 flex-col">
      <Header>
        <BackButton />
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-semibold">{t("customize.title", { theme: detail.title })}</div>
          <div className={cn("truncate text-xs text-muted-foreground", form.dirty && "text-warning")}>
            {problems > 0 ? (
              <span className="text-destructive">{t("customize.problems", { count: problems })}</span>
            ) : (
              [detail.version, state].filter(Boolean).join(" · ")
            )}
          </div>
        </div>

        {wide && !detail.problem && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant={open ? "secondary" : "outline"}
                size="sm"
                aria-pressed={open}
                onClick={() => setOpen(!open)}
                className={cn(open && "border border-primary/25 bg-primary/10 text-primary hover:bg-primary/15")}
              >
                <PanelRight />
                {t("customize.preview")}
              </Button>
            </TooltipTrigger>
            <TooltipContent>{open ? t("customize.hidePreview") : t("customize.showPreview")}</TooltipContent>
          </Tooltip>
        )}
        {!detail.problem && values && fields.length > 0 && (
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="size-8 shrink-0" aria-label={t("themes.more")}>
                <MoreHorizontal />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onSelect={() => setResetting(true)}>
                <RotateCcw />
                {t("customize.resetAll")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
        <Button
          size="sm"
          disabled={!values || problems > 0 || save.isPending || !!detail.problem || (!form.dirty && !trying)}
          onClick={submit}
        >
          {save.isPending && <Spinner />}
          {trying ? t("customize.use") : t("customize.save")}
        </Button>
      </Header>

      {wide ? (
        <div ref={body} className="flex min-h-0 flex-1">
          {sections.length > 0 && nav}
          <div
            ref={formBox}
            className={cn("min-h-0 overflow-y-auto", showing ? "shrink-0" : "min-w-0 flex-1")}
            style={showing ? { width: formWidth } : undefined}
          >
            <div className={cn("grid gap-4 p-6 [&>*]:min-w-0", !showing && "mx-auto max-w-3xl")}>
              {alerts}
              {sectionForm}
            </div>
          </div>
          {showing && (
            <>
              <Splitter
                width={formWidth}
                onChange={setFormWidth}
                limit={() => (body.current ? body.current.clientWidth - 192 - 360 : 760)}
              />
              <section className="flex min-w-0 flex-1">{previewPanel}</section>
            </>
          )}
        </div>
      ) : (
        <>
          <div role="tablist" className="flex shrink-0 gap-1 border-b p-2">
            {(["settings", "preview"] as const).map((each) => (
              <button
                key={each}
                type="button"
                role="tab"
                aria-selected={tab === each}
                onClick={() => setTab(each)}
                className="flex-1 rounded-md py-1.5 text-sm text-muted-foreground aria-selected:bg-muted aria-selected:font-medium aria-selected:text-foreground"
              >
                {t(each === "settings" ? "customize.settingsTab" : "customize.previewTab")}
              </button>
            ))}
          </div>
          <div ref={formBox} className={cn("min-h-0 flex-1 overflow-y-auto", tab !== "settings" && "hidden")}>
            <div className="grid gap-4 p-4 [&>*]:min-w-0">
              {sections.length > 1 && (
                <Select value={current?.key} onValueChange={(key) => key && choose(key)}>
                  <SelectTrigger aria-label={t("customize.sections")} className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {sections.map((each) => (
                      <SelectItem key={each.key} value={each.key}>
                        {each.label || each.key}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
              {alerts}
              {sectionForm}
            </div>
          </div>
          {!detail.problem && <section className={cn("flex min-h-0 flex-1", tab !== "preview" && "hidden")}>{previewPanel}</section>}
        </>
      )}

      <ConfirmDialog
        open={resetting}
        onOpenChange={setResetting}
        title={t("customize.resetAll")}
        desc={t("customize.resetNote")}
        confirmText={t("form.resetDefault")}
        handleConfirm={() => {
          setResetting(false);
          if (values) {
            form.change({ ...values, ...Object.fromEntries(fields.map((field) => [field.key, defaultOf(field)])) });
          }
        }}
      />
      <ConfirmDialog
        open={guard.leaving}
        onOpenChange={(next) => !next && guard.cancel()}
        title={t("editor.discardTitle")}
        desc={t("settings.discardNote")}
        cancelBtnText={t("conflict.keepEditing")}
        confirmText={t("editor.discard")}
        destructive
        handleConfirm={guard.discard}
      />
    </div>
  );
}

function BackButton() {
  const { t } = useI18n();
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8 shrink-0" asChild>
          <Link to="/settings/theme" aria-label={t("customize.back")}>
            <ChevronLeft />
          </Link>
        </Button>
      </TooltipTrigger>
      <TooltipContent>{t("customize.back")}</TooltipContent>
    </Tooltip>
  );
}

function IconButton({ label, onClick, children }: { label: string; onClick: () => void; children: React.ReactNode }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8 shrink-0" aria-label={label} onClick={onClick}>
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

/**
 * Splitter sets the form's width by dragging the rule between it and the
 * preview, or with the arrow keys. limit is the widest the form may be.
 */
function Splitter({ width, onChange, limit }: { width: number; onChange: (width: number) => void; limit: () => number }) {
  const { t } = useI18n();
  const fit = (next: number) => Math.round(Math.min(Math.max(next, minForm), Math.max(minForm, limit())));
  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label={t("customize.resize")}
      aria-valuenow={width}
      tabIndex={0}
      title={t("customize.resize")}
      onPointerDown={(event) => {
        event.preventDefault();
        const handle = event.currentTarget;
        // Captured, the pointer keeps reporting here even over the preview's frame.
        try {
          handle.setPointerCapture(event.pointerId);
        } catch {
          // A pointer already gone cannot be captured; the drag still works off the frame.
        }
        const from = event.clientX;
        const start = width;
        const move = (next: PointerEvent) => onChange(fit(start + next.clientX - from));
        const up = () => {
          handle.removeEventListener("pointermove", move);
          handle.removeEventListener("pointerup", up);
          handle.removeEventListener("pointercancel", up);
        };
        handle.addEventListener("pointermove", move);
        handle.addEventListener("pointerup", up);
        handle.addEventListener("pointercancel", up);
      }}
      onKeyDown={(event) => {
        if (event.key === "ArrowLeft") onChange(fit(width - 24));
        if (event.key === "ArrowRight") onChange(fit(width + 24));
      }}
      className="group relative z-10 w-px shrink-0 cursor-col-resize touch-none bg-border outline-none after:absolute after:inset-y-0 after:-inset-x-1.5 after:content-[''] focus-visible:bg-primary"
    >
      <span className="absolute top-1/2 left-1/2 h-8 w-1 -translate-1/2 rounded-full bg-border group-hover:bg-muted-foreground/40" />
    </div>
  );
}

const minForm = 380;

function useFormWidth(): [number, (width: number) => void] {
  const key = "kite.theme.formWidth";
  const [width, setWidth] = useState(() => {
    const stored = Number(read(key));
    return Number.isFinite(stored) && stored >= minForm ? stored : 520;
  });
  const choose = (next: number) => {
    setWidth(next);
    write(key, String(next));
  };
  return [width, choose];
}

/**
 * usePreviewOpen is whether the preview is shown beside the form: at first,
 * when the window is wide enough to have room for both, and after that as
 * the person last left it.
 */
function usePreviewOpen(): [boolean, (open: boolean) => void] {
  const key = "kite.theme.preview";
  const [open, setOpen] = useState(() => {
    const stored = read(key);
    if (stored === "open" || stored === "closed") return stored === "open";
    return typeof window.matchMedia === "function" && window.matchMedia("(min-width: 1280px)").matches;
  });
  const choose = (next: boolean) => {
    setOpen(next);
    write(key, next ? "open" : "closed");
  };
  return [open, choose];
}

/**
 * usePreviewPages finds the page of the site a section names: the home page,
 * the newest post, a page, or the list of posts. It answers with the page's
 * path in the site, or null while it is not known.
 */
function usePreviewPages(): (kind: string) => string | null {
  const post = useLatest("post", 1, "-published_at");
  const page = useLatest("page", 1, "-updated_at");
  const types = useContentTypes();
  const site = useSite();
  // An item's address carries the path the site is published under; the
  // preview draws the site at its own root.
  const home = siteHome(site.data);
  const inSite = (url: string | undefined) =>
    url === undefined ? null : home !== "/" && url.startsWith(home) ? "/" + url.slice(home.length) : url;
  return (kind) => {
    switch (kind) {
      case "home":
        return "/";
      case "post":
        return inSite(post.data?.items[0]?.url);
      case "page":
        return inSite(page.data?.items[0]?.url);
      case "posts": {
        const route = types.data?.items.find((each) => each.kind === "post")?.route;
        return route ? listOf(route) : null;
      }
      default:
        return null;
    }
  };
}

/** listOf is where a content type lists its items: its route up to the first parameter. */
function listOf(route: string): string {
  const fixed: string[] = [];
  for (const segment of route.split("/").filter(Boolean)) {
    if (segment.includes(":")) break;
    fixed.push(segment);
  }
  return fixed.length ? `/${fixed.join("/")}/` : "/";
}

/**
 * arrange puts the fields a theme declares outside any section into one of
 * their own, so every setting is in a section the list can show.
 */
function arrange(fields: Field[], general: string): Field[] {
  const out: Field[] = [];
  let loose: Field[] = [];
  const flush = () => {
    if (loose.length > 0) out.push({ key: `general-${out.length}`, type: "section", label: general, fields: loose });
    loose = [];
  };
  for (const field of fields) {
    if (field.type === "section") {
      flush();
      out.push(field);
    } else {
      loose.push(field);
    }
  }
  flush();
  return out;
}

/** pathInside is where a page is in the site, from its address in the preview. */
function pathInside(path: string, home: string | null): string {
  if (!home) return path;
  const root = new URL(home, window.location.origin).pathname;
  return path.startsWith(root) ? "/" + path.slice(root.length) : path;
}

/** inPreview is the address in the preview of a page of the site. */
function inPreview(home: string, path: string): string {
  const root = new URL(home, window.location.origin);
  return new URL(path.replace(/^\/+/, ""), root.href.endsWith("/") ? root : `${root.href}/`).href;
}

/**
 * useSiteUploads stores an image a setting names among the site's own files,
 * and shows one by where the site serves it, under its base path.
 */
function useSiteUploads(): Uploads {
  const site = useSite();
  const home = siteHome(site.data);
  return {
    upload: uploadSiteMedia,
    resolve: (link) => resolveLink(link, undefined, home),
  };
}

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Remembering the layout is a convenience, not a requirement.
  }
}
