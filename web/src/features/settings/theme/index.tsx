import { useRef, useState, type ReactNode } from "react";
import { Link } from "@tanstack/react-router";
import {
  CheckCircle2,
  ExternalLink,
  Eye,
  MoreHorizontal,
  Paintbrush,
  Palette,
  Trash2,
  TriangleAlert,
  Upload,
} from "lucide-react";
import { toast } from "sonner";

import { ApiError, type ThemeInfo } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useSite } from "@/hooks/useContents";
import { useSettings } from "@/hooks/useSettings";
import {
  installTheme,
  useRemoveTheme,
  useSaveTheme,
  useThemes,
  useThemesChanged,
} from "@/hooks/useThemes";
import { siteHome } from "@/lib/links";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { QueryError } from "@/components/query-error";
import { ContentSection } from "../components/content-section";

/**
 * The themes a project has: the one in use, and the others it could switch
 * to, each previewed on the whole site before it is chosen. A theme is added
 * by uploading its zip archive, which lands in themes/ like one put there by
 * hand.
 */
export function Themes() {
  const { t } = useI18n();
  const themes = useThemes();
  const [using, setUsing] = useState<ThemeInfo | null>(null);
  const [removing, setRemoving] = useState<ThemeInfo | null>(null);
  const upload = useUpload();

  const active = themes.data?.find((theme) => theme.active);
  const others = themes.data?.filter((theme) => !theme.active) ?? [];

  return (
    <ContentSection title={t("themes.title")} desc={t("themes.description")} wide>
      <div className="grid gap-8">
        {themes.error && !themes.data ? (
          <QueryError error={themes.error} onRetry={() => void themes.refetch()} />
        ) : !themes.data ? (
          <>
            <Skeleton className="h-52 w-full rounded-xl" />
            <Skeleton className="h-64 w-full rounded-xl" />
          </>
        ) : (
          <>
            {active && <ActiveTheme theme={active} />}

            <section className="grid gap-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h4 className="text-sm font-semibold">
                  {t("themes.others")}
                  <span className="ms-1.5 font-normal text-muted-foreground tabular-nums">{others.length}</span>
                </h4>
                <Button variant="outline" size="sm" disabled={upload.busy} onClick={upload.choose}>
                  {upload.busy ? <Spinner /> : <Upload />}
                  {t("themes.upload")}
                </Button>
              </div>
              <div className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(15rem,1fr))]">
                {others.map((theme) => (
                  <ThemeCard
                    key={theme.name}
                    theme={theme}
                    onUse={() => setUsing(theme)}
                    onRemove={() => setRemoving(theme)}
                  />
                ))}
                <UploadTile upload={upload} empty={others.length === 0} />
              </div>
            </section>
          </>
        )}

        <input
          ref={upload.input}
          type="file"
          accept=".zip,application/zip"
          className="hidden"
          onChange={(event) => {
            const file = event.target.files?.[0];
            event.target.value = "";
            if (file) void upload.send(file);
          }}
        />
        <UseDialog theme={using} onClose={() => setUsing(null)} />
        <RemoveDialog theme={removing} onClose={() => setRemoving(null)} />
        <ConfirmDialog
          open={upload.asking !== null}
          onOpenChange={(open) => !open && upload.dismiss()}
          title={t("themes.replaceTitle", { theme: upload.asking?.installed.title ?? "" })}
          desc={t("themes.replaceNote", {
            installed: upload.asking?.installed.version ?? "?",
            uploaded: upload.asking?.uploaded.version ?? "?",
          })}
          confirmText={t("themes.replace")}
          handleConfirm={() => upload.replace()}
        />
      </div>
    </ContentSection>
  );
}

/** Screenshot is a theme's picture of itself, or a plain panel where it has none. */
function Screenshot({ theme, className }: { theme: ThemeInfo; className?: string }) {
  const { t } = useI18n();
  const [failed, setFailed] = useState(false);
  return (
    <div className={cn("relative overflow-hidden bg-muted", className)}>
      {theme.screenshot && !failed ? (
        <img
          src={theme.screenshot}
          alt=""
          loading="lazy"
          className="size-full object-cover object-top"
          onError={() => setFailed(true)}
        />
      ) : (
        <div className="flex size-full flex-col items-center justify-center gap-2 text-muted-foreground">
          <Palette className="size-6" />
          <span className="text-xs">{t("themes.noScreenshot")}</span>
        </div>
      )}
    </div>
  );
}

