import { NavLink, Link, useLocation, useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  ArrowLeft,
  ArrowRight,
  Database,
  FileText,
  FolderOpen,
  HardDrive,
  Info,
  KeyRound,
  Keyboard,
  LayoutDashboard,
  LogOut,
  Settings,
  ShieldCheck,
  Upload,
  User as UserIcon,
  Users,
  type LucideIcon,
} from "lucide-react";
import { cn, formatSize } from "@/lib/utils";
import { useAuth } from "@/hooks/use-auth";
import { useI18n } from "@/i18n";
import { adminStatsApi, statsApi } from "@/lib/api";
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { KiteLogo } from "@/components/kite-logo";
import { Progress } from "@/components/ui/progress";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface NavEntry {
  to: string;
  icon: LucideIcon;
  labelKey: string;
  groupKey: "overview" | "resources" | "account" | "operations";
  /** count key indicates which stats field drives the badge */
  countKey?: "files" | "users";
}

const userNavItems: NavEntry[] = [
  { to: "/user/dashboard", icon: LayoutDashboard, labelKey: "nav.dashboard", groupKey: "overview" },
  { to: "/user/files", icon: Upload, labelKey: "nav.files", groupKey: "resources", countKey: "files" },
  { to: "/user/folders", icon: FolderOpen, labelKey: "nav.albums", groupKey: "resources" },
  { to: "/user/tokens", icon: KeyRound, labelKey: "nav.tokens", groupKey: "account" },
  { to: "/user/profile", icon: UserIcon, labelKey: "profile.title", groupKey: "account" },
];

const adminNavItems: NavEntry[] = [
  { to: "/admin/dashboard", icon: LayoutDashboard, labelKey: "nav.dashboard", groupKey: "overview" },
  { to: "/admin/files", icon: FileText, labelKey: "nav.adminFiles", groupKey: "resources", countKey: "files" },
  { to: "/admin/storage", icon: HardDrive, labelKey: "nav.storage", groupKey: "resources" },
  { to: "/admin/users", icon: Users, labelKey: "nav.users", groupKey: "operations", countKey: "users" },
  { to: "/admin/settings", icon: Settings, labelKey: "nav.settings", groupKey: "operations" },
];

const groupLabelKey: Record<NavEntry["groupKey"], string> = {
  overview: "nav.groupOverview",
  resources: "nav.groupResources",
  account: "nav.groupAccount",
  operations: "nav.groupOperations",
};

interface SidebarStats {
  total_files: number;
  total_size: number;
  users?: number;
}

interface SidebarProps {
  onClose?: () => void;
  collapsed?: boolean;
  onOpenShortcuts?: () => void;
}

/** Format compact counts: 1234 → "1.2k", 12345 → "12.4k". */
function compact(n: number | undefined): string | null {
  if (n == null || n <= 0) return null;
  if (n < 1000) return String(n);
  if (n < 1_000_000) return `${(n / 1000).toFixed(n < 10_000 ? 1 : 0).replace(/\.0$/, "")}k`;
  return `${(n / 1_000_000).toFixed(1)}M`;
}

