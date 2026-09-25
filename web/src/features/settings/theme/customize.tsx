import { useMemo, useRef, useState } from "react";
import { Link } from "@tanstack/react-router";
import { ChevronLeft, ExternalLink, House, MoreHorizontal, Palette, RotateCcw, XCircle } from "lucide-react";
import { toast } from "sonner";

import { ApiError, type Field, type ThemeDetail } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useSite } from "@/hooks/useContents";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useMediaQuery } from "@/hooks/useMediaQuery";
import { useSettingsDraft } from "@/hooks/useSettingsDraft";
import { useThemePreview } from "@/hooks/useThemePreview";
import { uploadSiteMedia, useSaveTheme, useTheme } from "@/hooks/useThemes";
import { useUnsavedGuard } from "@/hooks/useUnsavedGuard";
import { resolveLink, siteHome } from "@/lib/links";
import { defaultOf, problemOf, sameValue, valueFields } from "@/lib/schema";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Header } from "@/components/layout/header";
import { SchemaForm, type Uploads } from "@/components/SchemaForm";
import { usePreviewWidth, WidthToggle } from "@/components/editor/Preview";
import { PreviewFrame, type FrameHandle, type Page } from "./preview-frame";

/**
 * Customizing a theme: its settings on one side, the whole site drawn with
 * them on the other, redrawn as they change. Nothing reaches the site until
 * it is saved, so a theme not in use yet can be tried the same way, and put
 * in use with the settings chosen for it.
 */
export function ThemeCustomizer({ name }: { name: string }) {
  const { t } = useI18n();
  const problem = useProblem();
  const theme = useTheme(name);
  const detail = theme.data;
  useDocumentTitle(t("customize.title", { theme: detail?.title ?? name }));

  const form = useSettingsDraft(detail?.values, detail?.revision);
  const values = form.values;
  const save = useSaveTheme();
  const guard = useUnsavedGuard(form.dirty);
  const preview = useThemePreview(name, detail?.problem ? undefined : values);
  const uploads = useSiteUploads();

  const [width, setWidth] = usePreviewWidth();
  const [page, setPage] = useState<Page | null>(null);
  const [tab, setTab] = useState<"settings" | "preview">("settings");
  const [resetting, setResetting] = useState(false);
  const frame = useRef<FrameHandle>(null);
  const wide = useMediaQuery("(min-width: 768px)");

  const panels = useMemo(() => arrange(detail?.schema ?? [], t("customize.general")), [detail?.schema, t]);
  const fields = useMemo(() => valueFields(detail?.schema), [detail?.schema]);
  const problems = values ? fields.filter((field) => problemOf(field, values[field.key], t)).length : 0;
  const trying = detail !== undefined && !detail.active;

  const submit = () => {
    if (!detail || !values) return;
    const changes = changesOf(fields, detail.stored, values, form.baseline ?? {});
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

  const settingsPanel = (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="grid gap-4 p-4 [&>*]:min-w-0">
        <Summary theme={detail} />
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
        {detail.problem ? (
          <Alert variant="destructive">
            <XCircle />
            <AlertTitle>{t("customize.unusable")}</AlertTitle>
            <AlertDescription>{detail.problem}</AlertDescription>
          </Alert>
        ) : !values ? (
          <Skeleton className="h-72 w-full" />
        ) : panels.length === 0 ? (
          <div className="rounded-lg border border-dashed p-6 text-center">
            <p className="font-medium">{t("customize.noSettings")}</p>
            <p className="text-sm text-muted-foreground">{t("customize.noSettingsNote")}</p>
          </div>
        ) : (
          <SchemaForm
            fields={panels}
            values={values}
            onChange={form.change}
            uploads={uploads}
            sections="panel"
            onReset={reset}
          />
        )}
      </div>
    </div>
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
            <span className="ms-2 font-mono text-xs text-muted-foreground">
              {pathInside(page.path, preview.url)}
            </span>
          )}
        </span>
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
        onPage={setPage}
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
              state
            )}
          </div>
        </div>

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

      {!wide && (
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
      )}

      <div className="flex min-h-0 flex-1">
        <aside
          className={cn(
            "flex min-h-0 w-full flex-col md:w-[380px] md:shrink-0 md:border-e",
            !wide && tab !== "settings" && "hidden",
          )}
        >
          {settingsPanel}
        </aside>
        {!detail.problem && (
          <section className={cn("flex min-w-0 flex-1", !wide && tab !== "preview" && "hidden")}>
            {previewPanel}
          </section>
        )}
      </div>

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
        onOpenChange={(open) => !open && guard.cancel()}
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

/** Summary names the theme being customized, and whether it is the one in use. */
function Summary({ theme }: { theme: ThemeDetail }) {
  const { t } = useI18n();
  return (
    <div className="flex items-center gap-3 rounded-lg border bg-card p-3">
      <div className="flex aspect-[16/10] w-20 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-muted">
        {theme.screenshot ? (
          <img src={theme.screenshot} alt="" className="size-full object-cover object-top" />
        ) : (
          <Palette className="size-4 text-muted-foreground" />
        )}
      </div>
      <div className="min-w-0">
        <p className="truncate text-sm font-medium">{theme.title}</p>
        <p className="truncate text-xs text-muted-foreground">
          {[theme.version, theme.active ? t("customize.inUse") : t("customize.trying")]
            .filter(Boolean)
            .join(" · ")}
        </p>
      </div>
    </div>
  );
}

/**
 * arrange puts the fields a theme declares outside any section into one of
 * their own, so the form is a column of cards that fold.
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

/**
 * changesOf is what a save writes, by dotted path: each setting that changed,
 * and for one put back to the theme's default, its removal, so the theme's
 * default applies again and follows the theme when it changes.
 */
function changesOf(
  fields: Field[],
  stored: string[],
  values: Record<string, unknown>,
  baseline: Record<string, unknown>,
): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const field of fields) {
    const value = values[field.key];
    if (sameValue(value, baseline[field.key])) continue;
    const fallback = defaultOf(field);
    const cleared = value === "" && fallback === undefined;
    if (cleared || sameValue(value, fallback)) {
      if (stored.includes(field.key)) out[`theme.settings.${field.key}`] = null;
    } else {
      out[`theme.settings.${field.key}`] = value;
    }
  }
  return out;
}

/** pathInside is where a page is in the site, from its address in the preview. */
function pathInside(path: string, home: string | null): string {
  return home && path.startsWith(home) ? "/" + path.slice(home.length) : path;
}

/**
 * useSiteUploads stores an image a setting names among the site's own files,
 * and shows one by where the site serves it, under its base path.
 */
function useSiteUploads(): Uploads {
  const site = useSite();
  const home = siteHome(site.data).replace(/\/$/, "");
  return {
    upload: uploadSiteMedia,
    resolve: (link) => (link.startsWith("/") && !link.startsWith("//") ? home + link : resolveLink(link)),
  };
}

