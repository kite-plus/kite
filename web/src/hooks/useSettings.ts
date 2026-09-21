import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, unwrap, type Settings } from "@/api/client";

const key = ["settings"];

export function useSettings() {
  return useQuery({
    queryKey: key,
    queryFn: async () => unwrap(await api.GET("/settings", {})),
  });
}

/** useSaveSettings writes values by dotted config path, e.g. "site.title". */
export function useSaveSettings() {
  const client = useQueryClient();

  return useMutation({
    mutationFn: async (changes: Record<string, unknown>) =>
      unwrap(await api.PUT("/settings", { body: changes })),
    onSuccess: async (settings: Settings) => {
      client.setQueryData(key, settings);
      await client.invalidateQueries({ queryKey: ["site"] });
    },
  });
}
