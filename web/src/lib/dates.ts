/** isoDate writes an instant as its local calendar date, 2026-09-18. */
export function isoDate(iso: string | undefined): string {
  if (!iso) return "—";
  const at = new Date(iso);
  if (Number.isNaN(at.getTime())) return "—";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${at.getFullYear()}-${pad(at.getMonth() + 1)}-${pad(at.getDate())}`;
}

/**
 * longDay names a day with its weekday. Chinese is written with spaces
 * around the numbers, 9 月 22 日 星期二, which Intl does not put in.
 */
export function longDay(locale: string, at = new Date()): string {
  if (locale.startsWith("zh")) {
    const weekday = new Intl.DateTimeFormat(locale, { weekday: "long" }).format(at);
    return `${at.getMonth() + 1} 月 ${at.getDate()} 日 ${weekday}`;
  }
  return new Intl.DateTimeFormat(locale, {
    month: "long",
    day: "numeric",
    weekday: "long",
  }).format(at);
}
