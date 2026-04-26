import { createContext, useContext, useState, useCallback } from 'react'
import en, { type Translations } from './locales/en'
import zh from './locales/zh'

export type Locale = 'en' | 'zh'

const locales: Record<Locale, Translations> = { en, zh }

// STORAGE_KEY and the COOKIE_NAME below are kept in lockstep with the
// backend's i18n.LocaleCookieName. The middleware reads the cookie before
// Accept-Language, so writing it in setLocale below is what makes the
// SPA's language pick affect server-rendered surfaces (the admin shell,
// the public /share / /upload pages a user might navigate to).
const STORAGE_KEY = 'kite_locale'
const COOKIE_NAME = 'kite_locale'

function getInitialLocale(): Locale {
  const stored = localStorage.getItem(STORAGE_KEY)
  if (stored === 'en' || stored === 'zh') return stored
  // Cookie is the second-best signal: it survives a localStorage wipe
  // (private-window-to-private-window navigations) and mirrors the
  // value the public header's switcher writes.
  const cookie = readCookie(COOKIE_NAME)
  if (cookie === 'en' || cookie === 'zh') return cookie
  const browserLang = navigator.language.toLowerCase()
  if (browserLang.startsWith('zh')) return 'zh'
  return 'en'
}

function readCookie(name: string): string | null {
  const prefix = `${name}=`
  for (const part of document.cookie.split(';')) {
    const trimmed = part.trim()
    if (trimmed.startsWith(prefix)) {
      return decodeURIComponent(trimmed.slice(prefix.length))
    }
  }
  return null
}

function writeLocaleCookie(value: Locale) {
  // 1-year lifetime is plenty — the cookie is purely a UX convenience;
  // the SPA's localStorage is the canonical store. Path=/ so /share and
  // /api requests share the cookie. SameSite=Lax keeps it on top-level
  // navigations (which is when the server renders templates).
  const oneYear = 60 * 60 * 24 * 365
  document.cookie = `${COOKIE_NAME}=${value}; path=/; max-age=${oneYear}; SameSite=Lax`
}

// 按 "nav.dashboard" 格式的 key 取嵌套值
function resolve(obj: Record<string, unknown>, path: string): string {
  const keys = path.split('.')
  let cur: unknown = obj
  for (const k of keys) {
    if (cur == null || typeof cur !== 'object') return path
    cur = (cur as Record<string, unknown>)[k]
  }
  return typeof cur === 'string' ? cur : path
}

export interface I18nContextValue {
  locale: Locale
  setLocale: (l: Locale) => void
  t: (key: string) => string
}

export const I18nContext = createContext<I18nContextValue | null>(null)

export function useI18n() {
  const ctx = useContext(I18nContext)
  if (!ctx) throw new Error('useI18n must be used within I18nProvider')
  return ctx
}

export function useI18nProvider(): I18nContextValue {
  const [locale, setLocaleState] = useState<Locale>(getInitialLocale)

  const setLocale = useCallback((l: Locale) => {
    localStorage.setItem(STORAGE_KEY, l)
    writeLocaleCookie(l)
    setLocaleState(l)
  }, [])

  const t = useCallback(
    (key: string) =>
      resolve(locales[locale] as unknown as Record<string, unknown>, key),
    [locale]
  )

  return { locale, setLocale, t }
}

export const localeLabels: Record<Locale, string> = {
  en: 'English',
  zh: '中文',
}

// Standalone translator for contexts that can't use the useI18n hook
// (e.g. imperative helpers, hooks that run before provider mounts).
// Reads the active locale from localStorage with the same fallback as the provider.
export function translate(key: string): string {
  const locale = getInitialLocale()
  return resolve(locales[locale] as unknown as Record<string, unknown>, key)
}
