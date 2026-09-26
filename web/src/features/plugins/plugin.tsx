import { useMemo } from "react";
import { Link } from "@tanstack/react-router";
import { ChevronLeft, Globe, TriangleAlert, XCircle } from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { usePlugin, useSavePlugins, useSwitchPlugin } from "@/hooks/usePlugins";
import { useSettingsDraft } from "@/hooks/useSettingsDraft";
import { uploadSiteMedia } from "@/hooks/useThemes";
import { useUnsavedGuard } from "@/hooks/useUnsavedGuard";
import { changesOf, defaultOf, problemOf, valueFields } from "@/lib/schema";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { SchemaForm } from "@/components/SchemaForm";
import { PublishBar } from "@/features/settings/components/publish-bar";

/**
 * One plugin's settings, drawn from the form its plugin.yaml declares and
 * kept in kite.yaml under plugins.settings, with its switch beside its name.
 */
export function PluginSettings({ id }: { id: string }) {
  const { t } = useI18n();
  const problem = useProblem();
  const plugin = usePlugin(id);
  const detail = plugin.data;
  useDocumentTitle(detail?.name ?? id);

  const form = useSettingsDraft(detail?.values, detail?.revision);
  const values = form.values;
  const save = useSavePlugins();
  const flip = useSwitchPlugin();
  const guard = useUnsavedGuard(form.dirty);

  const fields = useMemo(() => valueFields(detail?.schema), [detail?.schema]);
  const problems = values ? fields.filter((field) => problemOf(field, values[field.key], t)).length : 0;

  const submit = () => {
    if (!detail || !values) return;
    const changes = changesOf(`plugins.settings.${id}.`, fields, detail.stored, values, form.baseline ?? {});
    save.mutate(
      { changes, revision: form.revision ?? detail.revision },
      {
        onSuccess: (result) => {
          form.saved(values, result.revision);
          toast.success(t("plugins.saved"));
        },
      },
    );
  };

  const failure = save.error ?? flip.error ?? (plugin.error && detail ? plugin.error : null);
  const said = failure
    ? failure instanceof ApiError && failure.code === "conflict"
      ? { title: t("settings.conflict"), detail: undefined }
      : problem(failure instanceof ApiError ? failure.code : undefined, failure.message)
    : null;

  return (
    <>
      <AppHeader />
      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <Link
          to="/plugins"
          className="-mb-2 inline-flex w-fit items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          <ChevronLeft className="size-4" />
          {t("nav.plugins")}
        </Link>

        {!detail ? (
          plugin.error ? (
            <Alert variant="destructive" className="max-w-2xl">
              <XCircle />
              <AlertTitle>
                {plugin.error instanceof ApiError && plugin.error.code === "not_found"
                  ? t("plugins.missing")
                  : problem(plugin.error instanceof ApiError ? plugin.error.code : undefined, plugin.error.message)
                      .title}
              </AlertTitle>
            </Alert>
          ) : (
            <Skeleton className="h-72 w-full max-w-2xl" />
          )
        ) : (
          <>
            <PageTitle title={detail.name} description={detail.description}>
              <label className="flex items-center gap-2 text-sm">
                {t(detail.enabled ? "plugins.on" : "plugins.off")}
                <Switch
                  checked={detail.enabled}
                  disabled={flip.isPending || (!detail.enabled && Boolean(detail.problem))}
                  onCheckedChange={(enabled) =>
                    flip.mutate(
                      { id, enabled },
                      {
                        onSuccess: () =>
                          toast.success(t(enabled ? "plugins.turnedOn" : "plugins.turnedOff", { plugin: detail.name })),
                      },
                    )
                  }
                />
              </label>
            </PageTitle>
            <PublishBar className="mb-0 lg:mb-0" />

            <div className="grid max-w-2xl gap-6">
              {detail.problem && (
                <Alert variant="destructive">
                  <TriangleAlert />
                  <AlertTitle>{t("plugins.unusable")}</AlertTitle>
                  <AlertDescription className="break-words">{detail.problem}</AlertDescription>
                </Alert>
              )}
              {(detail.hosts?.length ?? 0) > 0 && (
                <Alert>
                  <Globe />
                  <AlertDescription>{t("plugins.hosts", { hosts: detail.hosts!.join(", ") })}</AlertDescription>
                </Alert>
              )}
              {said && (
                <Alert variant="destructive">
                  <XCircle />
                  <AlertTitle>{said.title}</AlertTitle>
                  {said.detail && <AlertDescription>{said.detail}</AlertDescription>}
                </Alert>
              )}

              {!values ? (
                !detail.problem && <Skeleton className="h-40 w-full" />
              ) : fields.length === 0 ? (
                <div className="rounded-lg border border-dashed p-6 text-center">
                  <p className="font-medium">{t("plugins.noSettings")}</p>
                  <p className="text-sm text-muted-foreground">{t("plugins.noSettingsNote")}</p>
                </div>
              ) : (
                <>
                  <SchemaForm
                    fields={detail.schema ?? []}
                    values={values}
                    onChange={form.change}
                    uploads={{ upload: uploadSiteMedia }}
                    onReset={(key) => {
                      const field = fields.find((each) => each.key === key);
                      if (field) form.change({ ...values, [key]: defaultOf(field) });
                    }}
                    idPrefix="plugin-"
                  />
                  <div className="flex flex-wrap items-center gap-3">
                    <Button onClick={submit} disabled={!form.dirty || problems > 0 || save.isPending}>
                      {save.isPending && <Spinner />}
                      {t("settings.save")}
                    </Button>
                    {problems > 0 && (
                      <span className="text-sm text-destructive">{t("customize.problems", { count: problems })}</span>
                    )}
                  </div>
                </>
              )}
            </div>
          </>
        )}
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
      </Main>
    </>
  );
}
