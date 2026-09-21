import type { Draft } from "@/api/client";
import { useI18n } from "@/i18n";
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
  const { t } = useI18n();
  const theirs = conflict.theirs;

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>{t("conflict.title")}</DialogTitle>
          <DialogDescription>{t("conflict.description")}</DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 sm:grid-cols-2">
          <Side
            title={t("conflict.ours")}
            subtitle={t("conflict.oursNote")}
            heading={ours.title}
            body={ours.body}
          />
          <Side
            title={t("conflict.theirs")}
            subtitle={t("conflict.theirsNote")}
            heading={theirs?.title ?? ""}
            body={theirs?.body ?? t("conflict.unreadable")}
          />
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={onCancel}>
            {t("conflict.keepEditing")}
          </Button>
          <Button variant="outline" onClick={onTakeTheirs}>
            {t("conflict.takeTheirs")}
          </Button>
          <Button onClick={onKeepOurs}>{t("conflict.keepOurs")}</Button>
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
