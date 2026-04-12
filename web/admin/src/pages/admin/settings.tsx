import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { settingsApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";

export default function SettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [form, setForm] = useState<Record<string, string>>({});

  const { data, isLoading } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsApi.get().then((r) => r.data.data),
  });

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  const mutation = useMutation({
    mutationFn: () => settingsApi.update(form),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["settings"] }),
  });

  const updateField = (key: string, value: string) =>
    setForm((prev) => ({ ...prev, [key]: value }));

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 rounded-lg" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("settings.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("settings.description")}</p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("settings.site")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label>{t("settings.siteName")}</Label>
              <Input
                value={form.site_name ?? ""}
                onChange={(e) => updateField("site_name", e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("settings.siteUrl")}</Label>
              <Input
                value={form.site_url ?? ""}
                onChange={(e) => updateField("site_url", e.target.value)}
                placeholder={t("settings.siteUrlPlaceholder")}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("settings.registration")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium">{t("settings.allowRegistration")}</p>
              <p className="text-xs text-muted-foreground">
                {t("settings.allowRegistrationDesc")}
              </p>
            </div>
            <Button
              variant={
                form.allow_registration === "true" ? "default" : "outline"
              }
              size="sm"
              onClick={() =>
                updateField(
                  "allow_registration",
                  form.allow_registration === "true" ? "false" : "true"
                )
              }
            >
              {form.allow_registration === "true" ? t("common.enabled") : t("common.disabled")}
            </Button>
          </div>
        </CardContent>
      </Card>

      <Separator />

      <div className="flex justify-end">
        <Button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
          {mutation.isPending ? t("settings.saving") : t("settings.saveSettings")}
        </Button>
      </div>
    </div>
  );
}
