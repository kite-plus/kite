import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, ApiError, unwrap, type Media, type ThemeExists, type ThemeInfo } from "@/api/client";

export function useThemes() {
  return useQuery({
    queryKey: ["themes"],
    queryFn: async () => unwrap(await api.GET("/themes", {})).items,
  });
}

/** useTheme reads one theme with its settings and the revision to save them against. */
export function useTheme(name: string) {
  return useQuery({
    queryKey: ["theme", name],
    queryFn: async () => {
      const result = await api.GET("/themes/{name}", { params: { path: { name } } });
      return { ...unwrap(result), revision: result.response.headers.get("ETag") ?? "" };
    },
  });
}

/**
 * useThemesChanged refreshes what a change of theme can alter: the themes and
 * which is in use, the settings, and the templates each content type offers.
 */
export function useThemesChanged() {
  const client = useQueryClient();
  return () =>
    Promise.all(
      [["themes"], ["theme"], ["settings"], ["content-types"], ["site"]].map((queryKey) =>
        client.invalidateQueries({ queryKey }),
      ),
    );
}

/** What installing a theme came to: installed, or waiting to be told to replace one. */
export type Installed =
  | { status: "installed"; theme: ThemeInfo }
  | { status: "exists"; installed: ThemeInfo; uploaded: ThemeInfo };

/**
 * installTheme sends a theme's zip archive. One already installed under the
 * same name is replaced only when asked, and otherwise both versions come
 * back to be asked about.
 */
export function installTheme(
  file: File,
  replace: boolean,
  onProgress?: (percent: number) => void,
): Promise<Installed> {
  const form = new FormData();
  form.append("file", file);

  // XMLHttpRequest rather than fetch: only it reports upload progress.
  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest();
    request.open("POST", `/api/v1/themes${replace ? "?replace=true" : ""}`);
    request.responseType = "json";
    request.upload.onprogress = (event) => {
      if (event.lengthComputable) onProgress?.(Math.round((event.loaded / event.total) * 100));
    };
    request.onload = () => {
      const body = request.response as
        | (ThemeInfo & Partial<ThemeExists> & { error?: { code?: string; message?: string } })
        | null;
      if (request.status === 201 && body) {
        resolve({ status: "installed", theme: body });
      } else if (request.status === 409 && body?.installed && body.uploaded) {
        resolve({ status: "exists", installed: body.installed, uploaded: body.uploaded });
      } else {
        reject(new ApiError(body?.error?.code ?? "internal", body?.error?.message ?? "install failed"));
      }
    };
    request.onerror = () => reject(new ApiError("unreachable", "install failed"));
    request.send(form);
  });
}

export function useRemoveTheme() {
  const changed = useThemesChanged();
  return useMutation({
    mutationFn: async (name: string) => {
      const result = await api.DELETE("/themes/{name}", { params: { path: { name } } });
      if (result.error) unwrap(result);
    },
    onSuccess: () => changed(),
  });
}

/**
 * useSaveTheme writes settings by dotted path, as the settings form does, and
 * switches to the theme when the changes name it.
 */
export function useSaveTheme() {
  const changed = useThemesChanged();
  return useMutation({
    mutationFn: async ({ changes, revision }: { changes: Record<string, unknown>; revision: string }) => {
      const result = await api.PUT("/settings", {
        body: changes,
        params: { header: { "If-Match": revision } },
      });
      return { ...unwrap(result), revision: result.response.headers.get("ETag") ?? "" };
    },
    onSuccess: () => changed(),
  });
}

/** uploadSiteMedia stores a file of the site's own and resolves to its path in the site. */
export async function uploadSiteMedia(file: File): Promise<string> {
  const form = new FormData();
  form.append("file", file);
  const response = await fetch("/api/v1/media", { method: "POST", body: form, credentials: "same-origin" });
  const body = (await response.json().catch(() => null)) as
    | (Media & { error?: { code?: string; message?: string } })
    | null;
  if (!response.ok || !body?.link) {
    throw new ApiError(body?.error?.code ?? "internal", body?.error?.message ?? "upload failed");
  }
  return body.link;
}
