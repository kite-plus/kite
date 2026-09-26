import { createFileRoute } from "@tanstack/react-router";

import { Deploy } from "@/features/deploy";

export const Route = createFileRoute("/_authenticated/deploy")({
  component: Deploy,
});
