import { XCircle } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

/** A failure a settings form reached, said the same way on every page. */
export function SettingsProblem({ title, detail }: { title: string; detail?: string }) {
  return (
    <Alert variant="destructive">
      <XCircle />
      <AlertTitle>{title}</AlertTitle>
      {detail && <AlertDescription>{detail}</AlertDescription>}
    </Alert>
  );
}
