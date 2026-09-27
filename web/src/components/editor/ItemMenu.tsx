import { useState } from "react";
import { MoreHorizontal, SlidersHorizontal, Trash2 } from "lucide-react";

import type { Draft, Field } from "@/api/client";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { fieldLabel, SchemaForm, type Uploads } from "@/components/SchemaForm";

/**
 * ItemMenu holds what is set on an item now and then: a switch such as
 * pinning is one click, any other field the kind declares opens in a dialog,
 * and deleting comes last.
 */
export function ItemMenu({
  draft,
  fields,
  onEdit,
  uploads,
  onDelete,
}: {
  draft: Draft;
  fields: Field[];
  onEdit: (patch: Partial<Draft>) => void;
  uploads: Uploads;
  onDelete?: () => void;
}) {
  const { t } = useI18n();
  const [more, setMore] = useState(false);
  const meta = draft.meta ?? {};
  const switches = fields.filter((field) => field.type === "boolean");
  const rest = fields.filter((field) => field.type !== "boolean");
  if (switches.length === 0 && rest.length === 0 && !onDelete) return null;

  return (
    <>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="icon" className="size-8 shrink-0" aria-label={t("editor.moreActions")}>
            <MoreHorizontal />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-52">
          {switches.map((field) => (
            <DropdownMenuCheckboxItem
              key={field.key}
              checked={Boolean(meta[field.key] ?? field.default)}
              onCheckedChange={(checked) => onEdit({ meta: { ...meta, [field.key]: checked } })}
            >
              {fieldLabel(field, t)}
            </DropdownMenuCheckboxItem>
          ))}
          {rest.length > 0 && (
            <DropdownMenuItem onSelect={() => setMore(true)}>
              <SlidersHorizontal />
              {t("editor.moreSettings")}
            </DropdownMenuItem>
          )}
          {onDelete && (
            <>
              {(switches.length > 0 || rest.length > 0) && <DropdownMenuSeparator />}
              <DropdownMenuItem variant="destructive" onSelect={onDelete}>
                <Trash2 />
                {t("editor.delete")}
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={more} onOpenChange={setMore}>
        <DialogContent className="max-h-[85vh] overflow-auto sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("editor.moreSettings")}</DialogTitle>
            <DialogDescription>{t("editor.moreSettingsNote")}</DialogDescription>
          </DialogHeader>
          <SchemaForm
            fields={rest}
            values={meta}
            onChange={(next) => onEdit({ meta: next })}
            uploads={uploads}
          />
        </DialogContent>
      </Dialog>
    </>
  );
}
