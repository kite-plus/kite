import { Info } from "lucide-react";

import { useWritable } from "@/hooks/useContents";
import { useI18n } from "@/i18n";
import { Alert, AlertDescription } from "@/components/ui/alert";

/** ReadOnlyNote says, on a server that takes no changes, why none can be made. */
export function ReadOnlyNote({ className }: { className?: string }) {
  const { t } = useI18n();
  if (useWritable()) return null;
  return (
    <Alert className={className}>
      <Info />
      <AlertDescription>{t("session.readOnlyNote")}</AlertDescription>
    </Alert>
  );
}
