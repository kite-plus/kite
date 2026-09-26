import { createFileRoute } from "@tanstack/react-router";

import { PageError } from "@/features/errors/page-error";
import { PluginSettings } from "@/features/plugins/plugin";

export const Route = createFileRoute("/_authenticated/plugins/$id")({
  component: Plugin,
  errorComponent: PageError,
});

// Keyed by the plugin, so going from one plugin's page to another's starts over.
function Plugin() {
  const { id } = Route.useParams();
  return <PluginSettings key={id} id={id} />;
}
