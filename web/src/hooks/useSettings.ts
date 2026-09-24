import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, unwrap } from "@/api/client";

const key = ["settings"];

export function useSettings() {
  return useQuery({
    queryKey: key,
    queryFn: async () => {
      const result = await api.GET("/settings", {});
      return { ...unwrap(result), revision: result.response.headers.get("ETag") ?? "" };
    },
  });
}

/** useSaveSettings writes values by dotted config path, e.g. "site.title". */
export function useSaveSettings() {
  const client = useQueryClient();

  return useMutation({
    mutationFn: async ({ changes, revision }: { changes: Record<string, unknown>; revision: string }) => {
      const result = await api.PUT("/settings", {
        body: changes,
        params: { header: { "If-Match": revision } },
      });
      return { ...unwrap(result), revision: result.response.headers.get("ETag") ?? "" };
    },
    onSuccess: async (settings) => {
      client.setQueryData(key, settings);
      await client.invalidateQueries({ queryKey: ["site"] });
    },
  });
}
