import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthContext, useAuthProvider } from "@/hooks/use-auth";
import { I18nContext, useI18nProvider } from "@/i18n";
import { ThemeProvider } from "@/components/theme-provider";

// Layouts
import { UserCenterLayout, AdminPanelLayout } from "@/components/layouts/app-layout";
import { AuthLayout } from "@/components/layout";

// Auth pages
import LoginPage from "@/pages/login";
import RegisterPage from "@/pages/register";

// User pages
import UserDashboard from "@/pages/user/dashboard";
import FilesPage from "@/pages/files";
import AlbumsPage from "@/pages/albums";
import TokensPage from "@/pages/tokens";

// Admin pages
import AdminDashboard from "@/pages/dashboard";
import AdminFilesPage from "@/pages/admin/files";
import StoragePage from "@/pages/admin/storage";
import UsersPage from "@/pages/admin/users";
import SettingsPage from "@/pages/admin/settings";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
});

function AppRoutes() {
  const auth = useAuthProvider();

  return (
    <AuthContext.Provider value={auth}>
      <Routes>
        {/* Auth (centered card) */}
        <Route element={<AuthLayout />}>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
        </Route>

        {/* User center (top nav) */}
        <Route element={<UserCenterLayout />}>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<UserDashboard />} />
          <Route path="/files" element={<FilesPage />} />
          <Route path="/albums" element={<AlbumsPage />} />
          <Route path="/tokens" element={<TokensPage />} />
        </Route>

        {/* Admin panel (top nav, admin-only) */}
        <Route element={<AdminPanelLayout />}>
          <Route path="/admin" element={<AdminDashboard />} />
          <Route path="/admin/files" element={<AdminFilesPage />} />
          <Route path="/admin/storage" element={<StoragePage />} />
          <Route path="/admin/users" element={<UsersPage />} />
          <Route path="/admin/settings" element={<SettingsPage />} />
        </Route>

        {/* Fallback */}
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </AuthContext.Provider>
  );
}

export default function App() {
  const i18n = useI18nProvider();

  return (
    <ThemeProvider>
      <I18nContext.Provider value={i18n}>
        <QueryClientProvider client={queryClient}>
          <BrowserRouter>
            <AppRoutes />
          </BrowserRouter>
        </QueryClientProvider>
      </I18nContext.Provider>
    </ThemeProvider>
  );
}
