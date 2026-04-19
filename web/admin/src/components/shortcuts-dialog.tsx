import { Fragment } from "react";
import { Keyboard } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { useI18n } from "@/i18n";

interface ShortcutItem {
  keys: string[];
  descKey: string;
}
interface ShortcutGroup {
  labelKey: string;
  items: ShortcutItem[];
}

const GROUPS: ShortcutGroup[] = [
  {
    labelKey: "shortcuts.groupNav",
    items: [
      { keys: ["G", "D"], descKey: "shortcuts.gotoDashboard" },
      { keys: ["G", "F"], descKey: "shortcuts.gotoFiles" },
      { keys: ["G", "A"], descKey: "shortcuts.gotoAlbums" },
      { keys: ["G", "T"], descKey: "shortcuts.gotoTokens" },
      { keys: ["G", "U"], descKey: "shortcuts.gotoUsers" },
    ],
  },
  {
    labelKey: "shortcuts.groupActions",
    items: [
      { keys: ["⌘", "K"], descKey: "shortcuts.openSearch" },
      { keys: ["⌘", "U"], descKey: "shortcuts.uploadFile" },
      { keys: ["⌘", "B"], descKey: "shortcuts.toggleSidebar" },
      { keys: ["⌘", "."], descKey: "shortcuts.toggleTheme" },
    ],
  },
  {
    labelKey: "shortcuts.groupSelection",
    items: [
      { keys: ["J"], descKey: "shortcuts.selectNext" },
      { keys: ["K"], descKey: "shortcuts.selectPrev" },
      { keys: ["X"], descKey: "shortcuts.toggleSelect" },
      { keys: ["⇧", "X"], descKey: "shortcuts.multiSelect" },
      { keys: ["Enter"], descKey: "shortcuts.openDetail" },
      { keys: ["Esc"], descKey: "shortcuts.close" },
    ],
  },
  {
    labelKey: "shortcuts.groupHelp",
    items: [{ keys: ["?"], descKey: "shortcuts.openShortcuts" }],
  },
];

interface ShortcutsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function ShortcutsDialog({ open, onOpenChange }: ShortcutsDialogProps) {
  const { t } = useI18n();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[min(85vh,640px)] w-[min(92vw,640px)] gap-0 overflow-hidden border-border/80 p-0 shadow-2xl [&>button]:hidden">
        <div className="flex items-center justify-between border-b px-5 py-3.5">
          <div className="flex items-center gap-2">
            <Keyboard className="size-4 text-muted-foreground" />
            <DialogTitle className="text-sm font-semibold">
              {t("shortcuts.title")}
            </DialogTitle>
          </div>
          <button
            type="button"
            className="rounded-md p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            onClick={() => onOpenChange(false)}
            aria-label={t("shortcuts.close")}
          >
            <kbd>Esc</kbd>
          </button>
        </div>

        <div className="grid gap-x-8 gap-y-5 overflow-y-auto p-5 sm:grid-cols-2">
          {GROUPS.map((g) => (
            <div key={g.labelKey} className="space-y-2">
              <h3 className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                {t(g.labelKey)}
              </h3>
              <ul className="space-y-1.5">
                {g.items.map((it, i) => (
                  <li
                    key={i}
                    className="flex items-center justify-between gap-3"
                  >
                    <span className="text-xs text-foreground/90">
                      {t(it.descKey)}
                    </span>
                    <span className="flex items-center gap-1">
                      {it.keys.map((k, j) => (
                        <Fragment key={j}>
                          <kbd>{k}</kbd>
                          {j < it.keys.length - 1 && (
                            <span className="text-[10px] text-muted-foreground">
                              +
                            </span>
                          )}
                        </Fragment>
                      ))}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="flex items-center justify-between border-t bg-muted/30 px-5 py-2.5 text-[11px] text-muted-foreground">
          <span>
            {t("shortcuts.footerHint")} <kbd>?</kbd>
          </span>
          <span>{t("shortcuts.footerBrand")}</span>
        </div>
      </DialogContent>
    </Dialog>
  );
}
