import { useEffect, useMemo, useState } from "react";
import { Navigate, Link, useLocation, useNavigate } from "react-router-dom";
import { PageTransition } from "@/components/page-transition";
import {
  ArrowLeft,
  Bell,
  ChevronRight,
  KeyRound,
  Keyboard,
  LogOut,
  Menu,
  Search,
  ShieldCheck,
  User as UserIcon,
} from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useI18n, type Locale } from "@/i18n";
import { Sidebar } from "@/components/layouts/sidebar";
import { ShortcutsDialog } from "@/components/shortcuts-dialog";
import { Button } from "@/components/ui/button";
import { KiteLogo } from "@/components/kite-logo";
import { ThemeToggle } from "@/components/theme-toggle";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Progress } from "@/components/ui/progress";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn, formatSize } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";
import {
  Sheet,
  SheetContent,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  Dialog,
  DialogContent,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "@/components/ui/command";
import { Badge } from "@/components/ui/badge";
import { getPrimaryModifierKeyLabel } from "@/lib/platform";

const routeLabelKeys: Record<string, string> = {
  "/user": "nav.general",
  "/user/dashboard": "nav.dashboard",
  "/user/files": "nav.files",
  "/user/albums": "nav.albums",
  "/user/folders": "nav.albums",
  "/user/tokens": "nav.tokens",
  "/user/profile": "profile.title",
  "/admin": "nav.adminPanel",
  "/admin/dashboard": "nav.dashboard",
  "/admin/files": "nav.adminFiles",
  "/admin/storage": "nav.storage",
  "/admin/users": "nav.users",
  "/admin/settings": "nav.settings",
};

type SearchTarget = {
  to: string;
  label: string;
  group: string;
  keywords: string;
};

export default function AppLayout() {
  const { user, loading, logout } = useAuth();
  const { t, locale, setLocale } = useI18n();
  const location = useLocation();
  const navigate = useNavigate();
  const [mobileOpen, setMobileOpen] = useState(false);
  const [globalQuery, setGlobalQuery] = useState("");
  const [commandOpen, setCommandOpen] = useState(false);
  const [shortcutsOpen, setShortcutsOpen] = useState(false);
  const displayName = user?.nickname?.trim() || user?.username;
  const modifierKeyLabel = getPrimaryModifierKeyLabel();

  const isAdminWorkspace = location.pathname.startsWith("/admin");
  const homePath = isAdminWorkspace ? "/admin/dashboard" : "/user/dashboard";

  const breadcrumbItems = useMemo(() => {
    const parts = location.pathname.split("/").filter(Boolean);
    if (!parts.length) {
      return [{ to: "/user/dashboard", label: t("nav.dashboard") }];
    }

    const items: Array<{ to: string; label: string }> = [];
    for (let i = 0; i < parts.length; i += 1) {
      const to = `/${parts.slice(0, i + 1).join("/")}`;
      const labelKey = routeLabelKeys[to];
      const label = labelKey ? t(labelKey) : parts[i];
      items.push({ to, label });
    }
    return items;
  }, [location.pathname, t]);

  const globalSearchTargets = useMemo<SearchTarget[]>(() => {
    const workspaceGroup = t("nav.general");
    const adminGroup = t("nav.admin");
    const base: SearchTarget[] = [
      { to: "/user/dashboard", label: t("nav.dashboard"), group: workspaceGroup, keywords: "home stats" },
      { to: "/user/files", label: t("nav.files"), group: workspaceGroup, keywords: "upload media" },
      { to: "/user/folders", label: t("nav.albums"), group: workspaceGroup, keywords: "folder directory hierarchy" },
      { to: "/user/tokens", label: t("nav.tokens"), group: workspaceGroup, keywords: "api key" },
      { to: "/user/profile", label: t("profile.title"), group: workspaceGroup, keywords: "account user" },
    ];

    if (user?.role === "admin") {
      base.push(
        { to: "/admin/dashboard", label: t("nav.dashboard"), group: adminGroup, keywords: "admin overview" },
        { to: "/admin/files", label: t("files.adminTitle"), group: adminGroup, keywords: "all files moderation" },
        { to: "/admin/storage", label: t("nav.storage"), group: adminGroup, keywords: "s3 local driver" },
        { to: "/admin/users", label: t("nav.users"), group: adminGroup, keywords: "members role" },
        { to: "/admin/settings", label: t("nav.settings"), group: adminGroup, keywords: "config system" }
      );
    }

    return base;
  }, [t, user?.role]);

  const filteredTargets = useMemo(() => {
    const query = globalQuery.trim().toLowerCase();
    if (!query) return globalSearchTargets.slice(0, 8);
    return globalSearchTargets
      .filter((item) => {
        const source = `${item.label} ${item.to} ${item.keywords}`.toLowerCase();
        return source.includes(query);
      })
      .slice(0, 8);
  }, [globalQuery, globalSearchTargets]);

  const groupedTargets = useMemo(() => {
    const groups: Record<string, SearchTarget[]> = {};
    filteredTargets.forEach((target) => {
      if (!groups[target.group]) groups[target.group] = [];
      groups[target.group].push(target);
    });
    return groups;
  }, [filteredTargets]);

  useEffect(() => {
    const toggleTheme = () => {
      const html = document.documentElement;
      const isDark = html.classList.contains("dark");
      html.classList.toggle("dark", !isDark);
      localStorage.setItem("theme", isDark ? "light" : "dark");
    };

    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const tag = target?.tagName ?? "";
      const inField =
        tag === "INPUT" ||
        tag === "TEXTAREA" ||
        target?.isContentEditable === true;
      const mod = event.metaKey || event.ctrlKey;

      // ⌘K / Ctrl+K — command menu
      if (mod && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setCommandOpen((prev) => {
          const next = !prev;
          if (!next) setGlobalQuery("");
          return next;
        });
        return;
      }

      // Esc closes dialogs (native behaviour handles it, but guard against stray)
      if (event.key === "Escape") {
        if (commandOpen) setCommandOpen(false);
        if (shortcutsOpen) setShortcutsOpen(false);
        return;
      }

      if (inField) return;

      // ? — open keyboard shortcuts
      if (event.key === "?") {
        event.preventDefault();
        setShortcutsOpen(true);
        return;
      }

      // ⌘U / Ctrl+U — quick upload (user workspace only: opens upload dialog
      // via the ?upload=1 query param consumed by the files page)
      if (mod && event.key.toLowerCase() === "u") {
        const isAdmin = location.pathname.startsWith("/admin");
        if (isAdmin) return;
        event.preventDefault();
        navigate("/user/files?upload=1");
        return;
      }

      // ⌘. — toggle theme
      if (mod && event.key === ".") {
        event.preventDefault();
        toggleTheme();
        return;
      }

      // Sequential "g X" navigation (Gmail-style)
      if (event.key === "g" || event.key === "G") {
        const nextKey = (e2: KeyboardEvent) => {
          const isAdmin = location.pathname.startsWith("/admin");
          const userMap: Record<string, string> = {
            d: "/user/dashboard",
            f: "/user/files",
            a: "/user/folders",
            t: "/user/tokens",
            p: "/user/profile",
          };
          const adminMap: Record<string, string> = {
            d: "/admin/dashboard",
            f: "/admin/files",
            s: "/admin/storage",
            u: "/admin/users",
            t: "/admin/settings",
          };
          const map = isAdmin ? adminMap : userMap;
          const key = e2.key.toLowerCase();
          const to = map[key];
          if (to) {
            e2.preventDefault();
            navigate(to);
          }
          window.removeEventListener("keydown", nextKey, true);
        };
        window.addEventListener("keydown", nextKey, true);
        setTimeout(
          () => window.removeEventListener("keydown", nextKey, true),
          1200
        );
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [commandOpen, shortcutsOpen, navigate, location.pathname]);

  const goToRoute = (to: string) => {
    navigate(to);
    setGlobalQuery("");
    setCommandOpen(false);
  };

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
    <div className="flex h-screen overflow-hidden bg-background">
      {/* Desktop sidebar */}
      <div className="hidden shrink-0 border-r md:flex">
        <Sidebar
          onOpenShortcuts={() => setShortcutsOpen(true)}
        />
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
                to={homePath}
                className="flex items-center gap-2 transition-opacity hover:opacity-80"
              >
                <KiteLogo className="size-6" />
                <span className="font-semibold tracking-tight">Kite</span>
                {isAdminWorkspace && (
                  <Badge variant="secondary" className="gap-1 text-[10px]">
                    <ShieldCheck className="size-3" />
                    {t("nav.admin")}
                  </Badge>
                )}
              </Link>
            </div>

            <button
              type="button"
              className="mx-3 flex h-8 min-w-0 flex-1 items-center justify-between rounded-md border border-input/90 bg-background px-2.5 text-xs text-muted-foreground"
              onClick={() => setCommandOpen(true)}
            >
              <span className="flex min-w-0 items-center gap-1.5">
                <Search className="size-3.5 shrink-0" />
                <span className="truncate">{t("common.search")}</span>
              </span>
            </button>

            <ThemeToggle />
          </header>
          <SheetContent side="left" className="w-60 p-0">
            <SheetTitle className="sr-only">Navigation</SheetTitle>
            <Sidebar
              onClose={() => setMobileOpen(false)}
              onOpenShortcuts={() => {
                setMobileOpen(false);
                setShortcutsOpen(true);
              }}
            />
          </SheetContent>
        </Sheet>

        {/* Desktop header */}
        <header className="hidden h-14 shrink-0 border-b md:flex">
          <div className="mx-auto flex w-full max-w-screen-2xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
            <div className="flex min-w-0 flex-1 items-center gap-3">
              <nav className="hidden min-w-max items-center gap-1 text-xs text-muted-foreground md:flex">
                {breadcrumbItems.map((item, index) => {
                  const isLast = index === breadcrumbItems.length - 1;
                  return (
                    <div key={item.to} className="flex items-center gap-1">
                      {index > 0 && <ChevronRight className="size-3 text-muted-foreground/70" />}
                      {isLast ? (
                        <span className="rounded bg-muted px-2 py-1 font-medium text-foreground">{item.label}</span>
                      ) : (
                        <Link to={item.to} className="rounded px-1.5 py-1 transition-colors hover:bg-muted hover:text-foreground">
                          {item.label}
                        </Link>
                      )}
                    </div>
                  );
                })}
              </nav>

              <button
                type="button"
                className="flex h-9 w-[clamp(180px,18vw,240px)] shrink-0 items-center justify-between rounded-lg border border-input/90 bg-background px-3 text-sm text-muted-foreground shadow-xs transition-colors hover:bg-accent/30"
                onClick={() => setCommandOpen(true)}
              >
                <span className="flex min-w-0 items-center gap-2">
                  <Search className="size-4 shrink-0" />
                  <span className="truncate">{t("common.search")}</span>
                </span>
                <Badge variant="secondary" className="ml-auto shrink-0">
                  {modifierKeyLabel === "⌘" ? "⌘K" : "Ctrl+K"}
                </Badge>
              </button>
            </div>

            <div className="flex items-center gap-1">
              <TooltipProvider delayDuration={200}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-muted-foreground hover:text-foreground"
                      onClick={() => setShortcutsOpen(true)}
                      aria-label={t("nav.shortcuts")}
                    >
                      <Keyboard className="size-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="bottom">
                    <span className="flex items-center gap-1.5">
                      {t("nav.shortcuts")}
                      <kbd>?</kbd>
                    </span>
                  </TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="relative text-muted-foreground hover:text-foreground"
                      aria-label="Notifications"
                    >
                      <Bell className="size-4" />
                      <span
                        className="absolute right-1.5 top-1.5 size-1.5 rounded-full"
                        style={{ background: "hsl(var(--chart-2))" }}
                      />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="bottom">Notifications</TooltipContent>
                </Tooltip>
              </TooltipProvider>
              <ThemeToggle />
              <div className="mx-1 h-5 w-px bg-border" />
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="gap-2 px-2 text-xs text-muted-foreground hover:bg-accent hover:text-foreground"
                  >
                    <Avatar className="size-5">
                      <AvatarImage src={user.avatar_url} alt={user.username ?? ""} />
                      <AvatarFallback className="text-[10px] font-medium">
                        {user.username?.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <span className="font-medium">{displayName}</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" sideOffset={8} className="w-70 overflow-hidden rounded-xl border-border/80 p-0 shadow-lg">
                  {(() => {
                    const storageLimit = user.storage_limit ?? 0;
                    const storageUsed = user.storage_used ?? 0;
                    const isUnlimited = storageLimit < 0;
                    const hasLimit = storageLimit > 0;
                    const storagePct = hasLimit
                      ? Math.min(100, (storageUsed / storageLimit) * 100)
                      : 0;
                    const isAdmin = user.role === "admin";
                    return (
                      <>
                        <div className="flex items-center gap-3 px-3.5 pt-3.5 pb-3">
                          <Avatar className="size-9 ring-1 ring-border/60">
                            <AvatarImage src={user.avatar_url} alt={user.username ?? ""} />
                            <AvatarFallback className="bg-foreground text-background text-[11px] font-medium">
                              {displayName?.charAt(0).toUpperCase()}
                            </AvatarFallback>
                          </Avatar>
                          <div className="min-w-0 flex-1 space-y-0.5">
                            <div className="truncate text-[13px] font-semibold leading-tight text-foreground">
                              {displayName}
                            </div>
                            <p className="truncate text-xs leading-tight text-muted-foreground">
                              {user.email || `@${user.username}`}
                            </p>
                          </div>
                        </div>

                        <div className="space-y-2 border-t border-border/60 px-3.5 pt-2.5 pb-3.5">
                          <div className="flex items-baseline justify-between">
                            <span className="text-[10px] font-medium uppercase tracking-[0.08em] text-muted-foreground">
                              {t("users.storageCol")}
                            </span>
                            <span className="text-[11px] tabular-nums text-foreground">
                              {formatSize(storageUsed)}
                              <span className="text-muted-foreground">
                                {" / "}
                                {isUnlimited ? t("users.unlimited") : formatSize(storageLimit)}
                              </span>
                            </span>
                          </div>
                          {isUnlimited || !hasLimit ? (
                            <div className="h-1 rounded-full bg-muted/70" />
                          ) : (
                            <Progress
                              value={storagePct}
                              className="h-1 rounded-full bg-muted/70"
                              indicatorClassName="bg-foreground rounded-full"
                            />
                          )}
                        </div>

                        <div className="border-t border-border/60 p-1.5">
                          <DropdownMenuItem
                            onClick={() => navigate("/user/profile")}
                            className="gap-2.5 rounded-md px-2.5 py-2 text-[13px] focus:bg-accent/80"
                          >
                            <UserIcon className="size-3.5 text-muted-foreground" />
                            {t("profile.title")}
                          </DropdownMenuItem>
                          <DropdownMenuItem
                            onClick={() => navigate("/user/tokens")}
                            className="gap-2.5 rounded-md px-2.5 py-2 text-[13px] focus:bg-accent/80"
                          >
                            <KeyRound className="size-3.5 text-muted-foreground" />
                            {t("nav.tokens")}
                          </DropdownMenuItem>
                          {isAdmin && (
                            <DropdownMenuItem
                              onClick={() =>
                                navigate(isAdminWorkspace ? "/user/dashboard" : "/admin/dashboard")
                              }
                              className="gap-2.5 rounded-md px-2.5 py-2 text-[13px] focus:bg-accent/80"
                            >
                              {isAdminWorkspace ? (
                                <ArrowLeft className="size-3.5 text-muted-foreground" />
                              ) : (
                                <ShieldCheck className="size-3.5 text-muted-foreground" />
                              )}
                              {isAdminWorkspace ? t("nav.backToUser") : t("nav.adminPanel")}
                            </DropdownMenuItem>
                          )}
                        </div>

                        <div className="flex items-center justify-between border-t border-border/60 px-3.5 py-2.5">
                          <span className="text-xs text-muted-foreground">
                            {t("settings.language")}
                          </span>
                          <div className="relative flex items-center rounded-md bg-muted/70 p-[3px]">
                            {(["en", "zh"] as Locale[]).map((code) => (
                              <button
                                key={code}
                                type="button"
                                onClick={() => setLocale(code)}
                                className={cn(
                                  "relative z-10 rounded-[5px] px-2.5 py-0.5 text-[11px] leading-[18px] transition-colors",
                                  locale === code
                                    ? "bg-background font-medium text-foreground"
                                    : "text-muted-foreground hover:text-foreground"
                                )}
                              >
                                {code === "en" ? "EN" : "中"}
                              </button>
                            ))}
                          </div>
                        </div>

                        <div className="border-t border-border/60 p-1.5">
                          <DropdownMenuItem
                            onClick={logout}
                            className="gap-2.5 rounded-md px-2.5 py-2 text-[13px] text-muted-foreground focus:bg-accent/80 focus:text-foreground"
                          >
                            <LogOut className="size-3.5" />
                            {t("auth.logout")}
                          </DropdownMenuItem>
                        </div>
                      </>
                    );
                  })()}
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </header>

        <ShortcutsDialog
          open={shortcutsOpen}
          onOpenChange={setShortcutsOpen}
        />

        <Dialog
          open={commandOpen}
          onOpenChange={(open) => {
            setCommandOpen(open);
            if (!open) setGlobalQuery("");
          }}
        >
          <DialogContent className="max-h-[min(85vh,600px)] w-[min(90vw,500px)] gap-0 overflow-hidden border-border/80 p-0 shadow-2xl md:max-h-150 md:w-170 [&>button]:hidden">
            <DialogTitle className="sr-only">Search</DialogTitle>
            <Command shouldFilter={false} className="rounded-none">
              <CommandInput
                autoFocus
                value={globalQuery}
                onValueChange={setGlobalQuery}
                placeholder={`${t("common.search")}...`}
              />
              <CommandList className="max-h-[min(70vh,520px)]">
                <CommandEmpty>{t("common.noData")}</CommandEmpty>
                {Object.entries(groupedTargets).map(([groupName, items], index) => (
                  <div key={groupName}>
                    {index > 0 && <CommandSeparator />}
                    <CommandGroup
                      heading={(
                        <span className="block px-2 py-1 text-[11px] font-semibold tracking-wide text-muted-foreground">
                          {groupName}
                        </span>
                      )}
                    >
                      {items.map((target) => (
                        <CommandItem
                          key={target.to}
                          value={`${target.label} ${target.to} ${target.keywords}`}
                          onSelect={() => goToRoute(target.to)}
                          className="h-10"
                        >
                          <span className="truncate font-medium">{target.label}</span>
                          <CommandShortcut>{target.to}</CommandShortcut>
                        </CommandItem>
                      ))}
                    </CommandGroup>
                  </div>
                ))}
              </CommandList>
            </Command>
          </DialogContent>
        </Dialog>

        {/* Scrollable content */}
        <main className="flex-1 overflow-y-auto">
          <div className="mx-auto w-full max-w-screen-2xl px-4 py-6 sm:px-6 lg:px-8">
            <PageTransition />
          </div>
        </main>

        {/* Footer — always a single row; mobile trims to essentials */}
        <footer className="shrink-0 border-t bg-background">
          <div className="mx-auto flex w-full max-w-screen-2xl items-center justify-between gap-3 px-4 py-2.5 text-[11px] text-muted-foreground sm:px-6 lg:px-8">
            {/* left cluster: brand · [year ·] version */}
            <div className="flex min-w-0 items-center gap-x-2 sm:gap-x-3">
              <div className="flex shrink-0 items-center gap-1.5">
                <KiteLogo className="size-3.5 opacity-80" />
                <span className="font-medium text-foreground/80">Kite</span>
              </div>
              <span className="hidden text-border sm:inline">·</span>
              <span className="hidden sm:inline">
                &copy; {new Date().getFullYear()}
              </span>
              <span className="hidden text-border sm:inline">·</span>
              <span className="shrink-0 rounded border px-1.5 py-0.5 font-mono text-[10px]">
                v{__APP_VERSION__}
              </span>
            </div>
            {/* right cluster: status · (⌘K · ? sm+) · GitHub */}
            <div className="flex shrink-0 items-center gap-x-2 sm:gap-x-3">
              <span className="inline-flex items-center gap-1">
                <span className="size-1.5 rounded-full bg-emerald-500" />
                <span className="hidden sm:inline">{t("footer.statusOk")}</span>
              </span>
              <button
                type="button"
                onClick={() => setCommandOpen(true)}
                className="hidden items-center gap-1 transition-colors hover:text-foreground sm:inline-flex"
              >
                <kbd className="rounded border bg-muted/60 px-1 font-mono text-[10px]">
                  {modifierKeyLabel}
                </kbd>
                <kbd className="rounded border bg-muted/60 px-1 font-mono text-[10px]">
                  K
                </kbd>
                <span>{t("footer.search")}</span>
              </button>
              <button
                type="button"
                onClick={() => setShortcutsOpen(true)}
                className="hidden items-center gap-1 transition-colors hover:text-foreground sm:inline-flex"
              >
                <kbd className="rounded border bg-muted/60 px-1 font-mono text-[10px]">
                  ?
                </kbd>
                <span>{t("footer.shortcuts")}</span>
              </button>
              <a
                href="https://github.com/amigoer/kite"
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 transition-colors hover:text-foreground"
                aria-label="GitHub"
              >
                <svg
                  className="size-3.5"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                >
                  <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
                </svg>
                <span className="hidden sm:inline">GitHub</span>
              </a>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}
