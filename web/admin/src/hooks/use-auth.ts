import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
} from "react";
import { authApi } from "@/lib/api";
import { toast } from "sonner";
import { translate } from "@/i18n";

export interface User {
  user_id: string;
  username: string;
  nickname?: string;
  email?: string;
  avatar_url?: string;
  role: string;
  password_must_change?: boolean;
  storage_limit?: number;
  storage_used?: number;
  created_at?: string;
}

export interface AuthContextValue {
  user: User | null;
  loading: boolean;
  login: (username: string, password: string) => Promise<void>;
  register: (
    username: string,
    email: string,
    password: string
  ) => Promise<void>;
  logout: () => void;
  applyTokensAndRefresh: (tokens: {
    access_token: string;
    refresh_token: string;
  }) => Promise<void>;
}

export const AuthContext = createContext<AuthContextValue | null>(null);

const PROFILE_CACHE_KEY = "user_profile";

function readCachedUser(): User | null {
  try {
    const raw = localStorage.getItem(PROFILE_CACHE_KEY);
    return raw ? (JSON.parse(raw) as User) : null;
  } catch {
    return null;
  }
}

function writeCachedUser(user: User | null) {
  if (user) {
    localStorage.setItem(PROFILE_CACHE_KEY, JSON.stringify(user));
  } else {
    localStorage.removeItem(PROFILE_CACHE_KEY);
  }
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}

export function useAuthProvider(): AuthContextValue {
  // Hydrate synchronously so the first render already reflects auth state.
  const [user, setUser] = useState<User | null>(() => {
    if (typeof window === "undefined") return null;
    if (!localStorage.getItem("access_token")) return null;
    return readCachedUser();
  });
  const [loading, setLoading] = useState<boolean>(() => {
    if (typeof window === "undefined") return false;
    // Only block on loading when a token exists AND we have no cached profile.
    return !!localStorage.getItem("access_token") && !readCachedUser();
  });

  const fetchProfile = useCallback(async () => {
    const token = localStorage.getItem("access_token");
    if (!token) {
      setLoading(false);
      return;
    }
    try {
      const { data } = await authApi.profile();
      setUser(data.data);
      writeCachedUser(data.data);
    } catch {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      writeCachedUser(null);
      setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchProfile();
  }, [fetchProfile]);

  const login = useCallback(async (username: string, password: string) => {
    const { data } = await authApi.login(username, password);
    const tokens = data.data;
    localStorage.setItem("access_token", tokens.access_token);
    localStorage.setItem("refresh_token", tokens.refresh_token);
    const profile = await authApi.profile();
    setUser(profile.data.data);
    writeCachedUser(profile.data.data);
  }, []);

  const register = useCallback(
    async (username: string, email: string, password: string) => {
      await authApi.register(username, email, password);
    },
    []
  );

  const logout = useCallback(() => {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    writeCachedUser(null);
    setUser(null);
    authApi.logout().catch(() => { });
    toast.success(translate("auth.loggedOut"));
  }, []);

  const applyTokensAndRefresh = useCallback(
    async (tokens: { access_token: string; refresh_token: string }) => {
      localStorage.setItem("access_token", tokens.access_token);
      localStorage.setItem("refresh_token", tokens.refresh_token);
      const profile = await authApi.profile();
      setUser(profile.data.data);
      writeCachedUser(profile.data.data);
    },
    []
  );

  return { user, loading, login, register, logout, applyTokensAndRefresh };
}
