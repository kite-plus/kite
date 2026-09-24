import { createFileRoute, notFound } from "@tanstack/react-router";

import { contentTypesQuery } from "@/hooks/useContents";

export const Route = createFileRoute("/_authenticated/content/$kind")({
  // A kind the project does not define has nothing to list and nothing to write.
  beforeLoad: async ({ context, params }) => {
    const types = await context.queryClient.ensureQueryData(contentTypesQuery);
    if (!types.items.some((type) => type.kind === params.kind)) throw notFound();
  },
});
