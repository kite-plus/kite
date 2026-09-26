import { useRef, useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  Code,
  Cpu,
  ExternalLink,
  Globe,
  MoreHorizontal,
  Puzzle,
  Settings2,
  Trash2,
  TriangleAlert,
  Upload,
} from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import {
  installPlugin,
  usePlugins,
  usePluginsChanged,
  useRemovePlugin,
  useSwitchPlugin,
  type PluginInfo,
} from "@/hooks/usePlugins";
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
import { Switch } from "@/components/ui/switch";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { QueryError } from "@/components/query-error";
import { PublishBar } from "@/features/settings/components/publish-bar";

/**
 * The plugins a site has: each one's switch, what it does to the site, and
 * the other sites its code loads from. A plugin is added by uploading its zip
 * archive, which lands in plugins/ and stays off until it is switched on.
 */
export function Plugins() {
  const { t } = useI18n();
  useDocumentTitle(t("nav.plugins"));
  const plugins = usePlugins();
  const upload = useUpload();
  const [removing, setRemoving] = useState<PluginInfo | null>(null);

  return (
    <>
      <AppHeader />
      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <PageTitle title={t("nav.plugins")} description={t("plugins.note")}>
          <Button variant="outline" disabled={upload.busy} onClick={upload.choose}>
            {upload.busy ? <Spinner /> : <Upload />}
            {t("plugins.upload")}
          </Button>
        </PageTitle>
        <PublishBar className="mb-0 lg:mb-0" />

        {plugins.error && !plugins.data ? (
          <QueryError error={plugins.error} onRetry={() => void plugins.refetch()} />
        ) : !plugins.data ? (
          <div className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(20rem,1fr))]">
            <Skeleton className="h-44 rounded-xl" />
            <Skeleton className="h-44 rounded-xl" />
          </div>
        ) : (
          <div className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(20rem,1fr))]">
            {plugins.data.map((plugin) => (
              <PluginCard key={plugin.id} plugin={plugin} onRemove={() => setRemoving(plugin)} />
            ))}
            <UploadTile upload={upload} empty={plugins.data.length === 0} />
          </div>
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
        <RemoveDialog plugin={removing} onClose={() => setRemoving(null)} />
        <ConfirmDialog
          open={upload.asking !== null}
          onOpenChange={(open) => !open && upload.dismiss()}
          title={t("plugins.replaceTitle", { plugin: upload.asking?.installed.name ?? "" })}
          desc={t("plugins.replaceNote", {
            installed: upload.asking?.installed.version ?? "?",
            uploaded: upload.asking?.uploaded.version ?? "?",
          })}
          confirmText={t("plugins.replace")}
          handleConfirm={() => upload.replace()}
        />
      </Main>
    </>
  );
}

