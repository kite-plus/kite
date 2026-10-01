import { useEffect, useMemo, useState } from "react";
import { getRouteApi } from "@tanstack/react-router";
import { Palette, Puzzle, RefreshCw, Search, TriangleAlert, WifiOff } from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useApps, useRefreshApps, type AppInfo, type AppKind } from "@/hooks/useApps";
import { cn } from "@/lib/utils";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { QueryError } from "@/components/query-error";
import { ReadOnlyNote } from "@/components/read-only-note";
import { PublishBar } from "@/features/settings/components/publish-bar";
import { AppSheet } from "./app-sheet";
import { InstallDialog, UpdateDialog } from "./dialogs";
import { AppAction, AppScreenshot, StatusBadges } from "./parts";
import type { AppsSearch } from "./search";

const route = getRouteApi("/_authenticated/apps/");

/**
 * The app center: the themes and plugins the index lists, installed and kept
 * up to date by this server, which fetches the index and every archive so
 * that the browser never has to. What is installed lands in themes/ and
 * plugins/ like an uploaded archive, with kite.lock saying where it came from.
 */
export function Apps() {
  const { t, relative } = useI18n();
  useDocumentTitle(t("nav.apps"));
  const search = route.useSearch();
  const navigate = route.useNavigate();
  const apps = useApps();
  const refresh = useRefreshApps();
  const problem = useProblem();
  const kind: AppKind = search.kind ?? "theme";
  const [installing, setInstalling] = useState<AppInfo | null>(null);
  const [updating, setUpdating] = useState<AppInfo | null>(null);

  const set = (patch: Partial<AppsSearch>) =>
    void navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true });

  const items = useMemo(() => apps.data?.items ?? [], [apps.data]);
  const shown = useMemo(() => {
    const words = (search.q ?? "").toLowerCase().split(/\s+/).filter(Boolean);
    return items
      .filter((app) => app.kind === kind)
      .filter((app) => !search.source || (search.source === "official") === app.official)
      .filter((app) => {
        const text = [app.id, app.title, app.description ?? "", ...(app.tags ?? [])].join("\n").toLowerCase();
        return words.every((word) => text.includes(word));
      })
      .sort((a, b) => Number(Boolean(b.update)) - Number(Boolean(a.update)) || a.title.localeCompare(b.title));
  }, [items, kind, search.q, search.source]);
  const count = (k: AppKind) => items.filter((app) => app.kind === k).length;
  const open = search.id ? items.find((app) => app.kind === kind && app.id === search.id) : undefined;

  const fetchAgain = () =>
    refresh.mutate(undefined, {
      onError: (err) =>
        toast.error(err instanceof ApiError ? problem(err.code, err.message).title : String(err), {
          description: err instanceof ApiError ? err.message : undefined,
        }),
    });

  return (
    <>
      <AppHeader />
      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <PageTitle title={t("nav.apps")} description={t("apps.note")}>
          <Button variant="outline" disabled={refresh.isPending} onClick={fetchAgain}>
            {refresh.isPending ? <Spinner /> : <RefreshCw />}
            {t("apps.refresh")}
          </Button>
        </PageTitle>
        <ReadOnlyNote />
        <PublishBar className="mb-0 lg:mb-0" />
        {apps.data?.offline && (
          <Alert>
            <WifiOff />
            <AlertDescription>{t("apps.offline", { when: relative(apps.data.fetched) })}</AlertDescription>
          </Alert>
        )}

        <div className="flex flex-wrap items-center gap-2">
          <Tabs value={kind} onValueChange={(value) => set({ kind: value as AppKind, id: undefined })}>
            <TabsList>
              <TabsTrigger value="theme">
                <Palette />
                {t("apps.themes")}
                {apps.data && <span className="text-muted-foreground tabular-nums">{count("theme")}</span>}
              </TabsTrigger>
              <TabsTrigger value="plugin">
                <Puzzle />
                {t("apps.plugins")}
                {apps.data && <span className="text-muted-foreground tabular-nums">{count("plugin")}</span>}
              </TabsTrigger>
            </TabsList>
          </Tabs>
          <SearchBox value={search.q ?? ""} onChange={(q) => set({ q: q || undefined })} />
          <Select
            value={search.source ?? "all"}
            onValueChange={(value) => set({ source: value === "all" ? undefined : (value as AppsSearch["source"]) })}
          >
            <SelectTrigger size="sm" className="w-32" aria-label={t("apps.source")}>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">{t("apps.allSources")}</SelectItem>
              <SelectItem value="official">{t("apps.official")}</SelectItem>
              <SelectItem value="community">{t("apps.community")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {apps.error && !apps.data ? (
          <QueryError error={apps.error} onRetry={() => void apps.refetch()} />
        ) : !apps.data ? (
          <div className={grid(kind)}>
            {Array.from({ length: 3 }, (_, i) => (
              <Skeleton key={i} className={kind === "theme" ? "h-80 rounded-xl" : "h-44 rounded-xl"} />
            ))}
          </div>
        ) : shown.length === 0 ? (
          <p className="rounded-xl border border-dashed p-10 text-center text-sm text-muted-foreground">
            {t(kind === "theme" ? "apps.noThemes" : "apps.noPlugins")}
          </p>
        ) : (
          <div className={grid(kind)}>
            {shown.map((app) => (
              <AppCard
                key={app.id}
                app={app}
                onOpen={() => set({ id: app.id })}
                onInstall={() => setInstalling(app)}
                onUpdate={() => setUpdating(app)}
              />
            ))}
          </div>
        )}

        <AppSheet
          app={open}
          onClose={() => set({ id: undefined })}
          onInstall={(app) => setInstalling(app)}
          onUpdate={(app) => setUpdating(app)}
        />
        <InstallDialog app={installing} onClose={() => setInstalling(null)} />
        <UpdateDialog app={updating} onClose={() => setUpdating(null)} />
      </Main>
    </>
  );
}

