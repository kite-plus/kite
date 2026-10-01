import { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import { ExternalLink, TriangleAlert } from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useInstallApp, useUpdateApp, useUpdatePlan, type AppInfo, type AppKind } from "@/hooks/useApps";
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Does } from "./parts";

/**
 * InstallDialog says what a package will do to the site before it is
 * installed, and after it is offers the next step: a theme's preview, a
 * plugin's switch.
 */
export function InstallDialog({ app, onClose }: { app: AppInfo | null; onClose: () => void }) {
  const { t } = useI18n();
  const problem = useProblem();
  const navigate = useNavigate();
  const install = useInstallApp();
  const theme = app?.kind === "theme";

  const confirm = () => {
    if (!app) return;
    install.mutate(
      { kind: app.kind as AppKind, id: app.id, replace: Boolean(app.other) },
      {
        onSuccess: () => {
          onClose();
          toast.success(t("apps.installed", { title: app.title }), {
            description: t(theme ? "apps.installedTheme" : "apps.installedPlugin"),
            action: theme
              ? {
                  label: t("apps.preview"),
                  onClick: () => void navigate({ to: "/settings/theme/$name/{-$section}", params: { name: app.id } }),
                }
              : {
                  label: t("apps.openPlugins"),
                  onClick: () => void navigate({ to: "/plugins" }),
                },
          });
        },
        onError: (err) => {
          onClose();
          toast.error(t("apps.installFailed"), {
            description: err instanceof ApiError ? problem(err.code, err.message).detail ?? err.message : String(err),
          });
        },
      },
    );
  };

  return (
    <ConfirmDialog
      open={app !== null}
      onOpenChange={(open) => !open && !install.isPending && onClose()}
      title={t("apps.installTitle", { title: app?.title ?? "", version: app?.version ?? "" })}
      desc={
        <div className="grid gap-3">
          <p>{t(theme ? "apps.installThemeNote" : "apps.installPluginNote")}</p>
          {app && <Does loads={app.loads} inject={app.inject} hooks={app.hooks} plugin={!theme} />}
          {app && !app.official && <p>{t("apps.communityNote")}</p>}
          {app?.other && (
            <p className="flex gap-2 text-destructive">
              <TriangleAlert className="mt-0.5 size-4 shrink-0" />
              <span>{t(theme ? "apps.replacesTheme" : "apps.replacesPlugin", { id: app.id })}</span>
            </p>
          )}
        </div>
      }
      confirmText={t(app?.other ? "apps.installReplacing" : "apps.install")}
      destructive={Boolean(app?.other)}
      isLoading={install.isPending}
      handleConfirm={confirm}
    />
  );
}

/**
 * UpdateDialog says what an update will do before it is made: the version
 * it goes to, files changed by hand that it would replace, and anything a
 * plugin's new version does beyond what was agreed. Those have to be agreed
 * to, each, before the update can be made.
 */
export function UpdateDialog({ app, onClose }: { app: AppInfo | null; onClose: () => void }) {
  const { t } = useI18n();
  const problem = useProblem();
  const plan = useUpdatePlan((app?.kind ?? "theme") as AppKind, app?.id ?? "", app !== null);
  const update = useUpdateApp();
  const [overwrite, setOverwrite] = useState(false);
  const [grant, setGrant] = useState(false);
  useEffect(() => {
    setOverwrite(false);
    setGrant(false);
  }, [app]);

  const p = plan.data;
  const ready = Boolean(p) && (!p!.changed || overwrite) && (!p!.grows || grant);

  const confirm = () => {
    if (!app || !p) return;
    update.mutate(
      { kind: app.kind as AppKind, id: app.id, overwrite, grant },
      {
        onSuccess: () => {
          onClose();
          toast.success(t("apps.updated", { title: app.title, version: p.to }));
        },
        onError: (err) => {
          onClose();
          toast.error(t("apps.updateFailed"), {
            description: err instanceof ApiError ? problem(err.code, err.message).detail ?? err.message : String(err),
          });
        },
      },
    );
  };

  return (
    <ConfirmDialog
      open={app !== null}
      onOpenChange={(open) => !open && !update.isPending && onClose()}
      title={t("apps.updateTitle", { title: app?.title ?? "" })}
      desc={
        plan.error ? (
          <p className="text-destructive">
            {plan.error instanceof ApiError ? plan.error.message : String(plan.error)}
          </p>
        ) : !p ? (
          <div className="grid gap-2">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        ) : (
          <div className="grid gap-3">
            <p>
              {t("apps.updateNote", { from: p.from, to: p.to })}
              {p.notes && (
                <>
                  {" "}
                  <a href={p.notes} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 underline">
                    {t("apps.notes")}
                    <ExternalLink className="size-3" />
                  </a>
                </>
              )}
            </p>
            <Does loads={p.loads} inject={p.inject} hooks={p.hooks} plugin={p.kind === "plugin"} />
            {p.changed && (
              <Agree checked={overwrite} onChange={setOverwrite}>
                {t("apps.overwrite", { dir: `${p.kind === "theme" ? "themes" : "plugins"}/${p.id}` })}
              </Agree>
            )}
            {p.grows && (
              <Agree checked={grant} onChange={setGrant}>
                {t("apps.grant")}
                <ul className="mt-1 list-disc ps-5">
                  {p.more_loads.map((host) => (
                    <li key={host}>{t("apps.moreLoads", { host })}</li>
                  ))}
                  {p.more_hooks.map((hook) => (
                    <li key={hook}>{t("apps.moreHooks", { hook })}</li>
                  ))}
                  {p.inject > p.injected && <li>{t("apps.moreInject", { from: p.injected, to: p.inject })}</li>}
                </ul>
              </Agree>
            )}
          </div>
        )
      }
      confirmText={p ? t("apps.updateTo", { version: p.to }) : t("apps.update")}
      disabled={!ready}
      isLoading={update.isPending}
      handleConfirm={confirm}
    />
  );
}

/** Agree is one thing an update needs agreed to, said, with its box. */
function Agree({
  checked,
  onChange,
  children,
}: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  children: React.ReactNode;
}) {
  return (
    <label className="flex gap-2 rounded-md border border-destructive/30 bg-destructive/5 p-3 text-foreground">
      <Checkbox checked={checked} onCheckedChange={(value) => onChange(value === true)} className="mt-0.5" />
      <span className="min-w-0">{children}</span>
    </label>
  );
}
