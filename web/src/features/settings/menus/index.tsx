import { useDeferredValue, useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { FilePlus2 } from "lucide-react";

import { api, unwrap, type Field, type Settings } from "@/api/client";
import { useI18n } from "@/i18n";
import { useSite, useWritable } from "@/hooks/useContents";
import { siteHome } from "@/lib/links";
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Skeleton } from "@/components/ui/skeleton";
import { RepeatField } from "@/components/RepeatField";
import { ContentSection } from "../components/content-section";
import { Group } from "../components/form-group";
import { FormProblem, LeaveGuard, SaveButton, useSettingsForm } from "../use-settings-form";

type Link = { name?: string; url?: string; children?: Link[] };

// Each menu the theme draws is a value of the form, and so is each menu the
// site has written for another theme, so that saving one keeps the rest.
const read = (settings: Settings) => {
  const names = [...(settings.theme.menus ?? []).map((menu) => menu.name), ...Object.keys(settings.menus ?? {})];
  return Object.fromEntries(names.map((name) => [name, settings.menus?.[name] ?? []]));
};

export function MenuSettings() {
  const { t } = useI18n();
  const form = useSettingsForm(read, (key) => `menus.${key}`);
  const writable = useWritable();
  const values = form.values as Record<string, Link[] | undefined>;
  const set = (name: string, links: unknown) => form.change({ ...values, [name]: links });

  const drawn = form.settings?.theme.menus ?? [];
  const kept = Object.keys(values).filter((name) => !drawn.some((menu) => menu.name === name));

  return (
    <ContentSection title={t("menus.title")} desc={t("menus.note")} writes>
      <div>
        <FormProblem form={form} />
        {!form.loaded ? (
          <Skeleton className="h-72 w-full" />
        ) : (
          <fieldset disabled={!writable} className="grid gap-10">
            {drawn.length === 0 && (
              <div className="rounded-lg border border-dashed px-4 py-6 text-center">
                <p className="text-sm font-medium">{t("menus.none")}</p>
                <p className="mt-1 text-sm text-muted-foreground">{t("menus.noneNote")}</p>
              </div>
            )}
            {drawn.map((menu) => (
              <Menu
                key={menu.name}
                name={menu.name}
                title={menu.label || menu.name}
                note={menu.description}
                // Links written deeper than the theme draws are still shown,
                // so that none is hidden from the one editing them.
                depth={Math.max(menu.depth, depthOf(values[menu.name] ?? []))}
                links={values[menu.name] ?? []}
                onChange={(links) => set(menu.name, links)}
              />
            ))}
            {kept.map((name) => (
              <Menu
                key={name}
                name={name}
                title={name}
                note={t("menus.unusedNote")}
                depth={depthOf(values[name] ?? [])}
                links={values[name] ?? []}
                onChange={(links) => set(name, links)}
              />
            ))}
            {(drawn.length > 0 || kept.length > 0) && (
              <div>
                <SaveButton form={form} />
              </div>
            )}
          </fieldset>
        )}
        <LeaveGuard form={form} />
      </div>
    </ContentSection>
  );
}

function Menu({
  name,
  title,
  note,
  depth,
  links,
  onChange,
}: {
  name: string;
  title: string;
  note?: string;
  depth: number;
  links: Link[];
  onChange: (links: unknown) => void;
}) {
  const { t } = useI18n();
  const field: Field = { key: name, type: "repeat", label: title, fields: linkFields(t, depth) };
  return (
    <Group id={`menu-${name}`} title={title} note={note}>
      <div className="grid gap-2">
        <RepeatField id={`menu-${name}-links`} field={field} value={links} onChange={onChange} />
        <PickContent onPick={(link) => onChange([...links, link])} />
      </div>
    </Group>
  );
}

/**
 * linkFields are the fields of a link at a depth of the menu. A link that can
 * hold none under it needs an address; one that can may only head the rest.
 */
function linkFields(t: ReturnType<typeof useI18n>["t"], depth: number): Field[] {
  const leaf = depth <= 1;
  const fields: Field[] = [
    { key: "name", type: "string", label: t("menus.name"), required: true },
    { key: "url", type: "url", label: t("menus.url"), placeholder: "/about/", required: leaf },
  ];
  if (!leaf) fields.push({ key: "children", type: "repeat", label: t("menus.children"), fields: linkFields(t, depth - 1) });
  return fields;
}

function depthOf(links: Link[]): number {
  return Math.max(1, ...links.map((link) => 1 + (link.children?.length ? depthOf(link.children) : 0)));
}

/** PickContent adds a published page or post to a menu, by its title and address. */
function PickContent({ onPick }: { onPick: (link: Link) => void }) {
  const { t } = useI18n();
  const site = useSite();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const deferred = useDeferredValue(query.trim());
  const found = useQuery({
    queryKey: ["contents", "menu-pick", deferred],
    enabled: open,
    queryFn: async () =>
      unwrap(
        await api.GET("/contents", {
          params: { query: { q: deferred || undefined, status: ["published"], limit: 8 } },
        }),
      ),
  });
  const home = siteHome(site.data);
  const items = found.data?.items ?? [];

  // cmdk highlights the first row only among rows it filtered itself; these
  // arrive late, so Enter would pick nothing without this.
  const first = items[0]?.id ?? "";
  const [active, setActive] = useState("");
  useEffect(() => setActive(first), [first]);

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next);
        if (!next) setQuery("");
      }}
    >
      <PopoverTrigger asChild>
        <Button type="button" variant="ghost" size="sm" className="justify-self-start text-muted-foreground">
          <FilePlus2 />
          {t("menus.fromContent")}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80 p-0" align="start">
        <Command shouldFilter={false} value={active} onValueChange={setActive}>
          <CommandInput placeholder={t("menus.search")} value={query} onValueChange={setQuery} />
          <CommandList>
            {found.isSuccess && <CommandEmpty>{t("menus.nothingFound")}</CommandEmpty>}
            {items.map((item) => {
              const url = withinSite(item.url, home);
              return (
                <CommandItem
                  key={item.id}
                  value={item.id}
                  onSelect={() => {
                    onPick({ name: item.title, url });
                    setOpen(false);
                    setQuery("");
                  }}
                >
                  <span className="truncate">{item.title}</span>
                  <span className="ms-auto max-w-32 truncate font-mono text-xs text-muted-foreground">{url}</span>
                </CommandItem>
              );
            })}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

/**
 * withinSite is a page's address as a menu writes it: from the site's own
 * root, without the path the site is published under, which the theme puts
 * back.
 */
function withinSite(url: string, home: string): string {
  const root = home.replace(/\/$/, "");
  return root && url.startsWith(root + "/") ? url.slice(root.length) : url;
}
