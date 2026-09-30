import { createFileRoute } from "@tanstack/react-router";

import { MenuSettings } from "@/features/settings/menus";

export const Route = createFileRoute("/_authenticated/settings/menus")({
  component: MenuSettings,
});
