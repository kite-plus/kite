import { useState } from "react";
import { Outlet, NavLink, Link, Navigate, useNavigate, useLocation } from "react-router-dom";
import {
  LayoutDashboard,
  Upload,
  FolderOpen,
  KeyRound,
  HardDrive,
  Users,
  Settings,
  FileText,
  Shield,
  ArrowLeft,
  LogOut,
  Sun,
  Moon,
  Menu,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/hooks/use-auth";
import { useI18n } from "@/i18n";
import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetContent,
  SheetTrigger,
} from "@/components/ui/sheet";

interface NavItem {
  to: string;
  icon: React.ElementType;
  labelKey: string;
  end?: boolean;
}

const userNavItems: NavItem[] = [
  { to: "/dashboard", icon: LayoutDashboard, labelKey: "nav.dashboard" },
  { to: "/files", icon: Upload, labelKey: "nav.files" },
  { to: "/albums", icon: FolderOpen, labelKey: "nav.albums" },
  { to: "/tokens", icon: KeyRound, labelKey: "nav.tokens" },
];

const adminNavItems: NavItem[] = [
  { to: "/admin", icon: LayoutDashboard, labelKey: "nav.dashboard", end: true },
  { to: "/admin/files", icon: FileText, labelKey: "nav.adminFiles" },
  { to: "/admin/storage", icon: HardDrive, labelKey: "nav.storage" },
  { to: "/admin/users", icon: Users, labelKey: "nav.users" },
  { to: "/admin/settings", icon: Settings, labelKey: "nav.settings" },
];

function AppLayout({ context }: { context: "user" | "admin" }) {
  const { user, loading, logout } = useAuth();
  const { t } = useI18n();
  const { theme, setTheme } = useTheme();
  const navigate = useNavigate();
  const location = useLocation();
  const [mobileOpen, setMobileOpen] = useState(false);

  const navItems = context === "admin" ? adminNavItems : userNavItems;

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Skeleton className="h-8 w-32" />
      </div>
    );
  }

  if (!user) return <Navigate to="/login" replace />;
  if (context === "admin" && user.role !== "admin") {
    return <Navigate to="/dashboard" replace />;
  }

  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* ── Header ── */}
      <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="flex h-14 items-center gap-4 px-4 sm:px-6 lg:px-8">
          {/* Mobile menu trigger */}
          <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
            <SheetTrigger asChild>
              <Button variant="ghost" size="icon-sm" className="shrink-0 md:hidden">
                <Menu size={18} />
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="w-72 p-0">
              {/* Mobile nav brand */}
              <div className="flex h-14 items-center gap-2.5 border-b px-5">
                <div className="flex size-7 items-center justify-center rounded-lg bg-primary text-primary-foreground">
                  <span className="text-xs font-bold">K</span>
                </div>
                <span className="font-semibold">Kite</span>
                {context === "admin" && (
                  <Badge variant="secondary" className="px-1.5 text-[10px]">
                    Admin
                  </Badge>
                )}
              </div>

              {/* Mobile nav items */}
              <nav className="space-y-0.5 p-3">
                {navItems.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.end}
                    onClick={() => setMobileOpen(false)}
                    className={({ isActive }) =>
                      cn(
                        "flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors",
                        isActive
                          ? "bg-accent text-foreground"
                          : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                      )
                    }
                  >
                    <item.icon size={18} />
                    {t(item.labelKey)}
                  </NavLink>
                ))}

                {/* Context switch */}
                <div className="my-3 border-t" />
                {context === "user" && user.role === "admin" && (
                  <Link
                    to="/admin"
                    onClick={() => setMobileOpen(false)}
                    className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                  >
                    <Shield size={18} />
                    {t("nav.adminPanel")}
                  </Link>
                )}
                {context === "admin" && (
                  <Link
                    to="/dashboard"
                    onClick={() => setMobileOpen(false)}
                    className="flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                  >
                    <ArrowLeft size={18} />
                    {t("nav.backToUser")}
                  </Link>
                )}
              </nav>
            </SheetContent>
          </Sheet>

          {/* Logo */}
          <Link
            to={context === "admin" ? "/admin" : "/dashboard"}
            className="flex shrink-0 items-center gap-2"
          >
            <div className="flex size-7 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <span className="text-xs font-bold">K</span>
            </div>
            <span className="hidden font-semibold sm:inline">Kite</span>
            {context === "admin" && (
              <Badge variant="secondary" className="hidden px-1.5 text-[10px] sm:inline-flex">
                Admin
              </Badge>
            )}
          </Link>

          {/* Desktop nav */}
          <nav className="hidden flex-1 items-center gap-0.5 md:flex">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  cn(
                    "rounded-md px-3 py-1.5 text-sm font-medium transition-all duration-200",
                    isActive
                      ? "bg-accent text-foreground shadow-sm"
                      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground"
                  )
                }
              >
                {t(item.labelKey)}
              </NavLink>
            ))}
          </nav>

          {/* Spacer (mobile only) */}
          <div className="flex-1 md:hidden" />

          {/* Right actions */}
          <div className="flex items-center gap-1">
            {/* Theme toggle */}
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-muted-foreground"
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            >
              <Sun size={16} className="dark:hidden" />
              <Moon size={16} className="hidden dark:block" />
            </Button>

            {/* User menu */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-8 gap-2 px-2">
                  <Avatar className="size-6">
                    <AvatarFallback className="bg-primary/10 text-[10px] font-semibold text-primary">
                      {user.username?.charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <span className="hidden text-sm font-medium sm:inline">
                    {user.username}
                  </span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel className="font-normal">
                  <div className="flex flex-col gap-1">
                    <span className="text-sm font-medium">{user.username}</span>
                    <span className="text-xs text-muted-foreground">
                      {user.role === "admin" ? t("nav.roleAdmin") : t("nav.roleUser")}
                    </span>
                  </div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />

                {context === "user" && user.role === "admin" && (
                  <>
                    <DropdownMenuItem onClick={() => navigate("/admin")}>
                      <Shield size={14} />
                      {t("nav.adminPanel")}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                  </>
                )}
                {context === "admin" && (
                  <>
                    <DropdownMenuItem onClick={() => navigate("/dashboard")}>
                      <ArrowLeft size={14} />
                      {t("nav.backToUser")}
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                  </>
                )}

                <DropdownMenuItem variant="destructive" onClick={logout}>
                  <LogOut size={14} />
                  {t("auth.logout")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </header>

      {/* ── Content ── */}
      <main className="flex-1 px-4 py-6 sm:px-6 lg:px-8">
        <div key={location.pathname} className="page-transition">
          <Outlet />
        </div>
      </main>

      {/* ── Footer ── */}
      <footer className="border-t">
        <div className="flex h-14 items-center justify-between px-4 text-xs text-muted-foreground sm:px-6 lg:px-8">
          <div className="flex items-center gap-1.5">
            <div className="flex size-4 items-center justify-center rounded bg-primary text-primary-foreground text-[8px] font-bold">
              K
            </div>
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
    </div>
  );
}

export function UserCenterLayout() {
  return <AppLayout context="user" />;
}

export function AdminPanelLayout() {
  return <AppLayout context="admin" />;
}
