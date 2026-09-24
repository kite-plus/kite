import { createFileRoute, notFound } from "@tanstack/react-router";

import { taxonomiesQuery } from "@/hooks/useContents";
import { PageError } from "@/features/errors/page-error";
import { Terms } from "@/features/terms";

export const Route = createFileRoute("/_authenticated/taxonomies/$taxonomy")({
  // A taxonomy the project does not define has no terms to list.
  beforeLoad: async ({ context, params }) => {
    const taxonomies = await context.queryClient.ensureQueryData(taxonomiesQuery);
    if (!taxonomies.items.some((item) => item.name === params.taxonomy)) throw notFound();
  },
  component: TermsOf,
  errorComponent: PageError,
});

// Keyed by the taxonomy, so a search or a sort made among the tags is not
// carried over to the categories.
function TermsOf() {
  const { taxonomy } = Route.useParams();
  return <Terms key={taxonomy} />;
}
