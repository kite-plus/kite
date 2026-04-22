import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  Loader2,
  Camera,
  ShieldCheck,
  User,
  Key,
  Mail,
  Activity,
  Sun,
  Moon,
  Monitor,
  LayoutGrid,
  List as ListIcon,
  Link2,
} from 'lucide-react'

import { authApi, fileApi, statsApi } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { useI18n, localeLabels, type Locale } from '@/i18n'
import { useTheme } from '@/components/theme-provider'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { PageHeader } from '@/components/page-header'
import { SocialProviderLogo } from '@/components/social-provider-logo'
import { cn, formatSize } from '@/lib/utils'
import { toast } from 'sonner'

/* ─────────────────────────────────────────────────────────────
 * Types
 * ────────────────────────────────────────────────────────────*/
interface User {
  user_id: string
  username: string
  nickname?: string
  email?: string
  avatar_url?: string
  has_local_password?: boolean
  role: string
  created_at?: string
}

interface IdentityStatus {
  key: string
  label: string
  icon_key: string
  bound: boolean
  display_name?: string
  email?: string
}

const DEFAULT_VIEW_KEY = 'kite_files_default_view'
const EMAIL_NOTIFY_KEY = 'kite_email_notifications'
type ProfileTab = 'basic' | 'security' | 'identities' | 'preferences'

/* ─────────────────────────────────────────────────────────────
 * Page
 * ────────────────────────────────────────────────────────────*/
