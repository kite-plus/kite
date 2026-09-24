import { createFileRoute, redirect } from "@tanstack/react-router";

import { setupQuery } from "@/hooks/useSetup";
import { Setup } from "@/features/auth/setup";

export const Route = createFileRoute("/(auth)/setup")({
  // A server is set up once; after that this address leads into the studio.
  beforeLoad: async ({ context }) => {
    const setup = await context.queryClient.ensureQueryData(setupQuery);
    if (!setup.required) throw redirect({ to: "/" });
  },
  component: Setup,
});
