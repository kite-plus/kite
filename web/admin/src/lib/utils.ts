import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'
import { translate } from '@/i18n'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/** Format byte count into human-readable size string. */
export function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

/** Calculate percentage with 1 decimal place. */
export function calcPercent(part: number, total: number): string {
  if (total === 0) return '0.0'
  return ((part / total) * 100).toFixed(1)
}

/**
 * Format a date string into relative time (e.g. "5m ago" / "5 分钟前").
 *
 * Strings come from the i18n catalogue (`relativeTime.*`) so a single
 * source of truth covers every page that renders timestamps. The `locale`
 * argument is accepted for API compatibility with existing call sites;
 * the actual lookup goes through the catalogue's active locale (selected
 * by useI18nProvider on mount), so passing `locale` only matters when
 * the catalogue happens to be out of sync — which the tests guard
 * against in CI.
 */
export function formatRelativeTime(
  dateStr: string,
  locale: string = 'en'
): string {
  void locale // tracked for stability — see doc comment
  const now = Date.now()
  const date = new Date(dateStr).getTime()
  const diff = now - date

  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (seconds < 60) return translate('relativeTime.justNow')
  if (minutes < 60)
    return translate('relativeTime.minutesAgo').replace('{n}', String(minutes))
  if (hours < 24)
    return translate('relativeTime.hoursAgo').replace('{n}', String(hours))
  if (days === 1) return translate('relativeTime.yesterday')
  if (days < 30)
    return translate('relativeTime.daysAgo').replace('{n}', String(days))
  // Fallthrough to the platform's date formatter for anything older than
  // ~1 month — the relative phrasing stops being useful at that point.
  // Intl picks an appropriate format from the BCP-47 tag the SPA tracks.
  const tag = locale === 'zh' ? 'zh-CN' : 'en-US'
  return new Date(dateStr).toLocaleDateString(tag)
}