export default function ProfilePage() {
  const { user, applyTokensAndRefresh } = useAuth()
  const { t, locale, setLocale } = useI18n()
  const { theme, setTheme } = useTheme()
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()

  // Username is immutable and email rotation goes through the verified
  // flow in the dialog below; the self-serve form only edits these two.
  const [profileForm, setProfileForm] = useState({
    nickname: user?.nickname ?? '',
    avatarUrl: user?.avatar_url ?? '',
  })
  // Resync the form when the signed-in identity changes (e.g. after
  // applyTokensAndRefresh). Tracked via user_id so same-user updates
  // (like a successful save) don't clobber the user's edits.
  const [syncedUserId, setSyncedUserId] = useState<string | null>(
    user?.user_id ?? null
  )
  if (user && user.user_id !== syncedUserId) {
    setSyncedUserId(user.user_id)
    setProfileForm({
      nickname: user.nickname ?? '',
      avatarUrl: user.avatar_url ?? '',
    })
  }
  const [passwordDialogOpen, setPasswordDialogOpen] = useState(false)
  const [sessionDialogOpen, setSessionDialogOpen] = useState(false)
  // Email-change flow is a 2-step dialog: request a code (sent via SMTP to
  // the target address) and then confirm it. Owning this state at the page
  // level keeps the resend countdown alive across step transitions.
  const [emailDialogOpen, setEmailDialogOpen] = useState(false)
  const [emailStep, setEmailStep] = useState<'request' | 'verify'>('request')
  const [pendingNewEmail, setPendingNewEmail] = useState('')
  const [emailCode, setEmailCode] = useState('')
  const [resendAtMs, setResendAtMs] = useState<number | null>(null)
  const [resendTickNow, setResendTickNow] = useState<number>(() => Date.now())
  const [tab, setTab] = useState<ProfileTab>('basic')
  const [passwordForm, setPasswordForm] = useState({
    currentPassword: '',
    newPassword: '',
    confirmPassword: '',
  })
  // Cache "now" at mount time so joined-days math stays idempotent
  // during render (purity rule); one-day drift on a long session is fine.
  const [mountedAtMs] = useState<number>(() => Date.now())

  // Preferences: locale-stored and read lazily (default on SSR-safe fallback).
  const [defaultView, setDefaultView] = useState<'grid' | 'list'>(() => {
    const stored =
      typeof window !== 'undefined'
        ? localStorage.getItem(DEFAULT_VIEW_KEY)
        : null
    return stored === 'list' ? 'list' : 'grid'
  })
  const [emailNotify, setEmailNotify] = useState<boolean>(() => {
    const stored =
      typeof window !== 'undefined'
        ? localStorage.getItem(EMAIL_NOTIFY_KEY)
        : null
    return stored === null ? true : stored === 'true'
  })

  useEffect(() => {
    localStorage.setItem(DEFAULT_VIEW_KEY, defaultView)
  }, [defaultView])
  useEffect(() => {
    localStorage.setItem(EMAIL_NOTIFY_KEY, String(emailNotify))
  }, [emailNotify])

  // Ticker that drives the resend cooldown countdown in the email dialog.
  // The interval runs only while the dialog is open and a deadline is set,
  // and it stops as soon as the deadline elapses.
  useEffect(() => {
    if (!emailDialogOpen || !resendAtMs) return
    if (resendAtMs <= Date.now()) return
    const id = window.setInterval(() => setResendTickNow(Date.now()), 1000)
    return () => window.clearInterval(id)
  }, [emailDialogOpen, resendAtMs])

  const resendSecondsLeft = resendAtMs
    ? Math.max(0, Math.ceil((resendAtMs - resendTickNow) / 1000))
    : 0

  /* ── mutations ───────────────────────────────────────────── */
  const avatarUploadMutation = useMutation({
    mutationFn: async (file: File) => {
      const res = await fileApi.upload(file)
      const uploaded = res.data?.data?.links?.url as string | undefined
      if (!uploaded) throw new Error('avatar upload did not return URL')

      await authApi.updateProfile({
        nickname: profileForm.nickname,
        avatar_url: uploaded,
      })

      const tokens = {
        access_token: localStorage.getItem('access_token') ?? '',
        refresh_token: localStorage.getItem('refresh_token') ?? '',
      }
      if (tokens.access_token) await applyTokensAndRefresh(tokens)
      return uploaded
    },
    onSuccess: (url) => {
      setProfileForm((prev) => ({ ...prev, avatarUrl: url }))
      toast.success(t('profile.avatarUploaded'))
    },
    onError: () => toast.error(t('profile.avatarUploadFailed')),
  })

  const profileMutation = useMutation({
    mutationFn: () =>
      authApi.updateProfile({
        nickname: profileForm.nickname,
        avatar_url: profileForm.avatarUrl || undefined,
      }),
    onSuccess: async (res) => {
      const updated: User = res.data.data
      const tokens = {
        access_token: localStorage.getItem('access_token') ?? '',
        refresh_token: localStorage.getItem('refresh_token') ?? '',
      }
      if (tokens.access_token) await applyTokensAndRefresh(tokens)
      toast.success(t('profile.profileSaved'))
      setProfileForm({
        nickname: updated.nickname ?? '',
        avatarUrl: updated.avatar_url ?? '',
      })
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('profile.profileFailed')
      toast.error(msg)
    },
  })

  // Email-change mutations. RequestEmailChange sends a code via SMTP;
  // ConfirmEmailChange redeems it and rotates the user's email. On success
  // we rebuild the auth store so the display email updates without a
  // second round trip.
  const emailRequestMutation = useMutation({
    mutationFn: (newEmail: string) => authApi.requestEmailChange(newEmail),
    onSuccess: (res) => {
      const data = res.data?.data as { resend_after_s?: number } | undefined
      const after = data?.resend_after_s ?? 60
      const now = Date.now()
      setResendAtMs(now + after * 1000)
      setResendTickNow(now)
      setEmailStep('verify')
      toast.success(t('profile.emailCodeSentToast'))
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('profile.emailRequestFailed')
      toast.error(msg)
    },
  })

  const emailConfirmMutation = useMutation({
    mutationFn: (vars: { email: string; code: string }) =>
      authApi.confirmEmailChange(vars.email, vars.code),
    onSuccess: async () => {
      const tokens = {
        access_token: localStorage.getItem('access_token') ?? '',
        refresh_token: localStorage.getItem('refresh_token') ?? '',
      }
      if (tokens.access_token) await applyTokensAndRefresh(tokens)
      toast.success(t('profile.emailChangedToast'))
      setEmailDialogOpen(false)
      resetEmailDialog()
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('profile.emailConfirmFailed')
      toast.error(msg)
    },
  })

  const passwordMutation = useMutation({
    mutationFn: () =>
      authApi.changePassword({
        current_password: passwordForm.currentPassword,
        new_password: passwordForm.newPassword,
      }),
    onSuccess: () => {
      toast.success(t('profile.passwordSaved'))
      setPasswordForm({
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      })
      setPasswordDialogOpen(false)
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('profile.passwordFailed')
      toast.error(msg)
    },
  })

  const setPasswordMutation = useMutation({
    mutationFn: () =>
      authApi.setPassword({
        new_password: passwordForm.newPassword,
      }),
    onSuccess: async () => {
      const tokens = {
        access_token: localStorage.getItem('access_token') ?? '',
        refresh_token: localStorage.getItem('refresh_token') ?? '',
      }
      if (tokens.access_token) await applyTokensAndRefresh(tokens)
      queryClient.invalidateQueries({ queryKey: ['auth', 'identities'] })
      toast.success(t('profile.passwordSetSaved'))
      setPasswordForm({
        currentPassword: '',
        newPassword: '',
        confirmPassword: '',
      })
      setPasswordDialogOpen(false)
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('profile.passwordFailed')
      toast.error(msg)
    },
  })

  const { data: identityData } = useQuery<{
    has_local_password: boolean
    providers: IdentityStatus[]
  }>({
    queryKey: ['auth', 'identities'],
    queryFn: () => authApi.identities().then((r) => r.data.data),
    enabled: !!user,
  })

  interface StatsPayload {
    total_files: number
    total_size: number
    images: number
    videos: number
    audios: number
    others: number
    images_size: number
    videos_size: number
    audios_size: number
    others_size: number
  }
  const { data: statsData } = useQuery<StatsPayload>({
    queryKey: ['stats', 'me'],
    queryFn: () => statsApi.get().then((r) => r.data.data),
    enabled: !!user && tab === 'basic',
  })

  const unlinkMutation = useMutation({
    mutationFn: (provider: string) => authApi.unlinkIdentity(provider),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', 'identities'] })
      toast.success(t('profile.identityUnlinked'))
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('profile.identityUnlinkFailed')
      toast.error(msg)
    },
  })

  /* ── handlers ────────────────────────────────────────────── */
  const handleProfileSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    profileMutation.mutate()
  }

  const resetEmailDialog = () => {
    setEmailStep('request')
    setPendingNewEmail('')
    setEmailCode('')
    setResendAtMs(null)
  }

  const openEmailDialog = () => {
    resetEmailDialog()
    setEmailDialogOpen(true)
  }

  const handleEmailRequest = (e: React.FormEvent) => {
    e.preventDefault()
    const email = pendingNewEmail.trim()
    if (!email) {
      toast.error(t('profile.emailNewRequired'))
      return
    }
    if (email.toLowerCase() === (user?.email ?? '').toLowerCase()) {
      toast.error(t('profile.emailSameAsCurrent'))
      return
    }
    emailRequestMutation.mutate(email)
  }

  const handleEmailConfirm = (e: React.FormEvent) => {
    e.preventDefault()
    if (emailCode.length < 4) {
      toast.error(t('profile.emailCodeInvalid'))
      return
    }
    emailConfirmMutation.mutate({ email: pendingNewEmail, code: emailCode })
  }

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const hasLocalPassword = user?.has_local_password !== false
    if (!passwordForm.newPassword || !passwordForm.confirmPassword) {
      toast.error(t('profile.allFieldsRequired'))
      return
    }
    if (hasLocalPassword && !passwordForm.currentPassword) {
      toast.error(t('profile.allFieldsRequired'))
      return
    }
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      toast.error(t('auth.passwordMismatch'))
      return
    }
    if (
      hasLocalPassword &&
      passwordForm.newPassword === passwordForm.currentPassword
    ) {
      toast.error(t('profile.newPasswordMustDiffer'))
      return
    }
    if (hasLocalPassword) {
      passwordMutation.mutate()
      return
    }
    setPasswordMutation.mutate()
  }

  const handleAvatarFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (!file.type.startsWith('image/')) {
      toast.error(t('profile.avatarImageOnly'))
      e.target.value = ''
      return
    }
    avatarUploadMutation.mutate(file)
    e.target.value = ''
  }

  const resetProfile = () => {
    if (!user) return
    setProfileForm({
      nickname: user.nickname ?? '',
      avatarUrl: user.avatar_url ?? '',
    })
  }

  /* ── derived values ─────────────────────────────────────── */
  const displayName =
    profileForm.nickname.trim() ||
    user?.nickname?.trim() ||
    user?.username ||
    ''

  const joinedDate = user?.created_at
    ? new Date(user.created_at).toLocaleDateString()
    : t('profile.accountUnknownDate')

  const openSecuritySettings = () => {
    if (user?.role === 'admin') {
      navigate('/admin/settings')
      return
    }
    navigate('/user/tokens')
  }

  useEffect(() => {
    const params = new URLSearchParams(location.search)
    const status = params.get('social_status')
    const provider = params.get('provider')
    const oauthError = params.get('oauth_error')
    if (!status && !oauthError) return
    if (status === 'linked') {
      setTab('identities')
      toast.success(
        provider
          ? t('profile.identityLinked').replace('{provider}', provider)
          : t('profile.identityLinkedGeneric')
      )
      queryClient.invalidateQueries({ queryKey: ['auth', 'identities'] })
    } else if (oauthError) {
      setTab('identities')
      toast.error(oauthError)
    }
    navigate(location.pathname, { replace: true })
  }, [location.pathname, location.search, navigate, queryClient, t])

  if (!user) return null

  const hasLocalPassword =
    identityData?.has_local_password ?? user.has_local_password !== false
  const identities = identityData?.providers ?? []
  const linkedCount = identities.filter((i) => i.bound).length
  const pendingProvider = unlinkMutation.variables

  const storageLimit = user.storage_limit ?? 0
  const storageUsed = user.storage_used ?? 0
  const storageUnlimited = storageLimit <= 0

  const joinedDays = user.created_at
    ? Math.max(
        1,
        Math.floor(
          (mountedAtMs - new Date(user.created_at).getTime()) /
            (24 * 60 * 60 * 1000)
        )
      )
    : null

  // Per-type breakdown drives both the stacked bar and the legend list.
  // `barColor` is a solid Tailwind class applied to the bar segment and the
  // square swatch; order here controls both the bar order and list order.
  const usageBreakdown = [
    {
      key: 'images',
      label: t('profile.usageImages'),
      count: statsData?.images ?? 0,
      size: statsData?.images_size ?? 0,
      barColor: 'bg-sky-500',
    },
    {
      key: 'others',
      label: t('profile.usageOthers'),
      count: statsData?.others ?? 0,
      size: statsData?.others_size ?? 0,
      barColor: 'bg-emerald-500',
    },
    {
      key: 'videos',
      label: t('profile.usageVideos'),
      count: statsData?.videos ?? 0,
      size: statsData?.videos_size ?? 0,
      barColor: 'bg-violet-500',
    },
    {
      key: 'audios',
      label: t('profile.usageAudios'),
      count: statsData?.audios ?? 0,
      size: statsData?.audios_size ?? 0,
      barColor: 'bg-amber-500',
    },
  ]

  const copyUserId = async () => {
    try {
      await navigator.clipboard.writeText(user.user_id)
      toast.success(t('profile.metaUserIdCopied'))
    } catch {
      toast.error(t('profile.avatarUploadFailed'))
    }
  }
  const roleLabel =
    user.role === 'admin' ? t('nav.roleAdmin') : t('nav.roleUser')
  const tabs: { value: ProfileTab; label: string; icon: React.ElementType }[] =
    [
      { value: 'basic', label: t('profile.basicInfo'), icon: User },
      { value: 'security', label: t('profile.security'), icon: ShieldCheck },
      { value: 'identities', label: t('profile.socialAccounts'), icon: Link2 },
      { value: 'preferences', label: t('profile.preferences'), icon: Monitor },
    ]

  const startBindFlow = (provider: string) => {
    const params = new URLSearchParams({
      mode: 'bind',
      return_to: '/user/profile',
    })
    window.location.href = `/api/v1/auth/oauth/${provider}/start?${params.toString()}`
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={t('profile.title')}
        description={t('profile.description')}
      />

      <ProfileTabPills tabs={tabs} value={tab} onChange={setTab} />

      {tab === 'basic' && (
        <div className="space-y-3">
          {/* ── Profile header: larger hero-style row with clickable avatar.
               Both the avatar and the explicit button target the same hidden
               file input, so either entry point opens the picker. */}
          <section className="rounded-xl border bg-card p-5 sm:p-7">
            <div className="flex flex-col gap-5 sm:flex-row sm:items-center">
              <div className="flex min-w-0 items-center gap-5">
                <Label
                  htmlFor="avatar-file"
                  aria-label={t('profile.changeAvatar')}
                  className={cn(
                    'group/avatar relative shrink-0 cursor-pointer rounded-full ring-2 ring-background transition-[box-shadow] focus-within:ring-ring focus-within:ring-offset-2',
                    avatarUploadMutation.isPending && 'pointer-events-none'
                  )}
                >
                  <Avatar className="size-20 transition-opacity group-hover/avatar:opacity-80 sm:size-24">
                    <AvatarImage
                      src={profileForm.avatarUrl || user.avatar_url}
                      alt={displayName}
                      className="object-cover"
                    />
                    <AvatarFallback className="text-3xl font-semibold">
                      {displayName.charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  {/* Hover overlay with camera icon — absent at rest so the
                      avatar reads cleanly, appears on hover/focus for the
                      upload affordance. */}
                  <div
                    className={cn(
                      'pointer-events-none absolute inset-0 flex items-center justify-center rounded-full bg-black/45 text-white transition-opacity',
                      avatarUploadMutation.isPending
                        ? 'opacity-100'
                        : 'opacity-0 group-hover/avatar:opacity-100'
                    )}
                  >
                    {avatarUploadMutation.isPending ? (
                      <Loader2 className="size-6 animate-spin" />
                    ) : (
                      <Camera className="size-6" strokeWidth={1.8} />
                    )}
                  </div>
                </Label>
                <div className="min-w-0 flex-1">
                  <div className="flex min-w-0 flex-wrap items-baseline gap-x-2.5 gap-y-0.5">
                    <h2 className="max-w-full truncate text-2xl font-semibold leading-tight tracking-tight">
                      {displayName}
                    </h2>
                    <span className="truncate text-sm text-muted-foreground">
                      @{user.username}
                    </span>
                  </div>
                  <div className="mt-3 flex flex-wrap items-center gap-1.5">
                    <Badge
                      variant="outline"
                      className="gap-1 rounded-md border-sky-500/30 bg-sky-500/10 px-2 py-0.5 text-[12px] font-medium text-sky-700 dark:text-sky-300"
                    >
                      {roleLabel}
                    </Badge>
                    {/* User-ID badge doubles as the copy control — clicking
                        it hits the clipboard handler with a toast. */}
                    <button
                      type="button"
                      onClick={copyUserId}
                      title={t('profile.metaUserIdCopy')}
                      className="inline-flex rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                    >
                      <Badge
                        variant="secondary"
                        className="rounded-md px-2 py-0.5 font-mono text-[12px] font-medium transition-colors hover:bg-secondary/80"
                      >
                        ID {user.user_id.slice(0, 8)}
                      </Badge>
                    </button>
                    {joinedDays !== null && (
                      <Badge
                        variant="secondary"
                        className="rounded-md px-2 py-0.5 text-[12px] font-medium"
                      >
                        {t('profile.joinedBadge').replace(
                          '{n}',
                          String(joinedDays)
                        )}
                      </Badge>
                    )}
                  </div>
                </div>
              </div>
              <div className="shrink-0 sm:ml-auto">
                <Label
                  htmlFor="avatar-file"
                  className={cn(
                    'inline-flex h-9 cursor-pointer items-center justify-center gap-2 whitespace-nowrap rounded-md border bg-background px-3 text-sm font-medium shadow-xs transition-colors hover:bg-accent hover:text-accent-foreground',
                    avatarUploadMutation.isPending &&
                      'pointer-events-none opacity-60'
                  )}
                >
                  {avatarUploadMutation.isPending ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <Camera className="size-4" />
                  )}
                  {t('profile.changeAvatar')}
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
            </div>
          </section>

          {/* ── Two-col grid: storage/usage (narrow) · personal info form (wide) */}
          <div className="grid gap-3 lg:grid-cols-[1fr_1.4fr]">
            {/* Left: Storage & Usage */}
            <section className="flex flex-col rounded-xl border bg-card p-4">
              <div className="flex items-start justify-between gap-2">
                <h3 className="text-sm font-semibold tracking-tight">
                  {t('profile.storageUsageTitle')}
                </h3>
                <span className="text-xs text-muted-foreground">
                  {storageUnlimited
                    ? t('profile.storageUnlimited')
                    : t('profile.storageUsedFmt')
                        .replace('{used}', formatSize(storageUsed))
                        .replace('{total}', formatSize(storageLimit))}
                </span>
              </div>

              {/* Headline number — the integer value is emphasized; the
                  unit and file count trail in muted text. */}
              <div className="mt-3 flex items-baseline gap-1.5">
                {(() => {
                  const total = statsData?.total_size ?? 0
                  const [value, unit] = formatSize(total).split(' ')
                  return (
                    <>
                      <span className="text-[28px] font-bold leading-none tabular-nums">
                        {value}
                      </span>
                      <span className="text-xs text-muted-foreground">
                        {unit ?? ''}
                        {' · '}
                        {t('profile.usageFilesUnit').replace(
                          '{n}',
                          String(statsData?.total_files ?? 0)
                        )}
                      </span>
                    </>
                  )
                })()}
              </div>

              {/* Stacked bar — renders only non-zero segments so a single
                  dominant type doesn't pick up a phantom stripe. */}
              {(() => {
                const total = statsData?.total_size ?? 0
                return (
                  <div className="mt-3 flex h-2 w-full overflow-hidden rounded-full bg-muted">
                    {total > 0 &&
                      usageBreakdown
                        .filter((item) => item.size > 0)
                        .map((item) => (
                          <div
                            key={item.key}
                            className={item.barColor}
                            style={{
                              width: `${(item.size / total) * 100}%`,
                            }}
                          />
                        ))}
                  </div>
                )
              })()}

              {/* Legend list — colored square + label on left, size + count
                  on right. Zero rows are dimmed for visual hierarchy. */}
              <ul className="mt-4 space-y-2 text-[13px]">
                {usageBreakdown.map((item) => {
                  const empty = item.size === 0 && item.count === 0
                  return (
                    <li
                      key={item.key}
                      className="flex items-center justify-between gap-2"
                    >
                      <span className="flex items-center gap-2 min-w-0">
                        <span
                          className={cn(
                            'size-2.5 shrink-0 rounded-sm',
                            empty ? 'bg-muted-foreground/30' : item.barColor
                          )}
                        />
                        <span
                          className={cn(
                            'truncate',
                            empty && 'text-muted-foreground'
                          )}
                        >
                          {item.label}
                        </span>
                      </span>
                      <span
                        className={cn(
                          'shrink-0 tabular-nums',
                          empty ? 'text-muted-foreground' : 'text-foreground'
                        )}
                      >
                        {empty
                          ? '0'
                          : `${formatSize(item.size)} · ${item.count}`}
                      </span>
                    </li>
                  )
                })}
              </ul>

              <div className="mt-auto border-t pt-3">
                <div className="flex items-center justify-between text-[13px]">
                  <span className="text-muted-foreground">
                    {t('profile.linkedAccountsRow')}
                  </span>
                  <span className="font-medium tabular-nums">
                    {linkedCount} / {Math.max(identities.length, 3)}
                  </span>
                </div>
              </div>
            </section>

            {/* Right: Personal info form */}
            <section className="rounded-xl border bg-card p-4">
              <header className="mb-4">
                <h3 className="text-sm font-semibold tracking-tight">
                  {t('profile.personalInfoTitle')}
                </h3>
                <p className="mt-0.5 text-xs text-muted-foreground">
                  {t('profile.basicInfoCardDesc')}
                </p>
              </header>

              <form onSubmit={handleProfileSubmit} className="space-y-3">
                {/* Nickname — only truly editable field. */}
                <div className="grid gap-1.5">
                  <Label
                    htmlFor="nickname"
                    className="text-xs text-muted-foreground"
                  >
                    {t('profile.nickname')}
                  </Label>
                  <Input
                    id="nickname"
                    value={profileForm.nickname}
                    onChange={(e) =>
                      setProfileForm((p) => ({
                        ...p,
                        nickname: e.target.value,
                      }))
                    }
                    maxLength={32}
                    placeholder={t('profile.nicknamePlaceholder')}
                  />
                </div>

                {/* Email — read-only; change flow lives on the Security tab.
                    Styled like a normal input (no muted bg) to match the
                    mockup, with a subtle hint pointing users to security. */}
                <div className="grid gap-1.5">
                  <Label
                    htmlFor="email"
                    className="text-xs text-muted-foreground"
                  >
                    {t('auth.email')}
                  </Label>
                  <Input
                    id="email"
                    value={user.email ?? ''}
                    readOnly
                    aria-readonly="true"
                  />
                  <p className="text-[11px] text-muted-foreground">
                    {t('profile.emailChangeHint')}
                  </p>
                </div>

                {/* Locked pair: username + registered-at, shown side-by-side
                    with muted styling so they clearly read as immutable. */}
                <div className="grid gap-3 sm:grid-cols-2">
                  <div className="grid gap-1.5">
                    <Label
                      htmlFor="username"
                      className="text-xs text-muted-foreground"
                    >
                      {t('profile.usernameLockedLabel')}
                    </Label>
                    <Input
                      id="username"
                      value={user.username}
                      readOnly
                      disabled
                      className="bg-muted/40"
                    />
                  </div>
                  <div className="grid gap-1.5">
                    <Label
                      htmlFor="registered"
                      className="text-xs text-muted-foreground"
                    >
                      {t('profile.registered')}
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

                <div className="mt-2 flex items-center justify-end gap-2 border-t pt-3">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={resetProfile}
                  >
                    {t('profile.cancel')}
                  </Button>
                  <Button type="submit" disabled={profileMutation.isPending}>
                    {profileMutation.isPending && (
                      <Loader2 className="size-4 animate-spin" />
                    )}
                    {t('profile.save')}
                  </Button>
                </div>
              </form>
            </section>
          </div>
        </div>
      )}

      {tab === 'security' && (
        <section className="rounded-xl border bg-card p-6 sm:p-7">
          <header className="mb-5 flex items-start justify-between gap-4">
            <div>
              <h3 className="text-base font-semibold tracking-tight">
                {t('profile.security')}
              </h3>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('profile.securityDesc')}
              </p>
            </div>
            <Badge
              variant="outline"
              className="shrink-0 gap-1 border-emerald-500/30 bg-emerald-500/5 text-emerald-600 dark:text-emerald-400"
            >
              <ShieldCheck className="size-3" />
              {t('profile.securityHealthy')}
            </Badge>
          </header>

          <div className="divide-y">
            <SecurityRow
              icon={Key}
              title={
                hasLocalPassword
                  ? t('profile.passwordRow')
                  : t('profile.passwordSetup')
              }
              badge={
                !hasLocalPassword ? (
                  <Badge variant="outline" className="text-[10px]">
                    {t('profile.passwordUnset')}
                  </Badge>
                ) : undefined
              }
              description={
                hasLocalPassword
                  ? t('profile.passwordRowHintUnknown')
                  : t('profile.passwordSetupDesc')
              }
              actionLabel={
                hasLocalPassword
                  ? t('profile.passwordChange')
                  : t('profile.passwordSetup')
              }
              onAction={() => setPasswordDialogOpen(true)}
            />
            <SecurityRow
              icon={ShieldCheck}
              title={t('profile.twoFactor')}
              badge={
                <Badge variant="secondary" className="text-[10px]">
                  {t('profile.twoFactorOff')}
                </Badge>
              }
              description={t('profile.twoFactorDesc')}
              actionLabel={t('profile.enable')}
              onAction={openSecuritySettings}
            />
            <SecurityRow
              icon={Mail}
              title={t('profile.backupEmail')}
              badge={
                user.email ? (
                  <Badge variant="secondary" className="text-[10px]">
                    {t('profile.backupEmailVerified')}
                  </Badge>
                ) : (
                  <Badge variant="outline" className="text-[10px]">
                    {t('profile.backupEmailNone')}
                  </Badge>
                )
              }
              description={user.email ?? t('profile.backupEmailNone')}
              actionLabel={
                user.email
                  ? t('profile.backupEmailReplace')
                  : t('profile.backupEmailAdd')
              }
              onAction={openEmailDialog}
            />
            <SecurityRow
              icon={Activity}
              title={t('profile.loginActivity')}
              description={t('profile.loginActivityDesc')
                .replace('{n}', '1')
                .replace('{devices}', 'Web')}
              actionLabel={t('profile.viewAction')}
              onAction={() => setSessionDialogOpen(true)}
              actionIcon
            />
          </div>
        </section>
      )}

      {tab === 'identities' && (
        <section className="rounded-xl border bg-card p-6 sm:p-7">
          <header className="mb-5 flex items-start justify-between gap-3">
            <div>
              <h3 className="text-base font-semibold tracking-tight">
                {t('profile.socialAccounts')}
              </h3>
              <p className="mt-1 text-sm text-muted-foreground">
                {t('profile.socialAccountsDesc')}
              </p>
            </div>
            <Badge variant="outline" className="text-[10px]">
              {identities.filter((item) => item.bound).length}
              {' / '}
              {identities.length}
            </Badge>
          </header>

          <div className="space-y-3">
            {identities.map((identity) => (
              <div
                key={identity.key}
                className="flex flex-col gap-3 rounded-lg border bg-background/80 p-3 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <SocialProviderLogo
                    provider={identity.icon_key}
                    size={36}
                    rounded="rounded-lg"
                  />
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-sm font-medium">
                        {identity.label}
                      </span>
                      <Badge
                        variant={identity.bound ? 'secondary' : 'outline'}
                        className="text-[10px]"
                      >
                        {identity.bound
                          ? t('profile.identityBound')
                          : t('profile.identityUnbound')}
                      </Badge>
                    </div>
                    <p className="truncate text-xs text-muted-foreground">
                      {identity.bound
                        ? identity.email ||
                          identity.display_name ||
                          t('profile.identityBound')
                        : t('profile.identityUnboundDesc')}
                    </p>
                  </div>
                </div>

                {identity.bound ? (
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={
                      unlinkMutation.isPending &&
                      pendingProvider === identity.key
                    }
                    onClick={() => unlinkMutation.mutate(identity.key)}
                  >
                    {unlinkMutation.isPending &&
                    pendingProvider === identity.key ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      <Link2 className="size-4" />
                    )}
                    {t('profile.identityUnlink')}
                  </Button>
                ) : (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => startBindFlow(identity.key)}
                  >
                    <Link2 className="size-4" />
                    {t('profile.identityLink')}
                  </Button>
                )}
              </div>
            ))}
          </div>
        </section>
      )}

      {tab === 'preferences' && (
        <section className="rounded-xl border bg-card p-6 sm:p-7">
          <header className="mb-5">
            <h3 className="text-base font-semibold tracking-tight">
              {t('profile.preferences')}
            </h3>
          </header>

          <div className="divide-y">
            {/* Language */}
            <PrefRow
              title={t('profile.language')}
              description={t('profile.languageDesc')}
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
              title={t('settings.theme')}
              description={t('settings.themeDesc')}
            >
              <SegmentedControl
                options={[
                  {
                    value: 'light',
                    label: t('settings.themeLight'),
                    icon: Sun,
                  },
                  { value: 'dark', label: t('settings.themeDark'), icon: Moon },
                  {
                    value: 'system',
                    label: t('settings.themeSystem'),
                    icon: Monitor,
                  },
                ]}
                value={theme}
                onChange={(v) => setTheme(v as 'light' | 'dark' | 'system')}
              />
            </PrefRow>

            {/* Default view */}
            <PrefRow
              title={t('profile.defaultView')}
              description={t('profile.defaultViewDesc')}
            >
              <SegmentedControl
                options={[
                  {
                    value: 'grid',
                    label: t('profile.defaultViewGrid'),
                    icon: LayoutGrid,
                  },
                  {
                    value: 'list',
                    label: t('profile.defaultViewList'),
                    icon: ListIcon,
                  },
                ]}
                value={defaultView}
                onChange={(v) => setDefaultView(v as 'grid' | 'list')}
              />
            </PrefRow>

            {/* Email notifications */}
            <PrefRow
              title={t('profile.emailNotifications')}
              description={t('profile.emailNotificationsDesc')}
            >
              <Switch checked={emailNotify} onCheckedChange={setEmailNotify} />
            </PrefRow>
          </div>
        </section>
      )}

      {/* —— Password dialog —— */}
      <Dialog open={passwordDialogOpen} onOpenChange={setPasswordDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>
              {hasLocalPassword
                ? t('profile.changePassword')
                : t('profile.passwordSetup')}
            </DialogTitle>
            <DialogDescription>
              {hasLocalPassword
                ? t('profile.changePasswordDesc')
                : t('profile.passwordSetupDialogDesc')}
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handlePasswordSubmit} className="grid gap-4">
            {hasLocalPassword && (
              <div className="grid gap-2">
                <Label htmlFor="currentPassword">
                  {t('profile.currentPassword')}
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
            )}
            <div className="grid gap-2">
              <Label htmlFor="newPassword">{t('profile.newPassword')}</Label>
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
                placeholder={t('auth.passwordHint')}
                minLength={6}
                maxLength={64}
                required
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="confirmPassword">
                {t('auth.confirmPassword')}
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
                placeholder={t('auth.confirmPasswordPlaceholder')}
                required
              />
            </div>
            <p className="flex items-start gap-2 rounded-md border bg-muted/30 p-2.5 text-[11px] text-muted-foreground">
              <ShieldCheck className="mt-0.5 size-3 shrink-0" />
              <span>
                {hasLocalPassword
                  ? t('profile.passwordTip')
                  : t('profile.passwordSetupTip')}
              </span>
            </p>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setPasswordDialogOpen(false)}
              >
                {t('profile.cancel')}
              </Button>
              <Button
                type="submit"
                disabled={
                  passwordMutation.isPending || setPasswordMutation.isPending
                }
              >
                {(passwordMutation.isPending ||
                  setPasswordMutation.isPending) && (
                  <Loader2 className="size-4 animate-spin" />
                )}
                {hasLocalPassword
                  ? t('profile.savePassword')
                  : t('profile.savePasswordSetup')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* —— Email change dialog —— */}
      <Dialog
        open={emailDialogOpen}
        onOpenChange={(open) => {
          setEmailDialogOpen(open)
          // When the dialog closes (cancel, backdrop click, Esc) fully
          // reset state so reopening always starts from the request step.
          if (!open) resetEmailDialog()
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t('profile.emailChangeTitle')}</DialogTitle>
            <DialogDescription>
              {emailStep === 'request'
                ? t('profile.emailChangeDesc')
                : t('profile.emailCodeDesc').replace(
                    '{email}',
                    pendingNewEmail
                  )}
            </DialogDescription>
          </DialogHeader>

          <form
            onSubmit={
              emailStep === 'request' ? handleEmailRequest : handleEmailConfirm
            }
            className="grid gap-4"
          >
            {emailStep === 'request' ? (
              <>
                <div className="grid gap-1.5">
                  <Label
                    htmlFor="currentEmail"
                    className="text-xs text-muted-foreground"
                  >
                    {t('profile.emailCurrent')}
                  </Label>
                  <Input
                    id="currentEmail"
                    value={user.email ?? ''}
                    readOnly
                    disabled
                    className="bg-muted/40"
                  />
                </div>
                <div className="grid gap-1.5">
                  <Label
                    htmlFor="newEmail"
                    className="text-xs text-muted-foreground"
                  >
                    {t('profile.emailNew')}
                  </Label>
                  <Input
                    id="newEmail"
                    type="email"
                    autoComplete="email"
                    autoCapitalize="none"
                    autoCorrect="off"
                    value={pendingNewEmail}
                    onChange={(e) => setPendingNewEmail(e.target.value)}
                    placeholder={t('profile.emailNewPlaceholder')}
                    required
                  />
                </div>
                <p className="flex items-start gap-2 rounded-md border bg-muted/30 p-2.5 text-[11px] text-muted-foreground">
                  <ShieldCheck className="mt-0.5 size-3 shrink-0" />
                  <span>{t('profile.emailChangeTip')}</span>
                </p>
              </>
            ) : (
              <>
                <div className="grid gap-1.5">
                  <Label
                    htmlFor="emailCode"
                    className="text-xs text-muted-foreground"
                  >
                    {t('profile.emailCode')}
                  </Label>
                  <Input
                    id="emailCode"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    pattern="\d*"
                    maxLength={6}
                    value={emailCode}
                    onChange={(e) =>
                      setEmailCode(
                        e.target.value.replace(/\D/g, '').slice(0, 6)
                      )
                    }
                    placeholder="——————"
                    required
                    className="text-center font-mono text-base tracking-[0.4em]"
                  />
                </div>
                <div className="flex items-center justify-between gap-3 text-[11px] text-muted-foreground">
                  <span className="truncate">
                    {t('profile.emailCodeSent').replace(
                      '{email}',
                      pendingNewEmail
                    )}
                  </span>
                  <button
                    type="button"
                    disabled={
                      resendSecondsLeft > 0 || emailRequestMutation.isPending
                    }
                    onClick={() => emailRequestMutation.mutate(pendingNewEmail)}
                    className={cn(
                      'shrink-0 text-[11px] font-medium underline-offset-2',
                      resendSecondsLeft > 0 || emailRequestMutation.isPending
                        ? 'cursor-not-allowed text-muted-foreground'
                        : 'text-foreground hover:underline'
                    )}
                  >
                    {emailRequestMutation.isPending ? (
                      <Loader2 className="size-3 animate-spin" />
                    ) : resendSecondsLeft > 0 ? (
                      t('profile.emailResendIn').replace(
                        '{s}',
                        String(resendSecondsLeft)
                      )
                    ) : (
                      t('profile.emailResend')
                    )}
                  </button>
                </div>
              </>
            )}

            <DialogFooter>
              {emailStep === 'verify' ? (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setEmailStep('request')
                    setEmailCode('')
                  }}
                >
                  {t('profile.emailBack')}
                </Button>
              ) : (
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setEmailDialogOpen(false)}
                >
                  {t('profile.cancel')}
                </Button>
              )}
              <Button
                type="submit"
                disabled={
                  emailStep === 'request'
                    ? emailRequestMutation.isPending
                    : emailConfirmMutation.isPending
                }
              >
                {(emailStep === 'request'
                  ? emailRequestMutation.isPending
                  : emailConfirmMutation.isPending) && (
                  <Loader2 className="size-4 animate-spin" />
                )}
                {emailStep === 'request'
                  ? t('profile.emailSendCode')
                  : t('profile.emailConfirm')}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={sessionDialogOpen} onOpenChange={setSessionDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t('profile.loginActivity')}</DialogTitle>
            <DialogDescription>
              {t('profile.loginActivityDesc')
                .replace('{n}', '1')
                .replace('{devices}', 'Web')}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 rounded-lg border bg-muted/30 p-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">
                {t('auth.username')}
              </span>
              <span className="font-medium">{user.username}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t('common.type')}</span>
              <span className="font-medium">Web</span>
            </div>
            <div className="flex items-center justify-between gap-3">
              <span className="text-muted-foreground">User Agent</span>
              <span className="max-w-[220px] truncate text-right font-medium">
                {navigator.userAgent}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">{t('common.date')}</span>
              <span className="font-medium">{new Date().toLocaleString()}</span>
            </div>
          </div>

          <DialogFooter>
            <Button onClick={() => setSessionDialogOpen(false)}>
              {t('files.close')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

/* ─────────────────────────────────────────────────────────────
 * Sub-components
 * ────────────────────────────────────────────────────────────*/
function ProfileTabPills({
  tabs,
  value,
  onChange,
}: {
  tabs: { value: ProfileTab; label: string; icon: React.ElementType }[]
  value: ProfileTab
  onChange: (value: ProfileTab) => void
}) {
  return (
    <div className="overflow-x-auto pb-1">
      <div className="inline-flex min-w-max items-center rounded-xl border bg-muted/40 p-1 text-muted-foreground">
        {tabs.map((tab) => {
          const active = value === tab.value
          return (
            <button
              key={tab.value}
              type="button"
              onClick={() => onChange(tab.value)}
              className={cn(
                'inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-lg px-3 py-2 text-xs font-medium transition-all',
                active
                  ? 'bg-background text-foreground shadow-sm'
                  : 'hover:text-foreground'
              )}
            >
              <tab.icon className="size-3.5" strokeWidth={1.8} />
              {tab.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}

function SecurityRow({
  icon: Icon,
  title,
  badge,
  description,
  actionLabel,
  onAction,
  actionIcon,
}: {
  icon: React.ElementType
  title: string
  badge?: React.ReactNode
  description: string
  actionLabel: string
  onAction: () => void
  actionIcon?: boolean
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
  )
}

function PrefRow({
  title,
  description,
  children,
}: {
  title: string
  description: string
  children: React.ReactNode
}) {
  return (
    <div className="flex flex-col gap-3 py-4 first:pt-0 last:pb-0 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
      <div className="min-w-0 sm:flex-1">
        <p className="break-words text-sm font-medium">{title}</p>
        {description && (
          <p className="mt-0.5 break-words text-xs text-muted-foreground">
            {description}
          </p>
        )}
      </div>
      <div className="w-full sm:w-auto sm:shrink-0">{children}</div>
    </div>
  )
}

interface SegOption {
  value: string
  label: string
  icon?: React.ElementType
}

function SegmentedControl({
  options,
  value,
  onChange,
}: {
  options: SegOption[]
  value: string
  onChange: (value: string) => void
}) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [indicator, setIndicator] = useState<{
    left: number
    width: number
    ready: boolean
  }>({ left: 0, width: 0, ready: false })

  // Measure the active button and position the pill accordingly.
  // useLayoutEffect avoids a flicker on first paint when switching.
  useLayoutEffect(() => {
    const container = containerRef.current
    if (!container) return
    const el = container.querySelector<HTMLElement>(
      `[data-value="${CSS.escape(value)}"]`
    )
    if (!el) return
    setIndicator({
      left: el.offsetLeft,
      width: el.offsetWidth,
      ready: true,
    })
  }, [value, options.length])

  // Recompute on window resize so the pill tracks layout changes.
  useEffect(() => {
    const handle = () => {
      const container = containerRef.current
      if (!container) return
      const el = container.querySelector<HTMLElement>(
        `[data-value="${CSS.escape(value)}"]`
      )
      if (!el) return
      setIndicator((prev) => ({
        left: el.offsetLeft,
        width: el.offsetWidth,
        ready: prev.ready,
      }))
    }
    window.addEventListener('resize', handle)
    return () => window.removeEventListener('resize', handle)
  }, [value])

  return (
    <div className="w-full overflow-x-auto pb-1 sm:w-auto sm:pb-0">
      <div
        ref={containerRef}
        className="relative inline-flex min-w-max rounded-md border bg-muted/30 p-0.5"
      >
        {/* Sliding active pill */}
        <span
          aria-hidden
          className={cn(
            'pointer-events-none absolute top-0.5 bottom-0.5 rounded-sm bg-background shadow-sm',
            indicator.ready
              ? 'transition-[left,width] duration-300 ease-out'
              : 'opacity-0'
          )}
          style={{ left: indicator.left, width: indicator.width }}
        />
        {options.map((opt) => {
          const active = opt.value === value
          const Icon = opt.icon
          return (
            <button
              key={opt.value}
              type="button"
              data-value={opt.value}
              onClick={() => onChange(opt.value)}
              className={cn(
                'relative z-10 flex items-center gap-1.5 whitespace-nowrap rounded-sm px-3 py-1 text-xs font-medium transition-colors',
                active
                  ? 'text-foreground'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              {Icon && <Icon className="size-3.5" />}
              {opt.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}
