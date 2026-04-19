import axios from "axios";

const api = axios.create({
  baseURL: "/api/v1",
  headers: { "Content-Type": "application/json" },
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("access_token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    if (error.response?.status === 401) {
      const refreshToken = localStorage.getItem("refresh_token");
      if (refreshToken && !error.config._retry) {
        error.config._retry = true;
        try {
          const { data } = await axios.post("/api/v1/auth/refresh", {
            refresh_token: refreshToken,
          });
          const tokens = data.data;
          localStorage.setItem("access_token", tokens.access_token);
          localStorage.setItem("refresh_token", tokens.refresh_token);
          error.config.headers.Authorization = `Bearer ${tokens.access_token}`;
          return api(error.config);
        } catch {
          localStorage.removeItem("access_token");
          localStorage.removeItem("refresh_token");
          window.location.href = "/login";
        }
      }
    }
    return Promise.reject(error);
  }
);

// Auth
export const authApi = {
  login: (username: string, password: string) =>
    api.post("/auth/login", { username, password }),
  register: (username: string, email: string, password: string) =>
    api.post("/auth/register", { username, email, password }),
  logout: () => api.post("/auth/logout"),
  profile: () => api.get("/profile"),
  updateProfile: (data: { username: string; nickname?: string; email: string; avatar_url?: string }) =>
    api.put("/profile", data),
  changePassword: (data: { current_password: string; new_password: string }) =>
    api.post("/auth/change-password", data),
  firstLoginReset: (data: {
    new_username: string;
    new_email: string;
    new_password: string;
  }) => api.post("/auth/first-login-reset", data),
};

// Files
export const fileApi = {
  upload: (file: File, albumId?: string) => {
    const form = new FormData();
    form.append("file", file);
    if (albumId) form.append("album_id", albumId);
    return api.post("/upload", form, {
      headers: { "Content-Type": "multipart/form-data" },
    });
  },
  list: (params: Record<string, string | number | boolean>) =>
    api.get("/files", { params }),
  detail: (id: string) => api.get(`/files/${id}`),
  delete: (id: string) => api.delete(`/files/${id}`),
  batchDelete: (ids: string[]) => api.post("/files/batch-delete", { ids }),
  move: (id: string, folderId: string | null) =>
    api.patch(`/files/${id}/move`, { folder_id: folderId }),
};

// Albums / Folders
export const albumApi = {
  list: (params: Record<string, string | number>) =>
    api.get("/albums", { params }),
  create: (data: { name: string; description?: string; is_public?: boolean; parent_id?: string }) =>
    api.post("/albums", data),
  update: (id: string, data: Record<string, unknown>) =>
    api.put(`/albums/${id}`, data),
  delete: (id: string) => api.delete(`/albums/${id}`),
};

// Tokens
export const tokenApi = {
  list: () => api.get("/tokens"),
  create: (name: string, expiresIn?: number) =>
    api.post("/tokens", { name, expires_in: expiresIn }),
  delete: (id: string) => api.delete(`/tokens/${id}`),
};

// Storage (admin)
interface StoragePayload {
  name: string;
  driver: string;
  config: unknown;
  capacity_limit_bytes?: number;
  priority?: number;
  is_default?: boolean;
  is_active?: boolean;
}

export const storageApi = {
  list: () => api.get("/storage"),
  get: (id: string) => api.get(`/storage/${id}`),
  create: (data: StoragePayload) => api.post("/storage", data),
  update: (id: string, data: StoragePayload) => api.put(`/storage/${id}`, data),
  delete: (id: string) => api.delete(`/storage/${id}`),
  test: (id: string) => api.post(`/storage/${id}/test`),
  setDefault: (id: string) => api.post(`/storage/${id}/set-default`),
  reorder: (orderedIds: string[]) =>
    api.post("/storage/reorder", { ordered_ids: orderedIds }),
};

// Settings (admin)
export const settingsApi = {
  get: () => api.get("/settings"),
  update: (settings: Record<string, string>) =>
    api.put("/settings", { settings }),
};

// Admin Files
export const adminFileApi = {
  list: (params: Record<string, string | number>) =>
    api.get("/admin/files", { params }),
  delete: (id: string) => api.delete(`/admin/files/${id}`),
};

// Users (admin)
export const userApi = {
  list: (params: Record<string, string | number>) =>
    api.get("/admin/users", { params }),
  create: (data: Record<string, unknown>) => api.post("/admin/users", data),
  update: (id: string, data: Record<string, unknown>) =>
    api.put(`/admin/users/${id}`, data),
  delete: (id: string) => api.delete(`/admin/users/${id}`),
};

// Stats — 当前用户维度（仅展示自己的数据）
export const statsApi = {
  get: () => api.get("/stats"),
  daily: (days: number = 7) => api.get("/stats/daily", { params: { days } }),
  heatmap: (weeks: number = 12) => api.get("/stats/heatmap", { params: { weeks } }),
};

// Admin Stats — 全站维度（仅管理员可调用）
export const adminStatsApi = {
  get: () => api.get("/admin/stats"),
  daily: (days: number = 7) =>
    api.get("/admin/stats/daily", { params: { days } }),
  heatmap: (weeks: number = 12) =>
    api.get("/admin/stats/heatmap", { params: { weeks } }),
};

export const systemStatusApi = {
  wsTicket: () => api.post("/admin/system-status/ws-ticket"),
  ping: () => api.get("/health"),
};

// Setup
export const setupApi = {
  status: () => api.get("/setup/status"),
  setup: (data: Record<string, unknown>) => api.post("/setup", data),
};

export default api;
