import { LayoutTemplate } from "lucide-react";

import type { Settings } from "@/api/client";
import { useI18n } from "@/i18n";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { SchemaForm } from "@/components/SchemaForm";
import { ContentSection } from "../components/content-section";
import { FormProblem, LeaveGuard, SaveButton, useSettingsForm } from "../use-settings-form";

const read = ({ theme }: Settings) => theme.values ?? {};

/**
 * The theme in use, and the settings it asks for.
 *
 * The form is generated from the schema the theme declares, so configuring a
 * theme nobody here has seen needs no page written for it.
 */
export function ThemeSettings() {
  const { t } = useI18n();
  const form = useSettingsForm(read, (key) => `theme.settings.${key}`);
  const theme = form.settings?.theme;

  return (
    <ContentSection
      title={t("theme.title")}
      desc={theme ? t("settings.themeNote", { theme: theme.name }) : t("theme.description")}
    >
      <div>
        <FormProblem form={form} />
        {!theme ? (
          <Skeleton className="h-72 w-full" />
        ) : (
          <div className="grid gap-6">
            <div className="flex items-center gap-3 rounded-lg border p-4">
              <div className="flex size-9 items-center justify-center rounded-md bg-muted">
                <LayoutTemplate className="size-4" />
              </div>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium">{theme.name}</p>
                <p className="text-sm text-muted-foreground">{t("theme.activeNote")}</p>
              </div>
              <Badge variant="secondary">{t("theme.active")}</Badge>
            </div>

            {theme.schema && theme.schema.length > 0 ? (
              <>
                <SchemaForm fields={theme.schema} values={form.values} onChange={form.change} />
                <div>
                  <SaveButton form={form} />
                </div>
              </>
            ) : (
              <div className="rounded-lg border p-6 text-center">
                <p className="font-medium">{t("settings.themeEmpty")}</p>
                <p className="text-sm text-muted-foreground">{t("theme.emptyNote")}</p>
              </div>
            )}
          </div>
        )}
        <LeaveGuard form={form} />
      </div>
    </ContentSection>
  );
}
