import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError, api, unwrap, type components } from "@/api/client";

export type PluginInfo = components["schemas"]["PluginInfo"];
export type PluginDetail = components["schemas"]["PluginDetail"];

export function usePlugins() {
  return useQuery({
    queryKey: ["plugins"],
    queryFn: async () => unwrap(await api.GET("/plugins", {})).items,
  });
}

/** usePlugin reads one plugin with its settings and the revision to save them against. */
export function usePlugin(id: string) {
  return useQuery({
    queryKey: ["plugin", id],
    queryFn: async () => {
      const result = await api.GET("/plugins/{id}", { params: { path: { id } } });
      return { ...unwrap(result), revision: result.response.headers.get("ETag") ?? "" };
    },
  });
}

/**
 * usePluginsChanged refreshes what a plugin change can alter: the plugins,
 * the settings kite.yaml holds, and what is waiting to be published.
 */
export function usePluginsChanged() {
  const client = useQueryClient();
  return () =>
    Promise.all(
      [["plugins"], ["plugin"], ["settings"], ["publish"]].map((queryKey) =>
        client.invalidateQueries({ queryKey }),
      ),
    );
}

/**
 * useSavePlugins writes plugin settings by dotted path, as
 * plugins.enabled or plugins.settings.<id>.<key>, against a revision of
 * kite.yaml.
 */
export function useSavePlugins() {
  const changed = usePluginsChanged();
  return useMutation({
    mutationFn: async ({ changes, revision }: { changes: Record<string, unknown>; revision: string }) => {
      const result = await api.PUT("/settings", { body: changes, params: { header: { "If-Match": revision } } });
      return { ...unwrap(result), revision: result.response.headers.get("ETag") ?? "" };
    },
    onSuccess: () => changed(),
  });
}

/** useSwitchPlugin turns a plugin on, after the ones already on, or off. */
export function useSwitchPlugin() {
  const changed = usePluginsChanged();
  return useMutation({
    mutationFn: async ({ id, enabled }: { id: string; enabled: boolean }) =>
      unwrap(await api.PUT("/plugins/{id}/enabled", { params: { path: { id } }, body: { enabled } })),
    onSuccess: () => changed(),
  });
}

export function useRemovePlugin() {
  const changed = usePluginsChanged();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE("/plugins/{id}", { params: { path: { id } } });
      if (error) throw new ApiError(error.error.code, error.error.message, error.error.field);
    },
    onSuccess: () => changed(),
  });
}

/** What installing a plugin came to: installed, or waiting to be told to replace one. */
export type Installed =
  | { status: "installed"; plugin: PluginInfo }
  | { status: "exists"; installed: PluginInfo; uploaded: PluginInfo };

/**
 * installPlugin sends a plugin's zip archive. One already installed with the
 * same id is replaced only when asked; otherwise both versions come back to
 * be asked about.
 */
export async function installPlugin(file: File, replace: boolean): Promise<Installed> {
  const form = new FormData();
  form.append("file", file);
  let response: Response;
  try {
    response = await fetch(`/api/v1/plugins${replace ? "?replace=true" : ""}`, {
      method: "POST",
      body: form,
      credentials: "same-origin",
    });
  } catch (err) {
    throw new ApiError("unreachable", err instanceof Error ? err.message : String(err));
  }
  const body = (await response.json().catch(() => null)) as
    | (PluginInfo & { installed?: PluginInfo; uploaded?: PluginInfo; error?: { code?: string; message?: string } })
    | null;
  if (response.status === 201 && body) return { status: "installed", plugin: body };
  if (response.status === 409 && body?.installed && body.uploaded) {
    return { status: "exists", installed: body.installed, uploaded: body.uploaded };
  }
  throw new ApiError(body?.error?.code ?? "internal", body?.error?.message ?? "install failed");
}
