import { createFileRoute } from "@tanstack/react-router";

import { EditorPage } from "@/components/editor/EditorPage";

export const Route = createFileRoute("/_authenticated/content/$kind/$id")({
  component: Editor,
});

// Not keyed by the id: a new item's first save moves it to its own address,
// and the editor has to stay as it was across that move.
function Editor() {
  const { kind, id } = Route.useParams();
  return <EditorPage id={id === "new" ? null : id} kind={kind} />;
}
