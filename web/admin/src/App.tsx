import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { AuthContext, useAuthProvider } from "@/hooks/use-auth";
import { I18nContext, useI18nProvider } from "@/i18n";
import { AppLayout, AuthLayout } from "@/components/layout";

import LoginPage from "@/pages/login";
import RegisterPage from "@/pages/register";
import DashboardPage from "@/pages/dashboard";
import FilesPage from "@/pages/files";
import AlbumsPage from "@/pages/albums";
import TokensPage from "@/pages/tokens";
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
        <Route element={<AuthLayout />}>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
        </Route>
        <Route element={<AppLayout />}>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/files" element={<FilesPage />} />
          <Route path="/albums" element={<AlbumsPage />} />
          <Route path="/tokens" element={<TokensPage />} />
          <Route path="/admin/storage" element={<StoragePage />} />
          <Route path="/admin/users" element={<UsersPage />} />
          <Route path="/admin/settings" element={<SettingsPage />} />
        </Route>
      </Routes>
    </AuthContext.Provider>
  );
}

export default function App() {
  const i18n = useI18nProvider();

  return (
    <I18nContext.Provider value={i18n}>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <AppRoutes />
        </BrowserRouter>
      </QueryClientProvider>
    </I18nContext.Provider>
  );
}
