import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

// Tints for terms, picked by name so a term keeps its color wherever it shows,
// the way explore picks a blog's avatar color. Fixed classes, so Tailwind
// finds every one of them.
const palette = [
  "bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300",
  "bg-orange-100 text-orange-700 dark:bg-orange-500/15 dark:text-orange-300",
  "bg-amber-100 text-amber-800 dark:bg-amber-500/15 dark:text-amber-300",
  "bg-lime-100 text-lime-800 dark:bg-lime-500/15 dark:text-lime-300",
  "bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300",
  "bg-teal-100 text-teal-700 dark:bg-teal-500/15 dark:text-teal-300",
  "bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300",
  "bg-indigo-100 text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300",
  "bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300",
  "bg-fuchsia-100 text-fuchsia-700 dark:bg-fuchsia-500/15 dark:text-fuchsia-300",
];

function tint(term: string): string {
  let h = 0;
  for (const ch of term) h = (h * 31 + (ch.codePointAt(0) ?? 0)) >>> 0;
  return palette[h % palette.length];
}

/** TermBadge is a term, with how many items carry it, in the term's own color. */
export function TermBadge({
  term,
  count,
  className,
}: {
  term: string;
  count?: number;
  className?: string;
}) {
  return (
    <Badge variant="secondary" className={cn("gap-1.5 font-normal", tint(term), className)}>
      {term}
      {count !== undefined && <span className="tabular-nums opacity-60">{count}</span>}
    </Badge>
  );
}
