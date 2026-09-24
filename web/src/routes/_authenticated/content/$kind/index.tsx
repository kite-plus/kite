import { createFileRoute } from "@tanstack/react-router";

import { ContentList } from "@/features/contents";
import { validateContentSearch } from "@/features/contents/search";

export const Route = createFileRoute("/_authenticated/content/$kind/")({
  validateSearch: validateContentSearch,
  component: ContentList,
});
