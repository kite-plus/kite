import { useState } from "react";
import { Unlink } from "lucide-react";

import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface Props {
  href?: string;
  onApply: (href: string) => void;
  onRemove?: () => void;
  onCancel: () => void;
}

/** One field for a link's address, used wherever a link is made or changed. */
export function LinkForm({ href, onApply, onRemove, onCancel }: Props) {
  const { t } = useI18n();
  const [value, setValue] = useState(href ?? "");

  return (
    <form
      className="flex items-center gap-1.5"
      onSubmit={(event) => {
        event.preventDefault();
        onApply(value.trim());
      }}
    >
      <Input
        autoFocus
        value={value}
        onChange={(event) => setValue(event.target.value)}
        placeholder="https://"
        aria-label={t("editor.linkURL")}
        className="h-7 w-60 text-xs md:text-xs"
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            onCancel();
          }
        }}
      />
      <Button type="submit" size="sm">
        {t("editor.linkApply")}
      </Button>
      {onRemove && (
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label={t("editor.linkRemove")}
          onClick={onRemove}
        >
          <Unlink />
        </Button>
      )}
    </form>
  );
}
