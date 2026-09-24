import { createFileRoute } from "@tanstack/react-router";

import { Dashboard } from "@/features/dashboard";
import { PageError } from "@/features/errors/page-error";

export const Route = createFileRoute("/_authenticated/")({
  component: Dashboard,
  errorComponent: PageError,
});
