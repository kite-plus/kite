import type { ReactNode, SelectHTMLAttributes } from "react";
import { cn } from "@/lib/cn";

export function Panel({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "rounded-lg border border-[var(--border)] bg-[var(--background)]",
        className,
      )}
    >
      {children}
    </div>
  );
}

const statusTone: Record<string, string> = {
  published: "bg-emerald-500/12 text-emerald-700 dark:text-emerald-400",
  draft: "bg-amber-500/12 text-amber-700 dark:text-amber-400",
  scheduled: "bg-sky-500/12 text-sky-700 dark:text-sky-400",
  archived: "bg-zinc-500/12 text-zinc-600 dark:text-zinc-400",
};

export function StatusBadge({ status }: { status: string }) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
        statusTone[status] ?? statusTone.archived,
      )}
    >
      {status}
    </span>
  );
}

export function Tag({
  children,
  onClick,
  active,
}: {
  children: ReactNode;
  onClick?: () => void;
  active?: boolean;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded px-1.5 py-0.5 text-xs transition-colors",
        "bg-[var(--muted)] text-[var(--muted-foreground)]",
        onClick && "hover:bg-[var(--accent)] hover:text-[var(--foreground)]",
        active && "bg-brand/15 text-brand",
      )}
    >
      {children}
    </button>
  );
}

export function Select({
  label,
  className,
  ...props
}: { label: string } & SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <label className="flex items-center gap-2 text-sm">
      <span className="text-[var(--muted-foreground)]">{label}</span>
      <select
        {...props}
        className={cn(
          "rounded-md border border-[var(--border)] bg-[var(--background)]",
          "px-2 py-1.5 text-sm outline-none",
          "focus:border-brand focus:ring-2 focus:ring-brand/20",
          className,
        )}
      />
    </label>
  );
}

/** Failure reports what the server said, rather than that something failed. */
export function Failure({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : String(error);
  return (
    <Panel className="border-red-500/30 bg-red-500/5 p-4 text-sm text-red-700 dark:text-red-400">
      {message}
    </Panel>
  );
}
