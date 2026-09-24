import { createFileRoute } from "@tanstack/react-router";

import { ThemeSettings } from "@/features/settings/theme";

export const Route = createFileRoute("/_authenticated/settings/theme")({
  component: ThemeSettings,
});
