import {
  createContext,
  use,
  useCallback,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import { en } from "./en";
import { zhCN } from "./zh-CN";

/**
 * The admin speaks the operator's language, not the site's.
 *
 * What is frozen here is the call shape -- t("some.key", { count }) -- and
 * not what implements it. Plurals and dates come from Intl rather than from a
 * catalog format, so the parts that are genuinely hard are the platform's
 * problem; what is left is a lookup, and a lookup that is typed catches a
 * missing key at build time instead of showing a key to a person.
 */
export const locales = {
  en: { label: "English", catalog: en },
  "zh-CN": { label: "简体中文", catalog: zhCN },
} as const;

export type Locale = keyof typeof locales;

/**
 * PluralBase recovers "list.count" from "list.count_other", so a call site
 * names the thing rather than one of its forms.
 */
type PluralBase<K> = K extends `${infer Base}_other` ? Base : never;

/** Key is every string the admin can say. A typo will not compile. */
export type Key = keyof typeof en | PluralBase<keyof typeof en>;

export type Values = Record<string, string | number>;

const storageKey = "kite:locale";

/** detect picks a language from what the browser asks for. */
function detect(): Locale {
  try {
    const saved = localStorage.getItem(storageKey);
    if (saved && saved in locales) return saved as Locale;
  } catch {
    // A browser with storage blocked still gets a language.
  }
  for (const tag of navigator.languages ?? [navigator.language]) {
    if (tag in locales) return tag as Locale;
    const base = tag.split("-")[0];
    const match = Object.keys(locales).find((l) => l.split("-")[0] === base);
    if (match) return match as Locale;
  }
  return "en";
}

interface Context {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  t: (key: Key, values?: Values) => string;
  /** date formats an instant the way this locale writes dates. */
  date: (iso: string | undefined, style?: "short" | "long") => string;
}

const I18nContext = createContext<Context | null>(null);

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(detect);

  const setLocale = useCallback((next: Locale) => {
    setLocaleState(next);
    document.documentElement.lang = next;
    try {
      localStorage.setItem(storageKey, next);
    } catch {
      // Remembering the choice is a convenience, not a requirement.
    }
  }, []);

  const value = useMemo<Context>(() => {
    const catalog = locales[locale].catalog as Record<string, string>;
    const plural = new Intl.PluralRules(locale);

    const t = (key: Key, values?: Values): string => {
      let phrase = catalog[key];

      // A count selects between forms the same way the language does, so
      // "1 file" and "2 files" are not two call sites.
      if (phrase === undefined && values && "count" in values) {
        const form = plural.select(Number(values.count));
        phrase = catalog[`${key}_${form}`] ?? catalog[`${key}_other`];
      }
      // Falling back to English rather than to the key: a half-translated
      // catalog should read as English, not as source code.
      if (phrase === undefined) {
        phrase = (en as Record<string, string>)[key] ?? key;
      }
      if (!values) return phrase;

      return phrase.replace(/\{(\w+)\}/g, (whole, name: string) =>
        name in values ? String(values[name]) : whole,
      );
    };

    const date = (iso: string | undefined, style: "short" | "long" = "short") => {
      if (!iso) return "—";
      const at = new Date(iso);
      if (Number.isNaN(at.getTime())) return "—";
      return new Intl.DateTimeFormat(
        locale,
        style === "long"
          ? { dateStyle: "medium", timeStyle: "short" }
          : { dateStyle: "medium" },
      ).format(at);
    };

    return { locale, setLocale, t, date };
  }, [locale, setLocale]);

  return <I18nContext value={value}>{children}</I18nContext>;
}

export function useI18n(): Context {
  const context = use(I18nContext);
  if (!context) throw new Error("useI18n used outside I18nProvider");
  return context;
}

/**
 * problem turns a server code into a sentence in the operator's language.
 *
 * Every code the API returns is stable and documented; the English text
 * beside it is the fallback for a code this build has not learned yet. An
 * admin that translated its buttons and left every failure in English would
 * have translated everything except the moment it matters.
 */
export function useProblem() {
  const { t } = useI18n();
  return useCallback(
    (code: string | undefined, detail?: string, fix?: string) => {
      const key = `problem.${code}` as Key;
      const known =
        code !== undefined &&
        (locales.en.catalog as Record<string, string>)[key] !== undefined;

      // The headline is translated; what the server said stays underneath,
      // on its own line, because it is where the specifics live -- which
      // file, how large, how many commits behind. A phrase written in
      // advance cannot carry those, and dropping them to avoid two
      // languages on screen would cost more than it saves.
      return known
        ? { title: t(key), detail, fix }
        : { title: detail ?? code ?? "", detail: undefined, fix };
    },
    [t],
  );
}
