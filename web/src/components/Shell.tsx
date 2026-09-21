import type { ReactNode } from "react";
import { cn } from "cn";
import { FileText, Files, Languages, Settings2, Tags, Search } from "lucide-react";

import { locales, useI18n, type Locale } from "@/i18n";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export type Section = "post" | "page" | "taxonomies" | "settings";

interface Props {
  site?: { title?: string; store?: string; runtime?: string };
  counts?: Record<string, number>;
  section: Section;
  onSection: (section: Section) => void;
  onSearch?: () => void;
  footer?: ReactNode;
  children: ReactNode;
}

/**
 * The frame every screen sits in: a fixed rail on the left, the work on the
 * right.
 *
 * Navigation stays in one place rather than living inside the page, because
 * an author moves between writing and settings constantly, and a tool that
 * makes them find their way back each time feels slower than it is.
 */
export function Shell({
  site,
  counts,
  section,
  onSection,
  onSearch,
  footer,
  children,
}: Props) {
  const { t } = useI18n();

  return (
    <div className="flex min-h-svh bg-background">
      <aside className="fixed inset-y-0 left-0 hidden w-60 flex-col border-r bg-sidebar md:flex">
        <div className="flex h-14 items-center gap-2 px-4">
          <KiteMark />
          <div className="min-w-0 flex-1">
            <div className="truncate text-sm font-semibold">
              {site?.title ?? "Kite Studio"}
            </div>
          </div>
        </div>

        <div className="px-3">
          <button
            type="button"
            onClick={onSearch}
            className="flex w-full items-center gap-2 rounded-md bg-muted px-2.5 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            <Search className="size-3.5" />
            <span className="flex-1 text-left">{t("nav.search")}</span>
          </button>
        </div>

        <nav className="mt-4 flex-1 space-y-6 px-3">
          <Group label={t("nav.content")}>
            <Item
              icon={<FileText className="size-4" />}
              label={t("nav.posts")}
              count={counts?.post}
              active={section === "post"}
              onClick={() => onSection("post")}
            />
            <Item
              icon={<Files className="size-4" />}
              label={t("nav.pages")}
              count={counts?.page}
              active={section === "page"}
              onClick={() => onSection("page")}
            />
            <Item
              icon={<Tags className="size-4" />}
              label={t("nav.taxonomies")}
              active={section === "taxonomies"}
              onClick={() => onSection("taxonomies")}
            />
          </Group>

          <Group label={t("nav.site")}>
            <Item
              icon={<Settings2 className="size-4" />}
              label={t("nav.settings")}
              active={section === "settings"}
              onClick={() => onSection("settings")}
            />
          </Group>
        </nav>

        <div className="p-3">
          {footer}
          <Separator className="my-3" />
          <div className="flex items-center justify-between gap-2 px-2">
            <p className="min-w-0 truncate text-xs text-muted-foreground">
              {site
                ? t("shell.runtime", { store: site.store ?? "", runtime: site.runtime ?? "" })
                : t("shell.connecting")}
            </p>
            <LanguagePicker />
          </div>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col md:ml-60">{children}</div>
    </div>
  );
}

/**
 * The admin speaks the operator's language, which is not the site's: a
 * Chinese author may well publish in English, and the reverse.
 */
function LanguagePicker() {
  const { locale, setLocale, t } = useI18n();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button variant="ghost" size="icon" className="size-6" title={t("nav.language")}>
            <Languages className="size-3.5" />
          </Button>
        }
      />
      <DropdownMenuContent align="end">
        {(Object.keys(locales) as Locale[]).map((code) => (
          <DropdownMenuItem
            key={code}
            onClick={() => setLocale(code)}
            className={cn(code === locale && "font-medium text-primary")}
          >
            {locales[code].label}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

/** Page is the header and body of one screen inside the shell. */
export function Page({
  title,
  description,
  actions,
  children,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <>
      <header className="sticky top-0 z-10 flex min-h-14 flex-wrap items-center gap-3 border-b bg-background/80 px-4 py-2 backdrop-blur sm:px-6">
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-base font-semibold">{title}</h1>
          {description && (
            <p className="truncate text-xs text-muted-foreground">{description}</p>
          )}
        </div>
        {actions}
      </header>
      <div className="min-w-0 flex-1 p-4 sm:p-6">{children}</div>
    </>
  );
}

function Group({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <p className="mb-1 px-2 text-xs font-medium tracking-wide text-muted-foreground/70 uppercase">
        {label}
      </p>
      <div className="space-y-0.5">{children}</div>
    </div>
  );
}

function Item({
  icon,
  label,
  count,
  active,
  onClick,
}: {
  icon: ReactNode;
  label: string;
  count?: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      variant="ghost"
      onClick={onClick}
      className={cn(
        "h-8 w-full justify-start gap-2 px-2 font-normal",
        active && "bg-sidebar-accent font-medium text-sidebar-accent-foreground",
      )}
    >
      <span className={cn("text-muted-foreground", active && "text-primary")}>
        {icon}
      </span>
      <span className="flex-1 text-left">{label}</span>
      {count !== undefined && (
        <Badge variant="secondary" className="h-5 px-1.5 text-[11px] font-normal">
          {count}
        </Badge>
      )}
    </Button>
  );
}

/** The logo, drawn rather than fetched so the rail paints with the shell. */
function KiteMark() {
  return (
    <svg viewBox="0 0 24 24" className="size-6 shrink-0" aria-hidden>
      <path d="M12 2 4 10l8 12 8-12z" className="fill-primary" />
      <path d="M12 2v20" className="stroke-primary-foreground/40" strokeWidth="1" />
    </svg>
  );
}
