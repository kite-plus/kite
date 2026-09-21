import createClient from "openapi-fetch";
import type { paths, components } from "./schema";

/**
 * The one place a request is made.
 *
 * Paths, parameters and response shapes are checked against schema.d.ts,
 * which is generated from the server's own description by `pnpm gen`. A
 * hand-written fetch would compile happily against an endpoint that no longer
 * exists; this will not.
 */
export const api = createClient<paths>({ baseUrl: "/api/v1" });

export type Summary = components["schemas"]["Summary"];
export type Item = components["schemas"]["Item"];
export type Draft = components["schemas"]["Draft"];
export type SiteInfo = components["schemas"]["SiteInfo"];
export type ContentType = components["schemas"]["ContentType"];
export type Taxonomy = components["schemas"]["Taxonomy"];
export type TermCount = components["schemas"]["TermCount"];
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
