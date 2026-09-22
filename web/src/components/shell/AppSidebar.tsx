import type { ReactNode } from "react";
import { ExternalLink, Languages, LogOut, SunMoon } from "lucide-react";

import { locales, useI18n, type Key, type Locale } from "@/i18n";
import { useContentTypes, useSite } from "@/hooks/useContents";
import { useKindLabel } from "@/hooks/useKindLabel";
import { useSession, useSignOut } from "@/hooks/useSession";
import { linkProps, type Route } from "@/lib/router";
import { useTheme, type Theme } from "@/lib/theme";

import { KiteMark } from "@/components/KiteMark";
import { Soon } from "@/components/Soon";
import {
  IconAttachments,
  IconBackups,
  IconComments,
  IconDashboard,
  IconMenus,
  IconPages,
  IconPlugins,
  IconPosts,
  IconSearch,
  IconSettings,
  IconSignOut,
  IconTags,
  IconTheme,
  IconUsers,
} from "@/components/icons";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  useSidebar,
} from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

type Icon = (props: { className?: string }) => ReactNode;

const kindIcons: Record<string, Icon> = {
  post: IconPosts,
  page: IconPages,
};

const appearances: { value: Theme; label: Key }[] = [
  { value: "light", label: "settings.light" },
  { value: "dark", label: "settings.dark" },
  { value: "system", label: "settings.system" },
];

interface Entry {
  key: string;
  icon: Icon;
  label: string;
  /** route is absent on an entry the design has and Kite does not yet. */
  route?: Route;
  active?: boolean;
}

