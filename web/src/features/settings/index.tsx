import { Outlet } from "@tanstack/react-router";
import { Palette, SlidersHorizontal, UserCog } from "lucide-react";

import { useI18n } from "@/i18n";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { Separator } from "@/components/ui/separator";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PublishBar } from "./components/publish-bar";
import { SidebarNav } from "./components/sidebar-nav";

export function Settings() {
  const { t } = useI18n();
  useDocumentTitle(t("settings.title"));
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
        <Separator className="my-4 lg:my-6" />
        <PublishBar />
        <div className="flex flex-1 flex-col space-y-2 overflow-hidden md:space-y-2 lg:flex-row lg:space-y-0 lg:space-x-12">
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
