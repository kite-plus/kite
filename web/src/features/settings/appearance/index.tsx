import { Monitor, Moon, Sun } from "lucide-react";

import { locales, useI18n, type Locale } from "@/i18n";
import { useTheme, type Theme } from "@/lib/theme";
import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ContentSection } from "../components/content-section";

const themes = [
  { value: "light", icon: Sun, label: "settings.light" },
  { value: "dark", icon: Moon, label: "settings.dark" },
  { value: "system", icon: Monitor, label: "settings.system" },
] as const;

/**
 * Appearance is the operator's, not the site's: the studio's language and
 * colors, kept in this browser.
 */
export function AppearanceSettings() {
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();

  return (
    <ContentSection title={t("settings.interface")} desc={t("settings.interfaceNote")}>
      <div className="grid gap-8">
        <div className="grid gap-2">
          <Label htmlFor="locale">{t("nav.language")}</Label>
          <Select value={locale} onValueChange={(next) => setLocale(next as Locale)}>
            <SelectTrigger id="locale" className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(Object.keys(locales) as Locale[]).map((code) => (
                <SelectItem key={code} value={code}>
                  {locales[code].label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-sm text-muted-foreground">{t("settings.languageNote")}</p>
        </div>

        <div className="grid gap-2">
          <Label>{t("settings.appearance")}</Label>
          <p className="text-sm text-muted-foreground">{t("settings.appearanceNote")}</p>
          <div className="grid max-w-md grid-cols-3 gap-3 pt-2" role="radiogroup">
            {themes.map((item) => (
              <button
                key={item.value}
                type="button"
                role="radio"
                aria-checked={theme === item.value}
                onClick={() => setTheme(item.value as Theme)}
                className={cn(
                  "flex flex-col items-center gap-2 rounded-md border-2 p-4 text-sm transition-colors hover:bg-accent",
                  theme === item.value ? "border-primary" : "border-muted",
                )}
              >
                <item.icon className="size-5" />
                {t(item.label)}
              </button>
            ))}
          </div>
        </div>
      </div>
    </ContentSection>
  );
}
