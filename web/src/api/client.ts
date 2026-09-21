import createClient from "openapi-fetch";
import type { paths, components } from "./schema";
export type { components };

/**
 * The one place a request is made.
 *
 * Paths, parameters and response shapes are checked against schema.d.ts,
 * which is generated from the server's own description by `pnpm gen`. A
 * hand-written fetch would compile happily against an endpoint that no longer
 * exists; this will not.
 */
export const api = createClient<paths>({
  baseUrl: "/api/v1",
  // The session is a cookie, so every call carries it. Saying so explicitly
  // is the only note in this file about how the API knows who is asking.
  credentials: "same-origin",
});

/**
 * Listeners for the moment a request comes back refused.
 *
 * A session ends without the page being told: the cookie runs out, the server
 * is restarted under a new password, somebody signs out in another tab. The
 * first anybody hears of it is a 401 on whatever was asked for next, so that
 * is what puts the sign-in form back on screen.
 */
const refused = new Set<() => void>();

export function onRefused(listener: () => void): () => void {
  refused.add(listener);
  return () => {
    refused.delete(listener);
  };
}

api.use({
  onResponse({ response }) {
    if (response.status === 401) {
      for (const listener of refused) listener();
    }
    return response;
  },
});

export type Summary = components["schemas"]["Summary"];
export type Item = components["schemas"]["Item"];
export type Draft = components["schemas"]["Draft"];
export type SiteInfo = components["schemas"]["SiteInfo"];
export type ContentType = components["schemas"]["ContentType"];
export type Taxonomy = components["schemas"]["Taxonomy"];
export type TermCount = components["schemas"]["TermCount"];
export type Settings = components["schemas"]["Settings"];
export type ErrorBody = components["schemas"]["ErrorBody"];

/** ApiError carries the server's machine-readable code, not just a message. */
export class ApiError extends Error {
  constructor(
    readonly code: string,
    message: string,
    readonly field?: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

/** unwrap turns an openapi-fetch result into a value or a thrown ApiError. */
export function unwrap<T>(result: {
  data?: T;
  error?: ErrorBody;
  response: Response;
}): T {
  if (result.error) {
    const { code, message, field } = result.error.error;
    throw new ApiError(code, message, field);
  }
  if (result.data === undefined) {
    throw new ApiError("internal", `${result.response.status} with no body`);
  }
  return result.data;
}
