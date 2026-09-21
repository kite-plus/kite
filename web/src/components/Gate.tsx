import type { ReactNode } from "react";
import { XCircle } from "lucide-react";

import { useI18n } from "@/i18n";
import { useSession } from "@/hooks/useSession";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { LoginPage } from "@/components/LoginPage";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

/**
 * What stands between a browser and the studio.
 *
 * Whether somebody is signed in is asked of the server rather than worked out
 * here: the cookie is HttpOnly, so this page cannot read it, and that is the
 * point -- a client that decided for itself would be deciding something only
 * the server is in a position to know.
 *
 * It wraps the studio rather than living inside it so that a signed-out page
 * asks for nothing else. There is no use loading a content list that is about
 * to come back refused.
 */
export function Gate({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  const session = useSession();

  if (session.isPending) {
    return (
      <div className="flex min-h-svh items-center justify-center gap-2 text-sm text-muted-foreground">
        <Spinner />
        {t("session.checking")}
      </div>
    );
  }

  if (session.isError) {
    return (
      <div className="flex min-h-svh items-center justify-center p-4">
        <Alert variant="destructive" className="max-w-sm">
          <XCircle />
          <AlertTitle>{t("session.unreachable")}</AlertTitle>
          <AlertDescription>
            <p className="font-mono text-xs">{String(session.error)}</p>
            <Button
              size="sm"
              variant="outline"
              onClick={() => void session.refetch()}
            >
              {t("session.retry")}
            </Button>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (session.data.required && !session.data.authenticated) {
    return <LoginPage />;
  }
  return <>{children}</>;
}
