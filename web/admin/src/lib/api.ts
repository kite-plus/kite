import axios from 'axios'

// withCredentials ensures the browser attaches HttpOnly auth cookies
// (access_token, refresh_token) on every request. The frontend never sees
// those values — the server reads them directly.
const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
})

// inflightRefresh serializes concurrent refreshes. When N requests 401 at
// once (common after a page wakes from sleep with a stale access token),
// we must NOT fire N parallel /auth/refresh calls — the first rotates the
// refresh cookie, invalidating the token the others would present. Every
// concurrent 401 awaits the same in-flight promise and then retries.
let inflightRefresh: Promise<void> | null = null

function refreshOnce(): Promise<void> {
  if (!inflightRefresh) {
    inflightRefresh = axios
      .post('/api/v1/auth/refresh', {}, { withCredentials: true })
      .then(() => undefined)
      .finally(() => {
        inflightRefresh = null
      })
  }
  return inflightRefresh
}

// On 401, try exactly one silent refresh per request. The refresh endpoint
// reads the refresh cookie server-side and rotates both cookies on success,
// so we just need to retry the original request. We use the bare axios
// import (not `api`) for the refresh call so this interceptor doesn't loop
// on the refresh POST itself.
api.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (error.response?.status === 401 && !error.config?._retry) {
      error.config._retry = true
      try {
        await refreshOnce()
        return api(error.config)
      } catch {
        // Refresh failed — credentials are fully stale. Bounce to login
        // unless the user is already there, which would cause a loop.
        if (
          typeof window !== 'undefined' &&
          !window.location.pathname.startsWith('/login') &&
          !window.location.pathname.startsWith('/register')
        ) {
          window.location.href = '/login'
        }
      }
    }
    return Promise.reject(error)
  }
)

// Auth
export const authApi = {
  options: () => api.get('/auth/options'),
  exchangeOAuth: (ticket: string) =>
    api.post('/auth/oauth/exchange', { ticket }),
  onboardOAuth: (data: { ticket: string; username: string; email: string }) =>
    api.post('/auth/oauth/onboard', data),
  login: (username: string, password: string) =>
    api.post('/auth/login', { username, password }),
  register: (username: string, email: string, password: string) =>
    api.post('/auth/register', { username, email, password }),
  logout: () => api.post('/auth/logout'),
  profile: () => api.get('/profile'),
  identities: () => api.get('/auth/identities'),
  unlinkIdentity: (provider: string) =>
    api.delete(`/auth/identities/${provider}`),
  // Only nickname + avatar are mutable on the self-service profile endpoint.
  // Username is immutable and email rotation goes through the verified flow
  // exposed at /auth/email-change/*.
  updateProfile: (data: { nickname?: string; avatar_url?: string }) =>
    api.put('/profile', data),
  changePassword: (data: { current_password: string; new_password: string }) =>
    api.post('/auth/change-password', data),
  setPassword: (data: { new_password: string }) =>
    api.post('/auth/set-password', data),
  firstLoginReset: (data: {
    new_username: string
    new_email: string
    new_password: string
  }) => api.post('/auth/first-login-reset', data),
  requestEmailChange: (newEmail: string) =>
    api.post('/auth/email-change/request', { new_email: newEmail }),
  confirmEmailChange: (newEmail: string, code: string) =>
    api.post('/auth/email-change/confirm', { new_email: newEmail, code }),
  // TOTP 2FA — setup returns the otpauth URI + raw secret; enable
  // confirms with a 6-digit code; disable requires password + code;
  // verify exchanges a login challenge for real tokens.
  setupTotp: () => api.post('/auth/2fa/setup'),
  enableTotp: (code: string) => api.post('/auth/2fa/enable', { code }),
  disableTotp: (password: string, code: string) =>
    api.post('/auth/2fa/disable', { password, code }),
  verifyTotp: (challengeToken: string, code: string) =>
    api.post('/auth/2fa/verify', { challenge_token: challengeToken, code }),
  // Forgot-password — request ships a 6-digit code to the account's
  // registered email; confirm rotates the password. The endpoints are
  // public, so we intentionally don't rely on `withCredentials`.
  requestPasswordReset: (identifier: string) =>
    api.post('/auth/password-reset/request', { identifier }),
  confirmPasswordReset: (
    identifier: string,
    code: string,
    newPassword: string
  ) =>
    api.post('/auth/password-reset/confirm', {
      identifier,
      code,
      new_password: newPassword,
    }),
}

