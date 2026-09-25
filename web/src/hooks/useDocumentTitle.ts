import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { siteQuery } from "@/hooks/useContents";

/**
 * useDocumentTitle names the browser tab after the page and the site, as
 * "Posts - My Blog", so each open tab of the studio says where it is.
 *
 * The site is read from what the rail has fetched rather than fetched here:
 * reading it takes a session, which the sign-in and setup pages do not have,
 * so there "Kite" stands in for its name.
 */
export function useDocumentTitle(page?: string) {
  const site = useQuery({ ...siteQuery, enabled: false }).data?.title;

  useEffect(() => {
    document.title = [page, site || "Kite"].filter(Boolean).join(" - ");
  }, [page, site]);
}
