import { createFileRoute } from "@tanstack/react-router";

import { SiteSettings } from "@/features/settings/site";

export const Route = createFileRoute("/_authenticated/settings/")({
  component: SiteSettings,
});
