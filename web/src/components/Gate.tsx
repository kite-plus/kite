import type { ReactNode } from "react";
import { XCircle } from "lucide-react";

import { useI18n } from "@/i18n";
import { useSession } from "@/hooks/useSession";
import { useSetup } from "@/hooks/useSetup";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { LoginPage } from "@/components/LoginPage";
import { SetupPage } from "@/components/SetupPage";
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
 *
 * Setup is asked about before the session, because a server that has never
 * been configured has no account for a session to belong to and will refuse
 * every other question. Answering them in the other order would put an error
 * on screen where an installer belongs.
 */
export function Gate({ children }: { children: ReactNode }) {
  const { t } = useI18n();
  const setup = useSetup();
  const session = useSession();

  if (setup.isPending || session.isPending) {
    return (
      <div className="flex min-h-svh items-center justify-center gap-2 text-sm text-muted-foreground">
        <Spinner />
        {t("session.checking")}
      </div>
    );
  }

  const unreachable = setup.error ?? session.error;
  if (unreachable) {
    return (
      <div className="flex min-h-svh items-center justify-center p-4">
        <Alert variant="destructive" className="max-w-sm">
          <XCircle />
          <AlertTitle>{t("session.unreachable")}</AlertTitle>
          <AlertDescription>
            <p className="font-mono text-xs">{String(unreachable)}</p>
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                void setup.refetch();
                void session.refetch();
              }}
            >
              {t("session.retry")}
            </Button>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (setup.data?.required) {
    return <SetupPage state={setup.data} />;
  }
  if (session.data?.required && !session.data.authenticated) {
    return <LoginPage />;
  }
  return <>{children}</>;
}
