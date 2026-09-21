import { useDeferredValue, useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  ExternalLink,
  FileText,
  LayoutDashboard,
  LayoutTemplate,
  Plus,
  Settings2,
  Tags,
  type LucideIcon,
} from "lucide-react";

import { api, unwrap } from "@/api/client";
import { useI18n } from "@/i18n";
import { useContentTypes } from "@/hooks/useContents";
import { useKindLabel } from "@/hooks/useKindLabel";
import { navigate, type Route } from "@/lib/router";

import { StatusDot } from "@/components/StatusDot";
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";

interface Action {
  label: string;
  icon: LucideIcon;
  run: () => void;
}

/** actionValues names each row of a group, uniquely across the palette. */
function actionValues(group: { heading: string; entries: Action[] }): string[] {
  return group.entries.map((action) => `${group.heading}:${action.label}`);
}

/** The palette behind the rail's search box: find content, or go somewhere. */
export function CommandMenu({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const { t } = useI18n();
  const types = useContentTypes();
  const kindLabel = useKindLabel();

  const [query, setQuery] = useState("");
  const deferred = useDeferredValue(query.trim());

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() === "k" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        onOpenChange(!open);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onOpenChange]);

  const found = useQuery({
    queryKey: ["contents", "palette", deferred],
    enabled: open && deferred.length > 0,
    queryFn: async () =>
      unwrap(await api.GET("/contents", { params: { query: { q: deferred, limit: 8 } } })),
  });

  const go = (route: Route) => () => navigate(route);
  const kinds = types.data?.items.map((type) => type.kind) ?? [];

  const places: Action[] = [
    { label: t("nav.dashboard"), icon: LayoutDashboard, run: go({ name: "dashboard" }) },
    ...kinds.map((kind) => ({
      label: kindLabel.many(kind),
      icon: FileText,
      run: go({ name: "list", kind }),
    })),
    { label: t("nav.taxonomies"), icon: Tags, run: go({ name: "taxonomies" }) },
    { label: t("nav.theme"), icon: LayoutTemplate, run: go({ name: "theme" }) },
    { label: t("nav.settings"), icon: Settings2, run: go({ name: "settings" }) },
  ];
  const actions: Action[] = [
    ...kinds.map((kind) => ({
      label: t("list.newKind", { kind: kindLabel.one(kind) }),
      icon: Plus,
      run: go({ name: "edit", kind, id: null }),
    })),
    {
      label: t("nav.viewSite"),
      icon: ExternalLink,
      run: () => void window.open("/", "_blank", "noreferrer"),
    },
  ];

  // The server answers the content search, so cmdk's own filter is off and
  // the fixed entries are matched here.
  const matches = (action: Action) =>
    action.label.toLowerCase().includes(deferred.toLowerCase());

  const groups = [
    { heading: t("palette.goTo"), entries: places.filter(matches) },
    { heading: t("palette.actions"), entries: actions.filter(matches) },
  ];
  const results = found.data?.items ?? [];

  // cmdk highlights the first row only for rows it filtered itself. These
  // arrive late, so the highlight is moved to the top whenever they change.
  const first = results[0]?.id ?? groups.flatMap(actionValues)[0] ?? "";
  const [active, setActive] = useState("");
  useEffect(() => setActive(first), [first]);

  const pick = (run: () => void) => () => {
    onOpenChange(false);
    setQuery("");
    run();
  };

  return (
    <CommandDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t("palette.title")}
      description={t("palette.description")}
    >
      <Command shouldFilter={false} value={active} onValueChange={setActive}>
        <CommandInput
          value={query}
          onValueChange={setQuery}
          placeholder={t("palette.placeholder")}
        />
        <CommandList>
          <CommandEmpty>{t("palette.empty")}</CommandEmpty>

          {results.length > 0 && (
            <CommandGroup heading={t("nav.content")}>
              {results.map((item) => (
                <CommandItem
                  key={item.id}
                  value={item.id}
                  onSelect={pick(go({ name: "edit", kind: item.kind, id: item.id }))}
                >
                  <FileText />
                  <span className="truncate">{item.title || item.slug}</span>
                  <StatusDot status={item.status} className="ml-auto shrink-0 text-xs" />
                </CommandItem>
              ))}
            </CommandGroup>
          )}

          {groups.map(
            (group) =>
              group.entries.length > 0 && (
                <CommandGroup key={group.heading} heading={group.heading}>
                  {group.entries.map((action, i) => (
                    <CommandItem
                      key={action.label}
                      value={actionValues(group)[i]}
                      onSelect={pick(action.run)}
                    >
                      <action.icon />
                      {action.label}
                    </CommandItem>
                  ))}
                </CommandGroup>
              ),
          )}
        </CommandList>
      </Command>
    </CommandDialog>
  );
}
