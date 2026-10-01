import { type ReactNode } from "react";
import { ExternalLink, TriangleAlert } from "lucide-react";

import { useI18n } from "@/i18n";
import { useApp, type AppInfo } from "@/hooks/useApps";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { AppAction, AppIcon, AppScreenshot, Does, StatusBadges } from "./parts";

/**
 * AppSheet is everything the index says about one package: who makes it,
 * what it does to a site, and each version with what it needs.
 */
export function AppSheet({
  app,
  onClose,
  onInstall,
  onUpdate,
}: {
  app: AppInfo | undefined;
  onClose: () => void;
  onInstall: (app: AppInfo) => void;
  onUpdate: (app: AppInfo) => void;
}) {
  return (
    <Sheet open={app !== undefined} onOpenChange={(open) => !open && onClose()}>
      <SheetContent className="w-full gap-0 overflow-y-auto sm:max-w-xl">
        {app && <Details app={app} onInstall={() => onInstall(app)} onUpdate={() => onUpdate(app)} />}
      </SheetContent>
    </Sheet>
  );
}

function Details({ app, onInstall, onUpdate }: { app: AppInfo; onInstall: () => void; onUpdate: () => void }) {
  const { t, date } = useI18n();
  const detail = useApp(app.kind as "theme" | "plugin", app.id);

  return (
    <>
      <SheetHeader className="gap-3 border-b">
        {app.kind === "theme" ? (
          <AppScreenshot app={app} className="aspect-[16/10] rounded-lg border" />
        ) : (
          <AppIcon app={app} className="size-12" />
        )}
        <div className="grid gap-1.5">
          <SheetTitle className="flex flex-wrap items-baseline gap-2 text-lg">
            {app.title}
            {app.version && <span className="text-sm font-normal text-muted-foreground">{app.version}</span>}
          </SheetTitle>
          {app.description && <SheetDescription>{app.description}</SheetDescription>}
        </div>
        <StatusBadges app={app} />
        <div>
          <AppAction app={app} onInstall={onInstall} onUpdate={onUpdate} size="default" />
        </div>
      </SheetHeader>

      <div className="grid gap-6 p-4">
        {app.problem && (
          <p className="flex gap-2 rounded-md bg-muted px-3 py-2 text-sm">
            <TriangleAlert className="mt-0.5 size-4 shrink-0" />
            <span className="min-w-0 break-words">{app.problem}</span>
          </p>
        )}
        {app.delisted && (
          <p className="flex gap-2 rounded-md bg-destructive/5 px-3 py-2 text-sm text-destructive">
            <TriangleAlert className="mt-0.5 size-4 shrink-0" />
            <span>{t("apps.delisted", { reason: app.delisted })}</span>
          </p>
        )}

        <section className="grid gap-2">
          <h4 className="text-sm font-semibold">{t("apps.does")}</h4>
          <Does loads={app.loads} inject={app.inject} hooks={app.hooks} plugin={app.kind === "plugin"} />
        </section>

        <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1.5 text-sm">
          {app.author && (
            <Fact term={t("themes.author")}>
              {app.author.url ? <Out href={app.author.url}>{app.author.name}</Out> : app.author.name}
            </Fact>
          )}
          {app.license && <Fact term={t("themes.license")}>{app.license}</Fact>}
          {app.homepage && (
            <Fact term={t("themes.homepage")}>
              <Out href={app.homepage}>{app.homepage.replace(/^https?:\/\//, "")}</Out>
            </Fact>
          )}
          <Fact term={t("apps.repo")}>
            <Out href={`https://github.com/${app.repo}`}>{app.repo}</Out>
          </Fact>
          {(app.tags?.length ?? 0) > 0 && (
            <Fact term={t("apps.tags")}>
              <span className="flex flex-wrap gap-1">
                {app.tags!.map((tag) => (
                  <Badge key={tag} variant="outline" className="font-normal">
                    {tag}
                  </Badge>
                ))}
              </span>
            </Fact>
          )}
        </dl>

        <Separator />

        <section className="grid gap-3">
          <h4 className="text-sm font-semibold">{t("apps.versions")}</h4>
          {!detail.data ? (
            <Skeleton className="h-24 w-full" />
          ) : (
            <ol className="grid gap-3">
              {detail.data.versions.map((v) => (
                <li key={v.version} className="grid gap-1 rounded-lg border p-3 text-sm">
                  <div className="flex flex-wrap items-baseline gap-2">
                    <span className="font-medium">{v.version}</span>
                    {v.published && <span className="text-xs text-muted-foreground">{date(v.published)}</span>}
                    {v.version === app.installed && (
                      <Badge className="border-transparent bg-success/10 font-normal text-success">
                        {t("apps.installedHere")}
                      </Badge>
                    )}
                    {v.yanked && (
                      <Badge variant="destructive" className="font-normal">
                        {t("apps.yanked")}
                      </Badge>
                    )}
                    {v.notes && (
                      <a
                        href={v.notes}
                        target="_blank"
                        rel="noreferrer"
                        className="ms-auto inline-flex items-center gap-1 text-xs hover:underline"
                      >
                        {t("apps.notes")}
                        <ExternalLink className="size-3" />
                      </a>
                    )}
                  </div>
                  {v.requires && (
                    <p className="text-xs text-muted-foreground">{t("apps.requires", { range: v.requires })}</p>
                  )}
                  {v.problem && !v.yanked && <p className="text-xs text-muted-foreground">{t("apps.unfit")}</p>}
                </li>
              ))}
            </ol>
          )}
        </section>
      </div>
    </>
  );
}

function Fact({ term, children }: { term: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{term}</dt>
      <dd className="min-w-0">{children}</dd>
    </>
  );
}

function Out({ href, children }: { href: string; children: ReactNode }) {
  return (
    <a href={href} target="_blank" rel="noreferrer" className="inline-flex max-w-full items-center gap-1 hover:underline">
      <span className="truncate">{children}</span>
      <ExternalLink className="size-3 shrink-0" />
    </a>
  );
}
