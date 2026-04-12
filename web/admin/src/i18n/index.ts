import { createContext, useContext, useState, useCallback } from "react";
import en, { type Translations } from "./locales/en";
import zh from "./locales/zh";

export type Locale = "en" | "zh";

const locales: Record<Locale, Translations> = { en, zh };

const STORAGE_KEY = "kite_locale";

function getInitialLocale(): Locale {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "en" || stored === "zh") return stored;
  const browserLang = navigator.language.toLowerCase();
  if (browserLang.startsWith("zh")) return "zh";
  return "en";
}

// 按 "nav.dashboard" 格式的 key 取嵌套值
function resolve(obj: Record<string, unknown>, path: string): string {
  const keys = path.split(".");
  let cur: unknown = obj;
  for (const k of keys) {
    if (cur == null || typeof cur !== "object") return path;
    cur = (cur as Record<string, unknown>)[k];
  }
  return typeof cur === "string" ? cur : path;
}

export interface I18nContextValue {
  locale: Locale;
  setLocale: (l: Locale) => void;
  t: (key: string) => string;
}

export const I18nContext = createContext<I18nContextValue | null>(null);

export function useI18n() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}

export function useI18nProvider(): I18nContextValue {
  const [locale, setLocaleState] = useState<Locale>(getInitialLocale);

  const setLocale = useCallback((l: Locale) => {
    localStorage.setItem(STORAGE_KEY, l);
    setLocaleState(l);
  }, []);

  const t = useCallback(
    (key: string) => resolve(locales[locale] as unknown as Record<string, unknown>, key),
    [locale]
  );

  return { locale, setLocale, t };
}

export const localeLabels: Record<Locale, string> = {
  en: "English",
  zh: "中文",
};
