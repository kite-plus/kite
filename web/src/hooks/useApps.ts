import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError, api, unwrap, type components } from "@/api/client";

export type AppInfo = components["schemas"]["AppInfo"];
export type AppDetail = components["schemas"]["AppDetail"];
export type AppList = components["schemas"]["AppList"];
export type UpdatePlan = components["schemas"]["UpdatePlan"];
export type AppKind = "theme" | "plugin";

/**
 * useApps reads what the index lists, as the site stands with each package.
 * The server keeps the index for an hour, so this asks it, not the network,
 * unless refresh says to fetch it again.
 */
export function useApps() {
  return useQuery({
    queryKey: ["apps"],
    queryFn: async () => unwrap(await api.GET("/apps", {})),
    staleTime: 5 * 60 * 1000,
    retry: false,
  });
}

/** useRefreshApps fetches the index again, past the copy the server keeps. */
export function useRefreshApps() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => unwrap(await api.GET("/apps", { params: { query: { refresh: true } } })),
    onSuccess: (list) => client.setQueryData(["apps"], list),
  });
}

/** useApp reads one package with every version the index lists. */
export function useApp(kind: AppKind, id: string, enabled = true) {
  return useQuery({
    queryKey: ["app", kind, id],
    queryFn: async () => unwrap(await api.GET("/apps/{kind}/{id}", { params: { path: { kind, id } } })),
    enabled,
    retry: false,
  });
}

/**
 * useAppsChanged refreshes what installing or updating a package can alter:
 * the index's view of the site, the themes and plugins, and what is waiting
 * to be published.
 */
export function useAppsChanged() {
  const client = useQueryClient();
  return () =>
    Promise.all(
      [["apps"], ["app"], ["themes"], ["theme"], ["plugins"], ["plugin"], ["publish"]].map((queryKey) =>
        client.invalidateQueries({ queryKey }),
      ),
    );
}

export function useInstallApp() {
  const changed = useAppsChanged();
  return useMutation({
    mutationFn: async ({ kind, id, replace }: { kind: AppKind; id: string; replace: boolean }) =>
      unwrap(
        await api.POST("/apps/{kind}/{id}/install", {
          params: { path: { kind, id }, query: replace ? { replace: true } : {} },
          body: {},
        }),
      ),
    onSuccess: () => changed(),
  });
}

/** useUpdatePlan reads what updating a package would do, before it is asked to. */
export function useUpdatePlan(kind: AppKind, id: string, enabled: boolean) {
  return useQuery({
    queryKey: ["app-plan", kind, id],
    queryFn: async () => unwrap(await api.GET("/apps/{kind}/{id}/update", { params: { path: { kind, id } } })),
    enabled,
    retry: false,
    staleTime: 0,
    gcTime: 0,
  });
}

export function useUpdateApp() {
  const changed = useAppsChanged();
  return useMutation({
    mutationFn: async ({
      kind,
      id,
      overwrite,
      grant,
    }: {
      kind: AppKind;
      id: string;
      overwrite: boolean;
      grant: boolean;
    }) => {
      const { data, error } = await api.POST("/apps/{kind}/{id}/update", {
        params: { path: { kind, id } },
        body: { confirm: { overwrite, grant } },
      });
      if (error) throw new ApiError(error.error.code, error.error.message);
      return data;
    },
    onSuccess: () => changed(),
  });
}
