import { Link } from "@tanstack/react-router";
import { ExternalLink, MoreHorizontal, Pencil, RotateCcw, Trash2 } from "lucide-react";

import type { Summary } from "@/api/client";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface Props {
  item: Summary;
  trashed: boolean;
  onDelete: (item: Summary) => void;
  onRestore: (item: Summary) => void;
}

export function RowActions({ item, trashed, onDelete, onRestore }: Props) {
  const { t } = useI18n();

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" className="flex size-8 p-0 data-[state=open]:bg-muted">
          <MoreHorizontal className="size-4" />
          <span className="sr-only">{t("list.actions")}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        {trashed ? (
          <DropdownMenuItem onClick={() => onRestore(item)}>
            <RotateCcw />
            {t("list.restore")}
          </DropdownMenuItem>
        ) : (
          <>
            <DropdownMenuItem asChild>
              <Link to="/content/$kind/$id" params={{ kind: item.kind, id: item.id }}>
                <Pencil />
                {t("list.edit")}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <a href={item.url} target="_blank" rel="noreferrer">
                <ExternalLink />
                {t("list.open")}
              </a>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={() => onDelete(item)}>
              <Trash2 />
              {t("editor.delete")}
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
