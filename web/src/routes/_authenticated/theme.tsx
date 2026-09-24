import { createFileRoute, redirect } from "@tanstack/react-router";

// The theme's settings moved under the settings screen; an old link still lands.
export const Route = createFileRoute("/_authenticated/theme")({
  beforeLoad: () => {
    throw redirect({ to: "/settings/theme", replace: true });
  },
});
