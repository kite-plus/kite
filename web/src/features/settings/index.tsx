import { Outlet } from "@tanstack/react-router";
import { Palette, SlidersHorizontal, UserCog } from "lucide-react";

import { useI18n } from "@/i18n";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { SidebarNav } from "./components/sidebar-nav";

export function Settings() {
  const { t } = useI18n();
  const items = [
    { title: t("nav.site"), href: "/settings", icon: <SlidersHorizontal size={18} /> },
    { title: t("nav.theme"), href: "/settings/theme", icon: <Palette size={18} /> },
    { title: t("settings.interface"), href: "/settings/appearance", icon: <UserCog size={18} /> },
  ];

  return (
    <>
      <AppHeader />

      <Main fixed>
        <div className="space-y-0.5">
          <h1 className="text-2xl font-bold tracking-tight md:text-3xl">{t("settings.title")}</h1>
          <p className="text-muted-foreground">{t("settings.description")}</p>
        </div>
        <div className="mt-4 flex flex-1 flex-col space-y-2 overflow-hidden rounded-xl border bg-card p-4 shadow-xs md:space-y-2 lg:mt-6 lg:flex-row lg:space-y-0 lg:space-x-12">
          <aside className="top-0 lg:sticky lg:w-1/5">
            <SidebarNav items={items} />
          </aside>
          <div className="flex w-full overflow-y-hidden p-1">
            <Outlet />
          </div>
        </div>
      </Main>
    </>
  );
}
