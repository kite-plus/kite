import { useEffect, useState } from "react";
import { Monitor, Moon, Sun } from "lucide-react";
import { toast } from "sonner";

import { ApiError } from "@/api/client";
import { locales, useI18n, useProblem, type Key, type Locale } from "@/i18n";
import { useSaveSettings, useSettings } from "@/hooks/useSettings";
import { useTheme, type Theme } from "@/lib/theme";

import { LanguageSelect } from "@/components/LanguageSelect";
import { SettingsProblem } from "@/components/settings/SettingsProblem";
import { Page } from "@/components/shell/Page";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

/** sitePaths maps a field of SiteSettings to the config path that holds it. */
const sitePaths = {
  title: "site.title",
  description: "site.description",
  base_url: "site.baseURL",
  language: "site.language",
} as const;

type SiteKey = keyof typeof sitePaths;

const labels: Record<SiteKey, Key> = {
  title: "settings.siteTitle",
  description: "settings.siteDescription",
  base_url: "settings.siteBaseURL",
  language: "settings.siteLanguage",
};

const appearances = [
  { value: "light", icon: Sun, label: "settings.light" },
  { value: "dark", icon: Moon, label: "settings.dark" },
  { value: "system", icon: Monitor, label: "settings.system" },
] as const;

export function SettingsPage() {
  const { t, locale, setLocale } = useI18n();
  const problem = useProblem();
  const settings = useSettings();
  const save = useSaveSettings();
  const { theme, setTheme } = useTheme();

  const [site, setSite] = useState<Record<string, string>>({});
  const stored = settings.data?.site as Record<string, string | undefined> | undefined;

  useEffect(() => {
    if (stored) setSite(Object.fromEntries(Object.entries(stored).map(([k, v]) => [k, v ?? ""])));
  }, [stored]);

  // Only what changed is sent, so a file is not rewritten for untouched values.
  const changes = Object.fromEntries(
    (Object.keys(sitePaths) as SiteKey[])
      .filter((key) => stored && (site[key] ?? "") !== (stored[key] ?? ""))
      .map((key) => [sitePaths[key], site[key] ?? ""]),
  );
  const dirty = Object.keys(changes).length > 0;

  const failure = settings.error ?? save.error;
  const said = failure
    ? problem(failure instanceof ApiError ? failure.code : undefined, failure.message)
    : null;

  return (
    <Page
      title={t("settings.title")}
      description={t("settings.description")}
      actions={
        <Button
          disabled={!dirty || save.isPending}
          onClick={() => save.mutate(changes, { onSuccess: () => toast.success(t("settings.saved")) })}
        >
          {save.isPending && <Spinner data-icon="inline-start" />}
          {t("settings.save")}
        </Button>
      }
    >
      <div className="flex max-w-2xl flex-col gap-3.5">
        {said && <SettingsProblem {...said} />}

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{t("settings.site")}</CardTitle>
            <CardDescription>{t("settings.siteNote")}</CardDescription>
          </CardHeader>
          <CardContent>
            {!stored ? (
              <Skeleton className="h-64 w-full" />
            ) : (
              <FieldGroup>
                {(Object.keys(sitePaths) as SiteKey[]).map((key) => (
                  <Field key={key}>
                    <FieldLabel htmlFor={key}>{t(labels[key])}</FieldLabel>
                    {key === "language" ? (
                      <LanguageSelect
                        id={key}
                        value={site[key] ?? ""}
                        onChange={(value) => setSite({ ...site, [key]: value })}
                      />
                    ) : (
                      <Input
                        id={key}
                        type={key === "base_url" ? "url" : "text"}
                        value={site[key] ?? ""}
                        onChange={(event) => setSite({ ...site, [key]: event.target.value })}
                      />
                    )}
                  </Field>
                ))}
              </FieldGroup>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">{t("settings.interface")}</CardTitle>
            <CardDescription>{t("settings.interfaceNote")}</CardDescription>
          </CardHeader>
          <CardContent>
            <FieldGroup>
              <Field orientation="horizontal">
                <FieldContent>
                  <FieldLabel htmlFor="locale">{t("nav.language")}</FieldLabel>
                  <FieldDescription>{t("settings.languageNote")}</FieldDescription>
                </FieldContent>
                <Select value={locale} onValueChange={(next) => next && setLocale(next as Locale)}>
                  <SelectTrigger id="locale" className="w-40">
                    <SelectValue>{(code: Locale) => locales[code]?.label ?? code}</SelectValue>
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      {(Object.keys(locales) as Locale[]).map((code) => (
                        <SelectItem key={code} value={code}>
                          {locales[code].label}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>
              </Field>

              <Field orientation="horizontal">
                <FieldContent>
                  <FieldTitle id="appearance">{t("settings.appearance")}</FieldTitle>
                  <FieldDescription>{t("settings.appearanceNote")}</FieldDescription>
                </FieldContent>
                <ToggleGroup
                  aria-labelledby="appearance"
                  variant="outline"
                  value={[theme]}
                  onValueChange={(value) => value[0] && setTheme(value[0] as Theme)}
                >
                  {appearances.map((item) => (
                    <ToggleGroupItem key={item.value} value={item.value}>
                      <item.icon />
                      {t(item.label)}
                    </ToggleGroupItem>
                  ))}
                </ToggleGroup>
              </Field>
            </FieldGroup>
          </CardContent>
        </Card>
      </div>
    </Page>
  );
}
