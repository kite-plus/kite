import type { Draft } from "@/api/client";
import type { Conflict } from "@/hooks/useItem";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface Props {
  conflict: Conflict;
  ours: Draft;
  onTakeTheirs: () => void;
  onKeepOurs: () => void;
  onCancel: () => void;
}

/**
 * The two versions, side by side, when an edit lost a race.
 *
 * The server sends what is stored; the base is what this editor loaded and
 * still holds. Nothing merges on its own: which version is right is not a
 * question a program can answer, and guessing it is how a CMS loses work.
 */
export function ConflictDialog({
  conflict,
  ours,
  onTakeTheirs,
  onKeepOurs,
  onCancel,
}: Props) {
  const theirs = conflict.theirs;

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>This changed while you were editing</DialogTitle>
          <DialogDescription>
            Something else saved this item since you opened it. Nothing has been
            overwritten. Choose which version to keep.
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 sm:grid-cols-2">
          <Side title="Yours" subtitle="what you have been writing" heading={ours.title} body={ours.body} />
          <Side
            title="Stored"
            subtitle="what is on disk now"
            heading={theirs?.title ?? ""}
            body={theirs?.body ?? "(could not be read)"}
          />
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={onCancel}>
            Keep editing
          </Button>
          <Button variant="outline" onClick={onTakeTheirs}>
            Discard mine, load stored
          </Button>
          <Button onClick={onKeepOurs}>Overwrite with mine</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Side({
  title,
  subtitle,
  heading,
  body,
}: {
  title: string;
  subtitle: string;
  heading: string;
  body: string;
}) {
  return (
    <div className="min-w-0">
      <div className="mb-1 flex items-baseline gap-2">
        <span className="text-sm font-medium">{title}</span>
        <span className="text-xs text-muted-foreground">{subtitle}</span>
      </div>
      <div className="mb-1 truncate text-sm">{heading}</div>
      <pre className="max-h-72 overflow-auto rounded-md border bg-muted/50 p-3 text-xs whitespace-pre-wrap">
        {body}
      </pre>
    </div>
  );
}
