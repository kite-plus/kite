import type { Draft } from "@/api/client";
import type { Conflict } from "@/hooks/useItem";
import { Panel } from "@/components/ui";

interface Props {
  conflict: Conflict;
  ours: Draft;
  onTakeTheirs: () => void;
  onKeepOurs: () => void;
  onCancel: () => void;
}

/**
 * The three-way view an author sees when their edit lost a race.
 *
 * The server sends what is stored; the base is what this editor loaded and
 * still holds. Nothing is merged automatically: which version is right is not
 * a question a program can answer, and guessing it is how a CMS loses work.
 */
export function ConflictDialog({ conflict, ours, onTakeTheirs, onKeepOurs, onCancel }: Props) {
  const theirs = conflict.theirs;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <Panel className="max-h-full w-full max-w-4xl overflow-auto p-5">
        <h2 className="text-lg font-semibold">This changed while you were editing</h2>
        <p className="mt-1 text-sm text-[var(--muted-foreground)]">
          Something else saved this item since you opened it. Nothing has been
          overwritten. Choose which version to keep.
        </p>

        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <Side title="Yours" subtitle="what you have been writing" body={ours.body} meta={ours.title} />
          <Side
            title="Stored"
            subtitle="what is on disk now"
            body={theirs?.body ?? "(could not be read)"}
            meta={theirs?.title ?? ""}
          />
        </div>

        <div className="mt-5 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md border border-[var(--border)] px-3 py-1.5 text-sm hover:bg-[var(--accent)]"
          >
            Keep editing
          </button>
          <button
            type="button"
            onClick={onTakeTheirs}
            className="rounded-md border border-[var(--border)] px-3 py-1.5 text-sm hover:bg-[var(--accent)]"
          >
            Discard mine, load stored
          </button>
          <button
            type="button"
            onClick={onKeepOurs}
            className="rounded-md bg-brand px-3 py-1.5 text-sm text-white hover:opacity-90"
          >
            Overwrite with mine
          </button>
        </div>
      </Panel>
    </div>
  );
}

function Side({
  title,
  subtitle,
  body,
  meta,
}: {
  title: string;
  subtitle: string;
  body: string;
  meta: string;
}) {
  return (
    <div className="min-w-0">
      <div className="mb-1 flex items-baseline gap-2">
        <span className="text-sm font-medium">{title}</span>
        <span className="text-xs text-[var(--muted-foreground)]">{subtitle}</span>
      </div>
      <div className="mb-1 truncate text-sm">{meta}</div>
      <pre className="max-h-72 overflow-auto rounded-md bg-[var(--muted)] p-3 text-xs whitespace-pre-wrap">
        {body}
      </pre>
    </div>
  );
}