export function Sidebar({ onClose, collapsed = false, onOpenShortcuts }: SidebarProps) {
  const { user, logout } = useAuth();
  const { t } = useI18n();
  const location = useLocation();
  const navigate = useNavigate();
  const displayName = user?.nickname?.trim() || user?.username || "";

  const isAdminWorkspace = location.pathname.startsWith("/admin");
  const navItems = isAdminWorkspace ? adminNavItems : userNavItems;
  const homePath = isAdminWorkspace ? "/admin/dashboard" : "/user/dashboard";

  /* ── Shared stats query — same key as dashboard, so cache is deduped ── */
  const { data: stats } = useQuery<SidebarStats>({
    queryKey: ["dashboard", "stats", isAdminWorkspace ? "admin" : "user"],
    queryFn: () =>
      (isAdminWorkspace ? adminStatsApi.get() : statsApi.get()).then(
        (r) => r.data.data
      ),
    staleTime: 30_000,
    refetchInterval: 60_000,
    enabled: !!user,
  });

  const badgeFor = (entry: NavEntry): string | null => {
    if (!entry.countKey || !stats) return null;
    if (entry.countKey === "files") return compact(stats.total_files);
    if (entry.countKey === "users") return compact(stats.users);
    return null;
  };

  /* ── Storage meter values ───────────────────────────────────────── */
  const storageLimit = user?.storage_limit ?? 0;
  const storageUsed = user?.storage_used ?? 0;
  const isUnlimited = storageLimit < 0;
  const hasLimit = storageLimit > 0;
  const storagePct = hasLimit
    ? Math.min(100, (storageUsed / storageLimit) * 100)
    : 0;

  /* ── Group nav items ────────────────────────────────────────────── */
  const groups = navItems.reduce<Record<NavEntry["groupKey"], NavEntry[]>>(
    (acc, item) => {
      (acc[item.groupKey] ??= []).push(item);
      return acc;
    },
    {} as Record<NavEntry["groupKey"], NavEntry[]>
  );

  const handleWorkspaceSwitch = () => {
    navigate(isAdminWorkspace ? "/user/dashboard" : "/admin/dashboard");
    onClose?.();
  };

  const handleQuickUpload = () => {
    navigate(isAdminWorkspace ? "/admin/files" : "/user/files");
    onClose?.();
  };

  const groupOrder: NavEntry["groupKey"][] = [
    "overview",
    "resources",
    "account",
    "operations",
  ];

  return (
    <TooltipProvider delayDuration={200}>
      <aside
        className={cn(
          "flex h-full flex-col bg-background transition-[width] duration-200",
          collapsed ? "w-16" : "w-60"
        )}
      >
        {/* ── Brand ───────────────────────────────────────────── */}
        <div
          className={cn(
            "flex h-14 shrink-0 items-center border-b",
            collapsed ? "justify-center px-0" : "px-4"
          )}
        >
          <Link
            to={homePath}
            onClick={onClose}
            className={cn(
              "flex items-center transition-opacity hover:opacity-80",
              collapsed ? "justify-center" : "gap-2.5"
            )}
          >
            <KiteLogo className="size-6" />
            {!collapsed && (
              <div className="flex min-w-0 items-baseline gap-1.5">
                <span className="font-semibold tracking-tight">Kite</span>
                {isAdminWorkspace && (
                  <span className="inline-flex items-center gap-1 rounded bg-foreground/10 px-1.5 py-0.5 text-[10px] font-medium">
                    <ShieldCheck className="size-3" />
                    {t("nav.admin")}
                  </span>
                )}
              </div>
            )}
          </Link>
        </div>

        {/* ── Identity card (expanded-only) ───────────────────── */}
        {!collapsed && user && (
          <div className="px-3 pt-3">
            <div className="relative overflow-hidden rounded-xl border bg-card">
              <div className="sidebar-card-backdrop" />
              <div className="relative flex items-center gap-2.5 p-2.5">
                <Avatar className="size-8 shrink-0 ring-1 ring-background">
                  <AvatarImage src={user.avatar_url} alt={displayName} />
                  <AvatarFallback className="bg-foreground text-[11px] font-medium text-background">
                    {displayName.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="min-w-0 flex-1">
                  <div className="truncate text-[12px] font-semibold leading-tight">
                    {displayName}
                  </div>
                  <div className="truncate text-[10px] text-muted-foreground">
                    @{user.username}
                  </div>
                </div>
                {isAdminWorkspace && (
                  <span className="inline-flex shrink-0 items-center gap-0.5 rounded-full border bg-background/80 px-1.5 py-0.5 text-[9px] font-semibold tracking-wide text-foreground/70 backdrop-blur">
                    <ShieldCheck className="size-2.5" />
                    ADMIN
                  </span>
                )}
              </div>
            </div>
          </div>
        )}

        {/* ── Quick upload ────────────────────────────────────── */}
        <div className={cn("pt-3", collapsed ? "flex justify-center px-2" : "px-3")}>
          {collapsed ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="default"
                  size="icon-sm"
                  className="size-10 rounded-xl"
                  onClick={handleQuickUpload}
                  aria-label={t("files.uploadFile")}
                >
                  <Upload className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">{t("files.uploadFile")}</TooltipContent>
            </Tooltip>
          ) : (
            <Button
              size="sm"
              className="w-full justify-center gap-2"
              onClick={handleQuickUpload}
            >
              <Upload className="size-3.5" />
              {t("common.upload")}
              <span className="ml-auto flex items-center gap-0.5 opacity-70">
                <kbd>⌘</kbd>
                <kbd>U</kbd>
              </span>
            </Button>
          )}
        </div>

        {/* ── Grouped nav ─────────────────────────────────────── */}
        <nav className="mt-3 flex-1 overflow-y-auto px-2 pb-3">
          {groupOrder
            .filter((g) => groups[g] && groups[g].length > 0)
            .map((g, gi) => (
              <div key={g} className={cn(gi > 0 && (collapsed ? "mt-2" : "mt-4"))}>
                {!collapsed && (
                  <div className="px-3 py-1 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/80">
                    {t(groupLabelKey[g])}
                  </div>
                )}
                {collapsed && gi > 0 && (
                  <div className="mb-2 flex justify-center">
                    <div className="h-px w-6 bg-border/70" />
                  </div>
                )}
                <div className={cn(collapsed ? "flex flex-col items-center gap-1" : "space-y-0.5")}>
                  {groups[g].map((it) => {
                    const badge = badgeFor(it);
                    const content = (
                      <NavLink
                        key={it.to}
                        to={it.to}
                        onClick={onClose}
                        className={({ isActive }) =>
                          cn(
                            "relative flex items-center text-sm font-medium transition-colors",
                            collapsed
                              ? cn(
                                  "size-10 shrink-0 justify-center rounded-xl",
                                  isActive
                                    ? "bg-foreground/10 text-foreground"
                                    : "text-muted-foreground hover:bg-accent hover:text-foreground"
                                )
                              : cn(
                                  "w-full gap-2.5 rounded-lg px-3 py-2",
                                  isActive
                                    ? "bg-foreground/[0.06] text-foreground"
                                    : "text-muted-foreground hover:bg-accent/60 hover:text-foreground"
                                )
                          )
                        }
                      >
                        {({ isActive }) => (
                          <>
                            {/* active indicator rail (expanded only) */}
                            {isActive && !collapsed && (
                              <span className="absolute left-0 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-r-full bg-foreground" />
                            )}
                            <it.icon
                              className={cn(
                                "size-4 shrink-0",
                                isActive && !collapsed && "text-foreground",
                                isActive && collapsed && "stroke-[2.25]"
                              )}
                            />
                            {!collapsed && (
                              <>
                                <span className="flex-1 text-left">{t(it.labelKey)}</span>
                                {badge && (
                                  <span
                                    className={cn(
                                      "rounded-md border px-1.5 py-0 text-[10px] tabular-nums leading-[1.4]",
                                      isActive
                                        ? "border-foreground/20 bg-background text-foreground/80"
                                        : "border-border/80 text-muted-foreground"
                                    )}
                                  >
                                    {badge}
                                  </span>
                                )}
                              </>
                            )}
                            {collapsed && badge && !isActive && (
                              <span className="absolute right-1 top-1 size-1.5 rounded-full bg-foreground/70 ring-2 ring-background" />
                            )}
                          </>
                        )}
                      </NavLink>
                    );
                    return collapsed ? (
                      <Tooltip key={it.to}>
                        <TooltipTrigger asChild>{content}</TooltipTrigger>
                        <TooltipContent side="right">
                          {t(it.labelKey)}
                        </TooltipContent>
                      </Tooltip>
                    ) : (
                      content
                    );
                  })}
                </div>
              </div>
            ))}

          {/* ── Storage meter (expanded-only, non-admin workspace) ── */}
          {!collapsed && !isAdminWorkspace && user && (
            <div className="mt-5 rounded-xl border bg-muted/30 p-3">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
                  <Database className="size-3" />
                  {t("nav.storageLabel")}
                </div>
                {hasLimit && (
                  <span className="text-[10px] tabular-nums text-muted-foreground">
                    {Math.round(storagePct)}%
                  </span>
                )}
              </div>
              <div className="mt-2 flex items-baseline gap-1">
                <span className="text-[13px] font-semibold tabular-nums">
                  {formatSize(storageUsed)}
                </span>
                <span className="text-[10px] tabular-nums text-muted-foreground">
                  /{" "}
                  {isUnlimited
                    ? t("users.unlimited")
                    : hasLimit
                      ? formatSize(storageLimit)
                      : "—"}
                </span>
              </div>
              {hasLimit ? (
                <Progress
                  value={storagePct}
                  className="mt-2 h-1.5 rounded-full bg-muted/70"
                  indicatorClassName="bg-foreground rounded-full"
                />
              ) : (
                <div className="mt-2 h-1.5 rounded-full bg-muted/70" />
              )}
            </div>
          )}
        </nav>

        {/* ── Footer: workspace switch + utility row ─────────── */}
        <div className={cn("shrink-0 border-t", collapsed ? "px-2 py-2" : "p-3")}>
          {collapsed ? (
            <div className="flex flex-col items-center gap-1">
              {user?.role === "admin" && (
                <>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="icon-sm"
                        className="size-9 rounded-lg"
                        onClick={handleWorkspaceSwitch}
                        aria-label={
                          isAdminWorkspace ? t("nav.backToUser") : t("nav.adminPanel")
                        }
                      >
                        {isAdminWorkspace ? (
                          <UserIcon className="size-3.5" />
                        ) : (
                          <ShieldCheck className="size-3.5 text-violet-600 dark:text-violet-400" />
                        )}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent side="right">
                      {isAdminWorkspace ? t("nav.backToUser") : t("nav.adminPanel")}
                    </TooltipContent>
                  </Tooltip>
                  <div className="my-0.5 h-px w-6 bg-border/70" />
                </>
              )}
              {onOpenShortcuts && (
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="size-9 rounded-lg text-muted-foreground hover:text-foreground"
                      onClick={onOpenShortcuts}
                      aria-label={t("nav.shortcuts")}
                    >
                      <Keyboard className="size-3.5" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="right">{t("nav.shortcuts")}</TooltipContent>
                </Tooltip>
              )}
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={logout}
                    aria-label={t("auth.logout")}
                    className="size-9 rounded-lg text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                  >
                    <LogOut className="size-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="right">{t("auth.logout")}</TooltipContent>
              </Tooltip>
            </div>
          ) : (
            <div className="space-y-2">
              {user?.role === "admin" && (
                <button
                  type="button"
                  onClick={handleWorkspaceSwitch}
                  className="group/switch flex w-full items-center gap-2 rounded-lg border bg-background p-2 text-left transition-colors hover:border-foreground/20 hover:bg-accent/40"
                >
                  <div
                    className={cn(
                      "flex size-7 shrink-0 items-center justify-center rounded-md",
                      isAdminWorkspace
                        ? "bg-foreground/[0.06]"
                        : "bg-violet-500/10 text-violet-600 dark:text-violet-400"
                    )}
                  >
                    {isAdminWorkspace ? (
                      <UserIcon className="size-3.5" />
                    ) : (
                      <ShieldCheck className="size-3.5" />
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="text-[11px] font-medium leading-tight">
                      {isAdminWorkspace
                        ? t("nav.switchToUser")
                        : t("nav.adminPanel")}
                    </div>
                    <div className="truncate text-[10px] text-muted-foreground">
                      {isAdminWorkspace
                        ? t("nav.switchToUserDesc")
                        : t("nav.switchToAdminDesc")}
                    </div>
                  </div>
                  {isAdminWorkspace ? (
                    <ArrowLeft className="size-3 shrink-0 text-muted-foreground transition-transform group-hover/switch:-translate-x-0.5" />
                  ) : (
                    <ArrowRight className="size-3 shrink-0 text-muted-foreground transition-transform group-hover/switch:translate-x-0.5" />
                  )}
                </button>
              )}

              {/* Mini-button row: Shortcuts · Logout · Help */}
              <div className="flex items-center gap-1">
                {onOpenShortcuts && (
                  <>
                    <button
                      type="button"
                      onClick={onOpenShortcuts}
                      className="flex flex-1 items-center justify-center gap-1 rounded-md py-1 text-[10px] text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                    >
                      <Keyboard className="size-3" />
                      {t("nav.shortcuts")}
                    </button>
                    <div className="h-3 w-px bg-border" />
                  </>
                )}
                <Link
                  to="/user/profile"
                  onClick={onClose}
                  className="flex flex-1 items-center justify-center gap-1 rounded-md py-1 text-[10px] text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                >
                  <Info className="size-3" />
                  {t("nav.help")}
                </Link>
                <div className="h-3 w-px bg-border" />
                <button
                  type="button"
                  onClick={logout}
                  className="flex flex-1 items-center justify-center gap-1 rounded-md py-1 text-[10px] text-muted-foreground transition-colors hover:bg-accent hover:text-destructive"
                  aria-label={t("auth.logout")}
                >
                  <LogOut className="size-3" />
                  {t("auth.logout")}
                </button>
              </div>
            </div>
          )}
        </div>
      </aside>
    </TooltipProvider>
  );
}
