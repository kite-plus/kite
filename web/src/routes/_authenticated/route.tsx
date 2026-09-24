import { createFileRoute } from "@tanstack/react-router";

import { gate } from "@/lib/gate";
import { AuthenticatedLayout } from "@/components/layout/authenticated-layout";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: ({ context, location }) => gate(context.queryClient, location),
  component: AuthenticatedLayout,
});
