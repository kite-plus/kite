import type { Settings } from "@/api/client";
import { useI18n, type Key } from "@/i18n";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { LanguageSelect } from "@/components/LanguageSelect";
import { ContentSection } from "../components/content-section";
import { FormProblem, LeaveGuard, SaveButton, useSettingsForm } from "../use-settings-form";

/** paths maps a field of SiteSettings to the config path that holds it. */
const paths: Record<string, string> = {
  title: "site.title",
  description: "site.description",
  base_url: "site.baseURL",
  language: "site.language",
};

const labels: Record<string, Key> = {
  title: "settings.siteTitle",
  description: "settings.siteDescription",
  base_url: "settings.siteBaseURL",
  language: "settings.siteLanguage",
};

const read = ({ site }: Settings) => ({
  title: site.title,
  description: site.description ?? "",
  base_url: site.base_url,
  language: site.language ?? "",
});

export function SiteSettings() {
  const { t } = useI18n();
  const form = useSettingsForm(read, (key) => paths[key]);
  const values = form.values as Record<string, string>;
  const set = (key: string, value: string) => form.change({ ...values, [key]: value });

  return (
    <ContentSection title={t("settings.site")} desc={t("settings.siteNote")}>
      <div>
        <FormProblem form={form} />
        {!form.loaded ? (
          <Skeleton className="h-72 w-full" />
        ) : (
          <div className="grid gap-6">
            {Object.keys(paths).map((key) => (
              <div key={key} className="grid gap-2">
                <Label htmlFor={key}>{t(labels[key])}</Label>
                {key === "language" ? (
                  <LanguageSelect id={key} value={values[key] ?? ""} onChange={(v) => set(key, v)} />
                ) : (
                  <Input
                    id={key}
                    type={key === "base_url" ? "url" : "text"}
                    value={values[key] ?? ""}
                    onChange={(event) => set(key, event.target.value)}
                  />
                )}
              </div>
            ))}
            <div>
              <SaveButton form={form} />
            </div>
          </div>
        )}
        <LeaveGuard form={form} />
      </div>
    </ContentSection>
  );
}
