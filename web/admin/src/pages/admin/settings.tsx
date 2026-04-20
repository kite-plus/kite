import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Settings as SettingsIcon,
  Shield,
  HardDrive,
  Mail,
  Check,
  Copy,
  Loader2,
} from "lucide-react";
import { toast } from "sonner";

import { authProviderApi, settingsApi, storageApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { cn, formatSize } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { PageHeader } from "@/components/page-header";
import { SocialProviderLogo } from "@/components/social-provider-logo";
import { StorageLogo, resolveLogoVendor } from "@/components/storage-logo";

type Tab = "general" | "auth" | "storage" | "email";

interface StorageListItem {
  id: string;
  name: string;
  driver: string;
  provider?: string;
  capacity_limit_bytes: number;
  used_bytes: number;
  files_count?: number;
  priority: number;
  is_default: boolean;
  is_active: boolean;
}

interface OAuthProviderItem {
  key: string;
  label: string;
  icon_key: string;
  protocol: string;
  enabled: boolean;
  client_id: string;
  has_secret: boolean;
  callback_url: string;
  is_configured: boolean;
  scopes: string[];
  site_url: string;
  site_url_valid: boolean;
}

interface OAuthProviderDraft {
  enabled: boolean;
  client_id: string;
  client_secret: string;
}

/* ────────────────────────────────────────────────────────────
 * Preference row — label+hint on the left, control on the right.
 *  Uses `divide-y` on the parent Card content to get the
 *  separator lines in the target design.
 * ──────────────────────────────────────────────────────────── */
function Preference({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
      <div className="min-w-0">
        <div className="text-sm font-medium">{label}</div>
        {hint && (
          <div className="mt-0.5 text-[11px] text-muted-foreground">{hint}</div>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * Pill-style Tabs — mirrors the target `Tabs` component
 *  (inline-flex rounded bg-muted/70 with icon + label).
 * ──────────────────────────────────────────────────────────── */
function TabPills({
  tabs,
  value,
  onChange,
}: {
  tabs: { value: Tab; label: string; icon: React.ElementType }[];
  value: Tab;
  onChange: (v: Tab) => void;
}) {
  return (
    <div className="inline-flex h-9 items-center justify-center rounded-lg bg-muted/70 p-1 text-muted-foreground">
      {tabs.map((tab) => {
        const active = value === tab.value;
        return (
          <button
            key={tab.value}
            type="button"
            onClick={() => onChange(tab.value)}
            className={cn(
              "inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium transition-all",
              active
                ? "bg-background text-foreground shadow-sm"
                : "hover:text-foreground"
            )}
          >
            <tab.icon className="size-3.5" strokeWidth={1.8} />
            {tab.label}
          </button>
        );
      })}
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * SettingsPage
 * ──────────────────────────────────────────────────────────── */
export default function SettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<Tab>("general");
  const [form, setForm] = useState<Record<string, string>>({});
  const [saved, setSaved] = useState(false);
  const [providerDrafts, setProviderDrafts] = useState<
    Record<string, OAuthProviderDraft>
  >({});
  const [providerTouched, setProviderTouched] = useState<Record<string, boolean>>(
    {}
  );

  const { data, isLoading } = useQuery({
    queryKey: ["settings"],
    queryFn: () => settingsApi.get().then((r) => r.data.data),
  });

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  const { data: providerList } = useQuery<OAuthProviderItem[]>({
    queryKey: ["auth", "providers"],
    queryFn: () => authProviderApi.list().then((r) => r.data.data),
    enabled: tab === "auth",
  });

  useEffect(() => {
    if (!providerList) return;
    setProviderDrafts(
      Object.fromEntries(
        providerList.map((provider) => [
          provider.key,
          {
            enabled: provider.enabled,
            client_id: provider.client_id ?? "",
            client_secret: "",
          },
        ])
      )
    );
    setProviderTouched({});
  }, [providerList]);

  const mutation = useMutation({
    mutationFn: () => settingsApi.update(form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings"] });
      queryClient.invalidateQueries({ queryKey: ["auth", "providers"] });
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

  const resetForm = () => {
    if (data) setForm(data);
  };

  const boolOf = (key: string): boolean => form[key] === "true";

  /* ── storage tab ─────────────────────────────────────── */
  const { data: storageList } = useQuery<StorageListItem[]>({
    queryKey: ["storage", "list"],
    queryFn: () => storageApi.list().then((r) => r.data.data),
    enabled: tab === "storage",
  });

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => storageApi.setDefault(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["storage", "list"] });
      toast.success(t("settings.saved"));
    },
    onError: () => toast.error(t("toast.error")),
  });

  const saveProviderMutation = useMutation({
    mutationFn: ({
      provider,
      payload,
    }: {
      provider: string;
      payload: OAuthProviderDraft;
    }) =>
      authProviderApi.update(provider, {
        enabled: payload.enabled,
        client_id: payload.client_id,
        client_secret: payload.client_secret,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["auth", "providers"] });
      queryClient.invalidateQueries({ queryKey: ["auth", "options"] });
      toast.success(t("settings.saved"));
    },
    onError: (err: unknown) => {
      const msg =
        (
          err as { response?: { data?: { message?: string } } }
        )?.response?.data?.message ?? t("toast.error");
      toast.error(msg);
    },
  });

  /* ── early loading state ─────────────────────────────── */
  if (isLoading) {
    return (
      <div className="space-y-5">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-9 w-80" />
        <Skeleton className="h-64 rounded-xl" />
      </div>
    );
  }

  const tabs: { value: Tab; label: string; icon: React.ElementType }[] = [
    { value: "general", label: t("settings.general"), icon: SettingsIcon },
    { value: "auth", label: t("settings.auth"), icon: Shield },
    { value: "storage", label: t("settings.storageTab"), icon: HardDrive },
    { value: "email", label: t("settings.email"), icon: Mail },
  ];

  const updateProviderDraft = (
    provider: string,
    patch: Partial<OAuthProviderDraft>
  ) =>
    setProviderDrafts((prev) => ({
      ...prev,
      [provider]: {
        ...prev[provider],
        ...patch,
      },
    }));

  const getProviderDraft = (provider: OAuthProviderItem): OAuthProviderDraft =>
    providerDrafts[provider.key] ?? {
      enabled: provider.enabled,
      client_id: provider.client_id ?? "",
      client_secret: "",
    };

  const clientIdLabel = (provider: string) =>
    provider === "wechat" ? t("settings.oauthAppId") : t("settings.oauthClientId");

  const clientSecretLabel = (provider: string) =>
    provider === "wechat"
      ? t("settings.oauthAppSecret")
      : t("settings.oauthClientSecret");

  const getProviderStatus = (provider: OAuthProviderItem, draft: OAuthProviderDraft) => {
    if (draft.enabled) return t("settings.oauthStatusEnabled");
    if (provider.is_configured) return t("settings.oauthStatusConfigured");
    return t("settings.oauthStatusEmpty");
  };

  const providerMissingFields = (provider: OAuthProviderItem, draft: OAuthProviderDraft) => {
    if (!draft.enabled) return false;
    return !draft.client_id.trim() || (!draft.client_secret.trim() && !provider.has_secret);
  };

  const handleSaveProvider = (provider: OAuthProviderItem) => {
    const draft = getProviderDraft(provider);
    setProviderTouched((prev) => ({ ...prev, [provider.key]: true }));

    if (draft.enabled && !provider.site_url_valid) {
      toast.error(t("settings.oauthSiteUrlInvalid"));
      return;
    }
    if (providerMissingFields(provider, draft)) {
      toast.error(t("settings.oauthMissingFields"));
      return;
    }

    saveProviderMutation.mutate({
      provider: provider.key,
      payload: draft,
    });
  };

  /* ── render ──────────────────────────────────────────── */
  return (
    <div className="space-y-5">
      <PageHeader
        title={t("settings.title")}
        description={t("settings.description")}
      />

      <TabPills tabs={tabs} value={tab} onChange={setTab} />

      {/* ── General ──────────────────────────────────── */}
      {tab === "general" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("settings.generalDesc")}</CardTitle>
          </CardHeader>
          <CardContent className="divide-y">
            <Preference
              label={t("settings.siteName")}
              hint={t("settings.siteNameHint")}
            >
              <Input
                value={form.site_name ?? ""}
                onChange={(e) => updateField("site_name", e.target.value)}
                className="w-56"
              />
            </Preference>
            <Preference
              label={t("settings.siteUrl")}
              hint={t("settings.siteUrlHint")}
            >
              <Input
                value={form.site_url ?? ""}
                onChange={(e) => updateField("site_url", e.target.value)}
                placeholder={t("settings.siteUrlPlaceholder")}
                className="w-64"
              />
            </Preference>
            <Preference
              label={t("settings.allowRegistration")}
              hint={t("settings.allowRegistrationHint")}
            >
              <Switch
                checked={boolOf("allow_registration")}
                onCheckedChange={() => toggleField("allow_registration")}
              />
            </Preference>
            <Preference
              label={t("settings.defaultQuota")}
              hint={t("settings.defaultQuotaHint")}
            >
              <Input
                value={form.default_quota ?? ""}
                onChange={(e) => updateField("default_quota", e.target.value)}
                placeholder="10 GB"
                className="w-32"
              />
            </Preference>
          </CardContent>
          <CardFooter className="justify-end gap-2">
            <Button variant="outline" size="sm" onClick={resetForm}>
              {t("settings.reset")}
            </Button>
            <Button
              size="sm"
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending}
            >
              {saved ? (
                <>
                  <Check className="size-3.5" />
                  {t("settings.saved")}
                </>
              ) : mutation.isPending ? (
                t("settings.saving")
              ) : (
                t("settings.saveSettings")
              )}
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* ── Auth ─────────────────────────────────────── */}
      {tab === "auth" && (
        <div className="space-y-5">
          <Card>
            <CardHeader>
              <CardTitle>{t("settings.authDesc")}</CardTitle>
            </CardHeader>
            <CardContent className="divide-y">
              <Preference
                label={t("settings.twoFactor")}
                hint={t("settings.twoFactorHint")}
              >
                <Switch
                  checked={boolOf("two_factor_required")}
                  onCheckedChange={() => toggleField("two_factor_required")}
                />
              </Preference>
              <Preference label={t("settings.passwordMinLength")}>
                <Input
                  value={form.password_min_length ?? ""}
                  onChange={(e) =>
                    updateField("password_min_length", e.target.value)
                  }
                  placeholder="10"
                  className="w-20"
                />
              </Preference>
              <Preference
                label={t("settings.sessionTimeout")}
                hint={t("settings.sessionTimeoutHint")}
              >
                <Input
                  value={form.session_timeout ?? ""}
                  onChange={(e) =>
                    updateField("session_timeout", e.target.value)
                  }
                  placeholder="7d"
                  className="w-24"
                />
              </Preference>
              <Preference
                label={t("settings.allowGuestUpload")}
                hint={t("settings.allowGuestUploadHint")}
              >
                <Switch
                  checked={boolOf("allow_guest_upload")}
                  onCheckedChange={() => toggleField("allow_guest_upload")}
                />
              </Preference>
              <Preference
                label={t("settings.allowPublicGallery")}
                hint={t("settings.allowPublicGalleryHint")}
              >
                <Switch
                  checked={boolOf("allow_public_gallery")}
                  onCheckedChange={() => toggleField("allow_public_gallery")}
                />
              </Preference>
            </CardContent>
            <CardFooter className="justify-end gap-2">
              <Button variant="outline" size="sm" onClick={resetForm}>
                {t("settings.reset")}
              </Button>
              <Button
                size="sm"
                onClick={() => mutation.mutate()}
                disabled={mutation.isPending}
              >
                {saved ? (
                  <>
                    <Check className="size-3.5" />
                    {t("settings.saved")}
                  </>
                ) : mutation.isPending ? (
                  t("settings.saving")
                ) : (
                  t("settings.saveSettings")
                )}
              </Button>
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t("settings.oauthProvidersTitle")}</CardTitle>
              <CardDescription>{t("settings.oauthProvidersHint")}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
                {(providerList ?? []).map((provider) => {
                  const draft = getProviderDraft(provider);
                  const isPending =
                    saveProviderMutation.isPending &&
                    saveProviderMutation.variables?.provider === provider.key;
                  const missing = providerTouched[provider.key] && providerMissingFields(provider, draft);
                  return (
                    <div
                      key={provider.key}
                      className="flex h-full flex-col rounded-xl border bg-background/70 p-4"
                    >
                      <div className="mb-4 flex items-start justify-between gap-3">
                        <div className="flex items-center gap-3">
                          <SocialProviderLogo
                            provider={provider.icon_key}
                            size={40}
                            rounded="rounded-lg"
                          />
                          <div>
                            <div className="text-sm font-medium">
                              {provider.label}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {provider.protocol}
                            </div>
                          </div>
                        </div>
                        <Badge
                          variant={draft.enabled ? "secondary" : "outline"}
                          className="text-[10px]"
                        >
                          {getProviderStatus(provider, draft)}
                        </Badge>
                      </div>

                      {!provider.site_url_valid && (
                        <div className="mb-4 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-300">
                          {t("settings.oauthSiteUrlInvalid")}
                        </div>
                      )}

                      <div className="space-y-4">
                        <div className="flex items-center justify-between rounded-lg border bg-muted/20 px-3 py-2.5">
                          <div>
                            <div className="text-sm font-medium">
                              {t("settings.oauthEnable")}
                            </div>
                            <div className="text-[11px] text-muted-foreground">
                              {t("settings.oauthEnableHint")}
                            </div>
                          </div>
                          <Switch
                            checked={draft.enabled}
                            onCheckedChange={(checked) =>
                              updateProviderDraft(provider.key, { enabled: checked })
                            }
                          />
                        </div>

                        <div className="grid gap-2">
                          <Label>{clientIdLabel(provider.key)}</Label>
                          <Input
                            value={draft.client_id}
                            onChange={(e) =>
                              updateProviderDraft(provider.key, {
                                client_id: e.target.value,
                              })
                            }
                            className={cn(missing && !draft.client_id.trim() && "border-red-500")}
                            placeholder={provider.key === "wechat" ? "wx123..." : "client-id"}
                          />
                        </div>

                        <div className="grid gap-2">
                          <Label>{clientSecretLabel(provider.key)}</Label>
                          <Input
                            type="password"
                            value={draft.client_secret}
                            onChange={(e) =>
                              updateProviderDraft(provider.key, {
                                client_secret: e.target.value,
                              })
                            }
                            className={cn(
                              missing &&
                                !draft.client_secret.trim() &&
                                !provider.has_secret &&
                                "border-red-500"
                            )}
                            placeholder={
                              provider.has_secret
                                ? t("settings.oauthSecretConfigured")
                                : provider.key === "wechat"
                                  ? "app-secret"
                                  : "client-secret"
                            }
                          />
                          <div className="text-[11px] text-muted-foreground">
                            {provider.has_secret
                              ? t("settings.oauthSecretKeep")
                              : t("settings.oauthSecretEmpty")}
                          </div>
                        </div>

                        <div className="grid gap-2">
                          <Label>{t("settings.oauthCallbackUrl")}</Label>
                          <div className="flex items-center gap-2">
                            <Input value={provider.callback_url} readOnly className="font-mono text-xs" />
                            <Button
                              type="button"
                              variant="outline"
                              size="icon"
                              onClick={() => {
                                navigator.clipboard.writeText(provider.callback_url);
                                toast.success(t("toast.copied"));
                              }}
                            >
                              <Copy className="size-4" />
                            </Button>
                          </div>
                        </div>

                        <div className="grid gap-2">
                          <Label>{t("settings.oauthScopes")}</Label>
                          <div className="rounded-lg border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
                            {provider.scopes.join(" ")}
                          </div>
                        </div>
                      </div>

                      <div className="mt-5 flex justify-end">
                        <Button
                          size="sm"
                          onClick={() => handleSaveProvider(provider)}
                          disabled={isPending}
                        >
                          {isPending && <Loader2 className="size-4 animate-spin" />}
                          {t("settings.saveProvider")}
                        </Button>
                      </div>
                    </div>
                  );
                })}

                {providerList == null &&
                  Array.from({ length: 3 }).map((_, index) => (
                    <Skeleton key={index} className="h-[320px] rounded-xl" />
                  ))}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* ── Default storage ──────────────────────────── */}
      {tab === "storage" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("settings.storageTabDesc")}</CardTitle>
            <CardDescription>{t("settings.storageTabHint")}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {(storageList ?? []).map((d) => {
              const vendor = resolveLogoVendor(d.provider, d.driver);
              const checked = d.is_default;
              return (
                <label
                  key={d.id}
                  className={cn(
                    "flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition-colors",
                    checked
                      ? "border-foreground/30 bg-muted/30"
                      : "hover:bg-muted/30"
                  )}
                >
                  <input
                    type="radio"
                    name="default-driver"
                    checked={checked}
                    disabled={!d.is_active || setDefaultMutation.isPending}
                    onChange={() => {
                      if (!checked && d.is_active)
                        setDefaultMutation.mutate(d.id);
                    }}
                    className="size-4 accent-foreground"
                  />
                  <StorageLogo vendor={vendor} size={28} rounded="rounded-md" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium">
                        {d.name}
                      </span>
                      <Badge
                        variant="outline"
                        className="h-4 px-1.5 font-mono text-[10px] uppercase tracking-wide text-muted-foreground"
                      >
                        {d.driver}
                      </Badge>
                      {!d.is_active && (
                        <Badge
                          variant="outline"
                          className="h-4 px-1.5 text-[10px] uppercase tracking-wide text-muted-foreground"
                        >
                          {t("storage.idleBadge")}
                        </Badge>
                      )}
                    </div>
                    <div className="font-mono text-[10px] text-muted-foreground">
                      P{d.priority}
                    </div>
                  </div>
                  <span className="shrink-0 text-[11px] tabular-nums text-muted-foreground">
                    {formatSize(d.used_bytes)}
                  </span>
                </label>
              );
            })}
            {storageList != null && storageList.length === 0 && (
              <div className="py-8 text-center text-sm text-muted-foreground">
                {t("storage.noStorage")}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* ── Email ────────────────────────────────────── */}
      {tab === "email" && (
        <Card>
          <CardHeader>
            <CardTitle>{t("settings.emailDesc")}</CardTitle>
          </CardHeader>
          <CardContent className="divide-y">
            <Preference label={t("settings.smtpServer")}>
              <Input
                value={form.smtp_host ?? ""}
                onChange={(e) => updateField("smtp_host", e.target.value)}
                placeholder="smtp.kite.dev"
                className="w-64"
              />
            </Preference>
            <Preference label={t("settings.smtpPort")}>
              <Input
                value={form.smtp_port ?? ""}
                onChange={(e) => updateField("smtp_port", e.target.value)}
                placeholder="587"
                className="w-24"
              />
            </Preference>
            <Preference label={t("settings.smtpTls")}>
              <Switch
                checked={boolOf("smtp_tls")}
                onCheckedChange={() => toggleField("smtp_tls")}
              />
            </Preference>
            <Preference label={t("settings.smtpFrom")}>
              <Input
                value={form.smtp_from ?? ""}
                onChange={(e) => updateField("smtp_from", e.target.value)}
                placeholder="no-reply@kite.dev"
                className="w-64"
              />
            </Preference>
          </CardContent>
          <CardFooter className="justify-between gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => toast.info(t("settings.testMailSent"))}
            >
              <Mail className="size-3.5" />
              {t("settings.sendTestMail")}
            </Button>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={resetForm}>
                {t("settings.reset")}
              </Button>
              <Button
                size="sm"
                onClick={() => mutation.mutate()}
                disabled={mutation.isPending}
              >
                {saved ? (
                  <>
                    <Check className="size-3.5" />
                    {t("settings.saved")}
                  </>
                ) : mutation.isPending ? (
                  t("settings.saving")
                ) : (
                  t("settings.saveSettings")
                )}
              </Button>
            </div>
          </CardFooter>
        </Card>
      )}
    </div>
  );
}
