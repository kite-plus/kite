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
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { KiteLogo } from "@/components/kite-logo";

const commonNavItems = [
  { to: "/dashboard", icon: LayoutDashboard, labelKey: "nav.dashboard" },
];

const userExtraNavItems = [
  { to: "/albums", icon: FolderOpen, labelKey: "nav.albums" },
  { to: "/tokens", icon: KeyRound, labelKey: "nav.tokens" },
];

const adminNavItems = [
  { to: "/admin/storage", icon: HardDrive, labelKey: "nav.storage" },
  { to: "/admin/users", icon: Users, labelKey: "nav.users" },
  { to: "/admin/settings", icon: Settings, labelKey: "nav.settings" },
];

interface SidebarProps {
  onClose?: () => void;
  collapsed?: boolean;
}

export function Sidebar({ onClose, collapsed = false }: SidebarProps) {
  const { user, logout } = useAuth();
  const { t } = useI18n();
  const displayName = user?.nickname?.trim() || user?.username;
  const roleMainNavItems = user?.role === "admin"
    ? [{ to: "/admin/files", icon: FileText, labelKey: "nav.adminFiles" }]
    : [{ to: "/files", icon: Upload, labelKey: "nav.files" }];
  const roleExtraNavItems = user?.role === "admin" ? [] : userExtraNavItems;

  return (
    <aside className={cn("flex h-full flex-col bg-background transition-[width] duration-200", collapsed ? "w-16" : "w-55")}>
      {/* Logo — h-14 + border-b aligns with desktop header */}
      <div className={cn("flex h-14 shrink-0 items-center border-b", collapsed ? "justify-center px-0" : "justify-start px-4")}>
        <Link
          to="/dashboard"
          onClick={onClose}
          className={cn("flex items-center transition-opacity hover:opacity-80", collapsed ? "justify-center" : "gap-2.5")}
        >
          <KiteLogo className="size-6" />
          {!collapsed && <span className="font-semibold tracking-tight">Kite</span>}
        </Link>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-3">
        {[...commonNavItems, ...roleMainNavItems, ...roleExtraNavItems].map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            onClick={onClose}
            className={({ isActive }) =>
              cn(
                "flex items-center rounded-lg py-2 text-sm font-medium transition-colors",
                collapsed ? "justify-center px-2" : "gap-3 px-3",
                isActive
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              )
            }
          >
            <item.icon className="h-4 w-4" />
            {!collapsed && t(item.labelKey)}
          </NavLink>
        ))}
        {user?.role === "admin" && adminNavItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            onClick={onClose}
            className={({ isActive }) =>
              cn(
                "flex items-center rounded-lg py-2 text-sm font-medium transition-colors",
                collapsed ? "justify-center px-2" : "gap-3 px-3",
                isActive
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              )
            }
          >
            <item.icon className="h-4 w-4" />
            {!collapsed && t(item.labelKey)}
          </NavLink>
        ))}
      </nav>

      {/* Bottom user section — h-14 matches the footer */}
      <div className="flex h-14 shrink-0 items-center gap-2 border-t px-3 md:hidden">
        <Avatar className="size-7">
          <AvatarImage src={user?.avatar_url} alt={user?.username ?? ""} />
          <AvatarFallback className="text-[10px] font-medium">
            {user?.username?.charAt(0).toUpperCase()}
          </AvatarFallback>
        </Avatar>
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium leading-none">
            {displayName}
          </p>
          <p className="mt-0.5 text-[11px] text-muted-foreground">
            @{user?.username}
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
