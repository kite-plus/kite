import { LayoutTemplate } from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useSaveSettings, useSettings } from "@/hooks/useSettings";
import { useSettingsDraft } from "@/hooks/useSettingsDraft";
import { useUnsavedGuard } from "@/hooks/useUnsavedGuard";

import { ConfirmDialog } from "@/components/ConfirmDialog";
import { SchemaForm } from "@/components/SchemaForm";
import { SettingsProblem } from "@/components/settings/SettingsProblem";
import { Page } from "@/components/shell/Page";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Item, ItemActions, ItemContent, ItemDescription, ItemMedia, ItemTitle } from "@/components/ui/item";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";

/**
 * The theme in use, and the settings it asks for.
 *
 * The form is generated from the schema the theme declares, so configuring a
 * theme nobody here has seen needs no page written for it.
 */
export function ThemePage() {
  const { t } = useI18n();
  const problem = useProblem();
  const settings = useSettings();
  const save = useSaveSettings();

  const theme = settings.data?.theme;
  const form = useSettingsDraft(theme?.values ?? undefined, settings.data?.revision);
  const values = form.values ?? {};
  const guard = useUnsavedGuard(form.dirty);

  // Only what changed is sent, so a file is not rewritten for untouched values.
  const changes = Object.fromEntries(
    Object.entries(values)
      .filter(([key, value]) => JSON.stringify(value) !== JSON.stringify(form.baseline?.[key]))
      .map(([key, value]) => [`theme.settings.${key}`, value]),
  );
  const dirty = form.dirty;
  const conflict = save.error instanceof ApiError && save.error.code === "conflict";

  const failure = settings.error ?? save.error;
  const said = failure
    ? conflict
      ? { title: t("settings.conflict") }
      : problem(failure instanceof ApiError ? failure.code : undefined, failure.message)
    : null;

  return (
    <Page
      title={t("theme.title")}
      description={t("theme.description")}
      actions={
        <Button
          disabled={!dirty || save.isPending || conflict}
          onClick={() => save.mutate({ changes, revision: form.revision ?? "" }, {
            onSuccess: (result) => {
              form.saved(result.theme.values ?? {}, result.revision);
              toast.success(t("settings.saved"));
            },
          })}
        >
          {save.isPending && <Spinner data-icon="inline-start" />}
          {t("settings.save")}
        </Button>
      }
    >
      <div className="flex max-w-2xl flex-col gap-3.5">
        {said && <SettingsProblem {...said} />}
        {conflict && (
          <Button variant="outline" onClick={async () => {
            const result = await settings.refetch();
            if (result.data) {
              form.reset(result.data.theme.values ?? {}, result.data.revision);
              save.reset();
            }
          }}>{t("settings.reload")}</Button>
        )}

        {!theme ? (
          <Skeleton className="h-16 w-full" />
        ) : (
          <Item variant="outline">
            <ItemMedia variant="icon" className="size-9 rounded-md bg-brand/10 text-brand">
              <LayoutTemplate />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>{theme.name}</ItemTitle>
              <ItemDescription>{t("theme.activeNote")}</ItemDescription>
            </ItemContent>
            <ItemActions>
              <Badge variant="secondary">{t("theme.active")}</Badge>
            </ItemActions>
          </Item>
        )}

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{t("theme.settings")}</CardTitle>
            {theme && (
              <CardDescription>{t("settings.themeNote", { theme: theme.name })}</CardDescription>
            )}
          </CardHeader>
          <CardContent>
            {!theme ? (
              <Skeleton className="h-40 w-full" />
            ) : theme.schema && theme.schema.length > 0 ? (
              <SchemaForm fields={theme.schema} values={values} onChange={form.change} />
            ) : (
              <Empty>
                <EmptyHeader>
                  <EmptyTitle>{t("settings.themeEmpty")}</EmptyTitle>
                  <EmptyDescription>{t("theme.emptyNote")}</EmptyDescription>
                </EmptyHeader>
              </Empty>
            )}
          </CardContent>
        </Card>
      </div>
      <ConfirmDialog
        open={guard.leaving}
        onOpenChange={(open) => !open && guard.cancel()}
        title={t("editor.discardTitle")}
        description={t("settings.discardNote")}
        confirmLabel={t("editor.discard")}
        cancelLabel={t("conflict.keepEditing")}
        destructive
        onConfirm={guard.discard}
      />
    </Page>
  );
}
