import { useEffect, useState, type FormEvent } from "react";
import { ChevronDown } from "lucide-react";
import { toast } from "sonner";

import { locales, useI18n, type Locale } from "@/i18n";
import { useTheme, type Theme } from "@/lib/theme";
import { cn } from "@/lib/utils";
import { Button, buttonVariants } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { ContentSection } from "../components/content-section";

function LightPreview() {
  return (
    <div className="space-y-2 rounded-sm bg-[#f5f5f5] p-2">
      <div className="space-y-2 rounded-md bg-white p-2 shadow-xs">
        <div className="h-2 w-20 rounded-lg bg-[#e5e7eb]" />
        <div className="h-2 w-25 rounded-lg bg-[#e5e7eb]" />
      </div>
      <div className="flex items-center space-x-2 rounded-md bg-white p-2 shadow-xs">
        <div className="h-4 w-4 rounded-full bg-[#e5e7eb]" />
        <div className="h-2 w-25 rounded-lg bg-[#e5e7eb]" />
      </div>
      <div className="flex items-center space-x-2 rounded-md bg-white p-2 shadow-xs">
        <div className="h-4 w-4 rounded-full bg-[#e5e7eb]" />
        <div className="h-2 w-25 rounded-lg bg-[#e5e7eb]" />
      </div>
    </div>
  );
}

function DarkPreview() {
  return (
    <div className="space-y-2 rounded-sm bg-[#0f1013] p-2">
      <div className="space-y-2 rounded-md bg-[#17181c] p-2 shadow-xs">
        <div className="h-2 w-20 rounded-lg bg-[#6b7280]" />
        <div className="h-2 w-25 rounded-lg bg-[#6b7280]" />
      </div>
      <div className="flex items-center space-x-2 rounded-md bg-[#17181c] p-2 shadow-xs">
        <div className="h-4 w-4 rounded-full bg-[#6b7280]" />
        <div className="h-2 w-25 rounded-lg bg-[#6b7280]" />
      </div>
      <div className="flex items-center space-x-2 rounded-md bg-[#17181c] p-2 shadow-xs">
        <div className="h-4 w-4 rounded-full bg-[#6b7280]" />
        <div className="h-2 w-25 rounded-lg bg-[#6b7280]" />
      </div>
    </div>
  );
}

const themes = [
  { value: "light", label: "settings.light", preview: <LightPreview /> },
  { value: "dark", label: "settings.dark", preview: <DarkPreview /> },
  {
    value: "system",
    label: "settings.system",
    preview: (
      <div className="grid grid-cols-2 overflow-hidden rounded-sm">
        <div className="overflow-hidden">
          <LightPreview />
        </div>
        <div className="overflow-hidden">
          <DarkPreview />
        </div>
      </div>
    ),
  },
] as const;

/**
 * Appearance is the operator's, not the site's: the studio's language and
 * colors, kept in this browser. The form is shadcn-admin's, with a language
 * where it has a font, and the automatic theme explore adds.
 */
export function AppearanceSettings() {
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();
  const [chosenLocale, setChosenLocale] = useState<Locale>(locale);
  const [chosenTheme, setChosenTheme] = useState<Theme>(theme);
  // The header's toggle can change the theme under an open form.
  useEffect(() => setChosenTheme(theme), [theme]);
  useEffect(() => setChosenLocale(locale), [locale]);
  const dirty = chosenLocale !== locale || chosenTheme !== theme;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (chosenTheme !== theme) setTheme(chosenTheme);
    if (chosenLocale !== locale) setLocale(chosenLocale);
    toast.success(t("settings.appearanceSaved"));
  };

  return (
    <ContentSection title={t("settings.interface")} desc={t("settings.interfaceNote")}>
      <form onSubmit={submit} className="space-y-8">
        <div className="grid gap-2">
          <Label htmlFor="locale">{t("nav.language")}</Label>
          <div className="relative w-max">
            <select
              id="locale"
              value={chosenLocale}
              onChange={(e) => setChosenLocale(e.target.value as Locale)}
              className={cn(
                buttonVariants({ variant: "outline" }),
                "w-50 appearance-none font-normal",
                "dark:bg-background dark:hover:bg-background",
              )}
            >
              {(Object.keys(locales) as Locale[]).map((code) => (
                <option key={code} value={code}>
                  {locales[code].label}
                </option>
              ))}
            </select>
            <ChevronDown className="pointer-events-none absolute inset-e-3 top-2.5 size-4 opacity-50" />
          </div>
          <p className="text-sm text-muted-foreground">{t("settings.languageNote")}</p>
        </div>

        <div className="grid gap-2">
          <Label>{t("settings.appearance")}</Label>
          <p className="text-sm text-muted-foreground">{t("settings.appearanceNote")}</p>
          <RadioGroup
            value={chosenTheme}
            onValueChange={(value) => setChosenTheme(value as Theme)}
            className="grid max-w-2xl grid-cols-1 gap-8 pt-2 sm:grid-cols-3"
          >
            {themes.map((option) => (
              <Label
                key={option.value}
                className="flex-col items-stretch [&:has([data-state=checked])>div]:border-primary"
              >
                <RadioGroupItem value={option.value} className="sr-only" />
                <div className="items-center rounded-md border-2 border-muted p-1 hover:border-accent">
                  {option.preview}
                </div>
                <span className="block w-full p-2 text-center font-normal">{t(option.label)}</span>
              </Label>
            ))}
          </RadioGroup>
        </div>

        <Button type="submit" disabled={!dirty}>
          {t("settings.saveAppearance")}
        </Button>
      </form>
    </ContentSection>
  );
}
