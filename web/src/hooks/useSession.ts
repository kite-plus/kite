import { useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, onRefused, unwrap, type components } from "@/api/client";

export type Session = components["schemas"]["SessionInfo"];
export type Credentials = components["schemas"]["Credentials"];

const key = ["session"];

/**
 * Who the studio is being used by, and whether it needs to know.
 *
 * A server with no account answers required: false, and the admin then never
 * shows a sign-in form -- which is what keeps a local preview of a project
 * that has no password exactly as it was.
 */
export function useSession() {
  const client = useQueryClient();

  useEffect(
    () =>
      onRefused(() => {
        client.setQueryData<Session>(key, (current) =>
          current && { ...current, authenticated: false, user: undefined },
        );
      }),
    [client],
  );

  return useQuery({
    queryKey: key,
    queryFn: async () => unwrap(await api.GET("/auth/session", {})),
    // Asked for once per page load and then only when something says
    // otherwise: the answer changes on sign-in, sign-out and a refusal, and
    // all three already write it here.
    staleTime: Infinity,
    retry: false,
  });
}

export function useSignIn() {
  const client = useQueryClient();

  return useMutation({
    mutationFn: async (credentials: Credentials) =>
      unwrap(await api.POST("/auth/login", { body: credentials })),
    onSuccess: (session) => {
      client.setQueryData<Session>(key, session);
      // Everything already on screen was loaded, or refused, for nobody.
      void client.invalidateQueries();
    },
  });
}

export function useSignOut() {
  const client = useQueryClient();

  return useMutation({
    mutationFn: async () => {
      const { error } = await api.POST("/auth/logout", {});
      if (error) throw new Error(error.error.message);
    },
    onSuccess: () => {
      // The session first: it is what the sign-in form is put back on screen
      // by, and clearing the cache out from under a live query instead would
      // leave every observer holding what it last saw.
      client.setQueryData<Session>(key, { required: true, authenticated: false });

      // Then the project itself, because what somebody was reading should not
      // still be in this tab for whoever opens it next.
      client.removeQueries({
        predicate: (query) => query.queryKey[0] !== key[0],
      });
    },
  });
}