function ActiveTheme({ theme }: { theme: ThemeInfo }) {
  const { t } = useI18n();
  const site = useSite();

  return (
    <section className="flex flex-col gap-5 rounded-xl border bg-card p-4 sm:flex-row sm:p-5">
      <Screenshot theme={theme} className="aspect-[16/10] w-full shrink-0 rounded-lg border sm:w-72" />
      <div className="flex min-w-0 flex-1 flex-col gap-4">
        <div className="grid gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h4 className="text-lg font-semibold">{theme.title}</h4>
            <Badge className="border-transparent bg-success/10 text-success">
              <CheckCircle2 />
              {t("themes.inUse")}
            </Badge>
          </div>
          {theme.description && <p className="text-sm text-muted-foreground">{theme.description}</p>}
        </div>

        <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1.5 text-sm">
          {theme.version && <Fact term={t("themes.versionLabel")}>{theme.version}</Fact>}
          {theme.author && (
            <Fact term={t("themes.author")}>
              {theme.author.url ? (
                <a href={theme.author.url} target="_blank" rel="noreferrer" className="hover:underline">
                  {theme.author.name}
                </a>
              ) : (
                theme.author.name
              )}
            </Fact>
          )}
          {theme.license && <Fact term={t("themes.license")}>{theme.license}</Fact>}
          {theme.homepage && (
            <Fact term={t("themes.homepage")}>
              <a
                href={theme.homepage}
                target="_blank"
                rel="noreferrer"
                className="inline-flex max-w-full items-center gap-1 hover:underline"
              >
                <span className="truncate">{theme.homepage.replace(/^https?:\/\//, "")}</span>
                <ExternalLink className="size-3 shrink-0" />
              </a>
            </Fact>
          )}
          <Fact term={t("themes.location")}>
            {theme.builtin ? t("themes.builtin") : <code className="text-xs">themes/{theme.name}</code>}
          </Fact>
        </dl>

        {theme.problem && <ProblemNote problem={theme.problem} />}

        <div className="mt-auto flex flex-wrap gap-2">
          <Button asChild>
            <Link to="/settings/theme/$name" params={{ name: theme.name }}>
              <Paintbrush />
              {t("themes.customize")}
            </Link>
          </Button>
          <Button variant="outline" asChild>
            <a href={siteHome(site.data)} target="_blank" rel="noreferrer">
              <ExternalLink />
              {t("themes.visit")}
            </a>
          </Button>
        </div>
      </div>
    </section>
  );
}

function Fact({ term, children }: { term: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{term}</dt>
      <dd className="min-w-0 truncate">{children}</dd>
    </>
  );
}

function ProblemNote({ problem }: { problem: string }) {
  return (
    <p className="flex gap-2 rounded-md bg-destructive/5 px-3 py-2 text-sm text-destructive">
      <TriangleAlert className="mt-0.5 size-4 shrink-0" />
      <span className="min-w-0 break-words">{problem}</span>
    </p>
  );
}

function ThemeCard({ theme, onUse, onRemove }: { theme: ThemeInfo; onUse: () => void; onRemove: () => void }) {
  const { t } = useI18n();
  return (
    <article className="flex flex-col overflow-hidden rounded-xl border bg-card">
      <Screenshot theme={theme} className={cn("aspect-[16/10] border-b", theme.problem && "opacity-60")} />
      <div className="flex flex-1 flex-col gap-1.5 p-4">
        <div className="flex items-baseline gap-2">
          <h5 className="truncate font-medium">{theme.title}</h5>
          {theme.version && <span className="shrink-0 text-xs text-muted-foreground">{theme.version}</span>}
          {theme.builtin && (
            <Badge variant="secondary" className="ms-auto">
              {t("themes.builtin")}
            </Badge>
          )}
        </div>
        {theme.author && (
          <p className="truncate text-xs text-muted-foreground">{t("themes.by", { author: theme.author.name })}</p>
        )}
        {theme.problem ? (
          <ProblemNote problem={theme.problem} />
        ) : (
          theme.description && <p className="line-clamp-2 text-sm text-muted-foreground">{theme.description}</p>
        )}
      </div>
      <div className="flex items-center gap-2 border-t px-4 py-3">
        {theme.problem ? (
          <Badge variant="outline" className="border-destructive/30 text-destructive">
            {t("themes.unusable")}
          </Badge>
        ) : (
          <>
            <Button variant="outline" size="sm" asChild>
              <Link to="/settings/theme/$name" params={{ name: theme.name }}>
                <Eye />
                {t("themes.preview")}
              </Link>
            </Button>
            <Button size="sm" onClick={onUse}>
              {t("themes.use")}
            </Button>
          </>
        )}
        {!theme.builtin && (
          <DropdownMenu modal={false}>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="ms-auto size-8" aria-label={t("themes.more")}>
                <MoreHorizontal />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem variant="destructive" onSelect={onRemove}>
                <Trash2 />
                {t("themes.remove")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </article>
  );
}

type Upload = ReturnType<typeof useUpload>;

/**
 * useUpload installs the archive the author hands over, and asks before one
 * replaces a theme that is installed already.
 */
function useUpload() {
  const { t } = useI18n();
  const problem = useProblem();
  const changed = useThemesChanged();
  const input = useRef<HTMLInputElement>(null);
  const [sending, setSending] = useState<{ name: string; progress: number } | null>(null);
  const [asking, setAsking] = useState<{
    file: File;
    installed: ThemeInfo;
    uploaded: ThemeInfo;
  } | null>(null);

  const send = async (file: File, replace = false) => {
    setSending({ name: file.name, progress: 0 });
    try {
      const result = await installTheme(file, replace, (progress) => setSending({ name: file.name, progress }));
      if (result.status === "exists") {
        setAsking({ file, installed: result.installed, uploaded: result.uploaded });
        return;
      }
      await changed();
      toast.success(t("themes.installed", { theme: result.theme.title, version: result.theme.version ?? "" }));
    } catch (err) {
      const said =
        err instanceof ApiError && err.code !== "invalid_request"
          ? problem(err.code, err.message)
          : { title: t("themes.installFailed"), detail: err instanceof Error ? err.message : String(err) };
      toast.error(said.title, { description: said.detail });
    } finally {
      setSending(null);
    }
  };

  return {
    input,
    busy: sending !== null,
    sending,
    asking,
    send,
    choose: () => input.current?.click(),
    replace: () => {
      const file = asking?.file;
      setAsking(null);
      if (file) void send(file, true);
    },
    dismiss: () => setAsking(null),
  };
}

function UploadTile({ upload, empty }: { upload: Upload; empty: boolean }) {
  const { t } = useI18n();
  const [over, setOver] = useState(false);
  return (
    <button
      type="button"
      disabled={upload.busy}
      onClick={upload.choose}
      onDragOver={(event) => {
        event.preventDefault();
        setOver(true);
      }}
      onDragLeave={() => setOver(false)}
      onDrop={(event) => {
        event.preventDefault();
        setOver(false);
        const file = event.dataTransfer.files[0];
        if (file) void upload.send(file);
      }}
      className={cn(
        "flex min-h-64 flex-col items-center justify-center gap-2 rounded-xl border border-dashed p-6 text-center text-muted-foreground outline-none transition-colors hover:bg-accent/50 focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:pointer-events-none",
        over && "border-primary bg-primary/5",
      )}
    >
      {upload.busy ? <Spinner className="size-6" /> : <Upload className="size-6" />}
      {upload.sending ? (
        <span className="text-sm text-foreground">
          {t("themes.installing", { name: upload.sending.name })}
          {upload.sending.progress < 100 && ` ${upload.sending.progress}%`}
        </span>
      ) : (
        <>
          <span className="text-sm font-medium text-foreground">
            {empty ? t("themes.noOthers") : t("themes.upload")}
          </span>
          <span className="max-w-60 text-xs">{empty ? t("themes.noOthersNote") : t("themes.dropZip")}</span>
        </>
      )}
    </button>
  );
}

function UseDialog({ theme, onClose }: { theme: ThemeInfo | null; onClose: () => void }) {
  const { t } = useI18n();
  const problem = useProblem();
  const settings = useSettings();
  const save = useSaveTheme();

  return (
    <ConfirmDialog
      open={theme !== null}
      onOpenChange={(open) => !open && onClose()}
      title={t("themes.useTitle", { theme: theme?.title ?? "" })}
      desc={t("themes.useNote")}
      confirmText={t("themes.use")}
      isLoading={save.isPending}
      disabled={!settings.data}
      handleConfirm={() => {
        if (!theme || !settings.data) return;
        save.mutate(
          { changes: { "theme.name": theme.name }, revision: settings.data.revision },
          {
            onSuccess: () => {
              onClose();
              toast.success(t("themes.used", { theme: theme.title }));
            },
            onError: (err) => {
              const said = err instanceof ApiError ? problem(err.code, err.message) : { title: String(err) };
              toast.error(said.title, { description: "detail" in said ? said.detail : undefined });
            },
          },
        );
      }}
    />
  );
}

function RemoveDialog({ theme, onClose }: { theme: ThemeInfo | null; onClose: () => void }) {
  const { t } = useI18n();
  const problem = useProblem();
  const remove = useRemoveTheme();

  return (
    <ConfirmDialog
      open={theme !== null}
      onOpenChange={(open) => !open && onClose()}
      title={t("themes.removeTitle", { theme: theme?.title ?? "" })}
      desc={t("themes.removeNote", { name: theme?.name ?? "" })}
      confirmText={t("themes.remove")}
      destructive
      isLoading={remove.isPending}
      handleConfirm={() => {
        if (!theme) return;
        remove.mutate(theme.name, {
          onSuccess: () => {
            onClose();
            toast.success(t("themes.removed", { theme: theme.title }));
          },
          onError: (err) => {
            const said = err instanceof ApiError ? problem(err.code, err.message) : { title: String(err) };
            toast.error(said.title, { description: "detail" in said ? said.detail : undefined });
          },
        });
      }}
    />
  );
}
