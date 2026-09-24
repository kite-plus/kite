import { createFileRoute } from "@tanstack/react-router";

import { Taxonomies } from "@/features/taxonomies";

export const Route = createFileRoute("/_authenticated/taxonomies")({
  component: Taxonomies,
});
