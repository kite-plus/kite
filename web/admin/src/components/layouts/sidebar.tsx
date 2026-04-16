import { NavLink, Link } from "react-router-dom";
import {
  LayoutDashboard,
  Upload,
  FolderOpen,
  KeyRound,
  FileText,
  HardDrive,
  Users,
  Settings,
  LogOut,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/hooks/use-auth";
import { useI18n } from "@/i18n";
import { Separator } from "@/components/ui/separator";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { KiteLogo } from "@/components/kite-logo";

const userNavItems = [
  { to: "/dashboard", icon: LayoutDashboard, labelKey: "nav.dashboard" },
  { to: "/files", icon: Upload, labelKey: "nav.files" },
  { to: "/albums", icon: FolderOpen, labelKey: "nav.albums" },
  { to: "/tokens", icon: KeyRound, labelKey: "nav.tokens" },
];

const adminNavItems = [
  { to: "/admin/files", icon: FileText, labelKey: "nav.adminFiles" },
  { to: "/admin/storage", icon: HardDrive, labelKey: "nav.storage" },
  { to: "/admin/users", icon: Users, labelKey: "nav.users" },
  { to: "/admin/settings", icon: Settings, labelKey: "nav.settings" },
];

interface SidebarProps {
  onClose?: () => void;
}

export function Sidebar({ onClose }: SidebarProps) {
  const { user, logout } = useAuth();
  const { t } = useI18n();

  return (
    <aside className="flex h-full w-55 flex-col bg-background">
      {/* Logo — h-14 + border-b aligns with desktop header */}
      <div className="flex h-14 shrink-0 items-center border-b px-5">
        <Link
          to="/dashboard"
          onClick={onClose}
          className="flex items-center gap-2.5 transition-opacity hover:opacity-80"
        >
          <KiteLogo className="size-6" />
          <span className="font-semibold tracking-tight">Kite</span>
        </Link>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-3">
        {userNavItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            onClick={onClose}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              )
            }
          >
            <item.icon className="h-4 w-4" />
            {t(item.labelKey)}
          </NavLink>
        ))}

        {user?.role === "admin" && (
          <>
            <Separator className="my-3!" />
            <p className="px-3 py-1 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/60">
              {t("nav.admin")}
            </p>
            {adminNavItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                onClick={onClose}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-accent text-accent-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                  )
                }
              >
                <item.icon className="h-4 w-4" />
                {t(item.labelKey)}
              </NavLink>
            ))}
          </>
        )}
      </nav>

      {/* Bottom user section — h-14 matches the footer */}
      <div className="flex h-14 shrink-0 items-center gap-2 border-t px-3">
        <Avatar className="size-7">
          <AvatarFallback className="text-[10px] font-medium">
            {user?.username?.charAt(0).toUpperCase()}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium leading-none">
            {user?.username}
          </p>
          <p className="mt-0.5 text-[11px] text-muted-foreground">
            {user?.role === "admin" ? t("nav.roleAdmin") : t("nav.roleUser")}
          </p>
        </div>
        <Button
          variant="ghost"
          size="icon-sm"
          className="rounded-full text-muted-foreground hover:text-destructive"
          onClick={logout}
          aria-label={t("auth.logout")}
        >
          <LogOut className="h-4 w-4" />
        </Button>
      </div>
    </aside>
  );
}