export function AppSidebar({ route, onSearch }: { route: Route; onSearch: () => void }) {
  const { t } = useI18n();
  const site = useSite();
  const types = useContentTypes();
  const kindLabel = useKindLabel();

  // Until the types arrive the two built-in kinds hold the rail's shape.
  const kinds = types.data?.items.map((type) => type.kind) ?? ["post", "page"];
  const inKind = (kind: string) =>
    (route.name === "list" || route.name === "edit") && route.kind === kind;
  const soon = (key: string, icon: Icon, label: Key): Entry => ({ key, icon, label: t(label) });

  const groups: { label?: Key; entries: Entry[] }[] = [
    {
      entries: [
        {
          key: "dashboard",
          route: { name: "dashboard" },
          icon: IconDashboard,
          label: t("nav.dashboard"),
          active: route.name === "dashboard",
        },
      ],
    },
    {
      label: "nav.content",
      entries: [
        ...kinds.map((kind) => ({
          key: `kind-${kind}`,
          route: { name: "list", kind } as Route,
          icon: kindIcons[kind] ?? IconPosts,
          label: kindLabel.many(kind),
          active: inKind(kind),
        })),
        soon("comments", IconComments, "nav.comments"),
        soon("attachments", IconAttachments, "nav.attachments"),
        // Not in the design, which has no screen for it; kept last so the
        // entries it does have sit where it puts them.
        {
          key: "taxonomies",
          route: { name: "taxonomies" },
          icon: IconTags,
          label: t("nav.taxonomies"),
          active: route.name === "taxonomies",
        },
      ],
    },
    {
      label: "nav.appearance",
      entries: [
        {
          key: "theme",
          route: { name: "theme" },
          icon: IconTheme,
          label: t("nav.theme"),
          active: route.name === "theme",
        },
        soon("menus", IconMenus, "nav.menus"),
      ],
    },
    {
      label: "nav.system",
      entries: [
        soon("plugins", IconPlugins, "nav.plugins"),
        soon("users", IconUsers, "nav.users"),
        {
          key: "settings",
          route: { name: "settings" },
          icon: IconSettings,
          label: t("nav.settings"),
          active: route.name === "settings",
        },
        soon("backups", IconBackups, "nav.backups"),
      ],
    },
  ];

  return (
    // The rail is chrome: a stray drag should not highlight the navigation.
    // It folds to a strip of icons, so every button also carries its name.
    <Sidebar collapsible="icon" className="select-none">
      <SidebarHeader className="gap-0 px-3 pt-3.5 pb-0 group-data-[collapsible=icon]:px-2">
        <a
          {...linkProps({ name: "dashboard" })}
          className="flex h-7 items-center gap-[9px] rounded-md px-1.5 pt-0.5 outline-none focus-visible:ring-2 focus-visible:ring-ring/50 group-data-[collapsible=icon]:px-[3px]"
        >
          <KiteMark className="size-[26px]" />
          <span className="text-[17px] font-bold tracking-[-0.2px] group-data-[collapsible=icon]:hidden">
            Kite
          </span>
          {site.data?.version && (
            <span className="mt-0.5 max-w-24 truncate rounded-full border border-input bg-background px-[7px] py-px text-[10px] text-muted-foreground group-data-[collapsible=icon]:hidden">
              {site.data.version}
            </span>
          )}
        </a>
        <button
          type="button"
          onClick={onSearch}
          aria-label={t("nav.search")}
          className="mt-3.5 mb-1 flex h-8 items-center gap-2 rounded-[8px] border border-input bg-background pr-2 pl-2.5 text-[13px] text-muted-foreground outline-none transition-colors hover:border-border-strong focus-visible:ring-2 focus-visible:ring-ring/50 group-data-[collapsible=icon]:size-8 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:border-transparent group-data-[collapsible=icon]:p-0 dark:bg-input/30"
        >
          <IconSearch className="size-3.5" />
          <span className="flex-1 text-left group-data-[collapsible=icon]:hidden">
            {t("nav.search")}
          </span>
          <kbd className="rounded-[4px] border border-input bg-hover px-[5px] py-px font-mono text-[10px] text-subtle group-data-[collapsible=icon]:hidden">
            ⌘K
          </kbd>
        </button>
      </SidebarHeader>

      <SidebarContent className="mt-2 gap-px px-3 group-data-[collapsible=icon]:px-2">
        {groups.map((group, i) => (
          <SidebarGroup key={group.label ?? i} className="gap-px p-0">
            {group.label && <SidebarGroupLabel>{t(group.label)}</SidebarGroupLabel>}
            <SidebarGroupContent>
              <SidebarMenu>
                {group.entries.map((entry) => (
                  <NavItem key={entry.key} entry={entry} />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>

      <SidebarFooter className="mx-3 mt-2.5 mb-3 flex-row items-center gap-[9px] border-t p-0 pt-[11px] pr-0.5 pl-1 group-data-[collapsible=icon]:mx-2 group-data-[collapsible=icon]:px-0">
        <Account />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

function NavItem({ entry }: { entry: Entry }) {
  const { isMobile, setOpenMobile } = useSidebar();

  if (!entry.route) {
    return (
      <SidebarMenuItem>
        <Soon side="right">
          <SidebarMenuButton
            aria-disabled
            className="cursor-default text-subtle hover:bg-transparent hover:text-subtle aria-disabled:pointer-events-auto aria-disabled:opacity-100 active:bg-transparent active:text-subtle"
          >
            <entry.icon />
            <span>{entry.label}</span>
          </SidebarMenuButton>
        </Soon>
      </SidebarMenuItem>
    );
  }

  const link = linkProps(entry.route);
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        isActive={entry.active}
        tooltip={entry.label}
        aria-current={entry.active ? "page" : undefined}
        render={
          <a
            href={link.href}
            onClick={(event) => {
              link.onClick(event);
              if (isMobile) setOpenMobile(false);
            }}
          />
        }
      >
        <entry.icon />
        <span>{entry.label}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

/**
 * Who is signed in, and the things that are theirs rather than the site's:
 * the language the studio speaks, its appearance, and the way out.
 */
function Account() {
  const { t, locale, setLocale } = useI18n();
  const { isMobile } = useSidebar();
  const { theme, setTheme } = useTheme();
  const session = useSession();
  const signOut = useSignOut();

  const user = session.data?.required ? session.data.user : undefined;
  const name = user ?? t("session.local");

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger
          render={
            <button
              type="button"
              className="flex min-w-0 flex-1 items-center gap-[9px] rounded-md text-left outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
            />
          }
        >
          <span className="flex size-[29px] shrink-0 items-center justify-center rounded-full bg-primary text-[11.5px] font-semibold text-primary-foreground">
            {name.slice(0, 1).toUpperCase()}
          </span>
          <span className="min-w-0 flex-1 group-data-[collapsible=icon]:hidden">
            <span className="block truncate text-[12.5px] font-medium text-foreground">{name}</span>
            <span className="block truncate text-[11px] text-subtle">
              {user ? t("session.role") : t("session.localNote")}
            </span>
          </span>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          side={isMobile ? "bottom" : "right"}
          align="end"
          sideOffset={8}
          className="min-w-52"
        >
          <DropdownMenuGroup>
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <Languages />
                {t("nav.language")}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                {/* A radio group rather than a list of commands: one of
                    these is already true, and the menu should say which. */}
                <DropdownMenuRadioGroup
                  value={locale}
                  onValueChange={(next) => setLocale(next as Locale)}
                >
                  {(Object.keys(locales) as Locale[]).map((code) => (
                    <DropdownMenuRadioItem key={code} value={code}>
                      {locales[code].label}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
            <DropdownMenuSub>
              <DropdownMenuSubTrigger>
                <SunMoon />
                {t("settings.appearance")}
              </DropdownMenuSubTrigger>
              <DropdownMenuSubContent>
                <DropdownMenuRadioGroup
                  value={theme}
                  onValueChange={(next) => setTheme(next as Theme)}
                >
                  {appearances.map((item) => (
                    <DropdownMenuRadioItem key={item.value} value={item.value}>
                      {t(item.label)}
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
              </DropdownMenuSubContent>
            </DropdownMenuSub>
          </DropdownMenuGroup>
          <DropdownMenuSeparator />
          <DropdownMenuGroup>
            <DropdownMenuItem render={<a href="/" target="_blank" rel="noreferrer" />}>
              <ExternalLink />
              {t("nav.viewSite")}
            </DropdownMenuItem>
          </DropdownMenuGroup>
          {user && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem disabled={signOut.isPending} onClick={() => signOut.mutate()}>
                  <LogOut />
                  {t("session.signOut")}
                </DropdownMenuItem>
              </DropdownMenuGroup>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>

      {/* A local project has no session to end, so no way out is offered. */}
      {user && (
        <Tooltip>
          <TooltipTrigger
            render={
              <button
                type="button"
                aria-label={t("session.signOut")}
                disabled={signOut.isPending}
                onClick={() => signOut.mutate()}
                className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground outline-none transition-colors hover:bg-muted hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/50 disabled:opacity-50 group-data-[collapsible=icon]:hidden"
              />
            }
          >
            <IconSignOut className="size-[15px]" />
          </TooltipTrigger>
          <TooltipContent>{t("session.signOut")}</TooltipContent>
        </Tooltip>
      )}
    </>
  );
}
