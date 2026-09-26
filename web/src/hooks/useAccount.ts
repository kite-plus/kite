import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { ApiError, api, unwrap, type components } from "@/api/client";
import type { Session } from "@/hooks/useSession";

export type Account = components["schemas"]["AccountInfo"];
export type Profile = components["schemas"]["Profile"];
export type CredentialsChange = components["schemas"]["CredentialsChange"];

const key = ["account"];

/** Who uses the studio, how they are shown, and how they sign in. */
export function useAccountInfo() {
  return useQuery({
    queryKey: key,
    queryFn: async () => unwrap(await api.GET("/account", {})),
  });
}

/**
 * store keeps what the server answered with, and brings the session in line
 * with it: setting, changing or removing the password changes whether the
 * studio asks for one and who is signed in.
 */
function store(client: QueryClient, account: Account) {
  client.setQueryData(key, account);
  client.setQueryData<Session>(["session"], {
    required: account.protected,
    authenticated: Boolean(account.session),
    user: account.session ? account.user : undefined,
    expires_at: account.session?.expires_at,
  });
}

export function useUpdateProfile() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (profile: Profile) => unwrap(await api.PUT("/account/profile", { body: profile })),
    onSuccess: (account) => store(client, account),
  });
}

export function useUploadAvatar() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (picture: Blob) => {
      const form = new FormData();
      form.append("file", picture, "avatar");
      // The schema types the part as a string; openapi-fetch sends FormData
      // as it is and leaves the boundary to the browser.
      return unwrap(await api.PUT("/account/avatar", { body: form as unknown as { file: string } }));
    },
    onSuccess: (account) => store(client, account),
  });
}

export function useRemoveAvatar() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => unwrap(await api.DELETE("/account/avatar", {})),
    onSuccess: (account) => store(client, account),
  });
}

export function useSetCredentials() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (change: CredentialsChange) =>
      unwrap(await api.PUT("/account/credentials", { body: change })),
    onSuccess: (account) => store(client, account),
  });
}

export function useRemoveCredentials() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (currentPassword: string) =>
      unwrap(await api.DELETE("/account/credentials", { body: { current_password: currentPassword } })),
    onSuccess: (account) => store(client, account),
  });
}

export function useEndOtherSessions() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => unwrap(await api.DELETE("/account/sessions", {})),
    onSuccess: (account) => store(client, account),
  });
}

/** wrongPassword reports a refusal caused by the current password. */
export function wrongPassword(error: unknown): boolean {
  return error instanceof ApiError && error.code === "wrong_password";
}
