import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import App from "./App";
import { Gate } from "@/components/Gate";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { I18nProvider } from "@/i18n";
// Imported for its effect: it puts the theme on the document before a render.
import "@/lib/theme";
import "./index.css";

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

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <I18nProvider>
        <TooltipProvider delay={300}>
          <Gate>
            <App />
          </Gate>
        </TooltipProvider>
        <Toaster position="bottom-right" />
      </I18nProvider>
    </QueryClientProvider>
  </StrictMode>,
);
