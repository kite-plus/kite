import {
  ChevronsUpDown,
  ExternalLink,
  File,
  FileText,
  Languages,
  LayoutDashboard,
  LayoutTemplate,
  LogOut,
  PanelsTopLeft,
  Search,
  Settings2,
  SunMoon,
  Tags,
  type LucideIcon,
} from "lucide-react";

import { locales, useI18n, type Key, type Locale } from "@/i18n";
import { useContentTypes, useSite } from "@/hooks/useContents";
import { useKindLabel } from "@/hooks/useKindLabel";
import { useSession, useSignOut } from "@/hooks/useSession";
import { linkProps, type Route } from "@/lib/router";
import { useTheme, type Theme } from "@/lib/theme";

import { KiteMark } from "@/components/KiteMark";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
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
import { Kbd } from "@/components/ui/kbd";
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

const kindIcons: Record<string, LucideIcon> = {
  post: FileText,
  page: PanelsTopLeft,
};

const appearances: { value: Theme; label: Key }[] = [
  { value: "light", label: "settings.light" },
  { value: "dark", label: "settings.dark" },
  { value: "system", label: "settings.system" },
];

interface Entry {
  route: Route;
  icon: LucideIcon;
  label: string;
  active: boolean;
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

  const groups: { label?: Key; entries: Entry[] }[] = [
    {
      entries: [
        {
          route: { name: "dashboard" },
          icon: LayoutDashboard,
          label: t("nav.dashboard"),
          active: route.name === "dashboard",
        },
      ],
    },
    {
      label: "nav.content",
      entries: [
        ...kinds.map((kind) => ({
          route: { name: "list", kind } as Route,
          icon: kindIcons[kind] ?? File,
          label: kindLabel.many(kind),
          active: inKind(kind),
        })),
        {
          route: { name: "taxonomies" },
          icon: Tags,
          label: t("nav.taxonomies"),
          active: route.name === "taxonomies",
        },
      ],
    },
    {
      label: "nav.appearance",
      entries: [
        {
          route: { name: "theme" },
          icon: LayoutTemplate,
          label: t("nav.theme"),
          active: route.name === "theme",
        },
      ],
    },
    {
      label: "nav.system",
      entries: [
        {
          route: { name: "settings" },
          icon: Settings2,
          label: t("nav.settings"),
          active: route.name === "settings",
        },
      ],
    },
  ];

  return (
    // The rail is chrome: a stray drag should not highlight the navigation.
    // It folds to a strip of icons, so every button also carries its name.
    <Sidebar collapsible="icon" className="select-none">
      <SidebarHeader className="gap-1 p-2">
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip="Kite"
              className="h-10 gap-2.5 text-foreground group-data-[collapsible=icon]:size-8! group-data-[collapsible=icon]:p-1!"
              render={<a {...linkProps({ name: "dashboard" })} />}
            >
              <KiteMark className="size-6! shrink-0" />
              <span className="flex min-w-0 items-center gap-2">
                <span className="text-lg font-bold tracking-tight">Kite</span>
                {site.data?.version && (
                  <Badge
                    variant="outline"
                    className="mt-0.5 max-w-24 font-normal text-muted-foreground"
                  >
                    <span className="truncate">{site.data.version}</span>
                  </Badge>
                )}
              </span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip={t("nav.search")}
              onClick={onSearch}
              className="border border-border bg-background text-muted-foreground shadow-xs hover:bg-muted group-data-[collapsible=icon]:border-transparent group-data-[collapsible=icon]:bg-transparent group-data-[collapsible=icon]:shadow-none dark:border-input dark:bg-input/30"
            >
              <Search />
              <span>{t("nav.search")}</span>
              <Kbd className="ml-auto group-data-[collapsible=icon]:hidden">⌘K</Kbd>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        {groups.map((group, i) => (
          <SidebarGroup key={group.label ?? i} className="py-1">
            {group.label && <SidebarGroupLabel>{t(group.label)}</SidebarGroupLabel>}
            <SidebarGroupContent>
              <SidebarMenu>
                {group.entries.map((entry) => (
                  <NavItem key={entry.label} entry={entry} />
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>

      <SidebarFooter>
        <Account />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  );
}

function NavItem({ entry }: { entry: Entry }) {
  const { isMobile, setOpenMobile } = useSidebar();
  const link = linkProps(entry.route);

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        isActive={entry.active}
        tooltip={entry.label}
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
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                size="lg"
                tooltip={name}
                className="data-open:bg-sidebar-accent data-open:text-sidebar-accent-foreground"
              />
            }
          >
            <Avatar className="size-8 rounded-lg">
              <AvatarFallback className="rounded-lg bg-primary text-xs font-semibold text-primary-foreground">
                {name.slice(0, 1).toUpperCase()}
              </AvatarFallback>
            </Avatar>
            <span className="grid min-w-0 flex-1 text-left leading-tight">
              <span className="truncate text-sm font-medium">{name}</span>
              <span className="truncate text-xs text-muted-foreground">
                {user ? t("session.role") : t("session.localNote")}
              </span>
            </span>
            <ChevronsUpDown className="ml-auto text-muted-foreground" />
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
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
