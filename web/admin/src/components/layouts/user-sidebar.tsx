import { Link, NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  Upload,
  FolderOpen,
  KeyRound,
  LogOut,
  Shield,
  X,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/hooks/use-auth";
import { useI18n } from "@/i18n";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";

const navItems = [
  { to: "/dashboard", icon: LayoutDashboard, labelKey: "nav.dashboard" },
  { to: "/files", icon: Upload, labelKey: "nav.files" },
  { to: "/albums", icon: FolderOpen, labelKey: "nav.albums" },
  { to: "/tokens", icon: KeyRound, labelKey: "nav.tokens" },
];

interface UserSidebarProps {
  onClose?: () => void;
}

export function UserSidebar({ onClose }: UserSidebarProps) {
  const { user, logout } = useAuth();
  const { t } = useI18n();

  return (
    <aside className="flex h-full w-[240px] shrink-0 flex-col border-r border-sidebar-border bg-sidebar">
      {/* Brand */}
      <div className="flex h-16 items-center justify-between border-b border-sidebar-border px-5">
        <Link to="/" className="flex items-center gap-2.5 transition-opacity hover:opacity-80">
          <div className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm">
            <span className="text-sm font-bold">K</span>
          </div>
          <span className="text-base font-bold tracking-tight">Kite</span>
        </Link>
        {onClose && (
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground lg:hidden"
          >
            <X size={18} />
          </button>
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            onClick={onClose}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-[13px] font-medium transition-colors",
                isActive
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:bg-accent hover:text-foreground"
              )
            }
          >
            <item.icon size={18} />
            {t(item.labelKey)}
          </NavLink>
        ))}

        {/* Admin panel entry */}
        {user?.role === "admin" && (
          <>
            <div className="my-2 border-t border-sidebar-border" />
            <Link
              to="/admin"
              onClick={onClose}
              className="flex items-center gap-3 rounded-lg px-3 py-2 text-[13px] font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              <Shield size={18} />
              {t("nav.adminPanel")}
            </Link>
          </>
        )}
      </nav>

      {/* User */}
      <div className="border-t border-sidebar-border p-3">
        <div className="flex items-center gap-3 rounded-lg px-3 py-2">
          <Avatar className="size-8 border">
            <AvatarFallback className="bg-primary/10 text-xs font-semibold text-primary">
              {user?.username?.charAt(0).toUpperCase() ?? "U"}
            </AvatarFallback>
          </Avatar>
          <div className="flex-1 overflow-hidden">
            <p className="truncate text-sm font-medium">{user?.username}</p>
            <p className="truncate text-[11px] text-muted-foreground">
              {user?.role === "admin" ? t("nav.roleAdmin") : t("nav.roleUser")}
            </p>
          </div>
          <button
            onClick={logout}
            className="rounded p-1.5 text-muted-foreground transition-colors hover:bg-destructive/10 hover:text-destructive"
            title={t("auth.logout")}
          >
            <LogOut size={14} />
          </button>
        </div>
      </div>
    </aside>
  );
}
