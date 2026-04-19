import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { settingsApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Monitor,
  Sun,
  Moon,
  Check,
  Paintbrush,
  Globe,
  ShieldCheck,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useTheme } from "@/components/theme-provider";
import { localeLabels, type Locale } from "@/i18n";
import { PageHeader } from "@/components/page-header";
import { toast } from "sonner";

type Tab = "appearance" | "site" | "access";

export default function SettingsPage() {
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<Record<string, string>>({});
  const [saved, setSaved] = useState(false);
  const [activeTab, setActiveTab] = useState<Tab>("appearance");

  const { data, isLoading } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsApi.get().then((r) => r.data.data),
  });

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  const mutation = useMutation({
    mutationFn: () => settingsApi.update(form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings"] });
      setSaved(true);
      toast.success(t("settings.saved"));
      setTimeout(() => setSaved(false), 2000);
    },
    onError: () => toast.error(t("toast.error")),
  });

  const updateField = (key: string, value: string) =>
    setForm((prev) => ({ ...prev, [key]: value }));

  const toggleField = (key: string) =>
    setForm((prev) => ({
      ...prev,
      [key]: prev[key] === "true" ? "false" : "true",
    }));

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-10 w-96" />
        <Skeleton className="h-64" />
      </div>
    );
  }

  const tabs: {
    key: Tab;
    label: string;
    icon: React.ElementType;
    description: string;
  }[] = [
    {
      key: "appearance",
      label: t("settings.appearance"),
      icon: Paintbrush,
      description: t("settings.appearanceDesc"),
    },
    {
      key: "site",
      label: t("settings.site"),
      icon: Globe,
      description: t("settings.siteDesc"),
    },
    {
      key: "access",
      label: t("settings.accessControl"),
      icon: ShieldCheck,
      description: t("settings.accessControlDesc"),
    },
  ];

  const current = tabs.find((tab) => tab.key === activeTab)!;

  return (
    <div className="space-y-8">
      <PageHeader
        title={t("settings.title")}
        description={t("settings.description")}
      />

      {/* Tabs rail */}
      <div className="sticky top-0 z-10 -mx-4 overflow-x-auto border-b bg-background/80 px-4 backdrop-blur-md sm:mx-0 sm:rounded-lg sm:border sm:bg-card sm:px-1">
        <div role="tablist" className="flex gap-1">
          {tabs.map((tab) => {
            const active = tab.key === activeTab;
            return (
              <button
                key={tab.key}
                role="tab"
                aria-selected={active}
                onClick={() => setActiveTab(tab.key)}
                className={cn(
                  "group relative flex min-w-fit shrink-0 items-center gap-2 px-4 py-2.5 text-sm font-medium transition-colors sm:rounded-md",
                  active
                    ? "text-foreground"
                    : "text-muted-foreground hover:text-foreground",
                )}
              >
                <tab.icon className="size-4" />
                <span>{tab.label}</span>
                {active && (
                  <span className="absolute inset-x-3 -bottom-px h-0.5 rounded-full bg-foreground sm:inset-x-1 sm:bottom-0.5 sm:h-0.5" />
                )}
              </button>
            );
          })}
        </div>
      </div>

      <section
        role="tabpanel"
        aria-labelledby={`tab-${activeTab}`}
        className="space-y-6"
      >
        <div className="space-y-1">
          <h3 className="text-base font-semibold tracking-tight">
            {current.label}
          </h3>
          <p className="text-sm text-muted-foreground">{current.description}</p>
        </div>

        {activeTab === "appearance" && (
          <div className="grid max-w-2xl gap-5">
            <SettingRow label={t("settings.language")}>
              <div className="flex rounded-lg bg-muted p-1">
                {(Object.keys(localeLabels) as Locale[]).map((l) => (
                  <button
                    key={l}
                    className={cn(
                      "rounded-md px-4 py-1.5 text-[13px] font-medium transition-all",
                      locale === l
                        ? "bg-background text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground",
                    )}
                    onClick={() => setLocale(l)}
                  >
                    {localeLabels[l]}
                  </button>
                ))}
              </div>
            </SettingRow>

            <SettingRow label={t("settings.theme")}>
              <div className="flex rounded-lg bg-muted p-1">
                {(
                  [
                    {
                      value: "light",
                      icon: Sun,
                      label: t("settings.themeLight"),
                    },
                    {
                      value: "dark",
                      icon: Moon,
                      label: t("settings.themeDark"),
                    },
                    {
                      value: "system",
                      icon: Monitor,
                      label: t("settings.themeSystem"),
                    },
                  ] as const
                ).map((item) => (
                  <button
                    key={item.value}
                    className={cn(
                      "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-[13px] font-medium transition-all",
                      theme === item.value
                        ? "bg-background text-foreground shadow-sm"
                        : "text-muted-foreground hover:text-foreground",
                    )}
                    onClick={() => setTheme(item.value)}
                  >
                    <item.icon className="size-3.5" />
                    {item.label}
                  </button>
                ))}
              </div>
            </SettingRow>
          </div>
        )}

        {activeTab === "site" && (
          <div className="grid max-w-xl gap-4">
            <div className="grid gap-2">
              <Label htmlFor="site_name">{t("settings.siteName")}</Label>
              <Input
                id="site_name"
                value={form.site_name ?? ""}
                onChange={(e) => updateField("site_name", e.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="site_url">{t("settings.siteUrl")}</Label>
              <Input
                id="site_url"
                value={form.site_url ?? ""}
                onChange={(e) => updateField("site_url", e.target.value)}
                placeholder={t("settings.siteUrlPlaceholder")}
              />
            </div>
          </div>
        )}

        {activeTab === "access" && (
          <div className="grid max-w-2xl gap-5">
            <SettingToggle
              label={t("settings.allowRegistration")}
              description={t("settings.allowRegistrationDesc")}
              checked={form.allow_registration === "true"}
              onChange={() => toggleField("allow_registration")}
            />
            <SettingToggle
              label={t("settings.allowGuestUpload")}
              description={t("settings.allowGuestUploadDesc")}
              checked={form.allow_guest_upload === "true"}
              onChange={() => toggleField("allow_guest_upload")}
            />
            <SettingToggle
              label={t("settings.allowPublicGallery")}
              description={t("settings.allowPublicGalleryDesc")}
              checked={form.allow_public_gallery === "true"}
              onChange={() => toggleField("allow_public_gallery")}
            />
          </div>
        )}
      </section>

      <div className="flex">
        <Button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
          {saved ? (
            <>
              <Check className="size-4" />
              {t("settings.saved")}
            </>
          ) : mutation.isPending ? (
            t("settings.saving")
          ) : (
            t("settings.saveSettings")
          )}
        </Button>
      </div>
    </div>
  );
}

function SettingRow({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <span className="text-sm font-medium">{label}</span>
      {children}
    </div>
  );
}

function SettingToggle({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="min-w-0 space-y-0.5">
        <p className="text-sm font-medium">{label}</p>
        <p className="text-xs text-muted-foreground">{description}</p>
      </div>
      <Switch checked={checked} onCheckedChange={onChange} />
    </div>
  );
}
