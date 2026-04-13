import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { settingsApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Globe,
  LinkIcon,
  UserPlus,
  Languages,
  Monitor,
  Sun,
  Moon,
  Upload,
  Eye,
  Check,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useTheme } from "@/components/theme-provider";
import { localeLabels, type Locale } from "@/i18n";

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
      setTimeout(() => setSaved(false), 2000);
    },
  });

  const updateField = (key: string, value: string) =>
    setForm((prev) => ({ ...prev, [key]: value }));

  const toggleField = (key: string) =>
    setForm((prev) => ({ ...prev, [key]: prev[key] === "true" ? "false" : "true" }));

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-48" />
        <Skeleton className="h-32" />
      </div>
    );
  }

  return (
    <div className="space-y-6 pb-12">
      <div className="mb-8 mt-2">
        <h1 className="text-[22px] font-bold tracking-tight text-foreground">{t("settings.title")}</h1>
        <p className="mt-2 text-[14px] text-muted-foreground">{t("settings.description")}</p>
      </div>

      {/* Appearance */}
      <Card>
        <CardHeader className="pt-6 pb-4">
          <CardTitle className="text-base font-semibold">{t("settings.appearance")}</CardTitle>
          <p className="mt-1 text-[13px] text-muted-foreground">{t("settings.appearanceDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-5 pb-6">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Languages className="size-4 text-muted-foreground" />
              <span className="text-[14px] font-medium">{t("settings.language")}</span>
            </div>
            <div className="flex bg-muted p-1 rounded-lg">
              {(Object.keys(localeLabels) as Locale[]).map((l) => (
                <button
                  key={l}
                  className={cn(
                    "px-4 py-1.5 rounded-md text-[13px] font-medium transition-all shadow-none",
                    locale === l ? "bg-background shadow-sm text-foreground" : "text-muted-foreground hover:text-foreground"
                  )}
                  onClick={() => setLocale(l)}
                >
                  {localeLabels[l]}
                </button>
              ))}
            </div>
          </div>

          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Monitor className="size-4 text-muted-foreground" />
              <span className="text-[14px] font-medium">{t("settings.theme")}</span>
            </div>
            <div className="flex bg-muted p-1 rounded-lg">
              {([
                { value: "light", icon: Sun, label: t("settings.themeLight") },
                { value: "dark", icon: Moon, label: t("settings.themeDark") },
                { value: "system", icon: Monitor, label: t("settings.themeSystem") },
              ] as const).map((item) => (
                <button
                  key={item.value}
                  className={cn(
                    "flex items-center gap-2 px-4 py-1.5 rounded-md text-[13px] font-medium transition-all shadow-none",
                    theme === item.value ? "bg-background shadow-sm text-foreground" : "text-muted-foreground hover:text-foreground"
                  )}
                  onClick={() => setTheme(item.value)}
                >
                  <item.icon className="size-3.5" />
                  {item.label}
                </button>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Site config */}
      <Card>
        <CardHeader className="pt-6 pb-4">
          <CardTitle className="text-base font-semibold">{t("settings.site")}</CardTitle>
          <p className="mt-1 text-[13px] text-muted-foreground">{t("settings.siteDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-5 pb-6">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Globe className="size-4 text-muted-foreground" />
              <span className="text-[14px] font-medium">{t("settings.siteName")}</span>
            </div>
            <Input
              className="w-full sm:w-[320px]"
              value={form.site_name ?? ""}
              onChange={(e) => updateField("site_name", e.target.value)}
            />
          </div>

          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <LinkIcon className="size-4 text-muted-foreground" />
              <span className="text-[14px] font-medium">{t("settings.siteUrl")}</span>
            </div>
            <Input
              className="w-full sm:w-[320px]"
              value={form.site_url ?? ""}
              onChange={(e) => updateField("site_url", e.target.value)}
              placeholder={t("settings.siteUrlPlaceholder")}
            />
          </div>
        </CardContent>
      </Card>

      {/* Access control */}
      <Card>
        <CardHeader className="pt-6 pb-4">
          <CardTitle className="text-base font-semibold">{t("settings.accessControl")}</CardTitle>
          <p className="mt-1 text-[13px] text-muted-foreground">{t("settings.accessControlDesc")}</p>
        </CardHeader>
        <CardContent className="space-y-5 pb-6">
          {/* Registration */}
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <UserPlus className="size-4 text-muted-foreground" />
              <div className="flex flex-col">
                <span className="text-[14px] font-medium">{t("settings.allowRegistration")}</span>
                <span className="text-[12px] text-muted-foreground hidden sm:block">{t("settings.allowRegistrationDesc")}</span>
              </div>
            </div>
            <Button
              variant={form.allow_registration === "true" ? "default" : "secondary"}
              size="sm"
              className={cn(
                "h-8 px-4 text-[13px]",
                form.allow_registration !== "true" && "bg-muted hover:bg-muted/80 text-foreground"
              )}
              onClick={() => toggleField("allow_registration")}
            >
              {form.allow_registration === "true" ? t("common.enabled") : t("common.disabled")}
            </Button>
          </div>

          {/* Guest upload */}
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Upload className="size-4 text-muted-foreground" />
              <div className="flex flex-col">
                <span className="text-[14px] font-medium">{t("settings.allowGuestUpload")}</span>
                <span className="text-[12px] text-muted-foreground hidden sm:block">{t("settings.allowGuestUploadDesc")}</span>
              </div>
            </div>
            <Button
              variant={form.allow_guest_upload === "true" ? "default" : "secondary"}
              size="sm"
              className={cn(
                "h-8 px-4 text-[13px]",
                form.allow_guest_upload !== "true" && "bg-muted hover:bg-muted/80 text-foreground"
              )}
              onClick={() => toggleField("allow_guest_upload")}
            >
              {form.allow_guest_upload === "true" ? t("common.enabled") : t("common.disabled")}
            </Button>
          </div>

          {/* Public gallery */}
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Eye className="size-4 text-muted-foreground" />
              <div className="flex flex-col">
                <span className="text-[14px] font-medium">{t("settings.allowPublicGallery")}</span>
                <span className="text-[12px] text-muted-foreground hidden sm:block">{t("settings.allowPublicGalleryDesc")}</span>
              </div>
            </div>
            <Button
              variant={form.allow_public_gallery === "true" ? "default" : "secondary"}
              size="sm"
              className={cn(
                "h-8 px-4 text-[13px]",
                form.allow_public_gallery !== "true" && "bg-muted hover:bg-muted/80 text-foreground"
              )}
              onClick={() => toggleField("allow_public_gallery")}
            >
              {form.allow_public_gallery === "true" ? t("common.enabled") : t("common.disabled")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <div className="flex justify-start mt-8">
        <Button
          className="px-6"
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
