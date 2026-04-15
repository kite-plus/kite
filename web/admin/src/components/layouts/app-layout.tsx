import { useState } from "react";
import { Navigate, Link, useLocation } from "react-router-dom";
import { PageTransition } from "@/components/page-transition";
import { Menu } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useI18n } from "@/i18n";
import { Sidebar } from "@/components/layouts/sidebar";
import { Button } from "@/components/ui/button";
import { KiteLogo } from "@/components/kite-logo";
import { ThemeToggle } from "@/components/theme-toggle";
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";

export default function AppLayout() {
  const { user, loading } = useAuth();
  const { t } = useI18n();
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = useState(false);

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <KiteLogo className="size-8 animate-[splash-pulse_1.4s_ease-in-out_infinite]" />
      </div>
    );
  }

  if (!user) {
    return (
      <Navigate
        to="/login"
        replace
        state={{ from: location.pathname + location.search }}
      />
    );
  }

  return (
    <div className="flex h-screen overflow-hidden bg-background relative">
      {/* Floating Theme Toggle (Desktop only) */}
      <div className="absolute top-4 right-6 z-50 hidden md:block">
        <ThemeToggle />
      </div>

      {/* Desktop sidebar */}
      <div className="hidden shrink-0 border-r md:flex">
        <Sidebar />
      </div>

      {/* Right content area */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Mobile header */}
        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          <header className="flex h-14 shrink-0 items-center justify-between border-b px-4 md:hidden">
            <div className="flex items-center gap-2">
              <SheetTrigger asChild>
                <Button variant="ghost" size="icon-sm">
                  <Menu className="h-5 w-5" />
                </Button>
              </SheetTrigger>
              <Link
                to="/dashboard"
                className="flex items-center gap-2 transition-opacity hover:opacity-80"
              >
                <KiteLogo className="size-6" />
                <span className="font-semibold tracking-tight">Kite</span>
              </Link>
            </div>
            
            <ThemeToggle />
          </header>
          <SheetContent side="left" className="w-[220px] p-0">
            <SheetTitle className="sr-only">Navigation</SheetTitle>
            <Sidebar onClose={() => setMobileOpen(false)} />
          </SheetContent>
        </Sheet>

        {/* Scrollable content + footer */}
        <main className="flex flex-1 flex-col overflow-y-auto">
          <div className="mx-auto w-full max-w-screen-2xl flex-1 px-4 py-6 sm:px-6 lg:px-8">
            <PageTransition />
          </div>

          {/* Footer — h-14 + border-t matches sidebar bottom */}
          <footer className="mt-auto flex h-14 shrink-0 border-t">
            <div className="mx-auto flex w-full max-w-screen-2xl items-center justify-between px-4 text-xs text-muted-foreground sm:px-6 lg:px-8">
              <div className="flex items-center gap-1.5">
                <KiteLogo className="size-4" />
                <span className="font-medium text-foreground/70">Kite</span>
                <span className="mx-0.5 text-border">·</span>
                <span className="hidden sm:inline">{t("footer.description")}</span>
              </div>
              <div className="flex items-center gap-3">
                <a
                  href="https://github.com/amigoer/kite"
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex items-center gap-1 transition-colors hover:text-foreground"
                >
                  <svg className="size-3.5" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                  </svg>
                  <span className="hidden sm:inline">GitHub</span>
                </a>
                <span className="text-border">·</span>
                <span>&copy; 2026 Kite</span>
              </div>
            </div>
          </footer>
        </main>
      </div>
    </div>
  );
}
