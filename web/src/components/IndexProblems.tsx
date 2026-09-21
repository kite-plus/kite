import { TriangleAlert } from "lucide-react";

import { useI18n } from "@/i18n";
import { useSite } from "@/hooks/useContents";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

/** Files the index refused, which would otherwise be missing without a word. */
export function IndexProblems() {
  const { t } = useI18n();
  const site = useSite();
  const problems = site.data?.problems ?? [];
  if (problems.length === 0) return null;

  return (
    <Alert>
      <TriangleAlert />
      <AlertTitle>{t("problems.notIndexed", { count: problems.length })}</AlertTitle>
      <AlertDescription>
        <ul className="flex flex-col gap-0.5 font-mono text-xs">
          {problems.map((problem) => (
            <li key={problem}>{problem}</li>
          ))}
        </ul>
      </AlertDescription>
    </Alert>
  );
}
