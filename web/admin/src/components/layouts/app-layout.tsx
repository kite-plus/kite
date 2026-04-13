import { useState } from "react";
import { Outlet, NavLink, Link, Navigate, useNavigate } from "react-router-dom";
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
    <div className="min-h-screen bg-background">
      {/* ── Header ── */}
      <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
        <div className="mx-auto flex h-14 max-w-screen-xl items-center gap-4 px-4 sm:px-6">
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
                    "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-accent text-foreground"
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
      <main className="mx-auto max-w-screen-xl px-4 py-6 sm:px-6">
        <Outlet />
      </main>
    </div>
  );
}

export function UserCenterLayout() {
  return <AppLayout context="user" />;
}

export function AdminPanelLayout() {
  return <AppLayout context="admin" />;
}
