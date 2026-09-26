import { Badge } from "@/components/ui/badge";
import { tint } from "@/lib/tint";
import { cn } from "@/lib/utils";

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
