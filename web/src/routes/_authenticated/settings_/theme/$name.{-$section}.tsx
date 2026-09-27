import { createFileRoute } from "@tanstack/react-router";

import { PageError } from "@/features/errors/page-error";
import { ThemeCustomizer } from "@/features/settings/theme/customize";

// Outside the settings screen, like the editor: the preview wants the room.
// Each of the theme's sections has an address of its own.
export const Route = createFileRoute("/_authenticated/settings_/theme/$name/{-$section}")({
  component: Customizer,
  errorComponent: PageError,
});

// Keyed by the theme, so going from one theme's page to another's starts over.
function Customizer() {
  const { name, section } = Route.useParams();
  return <ThemeCustomizer key={name} name={name} section={section} />;
}
