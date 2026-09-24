import type { QueryClient } from "@tanstack/react-query";
import { redirect, type ParsedLocation } from "@tanstack/react-router";

import { sessionQuery } from "@/hooks/useSession";
import { setupQuery } from "@/hooks/useSetup";

/**
 * gate lets a screen be drawn only for a server that is set up, by somebody
 * signed in to it.
 *
 * Whether somebody is signed in is asked of the server rather than worked out
 * here: the cookie is HttpOnly, so this page cannot read it, and that is the
 * point. Setup is asked about first, because a server that has never been
 * configured has no account for a session to belong to and refuses every
 * other question.
 */
export async function gate(queryClient: QueryClient, location: ParsedLocation) {
  const setup = await queryClient.ensureQueryData(setupQuery);
  if (setup.required) throw redirect({ to: "/setup" });

  const session = await queryClient.ensureQueryData(sessionQuery);
  if (session.required && !session.authenticated) {
    throw redirect({ to: "/sign-in", search: { redirect: location.href } });
  }
}

/**
 * within keeps a path to return to only when it is one of the studio's own,
 * so a link crafted elsewhere cannot send a person off it after signing in.
 * It is an href within the studio, search included, and is followed as one.
 */
export function within(path: unknown): string | undefined {
  // Browsers read "/\" as "//", and drop tabs and newlines, so both lead off-site.
  return typeof path === "string" && /^\/(?![\t\n\r]*[\\/])/.test(path) ? path : undefined;
}
