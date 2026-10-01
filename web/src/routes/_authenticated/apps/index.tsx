import { createFileRoute } from "@tanstack/react-router";

import { Apps } from "@/features/apps";
import { validateAppsSearch } from "@/features/apps/search";
import { PageError } from "@/features/errors/page-error";

export const Route = createFileRoute("/_authenticated/apps/")({
  validateSearch: validateAppsSearch,
  component: Apps,
  errorComponent: PageError,
});
