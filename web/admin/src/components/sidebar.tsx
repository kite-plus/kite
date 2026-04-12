import { NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  Upload,
  FolderOpen,
  Key,
  Settings,
  Users,
  HardDrive,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuth } from "@/hooks/use-auth";
import { useI18n } from "@/i18n";

const navItems = [
  { to: "/", icon: LayoutDashboard, labelKey: "nav.dashboard" },
  { to: "/files", icon: Upload, labelKey: "nav.files" },
  { to: "/albums", icon: FolderOpen, labelKey: "nav.albums" },
  { to: "/tokens", icon: Key, labelKey: "nav.tokens" },
];

const adminItems = [
  { to: "/admin/storage", icon: HardDrive, labelKey: "nav.storage" },
  { to: "/admin/users", icon: Users, labelKey: "nav.users" },
  { to: "/admin/settings", icon: Settings, labelKey: "nav.settings" },
];

export function Sidebar() {
  const { user } = useAuth();
  const { t } = useI18n();
  const isAdmin = user?.role === "admin";

  return (
    <aside className="flex h-full w-56 flex-col border-r bg-sidebar">
      <div className="flex h-14 items-center gap-2 border-b px-4">
        <div className="flex size-7 items-center justify-center rounded-md bg-primary text-primary-foreground text-xs font-bold">
          K
        </div>
        <span className="text-sm font-semibold tracking-tight">Kite</span>
      </div>

      <nav className="flex-1 space-y-1 p-3">
        <div className="mb-2 px-2 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
          {t("nav.general")}
        </div>
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/"}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 rounded-md px-2.5 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-sidebar-accent text-sidebar-accent-foreground"
                  : "text-sidebar-foreground hover:bg-sidebar-accent/50"
              )
            }
          >
            <item.icon className="size-4" />
            {t(item.labelKey)}
          </NavLink>
        ))}

        {isAdmin && (
          <>
            <div className="mb-2 mt-6 px-2 text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
              {t("nav.admin")}
            </div>
            {adminItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-3 rounded-md px-2.5 py-2 text-sm font-medium transition-colors",
                    isActive
                      ? "bg-sidebar-accent text-sidebar-accent-foreground"
                      : "text-sidebar-foreground hover:bg-sidebar-accent/50"
                  )
                }
              >
                <item.icon className="size-4" />
                {t(item.labelKey)}
              </NavLink>
            ))}
          </>
        )}
      </nav>
    </aside>
  );
}
