import { createFileRoute } from "@tanstack/react-router";

import { AppearanceSettings } from "@/features/settings/appearance";

export const Route = createFileRoute("/_authenticated/settings/appearance")({
  component: AppearanceSettings,
});
