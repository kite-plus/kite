import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createRouter, RouterProvider } from "@tanstack/react-router";
import { Loader2 } from "lucide-react";

import { I18nProvider, useI18n } from "@/i18n";
// Imported for its effect: it puts the theme on the document before a render.
import "@/lib/theme";
import { routeTree } from "./routeTree.gen";
import "./styles/index.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // The files underneath the index change while this page is open, so a
      // cached answer is only ever a starting point.
      staleTime: 2_000,
      refetchOnWindowFocus: true,
      retry: 1,
    },
  },
});

function Pending() {
  const { t } = useI18n();
  return (
    <div className="flex min-h-svh items-center justify-center gap-2 text-sm text-muted-foreground">
      <Loader2 className="size-4 animate-spin" />
      {t("session.checking")}
    </div>
  );
}

const router = createRouter({
  routeTree,
  // The Go server mounts the studio here, and Vite builds it for this base.
  basepath: import.meta.env.BASE_URL,
  context: { queryClient },
  defaultPreload: "intent",
  // Queries own the freshness of what a screen shows, not the router.
  defaultPreloadStaleTime: 0,
  defaultPendingComponent: Pending,
  scrollRestoration: true,
});

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}

const root = document.getElementById("root")!;
// In development a change to the route tree runs this module again, over a
// root that is already mounted.
if (!root.innerHTML) {
  createRoot(root).render(
    <StrictMode>
      <QueryClientProvider client={queryClient}>
        <I18nProvider>
          <RouterProvider router={router} />
        </I18nProvider>
      </QueryClientProvider>
    </StrictMode>,
  );
}
