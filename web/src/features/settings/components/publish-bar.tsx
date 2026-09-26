import { useState } from "react";
import { CloudUpload, Upload } from "lucide-react";

import { useI18n } from "@/i18n";
import { canPublish, useDelivery } from "@/hooks/usePublish";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { PublishDialog } from "@/components/publish/PublishDialog";

/**
 * PublishBar offers what the settings screens change for publishing: kite.yaml,
 * the themes in themes/, the plugins in plugins/ and the pictures settings
 * name. They belong to no
 * item, so the editor never publishes them, and a theme switched to here
 * would otherwise reach the site only when someone committed it by hand.
 */
export function PublishBar({ className }: { className?: string }) {
  const { t } = useI18n();
  const delivery = useDelivery();
  const [open, setOpen] = useState(false);
  const paths = settingsPaths(delivery.data?.dirty ?? []);
  if (!canPublish(delivery.data) || paths.length === 0) return null;

  return (
    <>
      <div
        className={cn(
          "mb-4 flex flex-wrap items-center gap-x-3 gap-y-2 rounded-lg border border-warning/30 bg-warning/5 px-4 py-2.5 text-sm lg:mb-6",
          className,
        )}
      >
        <CloudUpload className="size-4 shrink-0 text-warning" />
        <span className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
          {t("publish.settingsPending")}
          {paths.map((path) => (
            <code key={path} className="rounded bg-background px-1.5 py-0.5 text-xs ring-1 ring-border">
              {path}
            </code>
          ))}
        </span>
        <Button size="sm" variant="outline" onClick={() => setOpen(true)}>
          <Upload />
          {t("publish.action")}
        </Button>
      </div>
      <PublishDialog
        ids={[]}
        paths={paths}
        open={open}
        onOpenChange={setOpen}
        onDone={() => undefined}
        title={t("publish.settingsTitle")}
        description={t("publish.settingsNote", { paths: paths.join(", ") })}
      />
    </>
  );
}

/** settingsPaths are the uncommitted files the settings screens wrote, a theme or plugin named by its folder. */
function settingsPaths(dirty: string[]): string[] {
  const out = new Set<string>();
  for (const path of dirty) {
    if (path === "kite.yaml") out.add(path);
    else if (path.startsWith("themes/") || path.startsWith("plugins/")) out.add(path.split("/").slice(0, 2).join("/"));
    else if (path.startsWith("static/uploads/")) out.add(path);
  }
  return [...out];
}
