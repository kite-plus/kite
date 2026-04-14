import { BrowserRouter, Routes, Route, Navigate, Outlet } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthContext, useAuthProvider, useAuth } from "@/hooks/use-auth";
import { I18nContext, useI18nProvider } from "@/i18n";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";

// Layouts
import AppLayout from "@/components/layouts/app-layout";
import { AuthLayout } from "@/components/layout";

// Auth pages
import LoginPage from "@/pages/login";
import RegisterPage from "@/pages/register";

// Pages
import DashboardPage from "@/pages/dashboard";
import FilesPage from "@/pages/files";
import AlbumsPage from "@/pages/albums";
import TokensPage from "@/pages/tokens";

// Admin pages
import AdminFilesPage from "@/pages/admin/files";
import StoragePage from "@/pages/admin/storage";
import UsersPage from "@/pages/admin/users";
import SettingsPage from "@/pages/admin/settings";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, refetchOnWindowFocus: false },
  },
});

function AdminRoute() {
  const { user } = useAuth();
  if (user?.role !== "admin") return <Navigate to="/dashboard" replace />;
  return <Outlet />;
}

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

        {/* App (sidebar layout) */}
        <Route element={<AppLayout />}>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/files" element={<FilesPage />} />
          <Route path="/albums" element={<AlbumsPage />} />
          <Route path="/tokens" element={<TokensPage />} />

          {/* Admin pages (role guard) */}
          <Route element={<AdminRoute />}>
            <Route path="/admin" element={<Navigate to="/dashboard" replace />} />
            <Route path="/admin/files" element={<AdminFilesPage />} />
            <Route path="/admin/storage" element={<StoragePage />} />
            <Route path="/admin/users" element={<UsersPage />} />
            <Route path="/admin/settings" element={<SettingsPage />} />
          </Route>
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
          <Toaster
            richColors
            closeButton
            position="top-center"
            offset={16}
            toastOptions={{
              classNames: {
                toast:
                  "rounded-lg border shadow-lg backdrop-blur-md bg-background/95",
              },
            }}
          />
        </QueryClientProvider>
      </I18nContext.Provider>
    </ThemeProvider>
  );
}
