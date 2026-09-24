import { createFileRoute, notFound, redirect } from "@tanstack/react-router";

import { taxonomiesQuery } from "@/hooks/useContents";

export const Route = createFileRoute("/_authenticated/taxonomies/")({
  // The page that once held every taxonomy leads to the first of them.
  beforeLoad: async ({ context }) => {
    const taxonomies = await context.queryClient.ensureQueryData(taxonomiesQuery);
    const first = taxonomies.items[0]?.name;
    if (!first) throw notFound();
    throw redirect({ to: "/taxonomies/$taxonomy", params: { taxonomy: first }, replace: true });
  },
});
