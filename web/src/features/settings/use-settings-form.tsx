import { useEffect, useRef } from "react";
import { XCircle } from "lucide-react";
import { toast } from "sonner";

import { ApiError, type Settings } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { useSaveSettings, useSettings } from "@/hooks/useSettings";
import { useSettingsDraft } from "@/hooks/useSettingsDraft";
import { useUnsavedGuard } from "@/hooks/useUnsavedGuard";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { ConfirmDialog } from "@/components/confirm-dialog";

/**
 * useSettingsForm is one settings form: a draft of what is stored, saved as
 * only what changed, against the revision of kite.yaml it was loaded at. A
 * file edited meanwhile is a conflict to reload, not something to overwrite.
 */
export function useSettingsForm(
  read: (settings: Settings) => Record<string, unknown> | undefined,
  /** path is the dotted configuration path a value of the form is saved to. */
  path: (key: string) => string,
) {
  const { t } = useI18n();
  const problem = useProblem();
  const settings = useSettings();
  const save = useSaveSettings();

  const incoming = settings.data ? read(settings.data) : undefined;
  const form = useSettingsDraft(incoming, settings.data?.revision);
  const values = form.values ?? {};
  const guard = useUnsavedGuard(form.dirty);

  // Only what changed is sent, so a file is not rewritten for untouched values.
  const changes = Object.fromEntries(
    Object.entries(values)
      .filter(([key, value]) => JSON.stringify(value) !== JSON.stringify(form.baseline?.[key]))
      .map(([key, value]) => [path(key), value]),
  );
  const conflict = save.error instanceof ApiError && save.error.code === "conflict";
  const failure = settings.error ?? save.error;
  const said = failure
    ? conflict
      ? { title: t("settings.conflict"), detail: undefined }
      : problem(failure instanceof ApiError ? failure.code : undefined, failure.message)
    : null;

  return {
    settings: settings.data,
    values,
    loaded: incoming !== undefined,
    dirty: form.dirty,
    conflict,
    said,
    pending: save.isPending,
    guard,
    change: form.change,
    submit: () =>
      save.mutate(
        { changes, revision: form.revision ?? "" },
        {
          onSuccess: (result) => {
            form.saved(read(result) ?? {}, result.revision);
            toast.success(t("settings.saved"));
          },
        },
      ),
    reload: async () => {
      const result = await settings.refetch();
      if (result.data) {
        form.reset(read(result.data) ?? {}, result.data.revision);
        save.reset();
      }
    },
  };
}

type Form = ReturnType<typeof useSettingsForm>;

/** FormProblem says what a save or a load ran into, with the way out of a conflict. */
export function FormProblem({ form }: { form: Form }) {
  const { t } = useI18n();
  const ref = useRef<HTMLDivElement>(null);
  const title = form.said?.title;
  // A save fails at the foot of a long form, out of sight of its head.
  useEffect(() => {
    if (title) ref.current?.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [title]);
  if (!form.said) return null;
  return (
    <Alert ref={ref} variant="destructive" className="mb-6 scroll-mt-4">
      <XCircle />
      <AlertTitle>{form.said.title}</AlertTitle>
      <AlertDescription>
        {form.said.detail && <p>{form.said.detail}</p>}
        {form.conflict && (
          <Button variant="outline" size="sm" className="mt-2" onClick={() => void form.reload()}>
            {t("settings.reload")}
          </Button>
        )}
      </AlertDescription>
    </Alert>
  );
}

/** SaveButton saves the form once there is something to save. */
export function SaveButton({ form }: { form: Form }) {
  const { t } = useI18n();
  return (
    <Button disabled={!form.dirty || form.pending || form.conflict} onClick={form.submit}>
      {form.pending && <Spinner />}
      {t("settings.save")}
    </Button>
  );
}

/** LeaveGuard asks before a navigation drops what has not been saved. */
export function LeaveGuard({ form }: { form: Form }) {
  const { t } = useI18n();
  return (
    <ConfirmDialog
      open={form.guard.leaving}
      onOpenChange={(open) => !open && form.guard.cancel()}
      title={t("editor.discardTitle")}
      desc={t("settings.discardNote")}
      cancelBtnText={t("conflict.keepEditing")}
      confirmText={t("editor.discard")}
      destructive
      handleConfirm={form.guard.discard}
    />
  );
}
