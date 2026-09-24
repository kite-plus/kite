import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, unwrap, type components } from "@/api/client";
import type { Session } from "@/hooks/useSession";

export type SetupState = components["schemas"]["SetupState"];
export type SetupRequest = components["schemas"]["SetupRequest"];

const key = ["setup"];

/** setupQuery asks whether this server has an owner yet. */
export const setupQuery = {
  queryKey: key,
  queryFn: async () => unwrap(await api.GET("/setup", {})),
  // A server is set up once. The answer changes exactly when this tab
  // changes it, and that path writes it here itself.
  staleTime: Infinity,
  retry: false,
};

/**
 * Whether this server has ever been configured.
 *
 * It is asked before the session, and it is the one question a server waiting
 * to be set up will answer: until it has an owner, everything else comes back
 * refused, so a studio that went looking for content first would render an
 * error page over a working installer.
 */
export function useSetup() {
  return useQuery(setupQuery);
}

/** useInstall finishes setup and leaves the browser signed in. */
export function useInstall() {
  const client = useQueryClient();

  return useMutation({
    mutationFn: async (request: SetupRequest) =>
      unwrap(await api.POST("/setup", { body: request })),
    onSuccess: (session) => {
      client.setQueryData<SetupState>(key, { required: false });
      client.setQueryData<Session>(["session"], session);
      // Nothing on screen was loaded for anybody, because nothing would have
      // answered until now.
      void client.invalidateQueries();
    },
  });
}
