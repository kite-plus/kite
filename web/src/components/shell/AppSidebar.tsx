import {
  File,
  FileText,
  LayoutDashboard,
  LayoutTemplate,
  LogOut,
  PanelsTopLeft,
  Search,
  Settings2,
  Tags,
  type LucideIcon,
} from "lucide-react";

import { useI18n, type Key } from "@/i18n";
import { useContentTypes, useSite } from "@/hooks/useContents";
import { useKindLabel } from "@/hooks/useKindLabel";
import { useSession, useSignOut } from "@/hooks/useSession";
import { linkProps, type Route } from "@/lib/router";

import { KiteMark } from "@/components/KiteMark";
import { LanguagePicker } from "@/components/LanguagePicker";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
  SidebarSeparator,
  useSidebar,
} from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

const kindIcons: Record<string, LucideIcon> = {
  post: FileText,
  page: PanelsTopLeft,
};

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
    <Sidebar className="select-none">
      <SidebarHeader className="gap-3 p-3">
        <a
          {...linkProps({ name: "dashboard" })}
          className="flex items-center gap-2 px-1.5 pt-0.5 text-foreground"
        >
          <KiteMark className="size-6.5" />
          <span className="text-lg font-bold tracking-tight">Kite</span>
          {site.data?.version && (
            <Badge variant="outline" className="mt-0.5 max-w-24 font-normal text-muted-foreground">
              <span className="truncate">{site.data.version}</span>
            </Badge>
          )}
        </a>

        <Button
          variant="outline"
          className="w-full justify-start font-normal text-muted-foreground"
          onClick={onSearch}
        >
          <Search data-icon="inline-start" />
          {t("nav.search")}
          <Kbd className="ml-auto">⌘K</Kbd>
        </Button>
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
        <SidebarSeparator className="mx-0" />
        <Account />
      </SidebarFooter>
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

/** Who is signed in. A server with no account says so instead. */
function Account() {
  const { t } = useI18n();
  const session = useSession();
  const signOut = useSignOut();

  const user = session.data?.required ? session.data.user : undefined;
  const name = user ?? t("session.local");

  return (
    <div className="flex items-center gap-2 px-1 pb-1">
      <Avatar>
        <AvatarFallback className="bg-primary text-xs font-semibold text-primary-foreground">
          {name.slice(0, 1).toUpperCase()}
        </AvatarFallback>
      </Avatar>
      <div className="min-w-0 flex-1 leading-tight">
        <div className="truncate text-sm font-medium text-foreground">{name}</div>
        <div className="truncate text-xs text-muted-foreground">
          {user ? t("session.role") : t("session.localNote")}
        </div>
      </div>
      <LanguagePicker />
      {user && (
        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("session.signOut")}
                disabled={signOut.isPending}
                onClick={() => signOut.mutate()}
              />
            }
          >
            <LogOut />
          </TooltipTrigger>
          <TooltipContent>{t("session.signOut")}</TooltipContent>
        </Tooltip>
      )}
    </div>
  );
}
