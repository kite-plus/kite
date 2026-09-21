import { Suspense, lazy, useState, type CSSProperties } from "react";

import { useI18n } from "@/i18n";
import { useRoute } from "@/lib/router";

import { ContentListPage } from "@/components/content/ContentListPage";
import { DashboardPage } from "@/components/dashboard/DashboardPage";
import { SettingsPage } from "@/components/settings/SettingsPage";
import { ThemePage } from "@/components/settings/ThemePage";
import { AppSidebar } from "@/components/shell/AppSidebar";
import { CommandMenu } from "@/components/shell/CommandMenu";
import { TaxonomiesPage } from "@/components/taxonomies/TaxonomiesPage";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Spinner } from "@/components/ui/spinner";

// The editor carries CodeMirror, the heaviest thing in the admin, and is one
// screen of several. Loading it when it opens keeps the rest light.
const EditorPage = lazy(() =>
  import("@/components/editor/EditorPage").then((module) => ({ default: module.EditorPage })),
);

export default function App() {
  const route = useRoute();
  const [searching, setSearching] = useState(false);

  return (
    <SidebarProvider style={{ "--sidebar-width": "14.5rem" } as CSSProperties}>
      <AppSidebar route={route} onSearch={() => setSearching(true)} />
      <SidebarInset className="min-w-0">
        <Screen />
      </SidebarInset>
      <CommandMenu open={searching} onOpenChange={setSearching} />
    </SidebarProvider>
  );
}

function Screen() {
  const { t } = useI18n();
  const route = useRoute();

  switch (route.name) {
    case "list":
      // Keyed so one kind's selection and paging never leak into another's.
      return <ContentListPage key={route.kind} kind={route.kind} />;
    case "edit":
      return (
        <Suspense
          fallback={
            <Empty className="h-svh">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <Spinner />
                </EmptyMedia>
                <EmptyTitle>{t("editor.loading")}</EmptyTitle>
              </EmptyHeader>
            </Empty>
          }
        >
          <EditorPage id={route.id} kind={route.kind} />
        </Suspense>
      );
    case "taxonomies":
      return <TaxonomiesPage />;
    case "theme":
      return <ThemePage />;
    case "settings":
      return <SettingsPage />;
    default:
      return <DashboardPage />;
  }
}
