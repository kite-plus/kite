import type { ReactElement } from "react";

import { useI18n } from "@/i18n";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

/**
 * A part of the design Kite has nothing behind yet.
 *
 * It is drawn where the design puts it, greyed, and says so on hover, rather
 * than filled with numbers nobody measured.
 */
export function Soon({
  children,
  side = "top",
}: {
  children: ReactElement;
  side?: "top" | "right" | "bottom" | "left";
}) {
  const { t } = useI18n();
  return (
    <Tooltip>
      <TooltipTrigger render={children} />
      <TooltipContent side={side}>{t("common.soon")}</TooltipContent>
    </Tooltip>
  );
}
