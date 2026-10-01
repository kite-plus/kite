import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { ArrowUpCircle, CheckCircle2, Code, Cpu, Download, Globe, Palette, Puzzle, ShieldCheck } from "lucide-react";

import { useI18n } from "@/i18n";
import { useWritable } from "@/hooks/useContents";
import { useApps, type AppInfo, type AppKind } from "@/hooks/useApps";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

/** Screenshot is a theme's picture of itself, or a plain panel without one. */
export function AppScreenshot({ app, className }: { app: AppInfo; className?: string }) {
  const [failed, setFailed] = useState(false);
  return (
    <div className={cn("relative overflow-hidden bg-muted", className)}>
      {app.screenshot && !failed ? (
        <img
          src={app.screenshot}
          alt=""
          loading="lazy"
          className="size-full object-cover object-top"
          onError={() => setFailed(true)}
        />
      ) : (
        <div className="flex size-full items-center justify-center text-muted-foreground">
          <Palette className="size-6" />
        </div>
      )}
    </div>
  );
}

/** AppIcon is a plugin's own icon, or a puzzle piece for one without. */
export function AppIcon({ app, className }: { app: AppInfo; className?: string }) {
  const [failed, setFailed] = useState(false);
  return (
    <div className={cn("flex shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground", className)}>
      {app.icon && !failed ? (
        <img src={app.icon} alt="" className="size-3/4 object-contain" onError={() => setFailed(true)} />
      ) : (
        <Puzzle className="size-1/2" />
      )}
    </div>
  );
}

/** StatusBadges says how the site stands with a package, and who makes it. */
export function StatusBadges({ app }: { app: AppInfo }) {
  const { t } = useI18n();
  return (
    <div className="flex flex-wrap gap-1.5">
      {app.official ? (
        <Badge variant="secondary" className="font-normal">
          <ShieldCheck />
          {t("apps.official")}
        </Badge>
      ) : (
        <Badge variant="outline" className="font-normal">
          {t("apps.community")}
        </Badge>
      )}
      {app.update ? (
        <Badge className="border-transparent bg-primary/10 font-normal text-primary">
          <ArrowUpCircle />
          {t("apps.updatable", { version: app.update })}
        </Badge>
      ) : (
        app.installed && (
          <Badge className="border-transparent bg-success/10 font-normal text-success">
            <CheckCircle2 />
            {t("apps.installedVersion", { version: app.installed })}
          </Badge>
        )
      )}
      {app.yanked && (
        <Badge variant="destructive" className="font-normal">
          {t("apps.yanked")}
        </Badge>
      )}
    </div>
  );
}

/** AppAction is what can be done with a package from here, if anything. */
export function AppAction({
  app,
  onInstall,
  onUpdate,
  size = "sm",
}: {
  app: AppInfo;
  onInstall: () => void;
  onUpdate: () => void;
  size?: "sm" | "default";
}) {
  const { t } = useI18n();
  const writable = useWritable();
  if (!writable) return null;
  if (app.update) {
    return (
      <Button size={size} onClick={onUpdate}>
        <ArrowUpCircle />
        {t("apps.updateTo", { version: app.update })}
      </Button>
    );
  }
  if (app.installed) return null;
  return (
    <Button size={size} variant="outline" disabled={!app.version} onClick={onInstall}>
      <Download />
      {t("apps.install")}
    </Button>
  );
}

/**
 * UpdateBadge marks an installed theme or plugin the index has a newer
 * version of, and leads to it in the app center. It says nothing while the
 * index cannot be read.
 */
export function UpdateBadge({ kind, name }: { kind: AppKind; name: string }) {
  const { t } = useI18n();
  const apps = useApps();
  const app = apps.data?.items.find((one) => one.kind === kind && one.id === name);
  if (!app?.update) return null;
  return (
    <Badge asChild className="border-transparent bg-primary/10 font-normal text-primary hover:bg-primary/15">
      <Link to="/apps" search={{ kind, id: name }}>
        <ArrowUpCircle />
        {t("apps.updatable", { version: app.update })}
      </Link>
    </Badge>
  );
}

/** Does says what a package does to a site that its owner should know. */
export function Does({
  loads,
  inject,
  hooks,
  plugin,
}: {
  loads: string[];
  inject: number;
  hooks: string[];
  plugin: boolean;
}) {
  const { t } = useI18n();
  return (
    <ul className="grid gap-1.5 text-sm text-muted-foreground">
      <li className="flex gap-2">
        <Globe className="mt-0.5 size-4 shrink-0" />
        <span>{loads.length > 0 ? t("apps.loadsFrom", { hosts: loads.join(", ") }) : t("apps.loadsNothing")}</span>
      </li>
      {plugin && (
        <li className="flex gap-2">
          <Code className="mt-0.5 size-4 shrink-0" />
          <span>{t("apps.injects", { count: inject })}</span>
        </li>
      )}
      {plugin && (
        <li className="flex gap-2">
          <Cpu className="mt-0.5 size-4 shrink-0" />
          <span>{hooks.length > 0 ? t("apps.runs", { hooks: hooks.join(", ") }) : t("apps.runsNothing")}</span>
        </li>
      )}
    </ul>
  );
}
