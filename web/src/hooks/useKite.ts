import { useMutation, useQueryClient } from "@tanstack/react-query";

import { api, unwrap } from "@/api/client";

/**
 * usePinKite pins the Kite release serving the studio in kite.lock. The
 * deploy builds with it once kite.lock is published, which the publish bar
 * then offers.
 */
export function usePinKite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async () => unwrap(await api.POST("/site/pin", {})),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["site"] });
      void queryClient.invalidateQueries({ queryKey: ["publish"] });
    },
  });
}
