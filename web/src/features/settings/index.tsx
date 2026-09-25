import { Outlet } from "@tanstack/react-router";

import { useI18n } from "@/i18n";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";

// Each section heads its own page: the app sidebar already lists the
// sections, so the page repeats neither a menu nor a title over them.
export function Settings() {
  const { t } = useI18n();
  useDocumentTitle(t("settings.title"));

  return (
    <>
      <AppHeader />

      <Main fixed>
        <Outlet />
      </Main>
    </>
  );
}
