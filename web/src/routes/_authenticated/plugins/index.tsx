import { createFileRoute } from "@tanstack/react-router";

import { PageError } from "@/features/errors/page-error";
import { Plugins } from "@/features/plugins";

export const Route = createFileRoute("/_authenticated/plugins/")({
  component: Plugins,
  errorComponent: PageError,
});