// Files
export const fileApi = {
  upload: (file: File, albumId?: string) => {
    const form = new FormData()
    form.append('file', file)
    if (albumId) form.append('album_id', albumId)
    return api.post('/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
  list: (params: Record<string, string | number | boolean>) =>
    api.get('/files', { params }),
  detail: (id: string) => api.get(`/files/${id}`),
  delete: (id: string) => api.delete(`/files/${id}`),
  batchDelete: (ids: string[]) => api.post('/files/batch-delete', { ids }),
  move: (id: string, folderId: string | null) =>
    api.patch(`/files/${id}/move`, { folder_id: folderId }),
}

// Albums / Folders
export const albumApi = {
  list: (params: Record<string, string | number>) =>
    api.get('/albums', { params }),
  create: (data: {
    name: string
    description?: string
    is_public?: boolean
    parent_id?: string
  }) => api.post('/albums', data),
  update: (id: string, data: Record<string, unknown>) =>
    api.put(`/albums/${id}`, data),
  delete: (id: string) => api.delete(`/albums/${id}`),
}

// Tokens
export const tokenApi = {
  list: () => api.get('/tokens'),
  create: (name: string, expiresIn?: number) =>
    api.post('/tokens', { name, expires_in: expiresIn }),
  delete: (id: string) => api.delete(`/tokens/${id}`),
}

// Storage (admin)
interface StoragePayload {
  name: string
  scheme_key: string
  config: unknown
  capacity_limit_bytes?: number
  priority?: number
  is_default?: boolean
  is_active?: boolean
}

export const storageApi = {
  list: () => api.get('/storage'),
  catalog: () => api.get('/storage/catalog'),
  get: (id: string) => api.get(`/storage/${id}`),
  create: (data: StoragePayload) => api.post('/storage', data),
  update: (id: string, data: StoragePayload) => api.put(`/storage/${id}`, data),
  delete: (id: string) => api.delete(`/storage/${id}`),
  test: (id: string) => api.post(`/storage/${id}/test`),
  setDefault: (id: string) => api.post(`/storage/${id}/set-default`),
  reorder: (orderedIds: string[]) =>
    api.post('/storage/reorder', { ordered_ids: orderedIds }),
}

// Settings (admin)
export const settingsApi = {
  get: () => api.get('/settings'),
  update: (settings: Record<string, string>) =>
    api.put('/settings', { settings }),
  testEmail: (settings: Record<string, string>) =>
    api.post('/settings/test-email', { settings }),
}

export const authProviderApi = {
  list: () => api.get('/admin/auth/providers'),
  update: (
    provider: string,
    data: { enabled: boolean; client_id: string; client_secret: string }
  ) => api.put(`/admin/auth/providers/${provider}`, data),
}

// Admin Files
export const adminFileApi = {
  list: (params: Record<string, string | number>) =>
    api.get('/admin/files', { params }),
  delete: (id: string) => api.delete(`/admin/files/${id}`),
}

// Users (admin)
export const userApi = {
  list: (params: Record<string, string | number>) =>
    api.get('/admin/users', { params }),
  create: (data: Record<string, unknown>) => api.post('/admin/users', data),
  update: (id: string, data: Record<string, unknown>) =>
    api.put(`/admin/users/${id}`, data),
  delete: (id: string) => api.delete(`/admin/users/${id}`),
  // Admin lockout recovery: strips TOTP from a user who lost their
  // authenticator device. Bumps token_version on the target user so
  // any session in flight on that account is forcibly rotated.
  resetTotp: (id: string) => api.post(`/admin/users/${id}/2fa/reset`),
}

// Stats — 当前用户维度（仅展示自己的数据）
export const statsApi = {
  get: () => api.get('/stats'),
  daily: (days: number = 7) => api.get('/stats/daily', { params: { days } }),
  heatmap: (weeks: number = 12) =>
    api.get('/stats/heatmap', { params: { weeks } }),
}

// Admin Stats — 全站维度（仅管理员可调用）
export const adminStatsApi = {
  get: () => api.get('/admin/stats'),
  daily: (days: number = 7) =>
    api.get('/admin/stats/daily', { params: { days } }),
  heatmap: (weeks: number = 12) =>
    api.get('/admin/stats/heatmap', { params: { weeks } }),
}

export const systemStatusApi = {
  wsTicket: () => api.post('/admin/system-status/ws-ticket'),
  ping: () => api.get('/health'),
}

// Setup
export const setupApi = {
  status: () => api.get('/setup/status'),
  setup: (data: Record<string, unknown>) => api.post('/setup', data),
}

// Share — public, no auth required. The endpoint is mounted under /public/* so
// the auth interceptor's silent-refresh path stays out of the way.
export const shareApi = {
  info: (hash: string) => api.get(`/public/share/${hash}`),
}

export default api
