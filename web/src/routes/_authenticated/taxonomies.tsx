import { createFileRoute } from "@tanstack/react-router";

import { Taxonomies } from "@/features/taxonomies";
import { PageError } from "@/features/errors/page-error";

export const Route = createFileRoute("/_authenticated/taxonomies")({
  component: Taxonomies,
  errorComponent: PageError,
});
