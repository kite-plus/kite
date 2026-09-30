import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, ApiError, heard, unwrap, type Media } from "@/api/client";

const key = (id: string) => ["media", id];

/** useBundle lists the files an item keeps beside it, which are published with its page. */
export function useBundle(id: string) {
  return useQuery({
    queryKey: key(id),
    queryFn: async () => unwrap(await api.GET("/contents/{id}/media", { params: { path: { id } } })),
  });
}

/** useBundleChanged has the list read again after a file was added. */
export function useBundleChanged(id: string) {
  const client = useQueryClient();
  return () => client.invalidateQueries({ queryKey: key(id) });
}

/**
 * useReplaceFile puts a new version of a file in its place, under its name,
 * and resolves to it as stored.
 */
export function useReplaceFile(id: string) {
  const changed = useBundleChanged(id);
  return useMutation({
    mutationFn: async ({ name, file }: { name: string; file: File }): Promise<Media> => {
      const form = new FormData();
      form.append("file", file);
      const response = await fetch(
        `/api/v1/contents/${encodeURIComponent(id)}/media/${encodeURIComponent(name)}`,
        { method: "PUT", body: form, credentials: "same-origin" },
      );
      heard(response.status);
      const body = (await response.json().catch(() => null)) as
        | (Media & { error?: { code?: string; message?: string } })
        | null;
      if (!response.ok || !body?.link) {
        throw new ApiError(body?.error?.code ?? "internal", body?.error?.message ?? "replace failed");
      }
      return body;
    },
    onSuccess: () => changed(),
  });
}

/** useRemoveFile takes a file out of an item's bundle. */
export function useRemoveFile(id: string) {
  const changed = useBundleChanged(id);
  return useMutation({
    mutationFn: async (name: string) => {
      const { error } = await api.DELETE("/contents/{id}/media/{name}", { params: { path: { id, name } } });
      if (error) throw new ApiError(error.error.code, error.error.message);
    },
    onSuccess: () => changed(),
  });
}
