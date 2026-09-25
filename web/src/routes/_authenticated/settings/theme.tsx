import { createFileRoute } from "@tanstack/react-router";

import { Themes } from "@/features/settings/theme";

export const Route = createFileRoute("/_authenticated/settings/theme")({
  component: Themes,
});
