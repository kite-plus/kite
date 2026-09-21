import type { ReactNode } from "react";
import { AlertTriangle, XCircle, Info } from "lucide-react";
import { cn } from "cn";

type Tone = "info" | "warn" | "stop";

const tones: Record<Tone, { box: string; icon: ReactNode }> = {
  info: {
    box: "border-border bg-muted/50 text-foreground",
    icon: <Info className="size-4 text-muted-foreground" />,
  },
  warn: {
    box: "border-amber-600/25 bg-amber-500/5 text-amber-900 dark:text-amber-300",
    icon: <AlertTriangle className="size-4 text-amber-600" />,
  },
  stop: {
    box: "border-destructive/25 bg-destructive/5 text-destructive",
    icon: <XCircle className="size-4" />,
  },
};

/**
 * One shape for everything the admin has to say: what could not be indexed,
 * what a publish found, what a save refused.
 *
 * The tone is the only thing that varies, so an author learns to read these
 * once rather than once per screen.
 */
export function Alert({
  tone = "info",
  title,
  children,
  className,
}: {
  tone?: Tone;
  title?: string;
  children?: ReactNode;
  className?: string;
}) {
  const { box, icon } = tones[tone];
  return (
    <div className={cn("flex gap-2.5 rounded-md border px-3 py-2.5 text-sm", box, className)}>
      <span className="mt-0.5 shrink-0">{icon}</span>
      <div className="min-w-0 flex-1">
        {title && <p className="font-medium">{title}</p>}
        {children && <div className="opacity-90">{children}</div>}
      </div>
    </div>
  );
}
