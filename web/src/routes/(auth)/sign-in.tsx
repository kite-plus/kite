import { createFileRoute, redirect } from "@tanstack/react-router";

import { sessionQuery } from "@/hooks/useSession";
import { setupQuery } from "@/hooks/useSetup";
import { within } from "@/lib/gate";
import { SignIn } from "@/features/auth/sign-in";

export const Route = createFileRoute("/(auth)/sign-in")({
  validateSearch: (search: Record<string, unknown>): { redirect?: string } => ({
    redirect: within(search.redirect),
  }),
  // Somebody already signed in, or a server with no password, has no use
  // for the form and goes where they were headed.
  beforeLoad: async ({ context, search }) => {
    const setup = await context.queryClient.ensureQueryData(setupQuery);
    if (setup.required) throw redirect({ to: "/setup" });
    const session = await context.queryClient.ensureQueryData(sessionQuery);
    if (!session.required || session.authenticated) {
      throw redirect({ href: search.redirect || "/" });
    }
  },
  component: SignIn,
});
