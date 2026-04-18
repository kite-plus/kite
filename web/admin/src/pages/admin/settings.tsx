import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { settingsApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Monitor,
  Sun,
  Moon,
  Check,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useTheme } from "@/components/theme-provider";
import { localeLabels, type Locale } from "@/i18n";
import { PageHeader, Section } from "@/components/page-header";
import { toast } from "sonner";

export default function SettingsPage() {
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<Record<string, string>>({});
  const [saved, setSaved] = useState(false);

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
    setForm((prev) => ({ ...prev, [key]: prev[key] === "true" ? "false" : "true" }));

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-40" />
        <Skeleton className="h-40" />
      </div>
    );
  }

  return (
    <div className="space-y-10">
      <PageHeader
        title={t("settings.title")}
        description={t("settings.description")}
      />

      {/* Appearance */}
      <Section
        title={t("settings.appearance")}
        description={t("settings.appearanceDesc")}
      >
        <div className="grid gap-5 max-w-2xl">
          <SettingRow label={t("settings.language")}>
            <div className="flex rounded-lg bg-muted p-1">
              {(Object.keys(localeLabels) as Locale[]).map((l) => (
                <button
                  key={l}
                  className={cn(
                    "rounded-md px-4 py-1.5 text-[13px] font-medium transition-all",
                    locale === l
                      ? "bg-background text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
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
              {([
                { value: "light", icon: Sun, label: t("settings.themeLight") },
                { value: "dark", icon: Moon, label: t("settings.themeDark") },
                { value: "system", icon: Monitor, label: t("settings.themeSystem") },
              ] as const).map((item) => (
                <button
                  key={item.value}
                  className={cn(
                    "flex items-center gap-1.5 rounded-md px-3 py-1.5 text-[13px] font-medium transition-all",
                    theme === item.value
                      ? "bg-background text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
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
      </Section>

      <Separator />

      {/* Site config */}
      <Section
        title={t("settings.site")}
        description={t("settings.siteDesc")}
      >
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
      </Section>

      <Separator />

      {/* Access control */}
      <Section
        title={t("settings.accessControl")}
        description={t("settings.accessControlDesc")}
      >
        <div className="grid gap-5 max-w-2xl">
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
      </Section>

      <div className="flex">
        <Button
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending}
        >
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
