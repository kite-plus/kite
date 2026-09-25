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

/**
 * The language the admin is shown in. A theme describes itself in it, from
 * its own language pack, and the browser's Accept-Language would name the
 * browser's language rather than the one chosen here.
 */
let language = "";

export function speak(locale: string) {
  language = locale;
}

api.use({
  onRequest({ request }) {
    if (language) request.headers.set("Accept-Language", language);
    return request;
  },
  onResponse({ response }) {
    if (response.status === 401) {
      for (const listener of refused) listener();
    }
    return response;
  },
  // fetch rejects with a TypeError, worded differently by every browser, when
  // the server cannot be reached at all. An abort is left as it is.
  onError({ error }) {
    if (error instanceof TypeError) return new ApiError("unreachable", error.message);
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
export type Field = components["schemas"]["Field"];
export type ThemeInfo = components["schemas"]["ThemeInfo"];
export type ThemeDetail = components["schemas"]["ThemeDetail"];
export type ThemeExists = components["schemas"]["ThemeExists"];
export type Media = components["schemas"]["Media"];
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