function PluginCard({ plugin, onRemove }: { plugin: PluginInfo; onRemove: () => void }) {
  const { t } = useI18n();
  const problem = useProblem();
  const flip = useSwitchPlugin();

  const toggle = (enabled: boolean) =>
    flip.mutate(
      { id: plugin.id, enabled },
      {
        onSuccess: () => toast.success(t(enabled ? "plugins.turnedOn" : "plugins.turnedOff", { plugin: plugin.name })),
        onError: (err) =>
          toast.error(err instanceof ApiError ? problem(err.code, err.message).title : String(err), {
            description: err instanceof ApiError ? err.message : undefined,
          }),
      },
    );

  return (
    <article className={cn("flex flex-col rounded-xl border bg-card", plugin.enabled && "border-primary/40")}>
      <div className="flex flex-1 items-start gap-3 p-4">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
          <Puzzle className="size-5" />
        </div>
        <div className="grid min-w-0 flex-1 gap-1.5">
          <div className="flex items-baseline gap-2">
            <h3 className="truncate font-medium">{plugin.name}</h3>
            {plugin.version && <span className="shrink-0 text-xs text-muted-foreground">{plugin.version}</span>}
          </div>
          {plugin.author && (
            <p className="truncate text-xs text-muted-foreground">{t("themes.by", { author: plugin.author.name })}</p>
          )}
          {plugin.problem ? (
            <p className="flex gap-2 rounded-md bg-destructive/5 px-3 py-2 text-sm text-destructive">
              <TriangleAlert className="mt-0.5 size-4 shrink-0" />
              <span className="min-w-0 break-words">{plugin.problem}</span>
            </p>
          ) : (
            plugin.description && <p className="line-clamp-2 text-sm text-muted-foreground">{plugin.description}</p>
          )}
          <div className="flex flex-wrap gap-1.5 pt-1">
            {plugin.injects && (
              <Badge variant="secondary" className="font-normal">
                <Code />
                {t("plugins.injects")}
              </Badge>
            )}
            {(plugin.hooks?.length ?? 0) > 0 && (
              <Badge variant="secondary" className="font-normal">
                <Cpu />
                {t("plugins.hooks")}
              </Badge>
            )}
          </div>
          {(plugin.hosts?.length ?? 0) > 0 && (
            <p className="flex gap-1.5 text-xs text-muted-foreground">
              <Globe className="mt-px size-3.5 shrink-0" />
              <span>{t("plugins.hosts", { hosts: plugin.hosts!.join(", ") })}</span>
            </p>
          )}
        </div>
        <Switch
          checked={plugin.enabled}
          disabled={flip.isPending || (!plugin.enabled && Boolean(plugin.problem))}
          onCheckedChange={toggle}
          aria-label={t(plugin.enabled ? "plugins.turnOff" : "plugins.turnOn", { plugin: plugin.name })}
        />
      </div>
      <div className="flex items-center gap-2 border-t px-4 py-3">
        <Button variant="outline" size="sm" asChild>
          <Link to="/plugins/$id" params={{ id: plugin.id }}>
            <Settings2 />
            {t("plugins.settings")}
          </Link>
        </Button>
        {plugin.homepage && (
          <Button variant="ghost" size="sm" asChild>
            <a href={plugin.homepage} target="_blank" rel="noreferrer">
              <ExternalLink />
              {t("plugins.homepage")}
            </a>
          </Button>
        )}
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="ms-auto size-8" aria-label={t("themes.more")}>
              <MoreHorizontal />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem variant="destructive" disabled={plugin.enabled} onSelect={onRemove}>
              <Trash2 />
              {t(plugin.enabled ? "plugins.removeOnFirst" : "plugins.remove")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </article>
  );
}

type Upload = ReturnType<typeof useUpload>;

/**
 * useUpload installs the archive the author hands over, and asks before one
 * replaces a plugin that is installed already.
 */
function useUpload() {
  const { t } = useI18n();
  const problem = useProblem();
  const changed = usePluginsChanged();
  const input = useRef<HTMLInputElement>(null);
  const [sending, setSending] = useState<string | null>(null);
  const [asking, setAsking] = useState<{ file: File; installed: PluginInfo; uploaded: PluginInfo } | null>(null);

  const send = async (file: File, replace = false) => {
    setSending(file.name);
    try {
      const result = await installPlugin(file, replace);
      if (result.status === "exists") {
        setAsking({ file, installed: result.installed, uploaded: result.uploaded });
        return;
      }
      await changed();
      toast.success(t("plugins.installed", { plugin: result.plugin.name }), {
        description: result.plugin.enabled ? undefined : t("plugins.installedNote"),
      });
    } catch (err) {
      const said =
        err instanceof ApiError && err.code !== "invalid_request"
          ? problem(err.code, err.message)
          : { title: t("plugins.installFailed"), detail: err instanceof Error ? err.message : String(err) };
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
        "flex min-h-44 flex-col items-center justify-center gap-2 rounded-xl border border-dashed p-6 text-center text-muted-foreground outline-none transition-colors hover:bg-accent/50 focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:pointer-events-none",
        over && "border-primary bg-primary/5",
      )}
    >
      {upload.busy ? <Spinner className="size-6" /> : <Upload className="size-6" />}
      {upload.sending ? (
        <span className="text-sm text-foreground">{t("plugins.installing", { name: upload.sending })}</span>
      ) : (
        <>
          <span className="text-sm font-medium text-foreground">
            {empty ? t("plugins.none") : t("plugins.upload")}
          </span>
          <span className="max-w-64 text-xs">{empty ? t("plugins.noneNote") : t("plugins.dropZip")}</span>
        </>
      )}
    </button>
  );
}

function RemoveDialog({ plugin, onClose }: { plugin: PluginInfo | null; onClose: () => void }) {
  const { t } = useI18n();
  const problem = useProblem();
  const remove = useRemovePlugin();
  return (
    <ConfirmDialog
      open={plugin !== null}
      onOpenChange={(open) => !open && onClose()}
      title={t("plugins.removeTitle", { plugin: plugin?.name ?? "" })}
      desc={t("plugins.removeNote")}
      confirmText={t("plugins.remove")}
      destructive
      isLoading={remove.isPending}
      handleConfirm={() =>
        plugin &&
        remove.mutate(plugin.id, {
          onSuccess: () => {
            onClose();
            toast.success(t("plugins.removed", { plugin: plugin.name }));
          },
          onError: (err) => {
            onClose();
            toast.error(err instanceof ApiError ? problem(err.code, err.message).title : String(err));
          },
        })
      }
    />
  );
}