function grid(kind: AppKind) {
  return cn(
    "grid gap-4",
    kind === "theme"
      ? "[grid-template-columns:repeat(auto-fill,minmax(17rem,1fr))]"
      : "[grid-template-columns:repeat(auto-fill,minmax(20rem,1fr))]",
  );
}

/** SearchBox waits for typing to pause before it changes the address. */
function SearchBox({ value, onChange }: { value: string; onChange: (q: string) => void }) {
  const { t } = useI18n();
  const [query, setQuery] = useState(value);
  useEffect(() => setQuery(value), [value]);
  useEffect(() => {
    if (query === value) return;
    const timer = setTimeout(() => onChange(query.trim()), 250);
    return () => clearTimeout(timer);
  }, [query, value, onChange]);
  return (
    <div className="relative w-full sm:w-64">
      <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        type="search"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={t("apps.search")}
        aria-label={t("apps.search")}
        className="h-8 ps-8"
      />
    </div>
  );
}

function AppCard({
  app,
  onOpen,
  onInstall,
  onUpdate,
}: {
  app: AppInfo;
  onOpen: () => void;
  onInstall: () => void;
  onUpdate: () => void;
}) {
  const { t } = useI18n();
  const theme = app.kind === "theme";
  return (
    <article className={cn("flex flex-col overflow-hidden rounded-xl border bg-card", app.update && "border-primary/40")}>
      {theme && (
        <button type="button" onClick={onOpen} className="outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50">
          <AppScreenshot app={app} className="aspect-[16/10] border-b" />
        </button>
      )}
      <div className="flex flex-1 items-start gap-3 p-4">
        {!theme && (
          <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
            <Puzzle className="size-5" />
          </div>
        )}
        <div className="grid min-w-0 flex-1 gap-1.5">
          <div className="flex items-baseline gap-2">
            <h3 className="truncate font-medium">
              <button type="button" onClick={onOpen} className="hover:underline">
                {app.title}
              </button>
            </h3>
            {app.version && <span className="shrink-0 text-xs text-muted-foreground">{app.version}</span>}
          </div>
          {app.author && (
            <p className="truncate text-xs text-muted-foreground">{t("themes.by", { author: app.author.name })}</p>
          )}
          {app.problem ? (
            <p className="flex gap-2 text-sm text-muted-foreground">
              <TriangleAlert className="mt-0.5 size-4 shrink-0" />
              <span className="line-clamp-2">{t("apps.unfit")}</span>
            </p>
          ) : (
            app.description && <p className="line-clamp-2 text-sm text-muted-foreground">{app.description}</p>
          )}
          <div className="pt-1">
            <StatusBadges app={app} />
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2 border-t px-4 py-3">
        <AppAction app={app} onInstall={onInstall} onUpdate={onUpdate} />
        <Button variant="ghost" size="sm" className="ms-auto" onClick={onOpen}>
          {t("apps.details")}
        </Button>
      </div>
    </article>
  );
}
