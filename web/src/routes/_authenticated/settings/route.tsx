import { createFileRoute } from "@tanstack/react-router";

import { Settings } from "@/features/settings";
import { PageError } from "@/features/errors/page-error";

export const Route = createFileRoute("/_authenticated/settings")({
  component: Settings,
  errorComponent: PageError,
});
