import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  Loader2,
  Camera,
  ShieldCheck,
  Key,
  Mail,
  Activity,
  Sun,
  Moon,
  Monitor,
  LayoutGrid,
  List as ListIcon,
} from "lucide-react";

import { authApi, fileApi } from "@/lib/api";
import { useAuth } from "@/hooks/use-auth";
import { useI18n, localeLabels, type Locale } from "@/i18n";
import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PageHeader } from "@/components/page-header";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

/* ─────────────────────────────────────────────────────────────
 * Types
 * ────────────────────────────────────────────────────────────*/
interface User {
  user_id: string;
  username: string;
  nickname?: string;
  email?: string;
  avatar_url?: string;
  role: string;
  created_at?: string;
}

const DEFAULT_VIEW_KEY = "kite_files_default_view";
const EMAIL_NOTIFY_KEY = "kite_email_notifications";

/* ─────────────────────────────────────────────────────────────
 * Page
 * ────────────────────────────────────────────────────────────*/
export default function ProfilePage() {
  const { user, applyTokensAndRefresh } = useAuth();
  const { t, locale, setLocale } = useI18n();
  const { theme, setTheme } = useTheme();
  const navigate = useNavigate();

  const [profileForm, setProfileForm] = useState({
    username: user?.username ?? "",
    nickname: user?.nickname ?? "",
    email: user?.email ?? "",
    avatarUrl: user?.avatar_url ?? "",
  });
  // Resync the form when the signed-in identity changes (e.g. after
  // applyTokensAndRefresh). Tracked via user_id so same-user updates
  // (like a successful save) don't clobber the user's edits.
  const [syncedUserId, setSyncedUserId] = useState<string | null>(
    user?.user_id ?? null,
  );
  if (user && user.user_id !== syncedUserId) {
    setSyncedUserId(user.user_id);
    setProfileForm({
      username: user.username ?? "",
      nickname: user.nickname ?? "",
      email: user.email ?? "",
      avatarUrl: user.avatar_url ?? "",
    });
  }
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false);
  const [sessionDialogOpen, setSessionDialogOpen] = useState(false);
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: "",
    newPassword: "",
    confirmPassword: "",
  });

  // Preferences: locale-stored and read lazily (default on SSR-safe fallback).
  const [defaultView, setDefaultView] = useState<"grid" | "list">(() => {
    const stored =
      typeof window !== "undefined"
        ? localStorage.getItem(DEFAULT_VIEW_KEY)
        : null;
    return stored === "list" ? "list" : "grid";
  });
  const [emailNotify, setEmailNotify] = useState<boolean>(() => {
    const stored =
      typeof window !== "undefined"
        ? localStorage.getItem(EMAIL_NOTIFY_KEY)
        : null;
    return stored === null ? true : stored === "true";
  });

  useEffect(() => {
    localStorage.setItem(DEFAULT_VIEW_KEY, defaultView);
  }, [defaultView]);
  useEffect(() => {
    localStorage.setItem(EMAIL_NOTIFY_KEY, String(emailNotify));
  }, [emailNotify]);

  /* ── mutations ───────────────────────────────────────────── */
  const avatarUploadMutation = useMutation({
    mutationFn: async (file: File) => {
      const res = await fileApi.upload(file);
      const uploaded = res.data?.data?.links?.url as string | undefined;
      if (!uploaded) throw new Error("avatar upload did not return URL");

      await authApi.updateProfile({
        username: profileForm.username,
        nickname: profileForm.nickname,
        email: profileForm.email,
        avatar_url: uploaded,
      });

      const tokens = {
        access_token: localStorage.getItem("access_token") ?? "",
        refresh_token: localStorage.getItem("refresh_token") ?? "",
      };
      if (tokens.access_token) await applyTokensAndRefresh(tokens);
      return uploaded;
    },
    onSuccess: (url) => {
      setProfileForm((prev) => ({ ...prev, avatarUrl: url }));
      toast.success(t("profile.avatarUploaded"));
    },
    onError: () => toast.error(t("profile.avatarUploadFailed")),
  });

  const profileMutation = useMutation({
    mutationFn: () =>
      authApi.updateProfile({
        username: profileForm.username,
        nickname: profileForm.nickname,
        email: profileForm.email,
        avatar_url: profileForm.avatarUrl || undefined,
      }),
    onSuccess: async (res) => {
      const updated: User = res.data.data;
      const tokens = {
        access_token: localStorage.getItem("access_token") ?? "",
        refresh_token: localStorage.getItem("refresh_token") ?? "",
      };
      if (tokens.access_token) await applyTokensAndRefresh(tokens);
      toast.success(t("profile.profileSaved"));
      setProfileForm({
        username: updated.username,
        nickname: updated.nickname ?? "",
        email: updated.email ?? "",
        avatarUrl: updated.avatar_url ?? "",
      });
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t("profile.profileFailed");
      toast.error(msg);
    },
  });

  const passwordMutation = useMutation({
    mutationFn: () =>
      authApi.changePassword({
        current_password: passwordForm.currentPassword,
        new_password: passwordForm.newPassword,
      }),
    onSuccess: () => {
      toast.success(t("profile.passwordSaved"));
      setPasswordForm({
        currentPassword: "",
        newPassword: "",
        confirmPassword: "",
      });
      setPasswordDialogOpen(false);
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t("profile.passwordFailed");
      toast.error(msg);
    },
  });

  /* ── handlers ────────────────────────────────────────────── */
  const handleProfileSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!profileForm.username.trim() || !profileForm.email.trim()) {
      toast.error(t("profile.allFieldsRequired"));
      return;
    }
    profileMutation.mutate();
  };

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (
      !passwordForm.currentPassword ||
      !passwordForm.newPassword ||
      !passwordForm.confirmPassword
    ) {
      toast.error(t("profile.allFieldsRequired"));
      return;
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      toast.error(t("auth.passwordMismatch"));
      return;
    }
    if (passwordForm.newPassword === passwordForm.currentPassword) {
      toast.error(t("profile.newPasswordMustDiffer"));
      return;
    }
    passwordMutation.mutate();
  };

  const handleAvatarFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    if (!file.type.startsWith("image/")) {
      toast.error(t("profile.avatarImageOnly"));
      e.target.value = "";
      return;
    }
    avatarUploadMutation.mutate(file);
    e.target.value = "";
  };

  const resetProfile = () => {
    if (!user) return;
    setProfileForm({
      username: user.username ?? "",
      nickname: user.nickname ?? "",
      email: user.email ?? "",
      avatarUrl: user.avatar_url ?? "",
    });
  };

  /* ── derived values ─────────────────────────────────────── */
  const displayName =
    profileForm.nickname.trim() || user?.nickname?.trim() || user?.username || "";

  const joinedDate = user?.created_at
    ? new Date(user.created_at).toLocaleDateString()
    : t("profile.accountUnknownDate");

  const openSecuritySettings = () => {
    if (user?.role === "admin") {
      navigate("/admin/settings");
      return;
    }
    navigate("/user/tokens");
  };

  const jumpToEmailField = () => {
    const el = document.getElementById("email") as HTMLInputElement | null;
    if (!el) return;
    el.scrollIntoView({ behavior: "smooth", block: "center" });
    window.setTimeout(() => el.focus(), 120);
  };

  if (!user) return null;

  return (
    <div className="space-y-6">
      <PageHeader
        title={t("profile.title")}
        description={t("profile.description")}
      />

      {/* —— 基本信息 Card —— */}
      <section className="rounded-xl border bg-card p-6 sm:p-7">
            <header className="mb-6">
              <h3 className="text-base font-semibold tracking-tight">
                {t("profile.basicInfo")}
              </h3>
              <p className="mt-1 text-sm text-muted-foreground">
                {t("profile.basicInfoCardDesc")}
              </p>
            </header>

            <form
              onSubmit={handleProfileSubmit}
              className="grid gap-x-6 gap-y-5 sm:grid-cols-[auto_minmax(0,1fr)] sm:items-center"
            >
              {/* Avatar — vertically centered with the 2×2 form block */}
              <div className="group relative shrink-0 self-center">
                <Avatar className="size-24 ring-2 ring-background/80">
                  <AvatarImage
                    src={profileForm.avatarUrl || user.avatar_url}
                    alt={displayName}
                    className="object-cover"
                  />
                  <AvatarFallback className="text-2xl font-semibold">
                    {displayName.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <Label
                  htmlFor="avatar-file"
                  className="absolute inset-0 flex cursor-pointer items-center justify-center rounded-full bg-black/55 text-white opacity-0 transition-opacity group-hover:opacity-100"
                >
                  {avatarUploadMutation.isPending ? (
                    <Loader2 className="size-5 animate-spin" />
                  ) : (
                    <Camera className="size-5" />
                  )}
                </Label>
                <Input
                  id="avatar-file"
                  type="file"
                  accept="image/*"
                  className="hidden"
                  onChange={handleAvatarFileChange}
                  disabled={avatarUploadMutation.isPending}
                />
              </div>

              {/* 2×2 form grid */}
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="grid gap-1.5">
                  <Label htmlFor="nickname" className="text-xs text-muted-foreground">
                    {t("profile.nickname")}
                  </Label>
                  <Input
                    id="nickname"
                    value={profileForm.nickname}
                    onChange={(e) =>
                      setProfileForm((p) => ({ ...p, nickname: e.target.value }))
                    }
                    maxLength={32}
                    placeholder={t("profile.nicknamePlaceholder")}
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="username" className="text-xs text-muted-foreground">
                    {t("auth.username")}
                  </Label>
                  <Input
                    id="username"
                    autoCapitalize="none"
                    autoCorrect="off"
                    value={profileForm.username}
                    onChange={(e) =>
                      setProfileForm((p) => ({ ...p, username: e.target.value }))
                    }
                    minLength={3}
                    maxLength={32}
                    required
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="email" className="text-xs text-muted-foreground">
                    {t("auth.email")}
                  </Label>
                  <Input
                    id="email"
                    type="email"
                    autoCapitalize="none"
                    autoCorrect="off"
                    value={profileForm.email}
                    onChange={(e) =>
                      setProfileForm((p) => ({ ...p, email: e.target.value }))
                    }
                    placeholder={t("auth.emailPlaceholder")}
                    required
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="registered" className="text-xs text-muted-foreground">
                    {t("profile.registered")}
                  </Label>
                  <Input
                    id="registered"
                    value={joinedDate}
                    readOnly
                    disabled
                    className="bg-muted/40"
                  />
                </div>
              </div>

              {/* Footer buttons span the whole width */}
              <div className="flex justify-end gap-2 sm:col-span-2">
                <Button type="button" variant="outline" onClick={resetProfile}>
                  {t("profile.cancel")}
                </Button>
                <Button type="submit" disabled={profileMutation.isPending}>
                  {profileMutation.isPending && (
                    <Loader2 className="size-4 animate-spin" />
                  )}
                  {t("profile.save")}
                </Button>
              </div>
            </form>
          </section>

      {/* ═════ Full-width rows below ═════ */}
      <div className="space-y-6">
        {/* —— 安全 Card —— */}
        <section className="rounded-xl border bg-card p-6 sm:p-7">
            <header className="mb-5 flex items-start justify-between gap-4">
              <div>
                <h3 className="text-base font-semibold tracking-tight">
                  {t("profile.security")}
                </h3>
                <p className="mt-1 text-sm text-muted-foreground">
                  {t("profile.securityDesc")}
                </p>
              </div>
              <Badge
                variant="outline"
                className="shrink-0 gap-1 border-emerald-500/30 bg-emerald-500/5 text-emerald-600 dark:text-emerald-400"
              >
                <ShieldCheck className="size-3" />
                {t("profile.securityHealthy")}
              </Badge>
            </header>

            <div className="divide-y">
              <SecurityRow
                icon={Key}
                title={t("profile.passwordRow")}
                description={t("profile.passwordRowHintUnknown")}
                actionLabel={t("profile.passwordChange")}
                onAction={() => setPasswordDialogOpen(true)}
              />
              <SecurityRow
                icon={ShieldCheck}
                title={t("profile.twoFactor")}
                badge={
                  <Badge variant="secondary" className="text-[10px]">
                    {t("profile.twoFactorOff")}
                  </Badge>
                }
                description={t("profile.twoFactorDesc")}
                actionLabel={t("profile.enable")}
                onAction={openSecuritySettings}
              />
              <SecurityRow
                icon={Mail}
                title={t("profile.backupEmail")}
                badge={
                  user.email ? (
                    <Badge variant="secondary" className="text-[10px]">
                      {t("profile.backupEmailVerified")}
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-[10px]">
                      {t("profile.backupEmailNone")}
                    </Badge>
                  )
                }
                description={user.email ?? t("profile.backupEmailNone")}
                actionLabel={
                  user.email
                    ? t("profile.backupEmailReplace")
                    : t("profile.backupEmailAdd")
                }
                onAction={jumpToEmailField}
              />
              <SecurityRow
                icon={Activity}
                title={t("profile.loginActivity")}
                description={t("profile.loginActivityDesc")
                  .replace("{n}", "1")
                  .replace("{devices}", "Web")}
                actionLabel={t("profile.viewAction")}
                onAction={() => setSessionDialogOpen(true)}
                actionIcon
              />
            </div>
          </section>

          {/* —— 偏好 Card —— */}
          <section className="rounded-xl border bg-card p-6 sm:p-7">
            <header className="mb-5">
              <h3 className="text-base font-semibold tracking-tight">
                {t("profile.preferences")}
              </h3>
            </header>

            <div className="divide-y">
              {/* Language */}
              <PrefRow
                title={t("profile.language")}
                description={t("profile.languageDesc")}
              >
                <SegmentedControl
                  options={(Object.keys(localeLabels) as Locale[]).map((l) => ({
                    value: l,
                    label: localeLabels[l],
                  }))}
                  value={locale}
                  onChange={(v) => setLocale(v as Locale)}
                />
              </PrefRow>

              {/* Theme */}
              <PrefRow
                title={t("settings.theme")}
                description={t("settings.themeDesc")}
              >
                <SegmentedControl
                  options={[
                    { value: "light", label: t("settings.themeLight"), icon: Sun },
                    { value: "dark", label: t("settings.themeDark"), icon: Moon },
                    { value: "system", label: t("settings.themeSystem"), icon: Monitor },
                  ]}
                  value={theme}
                  onChange={(v) => setTheme(v as "light" | "dark" | "system")}
                />
              </PrefRow>

              {/* Default view */}
              <PrefRow
                title={t("profile.defaultView")}
                description={t("profile.defaultViewDesc")}
              >
                <SegmentedControl
                  options={[
                    {
                      value: "grid",
                      label: t("profile.defaultViewGrid"),
                      icon: LayoutGrid,
                    },
                    {
                      value: "list",
                      label: t("profile.defaultViewList"),
                      icon: ListIcon,
                    },
                  ]}
                  value={defaultView}
                  onChange={(v) => setDefaultView(v as "grid" | "list")}
                />
              </PrefRow>

              {/* Email notifications */}
              <PrefRow
                title={t("profile.emailNotifications")}
                description={t("profile.emailNotificationsDesc")}
              >
                <Switch
                  checked={emailNotify}
                  onCheckedChange={setEmailNotify}
                />
              </PrefRow>
            </div>
          </section>
      </div>

      {/* —— Password dialog —— */}
      <Dialog open={passwordDialogOpen} onOpenChange={setPasswordDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("profile.changePassword")}</DialogTitle>
            <DialogDescription>
              {t("profile.changePasswordDesc")}
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handlePasswordSubmit} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="currentPassword">
                {t("profile.currentPassword")}
              </Label>
              <Input
                id="currentPassword"
                type="password"
                autoComplete="current-password"
                value={passwordForm.currentPassword}
                onChange={(e) =>
                  setPasswordForm((p) => ({
                    ...p,
                    currentPassword: e.target.value,
                  }))
                }
                required
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="newPassword">{t("profile.newPassword")}</Label>
              <Input
                id="newPassword"
                type="password"
                autoComplete="new-password"
                value={passwordForm.newPassword}
                onChange={(e) =>
                  setPasswordForm((p) => ({
                    ...p,
                    newPassword: e.target.value,
                  }))
                }
                placeholder={t("auth.passwordHint")}
                minLength={6}
                maxLength={64}
                required
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="confirmPassword">
                {t("auth.confirmPassword")}
              </Label>
              <Input
                id="confirmPassword"
                type="password"
                autoComplete="new-password"
                value={passwordForm.confirmPassword}
                onChange={(e) =>
                  setPasswordForm((p) => ({
                    ...p,
                    confirmPassword: e.target.value,
                  }))
                }
                placeholder={t("auth.confirmPasswordPlaceholder")}
                required
              />
            </div>
            <p className="flex items-start gap-2 rounded-md border bg-muted/30 p-2.5 text-[11px] text-muted-foreground">
              <ShieldCheck className="mt-0.5 size-3 shrink-0" />
              <span>{t("profile.passwordTip")}</span>
            </p>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setPasswordDialogOpen(false)}
              >
                {t("profile.cancel")}
              </Button>
              <Button type="submit" disabled={passwordMutation.isPending}>
                {passwordMutation.isPending && (
                  <Loader2 className="size-4 animate-spin" />
                )}
                {t("profile.savePassword")}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={sessionDialogOpen} onOpenChange={setSessionDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("profile.loginActivity")}</DialogTitle>
            <DialogDescription>
              {t("profile.loginActivityDesc")
                .replace("{n}", "1")
                .replace("{devices}", "Web")}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 rounded-lg border bg-muted/30 p-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t("auth.username")}</span>
              <span className="font-medium">{user.username}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t("common.type")}</span>
              <span className="font-medium">Web</span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="text-muted-foreground">User Agent</span>
              <span className="max-w-[220px] truncate text-right font-medium">
                {navigator.userAgent}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t("common.date")}</span>
              <span className="font-medium">{new Date().toLocaleString()}</span>
            </div>
          </div>

          <DialogFooter>
            <Button onClick={() => setSessionDialogOpen(false)}>
              {t("files.close")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* ─────────────────────────────────────────────────────────────
 * Sub-components
 * ────────────────────────────────────────────────────────────*/
function SecurityRow({
  icon: Icon,
  title,
  badge,
  description,
  actionLabel,
  onAction,
  actionIcon,
}: {
  icon: React.ElementType;
  title: string;
  badge?: React.ReactNode;
  description: string;
  actionLabel: string;
  onAction: () => void;
  actionIcon?: boolean;
}) {
  return (
    <div className="flex items-start justify-between gap-4 py-4 first:pt-0 last:pb-0">
      <div className="flex min-w-0 items-start gap-3">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-md border bg-muted/30">
          <Icon className="size-4 text-muted-foreground" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-medium">{title}</span>
            {badge}
          </div>
          <p className="mt-0.5 truncate text-xs text-muted-foreground">
            {description}
          </p>
        </div>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="shrink-0"
        onClick={onAction}
      >
        {actionLabel}
        {actionIcon && (
          <svg
            viewBox="0 0 16 16"
            fill="none"
            className="size-3"
            aria-hidden="true"
          >
            <path
              d="M6 4l4 4-4 4"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        )}
      </Button>
    </div>
  );
}

function PrefRow({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-4 first:pt-0 last:pb-0">
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium">{title}</p>
        {description && (
          <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

interface SegOption {
  value: string;
  label: string;
  icon?: React.ElementType;
}

function SegmentedControl({
  options,
  value,
  onChange,
}: {
  options: SegOption[];
  value: string;
  onChange: (value: string) => void;
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [indicator, setIndicator] = useState<{
    left: number;
    width: number;
    ready: boolean;
  }>({ left: 0, width: 0, ready: false });

  // Measure the active button and position the pill accordingly.
  // useLayoutEffect avoids a flicker on first paint when switching.
  useLayoutEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const el = container.querySelector<HTMLElement>(
      `[data-value="${CSS.escape(value)}"]`,
    );
    if (!el) return;
    setIndicator({
      left: el.offsetLeft,
      width: el.offsetWidth,
      ready: true,
    });
  }, [value, options.length]);

  // Recompute on window resize so the pill tracks layout changes.
  useEffect(() => {
    const handle = () => {
      const container = containerRef.current;
      if (!container) return;
      const el = container.querySelector<HTMLElement>(
        `[data-value="${CSS.escape(value)}"]`,
      );
      if (!el) return;
      setIndicator((prev) => ({
        left: el.offsetLeft,
        width: el.offsetWidth,
        ready: prev.ready,
      }));
    };
    window.addEventListener("resize", handle);
    return () => window.removeEventListener("resize", handle);
  }, [value]);

  return (
    <div
      ref={containerRef}
      className="relative inline-flex rounded-md border bg-muted/30 p-0.5"
    >
      {/* Sliding active pill */}
      <span
        aria-hidden
        className={cn(
          "pointer-events-none absolute top-0.5 bottom-0.5 rounded-sm bg-background shadow-sm",
          indicator.ready
            ? "transition-[left,width] duration-300 ease-out"
            : "opacity-0",
        )}
        style={{ left: indicator.left, width: indicator.width }}
      />
      {options.map((opt) => {
        const active = opt.value === value;
        const Icon = opt.icon;
        return (
          <button
            key={opt.value}
            type="button"
            data-value={opt.value}
            onClick={() => onChange(opt.value)}
            className={cn(
              "relative z-10 flex items-center gap-1.5 rounded-sm px-3 py-1 text-xs font-medium transition-colors",
              active
                ? "text-foreground"
                : "text-muted-foreground hover:text-foreground",
            )}
          >
            {Icon && <Icon className="size-3.5" />}
            {opt.label}
          </button>
        );
      })}
    </div>
  );
}
